package testutil

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

var ptyInstallChecked = false

// NewPtyCommand 构造一个带伪TTY的命令（使用 script 工具模拟 tty）
// name 是命令名，args 是参数列表
func NewPtyCommand(t *testing.T, name string, args ...string) *exec.Cmd {
	if !ptyInstallChecked {
		ptyInstallChecked = true
		cmd := exec.Command("script", "-q", "-c", "echo 'script installed'", "/dev/null")
		err := cmd.Run()
		if err != nil {
			// install script tool
			(&LinuxInstall{DefaultAppName: "util-linux", SpecAptName: "bsdutils"}).Install(t)
		}
	}

	fullCmd := append([]string{name}, args...)
	cmdStr := shellEscapeArgs(fullCmd)
	return exec.Command("script", "-q", "-c", "sh -c '"+cmdStr+"'; echo $? > /tmp/test_exit_code", "/dev/null")
}

// shellEscapeArgs 将参数数组安全拼接成 shell 字符串
// 使用 strconv.Quote 来正确转义参数中的空格、引号等特殊字符
func shellEscapeArgs(args []string) string {
	var escaped []string
	for _, arg := range args {
		escaped = append(escaped, strconv.Quote(arg))
	}
	return strings.Join(escaped, " ")
}

func GetPtyExitCode(t *testing.T, cmd *exec.Cmd) int {
	code, err := os.ReadFile("/tmp/test_exit_code")
	if err != nil {
		t.Fatalf("read exit code failed: %v", err)
	}
	return int(code[0])
}
