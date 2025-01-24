package app

import (
	"fmt"
	"os/exec"
)

type BinManagerSshpass struct{}

func (k BinManagerSshpass) CheckInstalled() bool {
	cmd := exec.Command("sshfs", "-V")
	err := cmd.Run()
	if err != nil {
		return false
	}
	return true
}

func (k BinManagerSshpass) BinName() string {
	return "sshpass"
}

func (k BinManagerSshpass) SpecInstallFunc() func() error {
	return func() error {
		// 执行安装命令
		cmd := exec.Command("sudo", "DEBIAN_FRONTEND=noninteractive", "apt", "install", "-y", "sshpass")

		// 获取命令的输出和错误信息
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("BinManagerSshpass.SpecInstallFunc: Error installing sshpass: %v\nOutput: %s", err, output)
		}
		return nil
	}
}
