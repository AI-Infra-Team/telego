package app

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"telego/util"
	"time"

	"github.com/fatih/color"
)

var FILESERVER = fmt.Sprintf("http://%s:8003/bin_telego", util.MainNodeIp)
var CHECKSUM = fmt.Sprintf("http://%s:8003/bin_telego/checksums.txt", util.MainNodeIp)

// getFileChecksum calculates the SHA-256 checksum of a local file
func getFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	hash := sha256.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// getChecksums retrieves the checksums from the remote checksums file
func getChecksums() (map[string]string, error) {
	client := &http.Client{
		Timeout: 3 * time.Second, // 设置超时为 5 秒
	}
	resp, err := client.Get(CHECKSUM)
	if err != nil {
		fmt.Printf("getChecksums http get err %v\n", err)
		return nil, fmt.Errorf("failed to fetch checksums: %v", err)
	}
	defer resp.Body.Close()

	checksums := make(map[string]string)
	var file string
	var checksum string
	for {
		_, err := fmt.Fscanf(resp.Body, "%s %s\n", &file, &checksum) // Expecting two columns: filename, checksum
		if err != nil {
			// fmt.Printf("getChecksums scan checksum err %v\n", err)
			break
		}
		checksums[checksum] = file
	}

	if len(checksums) == 0 {
		return nil, fmt.Errorf("no checksums found in file")
	}

	return checksums, nil
}

func needUpgrade() (bool, error) {

	// skip if contains NO_UPGRADE env
	if os.Getenv("NO_UPGRADE") == "true" {
		return false, nil
	}

	thisProcBinary, err := os.Executable()
	if err != nil {
		fmt.Printf("get this binary err: %v", err)
		return false, err
	}
	// if env FROM_GO_RUN is true, skip upgrade
	if os.Getenv("FROM_GO_RUN") == "true" {
		return false, nil
	}

	thisChecksum, err := getFileChecksum(thisProcBinary)
	if err != nil {
		fmt.Printf("getFileChecksum err %v\n", err)
		return false, err
	}
	targetChecksums, err := getChecksums()
	if err != nil {
		fmt.Printf("getChecksums err %v\n", err)
		return false, err
	}
	var _, find = targetChecksums[thisChecksum]
	if find {
		return false, nil
	}
	fmt.Printf("current checksum %s\n   does not match target checksums %v\n", thisChecksum, targetChecksums)
	return true, nil
}

// checkAndUpgrade checks the current binary hash and downloads the latest version if needed
func checkAndUpgrade() {
	// Placeholder for checking current binary hash (you would need to implement this)
	// If the hash does not match, proceed to download and upgrade the binary
	needUpgrade, err := needUpgrade() // Assume we need to upgrade for demonstration
	if err != nil {
		fmt.Println("checksum failed, skip upgrade")
		return
	}

	if !needUpgrade {
		fmt.Println("No upgrade needed")
		return
	}

	// Check the operating system and architecture
	switch runtime.GOOS {
	case "windows":
		upgradeWindows()
	case "linux":
		upgradeLinux()
	default:
		fmt.Println("Unsupported OS")
	}
	//sleep 3s
	time.Sleep(3 * time.Second)
}

func upgradeWindows() {
	fmt.Println("校验到版本更新，升级中，升级后请重新启动 telego")
	// Download binary to temp directory, then move to C:\Windows\System32
	tempDir := os.ExpandEnv("$USERPROFILE\\telego_install")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		fmt.Println("Error creating temp directory:", err)
		return
	}

	downloadPath := fmt.Sprintf("%s\\telego.exe", tempDir)
	util.DownloadFile(fmt.Sprintf("%s/telego_windows_amd64.exe", FILESERVER), downloadPath)

	// Use schetask Move the binary to System32

	moveCommand := fmt.Sprintf("echo \"telego升级中\" && timeout /t 2 && move %s C:\\Windows\\System32\\telego.exe", downloadPath)

	// 使用 exec.Command 启动一个独立进程执行 move 和重启
	cmd := exec.Command("cmd", "/C", "start", "cmd", "/C", moveCommand)

	// cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		fmt.Println("Error moving file:", err)
		os.Exit(1)
	}
	os.Exit(0)

	// fmt.Println("Upgrade completed for Windows")
}

func upgradeLinux() {
	tempDir, err := os.MkdirTemp("", "telego_upgrade")
	if err != nil {
		fmt.Println("Error creating temp directory:", err)
		return
	}

	defer os.RemoveAll(tempDir)

	if os.Geteuid() != 0 && !func() bool {
		// check sudo installed
		_, err := util.ModRunCmd.NewBuilder("sudo", "--version").BlockRun()
		return err == nil
	}() {
		fmt.Println(color.YellowString("Upgrade needed, please run as root"))
		os.Exit(1)
	}

	cmdPrefix := []string{}
	if os.Geteuid() != 0 {
		cmdPrefix = []string{"sudo"}
	}

	arch := util.GetCurrentArch()
	downloadPath := fmt.Sprintf("%s/telego", tempDir)
	util.DownloadFile(fmt.Sprintf("%s/telego_linux_%s", FILESERVER, arch), downloadPath)

	cmdArr := append(cmdPrefix, "chmod", "755", downloadPath)
	_, err = util.ModRunCmd.NewBuilder(cmdArr[0], cmdArr[1:]...).BlockRun()
	if err != nil {
		fmt.Println("Error setting executable permission:", err)
		return
	}

	// cmd := exec.Command("mv", downloadPath, "/usr/bin/telego")
	cmdArr = append(cmdPrefix, "mv", downloadPath, "/usr/bin/telego")
	_, err = util.ModRunCmd.NewBuilder(cmdArr[0], cmdArr[1:]...).BlockRun()
	if err != nil {
		fmt.Println("Error moving file:", err)
		return
	}

	fmt.Println("Upgrade completed on Linux, please rerun telego")
	os.Exit(0)
}
