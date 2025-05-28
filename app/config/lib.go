package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"telego/util/yamlext"

	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/util/homedir"
)

type Config struct {
	ProjectDir string `yaml:"project_dir"`
}

var config *Config

func SetFake(prjdir string) {
	fmt.Println(color.GreenString("SetFake %s", prjdir))
	config = &Config{
		ProjectDir: prjdir,
	}
}

func LoadFake() Config {
	return Config{
		ProjectDir: filepath.Join(homedir.HomeDir(), "fake_prj_dir"),
	}
}

// func Exists(workspace string) bool {
// 	// Construct the path to the config.yaml file
// 	configPath := filepath.Join(workspace, "config.yaml")

// 	// Check if config file exists
// 	_, err := os.Stat(configPath)
// 	return os.IsNotExist(err)
// }

var cacheMayFailLoad []bool = nil

// return false if load fake, true if load real
func MayFailLoad(workspace string) bool {
	if cacheMayFailLoad != nil {
		return cacheMayFailLoad[0]
	}
	configPath := filepath.Join(workspace, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		LoadFake()
		cacheMayFailLoad = []bool{false}
		return false
	}
	Load(workspace, nil)
	cacheMayFailLoad = []bool{true}
	return true
}

// Load loads configuration from a YAML file located at {workspace}/config.yaml
func Load(
	workspace string,
	StartTemporaryInputUI func(head string, placeholder string, tail string) (bool, string)) Config {
	if config != nil {
		return *config
	}

	// Construct the path to the config.yaml file
	configPath := filepath.Join(workspace, "config.yaml")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// If file doesn't exist, start Bubble Tea UI to get the project directory
		fmt.Println("Config file not found. Launching interactive mode.")
		set, projectDir := StartTemporaryInputUI(
			color.GreenString("为了进行自定义yml的管控，需要配置项目路径:"),
			"Enter your project directory",
			"(云桌面输入挂载项目目录, 本机输入git项目目录)\n\n(回车确认，ctrl+c取消)",
		)
		if set {
			// verify path is absolute
			absProjectDir, err := filepath.Abs(projectDir)
			if err != nil {
				fmt.Println(color.RedString("Invalid path: "), projectDir)
				os.Exit(1)
			}
			if strings.ReplaceAll(absProjectDir, "\\", "/") != strings.ReplaceAll(projectDir, "\\", "/") {
				fmt.Println(color.RedString("Require absolute path, input '%v'",
					strings.ReplaceAll(absProjectDir, "\\", "/"),
					strings.ReplaceAll(projectDir, "\\", "/")))
				os.Exit(1)
			}

			// Save the user input as a new config file
			cfg := Config{ProjectDir: projectDir}
			SaveConfig(configPath, cfg)
			return cfg
		} else {
			fmt.Println(color.BlueString("User cancelled configuration."))
			os.Exit(0)
		}
	}

	// Open and read the config.yaml file
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("failed to read config file: %v", err)
	}

	// Parse the YAML file into the Config struct
	var cfg Config
	err = yamlext.UnmarshalAndValidate(data, &cfg)
	if err != nil {
		log.Fatalf("failed to parse YAML config: %v", err)
	}
	config = &cfg
	return cfg
}

// SaveConfig saves the config to the specified path in YAML format
func SaveConfig(path string, cfg Config) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		log.Fatalf("failed to marshal config to YAML: %v", err)
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		log.Fatalf("failed to write config file: %v", err)
	}
}
