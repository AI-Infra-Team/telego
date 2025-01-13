package app

import (
	"telego/util"
)

type BinManagerWinfsp struct{}

func (k BinManagerWinfsp) CheckInstalled() bool {
	if !util.IsWindows() {
		return true
	}
	_, err := util.ModRunCmd.NewBuilder("rclone", "mount", "D:", "K:").BlockRun()

	if err != nil {
		return false
	}

	// 如果命令成功执行，则认为 Winfsp 已安装
	return true
}

func (k BinManagerWinfsp) BinName() string {
	return "winfsp"
}

func (k BinManagerWinfsp) SpecInstallFunc() func() error {
	return nil
}
