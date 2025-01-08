package util

// package main

// import "telego/app"

// func main() {
// 	app.Main()
// }

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
	"github.com/thoas/go-funk"
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

// check arch list in os.go Arch{XXX}
func GetRemoteArch(hosts []string, usePasswd string) []string {
	results := StartRemoteCmds(hosts, "python3 -c \"import platform; print({'x86_64': 'amd64', 'aarch64': 'arm64', 'armv7l': 'arm32', 'armv6l': 'arm32', 'i386': 'amd32', 'i686': 'amd32', 'ppc64le': 'ppc64le', 'mips': 'mips', 'mipsel': 'mips', 's390x': 's390x', 'riscv64': 'riscv64'}.get(platform.machine().lower(), 'unknown'))\"", usePasswd)
	Logger.Debugf("GetRemoteArch: %v", results)
	return funk.Map(results, func(result string) string {
		return strings.ReplaceAll(
			strings.ReplaceAll(
				strings.ReplaceAll(result, "\n", ""),
				"\r", ""),
			" ", "")
	}).([]string)
}

// hosts format is {user}@{ip}
// left usePasswd to "" if you want to use key
// return output if success
func StartRemoteCmds(hosts []string, remoteCmd string, usePasswd string) []string {
	fmt.Println()
	Logger.Debugf("Starting remote command: %s", remoteCmd)
	runRemoteCommand := func(host string, index int, logFile string, ch chan<- NodeMsg, remote_cmd string) {
		// 打开日志文件（追加模式），确保在函数退出时关闭文件
		file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			ch <- NodeMsg{Index: index, Output: fmt.Sprintf("Error opening log file: %v", err), Complete: true}
			return
		}
		defer file.Close()

		file.WriteString(fmt.Sprintf("Running command on host %s:\n  %s\n", host, remote_cmd))

		// 模拟运行远程指令，实际可以替换为 SSH 或其他工具
		hostsplit := strings.Split(host, "@")
		user := hostsplit[0]
		server := hostsplit[1]
		client, session, err := sshSession(server, user, usePasswd)
		if err != nil {
			ch <- NodeMsg{Index: index, Output: fmt.Sprintf("Error ssh: %v", err), Complete: true}
			return
		}

		// cmd := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", host, remote_cmd)

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
			// time.Sleep(500 * time.Millisecond) // 模拟延迟
			ch <- NodeMsg{Index: index, Output: line, Complete: false}

			// 将输出写入日志文件
			_, _ = writer.Write([]byte(line + "\n"))
		}

		// 等待命令完成
		if err := session.Wait(); err != nil {
			ch <- NodeMsg{Index: index, Output: fmt.Sprintf("Error %v with %s", err, logFile), Complete: true}
			return
		}

		session.Close()
		client.Close()

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
					path0 = path.Join(LogDir(), "remote_cmd_"+strings.ReplaceAll(host, "@", "_")+"_"+t+"_"+fmt.Sprintf("%v", randtag)+".log")
					break
				}
			}

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
		fmt.Println("Error starting program:", err)
	}

	return outputs
}
