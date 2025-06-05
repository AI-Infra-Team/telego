package testutil

import (
	"os/exec"
	"testing"
)

type LinuxInstall struct {
	DefaultAppName string
	SpecAptName    string
	SpecDnfName    string
	SpecYumName    string
	SpecApkName    string
	SpecBrewName   string
}

func (l *LinuxInstall) Install(t *testing.T) {
	// Define commands as arrays of strings
	aptCmd := []string{"apt-get", "install", "-y", l.DefaultAppName}
	dnfCmd := []string{"dnf", "install", "-y", l.DefaultAppName}
	yumCmd := []string{"yum", "install", "-y", l.DefaultAppName}
	apkCmd := []string{"apk", "add", l.DefaultAppName}       // Note: apk doesn't use -y flag
	brewCmd := []string{"brew", "install", l.DefaultAppName} // Note: brew doesn't need -y flag

	// Override with specific package names if provided
	if l.SpecAptName != "" {
		aptCmd[3] = l.SpecAptName
	}
	if l.SpecDnfName != "" {
		dnfCmd[3] = l.SpecDnfName
	}
	if l.SpecYumName != "" {
		yumCmd[3] = l.SpecYumName
	}
	if l.SpecApkName != "" {
		apkCmd[2] = l.SpecApkName
	}
	if l.SpecBrewName != "" {
		brewCmd[2] = l.SpecBrewName
	}

	// List of command arrays to try
	commands := [][]string{aptCmd, dnfCmd, yumCmd, apkCmd, brewCmd}

	for _, cmdArgs := range commands {
		if err := RunCommand(t, exec.Command(cmdArgs[0], cmdArgs[1:]...)); err != nil {
			t.Logf("安装 %s 失败: %v", l.DefaultAppName, err)
		} else {
			t.Logf("安装 %s 成功", l.DefaultAppName)
			return
		}
	}
	t.Fatalf("已经尝试若干安装方式，安装 %s 失败", l.DefaultAppName)
}
