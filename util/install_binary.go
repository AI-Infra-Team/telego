package util

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// the tempDir need to be existing
func InstallWindowsBin(url, tempDir, binname string) error {
	tempFilePath := filepath.Join(tempDir, filepath.Base(url))
	DownloadFile(url, tempFilePath)

	moveCommand := fmt.Sprintf("echo \"安装 %s 中\" && timeout /t 1 && move %s C:\\Windows\\System32\\%s", binname, strings.ReplaceAll(tempFilePath, "/", "\\"), fmt.Sprintf("%s.exe", binname))
	Logger.Debugf("move command: %s", moveCommand)

	// 使用 exec.Command 启动一个独立进程执行 move 和重启
	cmd := exec.Command("cmd", "/C", "start", "cmd", "/C", moveCommand)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Error moving file: %v", err)
	}
	time.Sleep(1500 * time.Millisecond)

	return nil
}

// the tempDir need to be existing
func InstallLinuxBin(url, tempDir, binname string) error {
	tempFilePath := filepath.Join(tempDir, filepath.Base(url))
	// file is owned by common user
	DownloadFile(url, tempFilePath)

	// we need root to do the help of move
	ModRunCmd.RequireRootRunCmd("mv", tempFilePath, fmt.Sprintf("/usr/bin/%s", binname))

	ModRunCmd.RequireRootRunCmd("chmod", "755", fmt.Sprintf("/usr/bin/%s", binname))
	return nil
}

func InstallLinuxPreparedBin(preparedBinFilePath string, targetBinName string) error {
	PrintStep("InstallLinuxPreparedBin", "installing "+targetBinName+" from "+preparedBinFilePath)
	// safely copy the preparedPath to /usr/bin/{targetBinName}
	_, err := ModRunCmd.RequireRootRunCmd("cp", preparedBinFilePath, fmt.Sprintf("/usr/bin/%s", targetBinName))
	if err != nil {
		return fmt.Errorf("InstallLinuxPreparedBin failed, err: %v", err)
	}
	_, err = ModRunCmd.RequireRootRunCmd("chmod", "755", fmt.Sprintf("/usr/bin/%s", targetBinName))
	if err != nil {
		return fmt.Errorf("InstallLinuxPreparedBin failed, err: %v", err)
	}
	return nil
}

func InstallWindowsPreparedBin(preparedBinFilePath string, targetBinName string) error {
	PrintStep("InstallWindowsPreparedBin", "installing "+targetBinName+" from "+preparedBinFilePath)
	// use go lib to do copy
	err := SafeCopyOverwrite(preparedBinFilePath, fmt.Sprintf("C:\\Windows\\System32\\%s", targetBinName))
	if err != nil {
		return fmt.Errorf("InstallWindowsPreparedBin failed, err: %v", err)
	}
	return nil
}
