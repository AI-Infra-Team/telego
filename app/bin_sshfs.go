package app

import "os/exec"

type BinManagerSshFs struct{}

func (k BinManagerSshFs) CheckInstalled() bool {
	cmd := exec.Command("sshfs", "version")
	err := cmd.Run()
	if err != nil {
		return false
	}
	return true
}

func (k BinManagerSshFs) BinName() string {
	return "sshfs"
}

func (k BinManagerSshFs) SpecInstallFunc() func() error {
	return nil
}
