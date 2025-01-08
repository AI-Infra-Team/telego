package app

import (
	"fmt"
	"os"
	"strings"
	"telego/util"
	"telego/util/yamlext"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type ModJobImgRepoStruct struct{}

var ModJobImgRepo ModJobImgRepoStruct

func (m ModJobImgRepoStruct) JobCmdName() string {
	return "img-repo"
}

func (m ModJobImgRepoStruct) ParseJob(applyCmd *cobra.Command) *cobra.Command {

	applyCmd.Run = func(_ *cobra.Command, _ []string) {
		err := m.startImgRepo()
		if err != nil {
			fmt.Println(color.RedString("start img repo failed: %s", err))
			os.Exit(1)
		}
		fmt.Println(color.GreenString("start img repo success"))
	}

	return applyCmd
}

func (m ModJobImgRepoStruct) NewCmd() []string {
	return []string{"telego", m.JobCmdName()}
}

func (m ModJobImgRepoStruct) RunCmd() error {
	cmd := m.NewCmd()
	_, err := util.ModRunCmd.NewBuilder(cmd[0], cmd[1:]...).BlockRun()
	return err
}

func (m ModJobImgRepoStruct) startImgRepo() error {
	harborVersion := "v2.4.0"
	confYaml, err := util.MainNodeConfReader{}.ReadSecretConf(util.SecretConfTypeImgRepo{})
	if err != nil {
		fmt.Println(color.YellowString("Failed to read image uploader url: %v"))

		return fmt.Errorf("failed to read image uploader url: %v\ncreate /teledeploy_secret/config/%s at main node like:\n%s",
			err, util.SecretConfTypeImgRepo{}.SecretConfPath(), util.SecretConfTypeImgRepo{}.Template())
	}

	config := util.ContainerRegistryConf{}
	err = yamlext.UnmarshalAndValidate([]byte(confYaml), &config)
	if err != nil {
		return err
	}

	err = os.MkdirAll("/teledeploy_secret/harbor", 0755)
	if err != nil {
		return err
	}

	_, err = os.Stat("/teledeploy_secret/harbor/harbor-offline-installer.tgz")
	if err != nil {
		err = util.DownloadFile(
			strings.ReplaceAll("https://github.com/goharbor/harbor/releases/download/${HARBOR_VERSION}/harbor-offline-installer-${HARBOR_VERSION}.tgz",
				"${HARBOR_VERSION}", harborVersion,
			), "/teledeploy_secret/harbor/harbor-offline-installer.tgz")

		if err != nil {
			return err
		}
	}

	err = util.UnzipFile("/teledeploy_secret/harbor/harbor-offline-installer.tgz", "/teledeploy_secret/harbor/pack") // 	// 生成 Dockerfile
	if err != nil {
		return err
	}

	// 生成 Harbor 配置文件
	generateHarborConfig := func() (string, error) {
		// yaml read tmpl
		// "/teledeploy_secret/harbor/pack/harbor/harbor.yml.tmpl"
		tmplPath := "/teledeploy_secret/harbor/pack/harbor/harbor.yml.tmpl"
		_, err := os.Stat(tmplPath)
		if err != nil {
			return "", err
		}
		templateContent, err := os.ReadFile(tmplPath)
		if err != nil {
			return "", err
		}
		// yaml parse as map
		yamlConfig := make(map[string]interface{})
		err = yamlext.UnmarshalAndValidate(templateContent, &yamlConfig)
		if err != nil {
			return "", err
		}
		yamlConfig["hostname"] = util.ImgRepoAddressNoPrefix
		yamlConfig["harbor_admin_password"] = config.Password

		port := "80"
		if strings.Contains(":", util.ImgRepoAddressNoPrefix) {
			port = strings.Split(util.ImgRepoAddressNoPrefix, ":")[1]
		}
		if strings.HasPrefix(util.ImgRepoAddressWithPrefix, "http://") {
			yamlConfig["http"] = map[string]string{"port": port}
			// remove https key
			delete(yamlConfig, "https")
		} else {
			if port == "80" {
				port = "443"
			}
			// 1.generate crt & key at /teledeploy_secret/harbor/
			_, err1 := os.Stat("/teledeploy_secret/harbor/harbor.crt")
			_, err2 := os.Stat("/teledeploy_secret/harbor/harbor.key")
			if err1 != nil || err2 != nil {
				err = util.GenerateCrtKey("/teledeploy_secret/harbor", "harbor")
				if err != nil {
					return "", err
				}
			}
			yamlConfig["http"] = map[string]string{"port": "5000"}
			yamlConfig["https"] = map[string]string{
				"port":        port,
				"certificate": "/teledeploy_secret/harbor/harbor.crt",
				"private_key": "/teledeploy_secret/harbor/harbor.key",
			}
		}

		delete(yamlConfig, "_version")

		// 创建临时文件
		configFilePath := "/teledeploy_secret/harbor/pack/harbor/harbor.yml"
		file, err := os.Create(configFilePath)
		if err != nil {
			return "", err
		}
		defer file.Close()

		// 执行模板，写入配置文件
		// encode to file as yaml
		encoded, err := yaml.Marshal(yamlConfig)
		if err != nil {
			return "", err
		}
		_, err = file.Write(encoded)
		if err != nil {
			return "", err
		}

		return configFilePath, nil
	}
	generateHarborConfig()

	_, err = util.ModRunCmd.NewBuilder("bash", "install.sh").
		WithRoot().
		ShowProgress().
		SetDir("/teledeploy_secret/harbor/pack/harbor").
		BlockRun()
	if err != nil {
		return err
	}

	return nil
	// 	generateDockerfile := func(config HarborConfig) (string, error) {
	// 		// Dockerfile 内容模板
	// 		const DockerfileContent = `
	// FROM alpine:3.18

	// # 安装必要的依赖
	// RUN apk update && apk add --no-cache \
	// 	bash \
	// 	curl \
	// 	git \
	// 	docker \
	// 	docker-compose \
	// 	ca-certificates \
	// 	&& rm -rf /var/cache/apk/*

	// # 设置 Harbor 版本
	// ENV HARBOR_VERSION={{.HarborVersion}}

	// # 下载并解压 Harbor 安装包
	// RUN curl -L https://github.com/goharbor/harbor/releases/download/${HARBOR_VERSION}/harbor-offline-installer-${HARBOR_VERSION}.tgz -o /tmp/harbor-offline-installer.tgz \
	// 	&& tar -zxvf /tmp/harbor-offline-installer.tgz -C /opt/ \
	// 	&& rm -f /tmp/harbor-offline-installer.tgz

	// WORKDIR /opt/harbor

	// RUN bash install.sh

	// # 复制 Harbor 配置文件
	// COPY harbor.cfg /opt/harbor/harbor.cfg

	// # 暴露 Harbor 默认的端口
	// EXPOSE 80 443

	// # 启动 Harbor
	// CMD ["/opt/harbor/start_harbor.sh"]
	// `
	// 		// 使用文本模板生成 Dockerfile
	// 		tmpl, err := template.New("dockerfile").Parse(DockerfileContent)
	// 		if err != nil {
	// 			return "", err
	// 		}

	// 		// 创建临时文件
	// 		dockerfilePath := "/tmp/Dockerfile"
	// 		file, err := os.Create(dockerfilePath)
	// 		if err != nil {
	// 			return "", err
	// 		}
	// 		defer file.Close()

	// 		// 执行模板，写入 Dockerfile
	// 		err = tmpl.Execute(file, map[string]string{
	// 			"HarborVersion": config.HarborVersion,
	// 		})
	// 		if err != nil {
	// 			return "", err
	// 		}

	// 		return dockerfilePath, nil
	// 	}

	// 	// 生成启动脚本
	// 	generateStartScript := func() (string, error) {
	// 		scriptPath := "/opt/harbor/start_harbor.sh"
	// 		file, err := os.Create(scriptPath)
	// 		if err != nil {
	// 			return "", err
	// 		}
	// 		defer file.Close()

	// 		// 写入启动脚本内容
	// 		// 启动脚本内容
	// 		const startHarborScript = `#!/bin/bash

	// # 设置 Harbor 配置
	// cd /opt/harbor

	// # 使用 docker-compose 启动 Harbor
	// docker-compose up -d

	// # 输出 Harbor 启动状态
	// docker-compose ps
	// `
	// 		_, err = file.WriteString(startHarborScript)
	// 		if err != nil {
	// 			return "", err
	// 		}

	// 		return scriptPath, nil
	// 	}

	// 	{
	// 		// fetch img repo conf
	// 		confYaml, err := util.MainNodeConfReader{}.ReadPubConf(util.PubConfTypeImgUploaderUrl{})
	// 		if err != nil {
	// 			fmt.Println(color.YellowString("Failed to read image uploader url: %v"))

	// 			return fmt.Errorf("failed to read image uploader url: %v", err)
	// 		}

	// 		// 加载 Harbor 配置文件
	// 		config, err := loadConfig(confYaml)
	// 		if err != nil {
	// 			return err
	// 		}

	// 		// Step 1: 生成 Harbor 配置文件
	// 		_, err = generateHarborConfig(config)
	// 		if err != nil {
	// 			fmt.Printf("Error generating Harbor config: %v\n", err)
	// 			return err
	// 		}

	// 		// Step 2: 生成 Dockerfile
	// 		dockerfilePath, err := generateDockerfile(config)
	// 		if err != nil {
	// 			fmt.Printf("Error generating Dockerfile: %v\n", err)
	// 			return err
	// 		}

	// 		// Step 3: 生成启动脚本
	// 		_, err = generateStartScript()
	// 		if err != nil {
	// 			fmt.Printf("Error generating start script: %v\n", err)
	// 			return err
	// 		}

	// 		// Step 4: 构建 Docker 镜像
	// 		imageTag := "imgrepo-harbor-alpine"
	// 		_, err = util.ModDocker.BuildDockerImage(dockerfilePath, imageTag).BlockRun()
	// 		if err != nil {
	// 			fmt.Printf("Error building Docker image: %v\n", err)
	// 			return err
	// 		}

	// 		// Step 5: 启动 Docker 容器
	// 		ports := []string{}
	// 		if config.UiUrlProtocol == "http" {
	// 			ports = []string{"80:" + config.HarborPort}
	// 		} else {
	// 			ports = []string{"443:" + config.HarborPort}
	// 		}
	// 		compose := util.NewDockerComposeBuilder().AddService(
	// 			util.DockerComposeService{
	// 				Name:  imageTag + "0",
	// 				Image: imageTag,
	// 				Ports: ports,
	// 				Volumes: []string{
	// 					config.DataVolume + ":/opt/harbor_data",
	// 				},
	// 			},
	// 		).Build()
	// 		// create compose file
	// 		// 创建文件
	// 		file, err := os.Create("/teledeploy_secret/harbor/docker-compose.yaml")
	// 		if err != nil {
	// 			return err
	// 		}

	// 		// 写入内容
	// 		_, err = file.WriteString(compose)
	// 		if err != nil {
	// 			file.Close()
	// 			return err
	// 		}
	// 		file.Close()

	//		fmt.Println("Harbor container started successfully!")
	//	}
}
