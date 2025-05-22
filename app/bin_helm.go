package app

import "os/exec"

type BinManagerHelm struct{}

func (h BinManagerHelm) CheckInstalled() bool {
	// 尝试运行 "helm version" 命令以验证 helm 是否可用
	cmd := exec.Command("helm", "version")
	err := cmd.Run()
	if err != nil {
		// 如果命令执行失败，则认为 helm 未安装
		return false
	}
	// 如果命令成功执行，则认为 helm 已安装
	return true
}

func (h BinManagerHelm) BinName() string {
	return "helm"
}

func (h BinManagerHelm) SpecInstallFunc() func() error {
	return nil
}
