package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"telego/util"
	"text/template"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/thoas/go-funk"
)

type StartFileserverMode = int

const (
	StartFileserverModeCaller = iota // 部署节点 为调用节点
	StartFileserverModeCallee        // main node 为被调用节点
)

type StartFileserverJob struct {
	Mode StartFileserverMode
}

func (s StartFileserverJob) ModeString() string {
	switch s.Mode {
	case StartFileserverModeCaller:
		return "start_fileserver_caller"
	case StartFileserverModeCallee:
		return "start_fileserver_callee"
	default:
		return "unknown"
	}
}

type ModJobStartFileserverStruct struct{}

var ModJobStartFileserver ModJobStartFileserverStruct

func (m ModJobStartFileserverStruct) JobCmdName() string {
	return "start-fileserver"
}

func (m ModJobStartFileserverStruct) ParseJob(applyCmd *cobra.Command) *cobra.Command {

	// 绑定命令行标志到结构体字段
	mode := ""
	pubkey := ""
	applyCmd.Flags().StringVar(&mode, "mode", "", "Sub operation of ssh")
	applyCmd.Flags().StringVar(&pubkey, "pubkey", "", "Pubkey to be set on this node")

	applyCmd.Run = func(_ *cobra.Command, _ []string) {
		TaskId := 0
		switch mode {
		case StartFileserverJob{Mode: StartFileserverModeCallee}.ModeString():
			TaskId = StartFileserverModeCallee
		case StartFileserverJob{Mode: StartFileserverModeCaller}.ModeString():
			TaskId = StartFileserverModeCaller
		default:
			fmt.Println(color.RedString("unsupported ssh ope mode: '%s'", mode))
			os.Exit(1)
		}

		ModJobStartFileserver.dispatchMode(StartFileserverJob{
			Mode: TaskId,
		})
	}

	return applyCmd
}

func (m ModJobStartFileserverStruct) dispatchMode(job StartFileserverJob) {
	switch job.Mode {
	case StartFileserverModeCaller:
		{
			// check dist dir with telego_* files
			distDir := filepath.Join(util.GetEntryDir(), "dist")
			files, err := os.ReadDir(distDir)
			if err != nil {
				fmt.Println(color.RedString("read dist dir failed, err: %v", err))
				os.Exit(1)
			}
			if !funk.Contains(files, func(file os.DirEntry) bool {
				return strings.HasPrefix(file.Name(), "telego_")
			}) {
				fmt.Println(color.RedString("run start-fileserver under telego project root dir, and make sure already compiled with 2.build.py"))
				os.Exit(1)
			}

			// ssh to main node and start the fileserver
			ok, mainnodePw := util.StartTemporaryInputUI(color.GreenString(
				"启动 MAIN_NODE 需要输入 MAIN_NODE密码"),
				"此处键入密码",
				"回车确认，ctrl+c取消")
			if !ok {
				fmt.Println("User canceled start fileserver")
				os.Exit(1)
			}

			mainNodeHostArr := []string{fmt.Sprintf("%s@%s", util.MainNodeUser, util.MainNodeIp)}

			util.PrintStep("StartFileserver", color.BlueString("getting remote arch"))
			arch := util.GetRemoteArch(mainNodeHostArr, mainnodePw)[0]
			switch arch {
			case util.ArchAmd64, util.ArchArm64:
				// fmt.Printf("entry dir %s\n", util.GetEntryDir())
				from := filepath.Join(util.GetEntryDir(), fmt.Sprintf("dist/telego_linux_%s", arch))
				to := "./"
				util.PrintStep("StartFileserver", color.BlueString("\ntransfering files (%s)->(%s)", from, to))
				// fmt.Println(color.BlueString("\ntransfering files (%s)->(%s)", from, to))
				util.UploadToMainNode(from, to)

				// fmt.Println(color.BlueString("\nstarting fileserver"))
				util.PrintStep("StartFileserver", color.BlueString("\nstarting fileserver"))
				remoteCmd0 := fmt.Sprintf("cp ~/telego_linux_%s /usr/bin/telego", arch)
				remoteCmd1 := "chmod +x /usr/bin/telego"
				remoteCmd2 := "chown {user} /usr/bin/telego"
				remoteCmd3 := strings.Join(m.NewStartFileserverCmd(
					StartFileserverJob{Mode: StartFileserverModeCallee}.ModeString()), " ")
				util.StartRemoteCmds(
					mainNodeHostArr,
					// install telego,
					fmt.Sprintf("python3 -c \"import os;os.system('%s' if os.geteuid() == 0 else 'sudo %s')\" && ", remoteCmd0, remoteCmd0)+
						fmt.Sprintf("python3 -c \"import os;os.system('%s' if os.geteuid() == 0 else 'sudo %s')\" && ", remoteCmd1, remoteCmd1)+
						fmt.Sprintf("python3 -c \"import os, getpass;user=getpass.getuser();os.system(f'%s' if os.geteuid() == 0 else f'sudo %s')\" && ", remoteCmd2, remoteCmd2)+
						fmt.Sprintf("python3 -c \"import os;os.system('%s' if os.geteuid() == 0 else 'sudo %s')\"", remoteCmd3, remoteCmd3),
					mainnodePw,
				)
			default:
				fmt.Println(color.RedString("unsupported arch: '%s'", arch))
			}

		}
	case StartFileserverModeCallee:
		{
			const serviceTemplate = `[Unit]
Description={{.Description}}
After=network.target

[Service]
Type=simple
User={{.User}}
WorkingDirectory={{.WorkingDirectory}}
ExecStart={{.ExecStart}}
Restart=always

[Install]
WantedBy=multi-user.target
`
			type ServiceConfig struct {
				Description      string
				User             string
				Group            string
				WorkingDirectory string
				ExecStart        string
			}

			config := ServiceConfig{
				Description: "Telego File Server",
				User:        "root", // 替换为实际用户
				// Group:            "your_group",                                    // 替换为实际用户组
				WorkingDirectory: "/teledeploy",                          // 替换为实际目录
				ExecStart:        "/usr/bin/python3 -m http.server 8003", // 替换为实际路径和端口
			}

			// 生成服务文件内容
			serviceFilePath := "/etc/systemd/system/python-fileserver.service"
			file, err := os.Create(serviceFilePath)
			if err != nil {
				fmt.Printf("无法创建服务文件: %v\n", err)
				os.Exit(1)
			}
			defer file.Close()

			tmpl := template.Must(template.New("service").Parse(serviceTemplate))
			if err := tmpl.Execute(file, config); err != nil {
				fmt.Printf("无法写入服务配置: %v\n", err)
				os.Exit(1)
			}

			// 设置文件权限
			if err := os.Chmod(serviceFilePath, 0644); err != nil {
				fmt.Printf("无法设置服务文件权限: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("服务文件已生成:", serviceFilePath)

			// 重新加载 systemd 配置
			if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
				fmt.Printf("无法重新加载 systemd 配置: %v\n", err)
				os.Exit(1)
			}

			// 启动服务
			if err := exec.Command("systemctl", "start", "python-fileserver.service").Run(); err != nil {
				fmt.Printf("无法启动服务: %v\n", err)
				os.Exit(1)
			}

			// 设置服务开机自启
			if err := exec.Command("systemctl", "enable", "python-fileserver.service").Run(); err != nil {
				fmt.Printf("无法设置服务开机自启: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("服务已启动并设置为开机自启")
		}
	default:
		fmt.Println(color.RedString("unsupported ssh ope mode: '%s'", job.Mode))
		os.Exit(1)
	}
}

func (m ModJobStartFileserverStruct) NewStartFileserverCmd(
	fileserverModeStr string,
) []string {
	switch fileserverModeStr {
	case StartFileserverJob{Mode: StartFileserverModeCallee}.ModeString(),
		StartFileserverJob{Mode: StartFileserverModeCaller}.ModeString():
		return []string{"telego", "start-fileserver",
			"--mode", fileserverModeStr}
	default:
		fmt.Println(color.RedString("unsupported ssh ope mode: '%s'", fileserverModeStr))
		os.Exit(1)
	}
	return []string{}
}

func (m ModJobStartFileserverStruct) ExecDelegate(sshModeStr string) DispatchExecRes {
	cmd := ModJobStartFileserver.NewStartFileserverCmd(sshModeStr)
	return DispatchExecRes{
		Exit: true,
		ExitWithDelegate: func() {
			util.PrintStep("StartFileserver", color.BlueString("Starting job ssh... cmds[%v]", cmd))
			_, err := util.ModRunCmd.ShowProgress(cmd[0], cmd[1:]...).BlockRun()
			if err != nil {
				fmt.Println(color.RedString("job ssh error: %v", err))
			}
			fmt.Println(color.GreenString("job ssh finished"))
		},
	}
}

// // in app entry
// func (_ ModJobStartFileserverStruct) StartFileserverLocal(k8sprj string, k8sdp *Deployment, clusterContextName string) {
// 	// cmds := []string{}
// 	cmds := ModJobStartFileserver.NewStartFileserverCmd(k8sprj, k8sdp.K8s, k8sdp.Helms, clusterContextName)
// 	util.ModRunCmd.RunCommandShowProgress(cmds[0], cmds[1:]...)
// 	// cmd += NewStartFileserverCmd(binpack, bin, binBin[bin])

// 	// util.Logger.Debugf("apply cmds split: %s", cmds)

// }
