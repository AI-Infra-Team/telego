package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"telego/util"
	"telego/util/yamlext"

	"github.com/barweiss/go-tuple"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type ModJobConfigExporterStruct struct{}

var ModJobConfigExporter ModJobConfigExporterStruct

func (ModJobConfigExporterStruct) JobCmdName() string {
	return "config-exporter"
}

type EnvExporterDistConfig struct {
	DistPrjName        string
	DistNode           string
	DistInstanceIdx    int
	DistConfigFilePath string
}

func (d *EnvExporterDistConfig) IsValid() bool {
	return d.DistPrjName != "" && d.DistNode != "" && d.DistInstanceIdx >= 0 && d.DistConfigFilePath != ""
}

func (ModJobConfigExporterStruct) ParseJob(Cmd *cobra.Command) *cobra.Command {
	secrets := []string{}
	publics := []string{}
	saveas := ""
	distconfig := EnvExporterDistConfig{}

	// 绑定命令行标志到结构体字段

	Cmd.Flags().StringArrayVar(&secrets, "secret", []string{}, "")
	Cmd.Flags().StringArrayVar(&publics, "public", []string{}, "")
	Cmd.Flags().StringVar(&saveas, "saveas", "", "导出shell文件在bash中使用source执行，以加载环境变量")

	Cmd.Flags().StringVar(&distconfig.DistPrjName, "dist", "", "分布式项目名称")
	Cmd.Flags().StringVar(&distconfig.DistNode, "dist-node", "", "分布式项目部署节点名称")
	Cmd.Flags().IntVar(&distconfig.DistInstanceIdx, "dist-instance-idx", -1, "分布式项目部署实例索引")
	Cmd.Flags().StringVar(&distconfig.DistConfigFilePath, "dist-config", "", "分布式项目部署配置文件路径")
	// Cmd.MarkFlagRequired("saveas")

	Cmd.Run = func(_ *cobra.Command, _ []string) {

		ModJobConfigExporter.DoJob(secrets, publics, saveas, distconfig)
	}

	return Cmd
}

type ConfigExporterPipeline struct {
	Key        string
	YamlMapGet []string // expr: {key}|yamlmapget({subkey1},{subkey2})|as({keyname})
	Value      string
	As         string
}

// 递归处理YAML MapGet获取最终的字符串值
func (pipeline *ConfigExporterPipeline) getFinalValue() (string, error) {
	// 如果没有YamlMapGet字段，直接返回原始值
	if len(pipeline.YamlMapGet) == 0 {
		return pipeline.Value, nil
	}

	var parsedMap_ map[string]interface{}
	err := yamlext.UnmarshalAndValidate([]byte(pipeline.Value), &parsedMap_)
	if err != nil {
		return "", fmt.Errorf("failed to parse value as YAML: %v", err)
	}
	// var parsedMap parseMapValueMap
	// err = mapstructure.Decode(parsedMap_, &parsedMap)
	// if err != nil {
	// 	return "", fmt.Errorf("failed to parse value by mapstructure: %v", err)
	// }

	// 递归地获取子键的最终值
	var currentValue = parsedMap_
	var resultValue string
	for i, subkey := range pipeline.YamlMapGet {
		// 如果当前值是map类型，则查找subkey
		currentValue_, ok := currentValue[subkey]
		if !ok {
			return "", fmt.Errorf("key '%s' not found in map", subkey)
		}
		if i == len(pipeline.YamlMapGet)-1 {
			switch v := currentValue_.(type) {
			case int:
				resultValue = strconv.Itoa(v)
			case string:
				resultValue = v
			default:
				// resultValue = fmt.Sprintf("%v", v)
				return "", fmt.Errorf("unsupported type: %T with value: %v", v, v)
			}
			// err := mapstructure.Decode(currentValue_, &resultValue)
			// if err != nil {
			// 	return "", fmt.Errorf("failed to decode map: %v", err)
			// }
		} else {

			// err := mapstructure.Decode(currentValue_, &currentValue)
			// if err != nil {
			// 	return "", fmt.Errorf("failed to decode map: %v", err)
			// }
		}
	}
	// else {
	// 	return "", fmt.Errorf("current value is not a map, unable to extract key '%s'", subkey)
	// }

	// // 将最终值转换为string并返回
	// switch v := currentValue.(type) {
	// case parseMapValueStr:
	// 	pipeline.Value = string(v)
	// default:
	// 	pipeline.Value = fmt.Sprintf("%v", v)
	// }
	pipeline.Value = resultValue
	return resultValue, nil
}

func (ModJobConfigExporterStruct) DoJob(secrets []string,
	publics []string,
	saveas string,
	distcmd EnvExporterDistConfig) {
	util.PrintStep("config-exporter", fmt.Sprintln(secrets, publics, saveas, distcmd))
	if !filepath.IsAbs(saveas) {
		saveas = filepath.Join(util.GetEntryDir(), saveas)
	}
	if distcmd.IsValid() && !filepath.IsAbs(distcmd.DistConfigFilePath) {
		distcmd.DistConfigFilePath = filepath.Join(util.GetEntryDir(), distcmd.DistConfigFilePath)
	}
	_, err := os.Stat(filepath.Dir(saveas))
	if err != nil {
		err := os.MkdirAll(filepath.Dir(saveas), 0755)
		if err != nil {
			fmt.Println(color.RedString("Error creating directory: %v", err))
			return
		}
	}
	if len(publics)+len(secrets) == 0 && !distcmd.IsValid() {
		fmt.Println(color.YellowString("No public or secret configs specified"))
		return
	}

	pubConfTypes := []tuple.T2[ConfigExporterPipeline, util.PubConfType]{}
	secretConfTypes := []tuple.T2[ConfigExporterPipeline, util.SecretConfType]{}
	if !func() bool {
		// Check if all config is valid
		for _, secret := range secrets {
			// Split the key and validate
			pipeline, err := parseKeyIntoPipeline(secret)
			if err != nil {
				fmt.Println(color.RedString("Invalid secret config key: %s, %v", secret, err))
				return false
			}

			// Check if the secret key is valid using parsed Key
			if util.NewSecretConfType(pipeline.Key) == nil {
				fmt.Println(color.RedString("Invalid secret config with key: %s", pipeline.Key))
				return false
			}

			// Add the secret configuration type and pipeline after validation
			secretConfTypes = append(secretConfTypes, tuple.New2(pipeline, util.NewSecretConfType(pipeline.Key)))
		}
		for _, public := range publics {
			// Split the key and validate
			pipeline, err := parseKeyIntoPipeline(public)
			if err != nil {
				fmt.Println(color.RedString("Invalid public config key: %s, %v", public, err))
				return false
			}

			// Check if the public key is valid using parsed Key
			if util.NewPubConfType(pipeline.Key) == nil {
				fmt.Println(color.RedString("Invalid public config with key: %s", pipeline.Key))
				return false
			}

			// Add the public configuration type and pipeline after validation
			pubConfTypes = append(pubConfTypes, tuple.New2(pipeline, util.NewPubConfType(pipeline.Key)))
		}
		return true
	}() {
		os.Exit(1)
	}

	if !func() bool {
		// Read config kvs
		for i, secretConfType := range secretConfTypes {
			secret, err := util.MainNodeConfReader{}.ReadSecretConf(secretConfType.V2)
			if err != nil {
				fmt.Println(color.RedString("Read secret conf failed: %v", err))
				return false
			}
			secretConfTypes[i].V1.Value = secret
		}
		for i, pubConfType := range pubConfTypes {
			pub, err := util.MainNodeConfReader{}.ReadPubConf(pubConfType.V2)
			if err != nil {
				fmt.Println(color.RedString("Read pub conf failed: %v", err))
				return false
			}
			pubConfTypes[i].V1.Value = pub
		}
		return true
	}() {
		os.Exit(1)
	}

	confKvs := []tuple.T2[string, string]{}
	if !func() bool {
		// secret
		// 遍历每个 pipeline，解析YamlMapGet获取最终的值
		util.PrintStep("config-exporter", fmt.Sprintln("exporting secrets"))
		for _, secretConfType := range secretConfTypes {
			finalValue, err := secretConfType.V1.getFinalValue()
			if err != nil {
				fmt.Println(color.RedString("Failed to get final value for secret '%s': %v", secretConfType.V1.Key, err))
				return false
			}
			// 输出解析后的值
			fmt.Println("Got Secret Key:", secretConfType.V1.Key)
			if secretConfType.V1.As != "" {
				confKvs = append(confKvs, tuple.New2(secretConfType.V1.As, finalValue))
			} else {
				confKvs = append(confKvs, tuple.New2(secretConfType.V1.Key, finalValue))
			}
		}

		// public
		util.PrintStep("config-exporter", fmt.Sprintln("exporting publics"))
		for _, pubConfType := range pubConfTypes {
			finalValue, err := pubConfType.V1.getFinalValue()
			if err != nil {
				fmt.Println(color.RedString("Failed to get final value for public '%s': %v", pubConfType.V1.Key, err))
				return false
			}
			// 输出解析后的值
			fmt.Println("Got Public Key:", pubConfType.V1.Key, " Value:", finalValue)
			if pubConfType.V1.As != "" {
				confKvs = append(confKvs, tuple.New2(pubConfType.V1.As, finalValue))
			} else {
				confKvs = append(confKvs, tuple.New2(pubConfType.V1.Key, finalValue))
			}
		}

		// dist
		if distcmd.IsValid() {
			util.PrintStep("config-exporter", fmt.Sprintln("exporting dist"))

			distyml, err := os.ReadFile(distcmd.DistConfigFilePath)
			if err != nil {
				fmt.Println(color.RedString("Read dist config file failed: %v", err))
				return false
			}
			distconf := DeploymentDistConfYaml{}
			err = yamlext.UnmarshalAndValidate(distyml, &distconf)
			if err != nil {
				fmt.Println(color.RedString("Unmarshal dist config file failed: %v", err))
				return false
			}
			fmt.Println("DeploymentDistConfYaml", distconf)
			// DIST_UNIQUE_ID
			_, contain := distconf.Distribution[distcmd.DistNode]
			if !contain {
				fmt.Println(color.RedString("Dist node not the deployment node: %s", distcmd.DistNode))
				return false
			}
			if distcmd.DistInstanceIdx >= len(distconf.Distribution[distcmd.DistNode]) {
				fmt.Println(color.RedString("Dist instance index out of range: %d, max: %d", distcmd.DistInstanceIdx, len(distconf.Distribution[distcmd.DistNode])))
				return false
			}
			distuid := distconf.Distribution[distcmd.DistNode][distcmd.DistInstanceIdx]
			confKvs = append(confKvs, tuple.New2("DIST_UNIQUE_ID", distuid))

			// DIST_CONF_{UNIQUE_ID}_NODE & DIST_CONF_{UNIQUE_ID}_NODE_IP
			for node, nodesvcs := range distconf.Distribution {
				for _, serviceuid := range nodesvcs {
					confKvs = append(confKvs, tuple.New2(
						fmt.Sprintf("DIST_CONF_%s_NODE", serviceuid),
						node,
					))
					confKvs = append(confKvs, tuple.New2(
						fmt.Sprintf("DIST_CONF_%s_NODE_IP", serviceuid),
						distconf.NodeIps[node],
					))
				}
			}

			// DIST_CONF_{UNIQUE_ID}_xxx
			eachUniqueConf := map[string]map[string]string{}
			for confkey, node := range distconf.Conf {
				if confkey != "global" {
					eachUniqueConf[confkey] = node
				}
			}
			// global conf will override conf not specified in eachUniqueConf
			if gconf, ok := distconf.Conf["global"]; ok {
				for gconfkey, gconfvalue := range gconf {
					for seviceuid, svcconf := range eachUniqueConf {
						if _, ok := svcconf[gconfkey]; !ok {
							eachUniqueConf[seviceuid][gconfkey] = gconfvalue
						}
					}
				}
			}
			for serviceuid, svcconf := range eachUniqueConf {
				for confkey, confvalue := range svcconf {
					confKvs = append(confKvs, tuple.New2(fmt.Sprintf("DIST_CONF_%s_%s", serviceuid, confkey), confvalue))
				}
			}

			// backup, install, recover, entrypoint .sh
			createSh := func(name string, content string) {
				name = filepath.Join(util.GetEntryDir(), name)
				err = os.MkdirAll(filepath.Dir(name), 0755)
				if err != nil {
					fmt.Println(color.RedString("mkdir %s failed: %s", filepath.Dir(name), err))
					return
				}
				err = os.WriteFile(name, []byte(content), 0744)
				if err != nil {
					fmt.Println(color.RedString("write sh file %s failed: %s", name, err))
					return
				}
				fmt.Println(color.GreenString("create sh file %s successfully", name))
			}

			createSh("backup.sh", distconf.StateBackup)
			createSh("install.sh", distconf.Install)
			createSh("restore.sh", distconf.StateRestore)
			createSh("entrypoint.sh", distconf.EntryPoint)
		}

		return true
	}() {
		os.Exit(1)
	}

	// output to export script
	{
		escapeForShell := func(value string) string {
			// 转义特殊字符以保证安全性
			replacer := strings.NewReplacer(
				`"`, `\"`, // 双引号
				`$`, `\$`, // 变量符号
				"`", "\\`", // 反引号
				"\n", `\n`, // 换行符
			)
			return replacer.Replace(value)
		}

		output := ""
		for _, kv := range confKvs {
			key := kv.V1
			value := escapeForShell(kv.V2)
			output += fmt.Sprintf("export %s=\"%s\"\n", key, value)
		}

		err := os.WriteFile(saveas, []byte(output), 0744)
		if err != nil {
			fmt.Println(color.RedString("Failed to write to file %s: %v", saveas, err))
			return
		}

		fmt.Println(color.GreenString("Export script saved successfully to: %s", saveas))
	}
}

func parseKeyIntoPipeline(key string) (ConfigExporterPipeline, error) {
	// Split by '|'
	parts := strings.Split(key, ":")

	mainKey := parts[0]
	var subkeys []string
	var asKey string

	for _, part := range parts {
		if strings.HasPrefix(part, "yamlmapget.") {
			// Extract subkeys between parentheses
			subkeyPart := part[len("yamlmapget."):]
			subkeys = strings.Split(subkeyPart, ".")
		} else if strings.HasPrefix(part, "as.") {
			// Extract 'keyname' for 'as'
			asKey = part[len("as."):]
		}
	}

	// Create the ConfigExporterPipeline
	pipeline := ConfigExporterPipeline{
		Key:        mainKey,
		YamlMapGet: subkeys,
		As:         asKey,
	}

	return pipeline, nil
}
