package app

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"telego/util"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type ModJobImgPrepareStruct struct{}

var ModJobImgPrepare *ModJobImgPrepareStruct = &ModJobImgPrepareStruct{}

func (m *ModJobImgPrepareStruct) JobCmdName() string {
	return "img-prepare"
}

func (m *ModJobImgPrepareStruct) ParseJob(ImgPrepareCmd *cobra.Command) *cobra.Command {
	imagesWithTag := ""
	// 绑定命令行标志到结构体字段
	ImgPrepareCmd.Flags().StringVar(&imagesWithTag, "images", "", "image name with tag")
	ImgPrepareCmd.Run = func(_ *cobra.Command, _ []string) {
		m.PrepareImages(strings.Split(imagesWithTag, ","))
	}
	// err := ImgPrepareCmd.Execute()
	// if err != nil {
	// 	return nil,nil
	// }

	return ImgPrepareCmd
}

// allow fail
// with print
func (m *ModJobImgPrepareStruct) PrepareImages(imagesWithTag []string) error {

	util.PrintStep("ImgPrepare", fmt.Sprintf("preparing images: %v", imagesWithTag))
	// check format
	for _, imageWithTag := range imagesWithTag {
		if !util.FmtCheck.CheckImagePath(imageWithTag) {
			return fmt.Errorf("image %s format error", imageWithTag)
		}
	}
	prepareImage := func(imageName string) {
		curDir := util.CurDir()
		defer os.Chdir(curDir)

		os.MkdirAll(path.Join(ConfigLoad().ProjectDir, "container_image"), 0755)
		os.Chdir(path.Join(ConfigLoad().ProjectDir, "container_image"))

		// 定义可用的平台
		availablePlatforms := []string{
			"linux/amd64",
			"linux/arm64",
		}

		// 获取镜像的名称和标签（如果没有标签默认为 "latest"）
		parts := strings.Split(imageName, ":")
		name := parts[0]
		tag := "latest"
		if len(parts) > 1 {
			tag = parts[1]
		}

		// 设置输出目录
		nameEndSplit := strings.Split(name, "/")
		nameEnd := nameEndSplit[len(nameEndSplit)-1]
		outputDir := fmt.Sprintf("image_%s_%s", nameEnd, tag)

		// 创建输出目录
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Fatalf("Failed to create directory: %v", err)
		}

		// 下载镜像并保存为 tar 文件
		for _, platform := range availablePlatforms {
			outputFile := fmt.Sprintf("%s/%s_%s_%s.tar", outputDir, nameEnd, strings.Split(platform, "/")[1], tag)

			// 如果文件已存在，跳过
			if _, err := os.Stat(outputFile); !os.IsNotExist(err) {
				fmt.Printf("Image already downloaded: %s\n", outputFile)
				continue
			}

			// 拉取 Docker 镜像
			fmt.Printf("Downloading %s:%s for platform %s...\n", name, tag, platform)
			pullCommand := []string{"docker", "pull", fmt.Sprintf("%s:%s", name, tag)}
			if platform != "" {
				pullCommand = append(pullCommand, "--platform", platform)
			}

			if _, err := util.ModRunCmd.ShowProgress(pullCommand[0], pullCommand[1:]...).BlockRun(); err != nil {
				fmt.Printf("Error downloading image, cmd: %v, err: %v\n", pullCommand, err)
				continue
			}

			// 保存镜像为 tar 文件
			exportCommand := []string{"docker", "save", fmt.Sprintf("%s:%s", name, tag), "-o", outputFile}
			if _, err := util.ModRunCmd.ShowProgress(exportCommand[0], exportCommand[1:]...).BlockRun(); err != nil {
				fmt.Printf("Error saving image to file: %v\n", err)
				continue
			}

			fmt.Println(color.GreenString("Image downloaded and saved successfully."))
		}
	}
	for _, imageWithTag := range imagesWithTag {
		util.PrintStep("ImgPrepare", "preparing "+imageWithTag)
		prepareImage(imageWithTag)
	}

	return nil
}
