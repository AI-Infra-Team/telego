package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"telego/util"

	"github.com/barweiss/go-tuple"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type ModJobConfigExporterStruct struct{}

var ModJobConfigExporter ModJobConfigExporterStruct

func (ModJobConfigExporterStruct) JobCmdName() string {
	return "config-exporter"
}

func (ModJobConfigExporterStruct) ParseJob(Cmd *cobra.Command) *cobra.Command {
	secrets := []string{}
	publics := []string{}
	saveas := ""

	// 绑定命令行标志到结构体字段
	Cmd.Flags().StringArrayVar(&secrets, "secret", []string{}, "")
	Cmd.Flags().StringArrayVar(&publics, "public", []string{}, "")
	Cmd.Flags().StringVar(&saveas, "saveas", "", "导出shell文件在bash中使用source执行，以加载环境变量")
	// Cmd.MarkFlagRequired("saveas")

	Cmd.Run = func(_ *cobra.Command, _ []string) {

		ModJobConfigExporter.DoJob(secrets, publics, saveas)
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

	// 将Value解析为map
	var parsedMap map[string]interface{}
	err := yaml.Unmarshal([]byte(pipeline.Value), &parsedMap)
	if err != nil {
		return "", fmt.Errorf("failed to parse value as YAML: %v", err)
	}

	// 递归地获取子键的最终值
	var currentValue interface{} = parsedMap
	for _, subkey := range pipeline.YamlMapGet {
		if subMap, ok := currentValue.(map[string]interface{}); ok {
			// 如果当前值是map类型，则查找subkey
			currentValue, ok = subMap[subkey]
			if !ok {
				return "", fmt.Errorf("key '%s' not found in map", subkey)
			}
		} else {
			return "", fmt.Errorf("current value is not a map, unable to extract key '%s'", subkey)
		}
	}

	// 将最终值转换为string并返回
	pipeline.Value = currentValue.(string)
	return pipeline.Value, nil
}

func (ModJobConfigExporterStruct) DoJob(secrets []string,
	publics []string,
	saveas string) {
	if !filepath.IsAbs(saveas) {
		saveas = filepath.Join(util.GetEntryDir(), saveas)
	}
	_, err := os.Stat(filepath.Dir(saveas))
	if err != nil {
		err := os.MkdirAll(filepath.Dir(saveas), 0755)
		if err != nil {
			fmt.Println("Error creating directory: %v", err)
			return
		}
	}
	if len(publics)+len(secrets) == 0 {
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
		return
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
		return
	}

	confKvs := []tuple.T2[string, string]{}
	if !func() bool {
		// 遍历每个 pipeline，解析YamlMapGet获取最终的值
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
		return true
	}() {
		return
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
