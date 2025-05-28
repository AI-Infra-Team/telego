package util

// package main

// import "telego/app"

// func main() {
// 	app.Main()
// }

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
	"github.com/thoas/go-funk"
	"gopkg.in/yaml.v2"
)

type NodeState struct {
	Host       string
	Output     string
	IsComplete bool
}

type RemoteControlModel struct {
	Nodes []NodeState
}

type NodeMsg struct {
	Index    int
	Output   string
	Complete bool
}

const (
	DonePrefix = "Done with "
)

func (m RemoteControlModel) Init() tea.Cmd {
	return nil
}

func (m RemoteControlModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	case NodeMsg:
		// 更新节点状态
		m.Nodes[msg.Index].Output = msg.Output
		m.Nodes[msg.Index].IsComplete = msg.Complete

		// 检查所有节点是否完成
		allComplete := true
		for _, node := range m.Nodes {
			if !node.IsComplete {
				allComplete = false
				break
			}
		}

		// 如果所有节点完成，退出程序
		if allComplete {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m RemoteControlModel) View() string {
	view := color.BlueString("Remote Command Execution:\n\n")
	for _, node := range m.Nodes {
		status := color.BlueString("Running")
		if node.IsComplete {
			status = color.GreenString("Completed")
		}
		view += fmt.Sprintf("[%s] %s: %s\n", status, node.Host, node.Output)
	}
	return view
}

// GetRemoteArch 获取远程主机的架构信息
// hosts: 远程主机列表
// usePasswd: 密码
// currentSystems: 每个主机对应的系统类型
func GetRemoteArch(hosts []string, usePasswd string, currentSystems []SystemType) []string {
	if len(hosts) != len(currentSystems) {
		Logger.Errorf("hosts and systems length mismatch: hosts=%d, systems=%d", len(hosts), len(currentSystems))
		return make([]string, len(hosts))
	}

	// 为每个主机执行对应的命令
	results := make([]string, len(hosts))
	for i, host := range hosts {
		cmd := currentSystems[i].GetArchCmd()
		hostResults, _ := StartRemoteCmds([]string{host}, cmd, usePasswd)
		if len(hostResults) > 0 {
			// lines := strings.Split(hostResults[0], "\n")
			// // assert line count >=2
			// if len(lines) < 3 {
			// 	Logger.Warnf("Command output for host %s has fewer than 3 lines: %v", host, hostResults[0])
			// 	continue
			// }

			// 从第3行开始处理
			result := hostResults[0]

			// 清理结果
			result = strings.ToLower(strings.TrimSpace(result))
			result = strings.ReplaceAll(result, "\n", "")
			result = strings.ReplaceAll(result, "\r", "")
			result = strings.ReplaceAll(result, " ", "")

			// 判断架构
			if result == "aarch64" || result == "arm64" || result == "arm64e" {
				results[i] = "arm64"
			} else if result == "x86_64" || result == "amd64" || result == "x64" {
				results[i] = "amd64"
			} else {
				results[i] = "unknown"
			}
		} else {
			results[i] = "unknown"
		}
	}
	fmt.Printf("GetRemoteArch: %v", results)
	Logger.Debugf("GetRemoteArch: %v", results)
	return results
}

// GetRemoteSys 获取远程主机的系统类型
// hosts: 远程主机列表
// usePasswd: 密码
func GetRemoteSys(hosts []string, usePasswd string) []SystemType {
	// 使用 uname 命令获取系统信息
	results, logpaths := StartRemoteCmds(hosts, "uname -s", usePasswd)

	fmt.Println(color.BlueString("GetRemoteSys raw output: %v", results))
	// Logger.Debugf("GetRemoteSys: %v", results)

	i := 0
	remoteSys := funk.Map(results, func(result string) SystemType {
		// remove first 2 line
		// lines := strings.Split(result, "\n")
		// // assert line count >=2
		// if len(lines) < 3 {
		// 	errMsg := fmt.Sprintf("GetRemoteSys command output has fewer than 3 lines: %v", result)
		// 	Logger.Warnf(errMsg)
		// 	logContet, err := os.ReadFile(logpaths[i])
		// 	if err != nil {
		// 		fmt.Println(color.BlueString("%s\n and we failed to read remote log"), errMsg)
		// 	} else {
		// 		fmt.Println(color.BlueString("%s,\n log content read begin >>> \n %s \n<<< log content read end", errMsg, string(logContet)))
		// 	}
		// 	return UnknownSystem{}
		// }

		result = results[0]

		// 清理结果
		result = strings.ToLower(strings.TrimSpace(result))
		result = strings.ReplaceAll(result, "\n", "")
		result = strings.ReplaceAll(result, "\r", "")
		result = strings.ReplaceAll(result, " ", "")

		i += 1
		// 判断系统类型
		switch result {
		case "linux", "gnu/linux", "gnu":
			return LinuxSystem{}
		case "darwin":
			return DarwinSystem{}
		case "windows":
			return WindowsSystem{}
		default:
			errMsg := fmt.Sprintf("GetRemoteSys not match system: %v", result)
			Logger.Warnf(errMsg)
			logContet, err := os.ReadFile(logpaths[i])
			if err != nil {
				fmt.Println(color.BlueString("%s\n and we failed to read remote log"), errMsg)
			} else {
				fmt.Println(color.BlueString("%s,\n log content read begin >>> \n %s \n<<< log content read end", errMsg, string(logContet)))
			}
			return UnknownSystem{}
		}

	}).([]SystemType)

	fmt.Println(color.BlueString("GetRemoteSys type: %v", reflect.TypeOf(remoteSys[0])))
	fmt.Println(color.BlueString("GetRemoteSys content: %+v", remoteSys))

	return remoteSys
}

// 为每个用户创建配置文件
func createUserConfigs(hosts []string, usePasswd string) error {
	// 使用 map 对用户进行去重
	uniqueUsers := make(map[string]bool)
	for _, host := range hosts {
		hostsplit := strings.Split(host, "@")
		if len(hostsplit) != 2 {
			return fmt.Errorf("invalid host format: %s", host)
		}
		uniqueUsers[hostsplit[0]] = true
	}

	// 为每个唯一用户创建配置文件
	for user := range uniqueUsers {
		// 创建用户特定的配置文件
		userConfig := AdminUserConfig{
			Username: user,
			Password: usePasswd,
		}

		// 创建用户特定的配置文件路径，添加 userconfig_ 前缀
		userConfigPath := fmt.Sprintf("/teledeploy_secret/config/userconfig_%s", user)

		// 确保目录存在
		if err := os.MkdirAll(filepath.Dir(userConfigPath), 0755); err != nil {
			return fmt.Errorf("error creating config directory for user %s: %w", user, err)
		}

		// 序列化配置为YAML
		data, err := yaml.Marshal(userConfig)
		if err != nil {
			return fmt.Errorf("error serializing config for user %s: %w", user, err)
		}

		// 写入文件
		if err := os.WriteFile(userConfigPath, data, 0600); err != nil {
			return fmt.Errorf("error writing config for user %s: %w", user, err)
		}
	}
	return nil
}

type LogPathStr = string

// hosts format is {user}@{ip}
// left usePasswd to "" if you want to use key
// return output if success
func StartRemoteCmds(hosts []string, remoteCmd string, usePasswd string) ([]string, []LogPathStr) {
	fmt.Println()
	Logger.Debugf("Starting remote command: %s", remoteCmd)

	// 如果提供了密码，为每个用户创建配置文件
	if usePasswd != "" {
		if err := createUserConfigs(hosts, usePasswd); err != nil {
			Logger.Warnf("Failed to create user configs: %v", err)
		}
	}

	// runRemoteCommand 执行单个远程命令
	runRemoteCommand := func(host string, index int, logFile string, ch chan<- NodeMsg, remote_cmd string) {

		// 打开日志文件（追加模式），确保在函数退出时关闭文件
		file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			ch <- NodeMsg{Index: index, Output: fmt.Sprintf("Error opening log file: %v", err), Complete: true}
			return
		}
		defer file.Close()
		debugFile, err := os.OpenFile(logFile+".debug", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			ch <- NodeMsg{Index: index, Output: fmt.Sprintf("Error opening debug file: %v", err), Complete: true}
			return
		}
		defer debugFile.Close()

		// 调试错误信息的辅助函数
		debugErr := func(stdout, stderr string, err error, note string, end bool) {
			errMsg := fmt.Sprintf("Error %s, err:%v, stdout:%v, stderr:%v", note, err, stdout, stderr)
			ch <- NodeMsg{Index: index, Output: errMsg, Complete: end}
			debugFile.WriteString(errMsg + "\n")
		}

		file.WriteString(fmt.Sprintf("Running command on host %s:\n  %s\n", host, remote_cmd))

		// 获取主机信息
		hostsplit := strings.Split(host, "@")
		if len(hostsplit) != 2 {
			ch <- NodeMsg{Index: index, Output: fmt.Sprintf("Invalid host format: %s", host), Complete: true}
			return
		}
		user := hostsplit[0]
		server := hostsplit[1]
		port := "22"

		// Parse port if it exists in the server address (format: hostname:port)
		if strings.Contains(server, ":") {
			parts := strings.Split(server, ":")
			server = parts[0]
			port = parts[1]
		}

		// remoteConfigPath := fmt.Sprintf("/teledeploy_secret/config/userconfig_%s", user)
		// localConfigPath := GetCurUserConfigPath()

		// 执行单个命令的辅助函数
		// execRemoteCmd := func(cmd string, desc string) error {
		// 	ch <- NodeMsg{Index: index, Output: desc, Complete: false}
		// 	client, session, err := sshSession(server, user, usePasswd, port)
		// 	if err != nil {
		// 		return fmt.Errorf("ssh error: %v", err)
		// 	}
		// 	defer client.Close()
		// 	defer session.Close()
		// 	return session.Run(cmd)
		// }

		// 执行命令并获取输出的辅助函数
		execRemoteCmdWithOutput := func(cmd string, desc string) (string, string, error) {
			ch <- NodeMsg{Index: index, Output: desc, Complete: false}
			client, session, err := sshSession(server, user, usePasswd, port)
			if err != nil {
				return "", "", fmt.Errorf("ssh error: %v", err)
			}
			defer client.Close()
			defer session.Close()

			var stdout, stderr bytes.Buffer
			session.Stdout = &stdout
			session.Stderr = &stderr
			if err := session.Run(cmd); err != nil {
				return stdout.String(), stderr.String(), err
			}
			return stdout.String(), stderr.String(), nil
		}

		// 2. 检查并配置 sudo 权限
		stdout, stderr, err := execRemoteCmdWithOutput(
			"if sudo -n true 2>/dev/null; then echo 'sudo_ok'; else echo 'sudo_need_config'; fi",
			fmt.Sprintf("checking sudo permissions for %s", host),
		)
		if err != nil {
			debugErr(stdout, stderr, err, "检查 sudo 权限时出错", true)
			return
		}
		debugFile.WriteString(fmt.Sprintf("Checking sudo permissions output: %s\n", stdout))

		rcloneName := ""
		configRcloneOnce := func() {
			if rcloneName != "" {
				return
			}
			// 定义需要在多个代码块中共享的变量
			rcloneName = base64.RawURLEncoding.EncodeToString([]byte(server))
			err = NewRcloneConfiger(RcloneConfigTypeSsh{}, rcloneName, server).
				WithUser(user, usePasswd).WithPort(port).DoConfig()
			if err != nil {
				debugErr("", "", err, "配置rclone失败", true)
			}
		}
		if strings.Contains(stdout, "sudo_need_config") {
			// 1. 创建本地临时密码文件
			localPasswdFile := "/tmp/sudo_passwd"
			if err := os.WriteFile(localPasswdFile, []byte(usePasswd), 0600); err != nil {
				debugErr("", "", err, "创建本地密码文件时出错", true)
				return
			}
			defer os.Remove(localPasswdFile)

			configRcloneOnce()

			// 2. 使用 rclone 传输密码文件到远程
			remotePasswdFile := fmt.Sprintf("sudo_passwd_%s", user)
			if err := RcloneSyncFileToFile(localPasswdFile, fmt.Sprintf("%s:%s", rcloneName, remotePasswdFile)); err != nil {
				debugErr("", "", err, "传输密码文件时出错", true)
				return
			}

			// 3. 在远程执行 sudo 命令，使用密码文件
			stdout, stderr, err = execRemoteCmdWithOutput(
				fmt.Sprintf("cat %s | sudo -S sh -c 'echo \"%s ALL=(ALL) NOPASSWD:ALL\" > /etc/sudoers.d/%s && chmod 440 /etc/sudoers.d/%s' && rm -f %s",
					remotePasswdFile, user, user, user, remotePasswdFile),
				fmt.Sprintf("configuring sudo for %s", host),
			)
			if err != nil {
				debugErr(stdout, stderr, err, "配置 sudo 权限时出错", true)
				return
			}
			file.WriteString(fmt.Sprintf("Configuring sudo output: %s\n", stdout))

			// 4. 验证配置是否生效
			stdout, stderr, err = execRemoteCmdWithOutput(
				"sudo -n true",
				fmt.Sprintf("verifying sudo config for %s", host),
			)
			if err != nil {
				debugErr(stdout, stderr, err, "验证 sudo 配置时出错", true)
				return
			}
			file.WriteString(fmt.Sprintf("Verifying sudo config output: %s\n", stdout))
		}

		// 1. 准备远程目录
		if stdout, stderr, err := execRemoteCmdWithOutput(
			fmt.Sprintf("mkdir -p /teledeploy_secret/config && chown -R %s:%s /teledeploy_secret/config && chmod 700 /teledeploy_secret/config", user, user),
			fmt.Sprintf("prepare remote %s secret directory", host),
		); err != nil {
			debugErr(stdout, stderr, err, "准备远程目录时出错，将使用sudo重试", false)
			// try with sudo
			if stdout, stderr, err := execRemoteCmdWithOutput(
				fmt.Sprintf("sudo mkdir -p /teledeploy_secret/config && sudo chown -R %s:%s /teledeploy_secret/config && sudo chmod 700 /teledeploy_secret/config", user, user),
				fmt.Sprintf("sudo prepare remote %s secret directory", host),
			); err != nil {
				debugErr(stdout, stderr, err, "sudo 准备远程目录时也出错", true)
				return
			}
			// ch <- NodeMsg{Index: index, Output: fmt.Sprintf("Error preparing directory: %v", err), Complete: true}
			// return
		}

		// cur user config
		curUserConfig := GetUserConfigPath(user)
		WriteAdminUserConfig(&AdminUserConfig{
			Username: user,
			Password: usePasswd,
		})
		configRcloneOnce()
		if err := RcloneSyncFileToFile(curUserConfig, fmt.Sprintf("%s:%s", rcloneName, curUserConfig)); err != nil {
			debugErr("", "", err, "传输当前用户配置时出错", true)
			return
		}

		// // 3. 配置 rclone
		// err = NewRcloneConfiger(RcloneConfigTypeSsh{}, rcloneName, server).
		// 	WithUser(user, usePasswd).
		// 	DoConfig()
		// if err != nil {
		// 	ch <- NodeMsg{Index: index, Output: fmt.Sprintf("Error configuring rclone: %v", err), Complete: true}
		// 	return
		// }

		// // 4. 传输配置文件
		// if err := RcloneSyncFileToFile(localConfigPath, fmt.Sprintf("%s:%s", rcloneName, remoteConfigPath)); err != nil {
		// 	ch <- NodeMsg{Index: index, Output: fmt.Sprintf("Error transferring config: %v", err), Complete: true}
		// 	return
		// }

		// // 5. 设置配置文件权限
		// if err := execRemoteCmd(
		// 	fmt.Sprintf("chmod 600 %s", remoteConfigPath),
		// 	fmt.Sprintf("set ssh config permissions for %s", host),
		// ); err != nil {
		// 	ch <- NodeMsg{Index: index, Output: fmt.Sprintf("Error setting config permissions: %v", err), Complete: true}
		// 	return
		// }

		// 6. 执行实际命令
		ch <- NodeMsg{Index: index, Output: fmt.Sprintf("executing command on %s", host), Complete: false}
		client, session, err := sshSession(server, user, usePasswd, port)
		if err != nil {
			ch <- NodeMsg{Index: index, Output: fmt.Sprintf("Error ssh: %v", err), Complete: true}
			return
		}
		defer client.Close()
		defer session.Close()

		// 创建管道用于读取 stdout 和 stderr
		stdoutPipe, _ := session.StdoutPipe()
		stderrPipe, _ := session.StderrPipe()

		// 合并 stdout 和 stderr
		reader := io.MultiReader(stdoutPipe, stderrPipe)

		// 同时输出到日志文件和通道
		writer := io.MultiWriter(file)

		// 创建 Scanner 读取合并流
		scanner := bufio.NewScanner(reader)

		// 启动命令
		if err := session.Start(remote_cmd); err != nil {
			ch <- NodeMsg{Index: index, Output: fmt.Sprintf("Error starting command: %v", err), Complete: true}
			return
		}

		// 扫描合并后的输出流
		for scanner.Scan() {
			line := scanner.Text()
			ch <- NodeMsg{Index: index, Output: line, Complete: false}

			// 将输出写入日志文件
			_, _ = writer.Write([]byte(line + "\n"))
		}

		// 等待命令完成
		if err := session.Wait(); err != nil {
			ch <- NodeMsg{Index: index, Output: fmt.Sprintf("Error %v with %s", err, logFile), Complete: true}
			return
		}

		// 发送完成消息
		ch <- NodeMsg{Index: index, Output: DonePrefix + logFile, Complete: true}
	}

	// hosts := []string{"127.0.0.1", "8.8.8.8", "1.1.1.1"}
	nodes := make([]NodeState, len(hosts))
	for i, host := range hosts {
		nodes[i] = NodeState{Host: host}
	}

	model := RemoteControlModel{Nodes: nodes}
	program := tea.NewProgram(model)

	logPaths := make([]LogPathStr, len(hosts))
	// 并行执行远程指令
	msgCh := make(chan NodeMsg)
	var wg sync.WaitGroup
	t := timestamp()
	for i, host := range hosts {
		wg.Add(1)
		go func(index int, host string) {
			defer wg.Done()
			path0 := ""
			for {
				randtag := rand.Intn(10000)
				if _, err := os.Stat(path0); err != nil {
					path0 = filepath.Join(LogDir(), "remote_cmd_"+strings.ReplaceAll(host, "@", "_")+"_"+t+"_"+fmt.Sprintf("%v", randtag)+".log")
					break
				}
			}
			logPaths[i] = path0
			runRemoteCommand(host, index, path0, msgCh, remoteCmd)
		}(i, host)
	}

	go func() {
		wg.Wait()
		close(msgCh)
	}()

	outputs := make([]string, len(hosts))
	matchUserIdx := func(output string) int {
		for i, host := range hosts {
			hostFormated := strings.ReplaceAll(host, "@", "_")
			if strings.Contains(output, hostFormated) {
				return i
			}
		}
		return -1
	}
	// 启动消息监听
	go func() {
		for msg := range msgCh {
			if strings.Contains(msg.Output, DonePrefix) {
				idx := matchUserIdx(msg.Output)
				if idx >= 0 {
					outputs[idx] = strings.Replace(msg.Output, DonePrefix, "", 1)
					// read from path
					if outputs[idx] != "" {
						contentbytes, err := os.ReadFile(outputs[idx])
						content := string(contentbytes)
						lines := strings.Split(string(content), "\n")
						if len(lines) >= 2 {
							// remove first two line
							content = strings.Join(lines[2:], "\n")
						}
						if err != nil {
							Logger.Warnf("Error reading log file(%s): %v", outputs[idx], err)
						} else {
							outputs[idx] = content
						}
					} else {
						Logger.Warnf("Remote output is empty")
					}
				} else {
					Logger.Warnf("Remote run maybe failed, output: %s", msg.Output)
				}
			}
			program.Send(msg)
		}
	}()

	// 启动 TUI
	if err := program.Start(); err != nil {
		fmt.Println("Error starting remote cmds program:", err)
	}

	return outputs, logPaths
}
