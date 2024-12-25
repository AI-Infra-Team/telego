package app

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"telego/util"

	"github.com/fatih/color"
)

type BinManagerRclone struct{}

func (k BinManagerRclone) CheckInstalled() bool {
	// 尝试运行 "Rclone version --client" 命令以验证 Rclone 是否可用
	cmd := exec.Command("rclone", "version")
	err := cmd.Run()
	if err != nil {
		// 如果命令执行失败，则认为 Rclone 未安装
		return false
	}
	// 如果命令成功执行，则认为 Rclone 已安装
	return true
}

func (k BinManagerRclone) BinName() string {
	return "rclone"
}

func (k BinManagerRclone) SpecInstallFunc() func() error {
	return func() error {
		if strings.Contains(runtime.GOOS, "darwin") {
			// macOS
			// util.ModRunCmd.RunCommandShowProgress("brew", "install", "rclone")
			os.MkdirAll("install_rclone", 0755)
			os.Chdir("install_rclone")
			// defer os.RemoveAll("install_rclone")
			err := util.DownloadFile("https://github.com/rclone/rclone/releases/download/v1.68.2/rclone-v1.68.2-osx-arm64.zip", "rclone.zip")
			if err != nil {
				// fmt.Println("\nFailed to download rclone on mac os, maybe u need proxy, err:", err)
				// os.Exit(1)
				return fmt.Errorf("Failed to download rclone on mac os, maybe u need proxy, err: %v", err)
			} else {
				err = util.UnzipFile("rclone.zip", "./")
				if err != nil {
					return err
				}
				_, err = util.ModRunCmd.ShowProgress(
					"bash",
					"-c",
					"chmod +x rclone-v1.68.2-osx-arm64/rclone "+
						"&& mv rclone-v1.68.2-osx-arm64/rclone /usr/local/bin").
					BlockRun()
				if err != nil {
					return err
				}

				fmt.Println(color.BlueString("please allow rclone to run in config"))
				// 打开 "安全性与隐私" 设置界面
				_, err := util.ModRunCmd.ShowProgress("open", "x-apple.systempreferences:com.apple.preference.security?General").BlockRun()
				if err != nil {
					// fmt.Println("无法打开系统偏好设置:", err)
					return fmt.Errorf("无法打开系统偏好设置: %v", err)
				}
				fmt.Println("已打开系统偏好设置，请手动允许应用运行。")
				os.Chdir("..")
				os.RemoveAll("install_rclone")
				return nil
			}
		} else if strings.Contains(runtime.GOOS, "linux") {
			armOrAmd := "arm64"
			if runtime.GOARCH == "amd64" {
				armOrAmd = "amd64"
			}
			installDir := path.Join(util.WorkspaceDir(), "install_rclone")
			os.MkdirAll(installDir, 0755)
			defer os.RemoveAll(installDir)
			// os.Chdir("install_rclone")
			if util.FileServerAccessible() {
				err := util.DownloadFile(fmt.Sprintf("http://%s:8003/bin_rclone/rclone_%s", util.MainNodeIp, armOrAmd), path.Join(installDir, "rclone"))
				if err != nil {
					return err
				}
				_, err = util.ModRunCmd.RequireRootRunCmd("mv", path.Join(installDir, "rclone"), "/usr/bin/")
				if err != nil {
					return err
				}
				_, err = util.ModRunCmd.RequireRootRunCmd("chmod", "555", "/usr/bin/rclone")
				if err != nil {
					return err
				}
				return nil
			} else {
				// _, err := util.ModRunCmd.NewBuilder("telego", "cmd", "--cmd", "deploy-templete/bin_rclone/install/local").ShowProgress().BlockRun()

				// https://github.com/rclone/rclone/releases/download/v1.68.1/rclone-v1.68.1-linux-amd64.zip
				err := util.DownloadFile(fmt.Sprintf("https://github.com/rclone/rclone/releases/download/v1.68.1/rclone-v1.68.1-linux-%s.zip", armOrAmd), "rclone.zip")
				if err != nil {
					return fmt.Errorf("Download failed %w", err)
				}
				err = util.UnzipFile("rclone.zip", "./")
				if err != nil {
					return fmt.Errorf("Unzip failed %w", err)
				}
				_, err = util.ModRunCmd.NewBuilder("mv", "rclone-v1.68.1-linux-amd64/rclone", "/usr/bin/").
					WithRoot().PrintCmd().ShowProgress().BlockRun()
				if err != nil {
					return fmt.Errorf("mv failed from %s to %s, %v", "rclone-v1.68.1-linux-amd64/rclone", "/usr/bin/", err)
				}
				return nil
			}

		} else if strings.Contains(runtime.GOOS, "windows") {
			// windows
			err := util.DownloadFile("https://github.com/rclone/rclone/releases/download/v1.68.1/rclone-v1.68.1-windows-amd64.zip", "rclone.zip")
			if err != nil {
				ModJobInstall.InstallLocalByJob(InstallJob{Bin: "rclone", BinPack: "bin_rclone"})
				return nil
				// ModRunCmd.RequireRootRunCmd(
				// 	pyCmdHead(), "-c",
				// 	fmt.Sprintf("import urllib.request, os; script = urllib.request.urlopen('http://%s:8003/bin_rclone/install_browser.py').read(); exec(script.decode());", MainNodeIp))
			} else {
				util.UnzipFile("rclone.zip", "./")
				return util.ModRunCmd.CopyDirContentOrFileTo("rclone.exe", "C:\\Windows\\System32\\")
			}
		} else {
			// fmt.Println(color.RedString("unsupported System %s", runtime.GOOS))
			// os.Exit(1)
			return fmt.Errorf("unsupported System %s", runtime.GOOS)
		}
	}
}
