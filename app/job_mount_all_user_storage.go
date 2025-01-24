package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"telego/util"
	"telego/util/gemini"
	"telego/util/platform_interface"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type ModJobMountAllUserStorageStruct struct{}

var ModJobMountAllUserStorage ModJobMountAllUserStorageStruct

func (m ModJobMountAllUserStorageStruct) JobCmdName() string {
	return "usmnt"
}

// 请求结构体
type GetAllUserStorageLinkRequest struct {
	GeminiUserInfo gemini.GeminiUserInfo // gemini 平台账号
}

// 返回结构体
type GetAllUserStorageLinkResponse struct {
	RemoteInfos []util.UserMountsInfo `json:"remote_infos"` // 用于挂载的远程链接信息
}

func (m ModJobMountAllUserStorageStruct) inputUserLoginInfoByPlatform(platform platform_interface.Platform) (username, password string, err error) {
	var ok bool
	ok, username = util.StartTemporaryInputUI(
		color.GreenString("ModJobMountAllUserStorageStruct.Run: Mount 用户存储空间需要各平台用户名"),
		fmt.Sprintf("输入 %s 平台用户名", platform.GetPlatformName()),
		"回车确认，ctrl + c 取消",
	)
	if !ok {
		fmt.Println("User canceled input")
		err = fmt.Errorf("ModJobMountAllUserStorageStruct.inputUserLoginInfo: error input")
		return
	}

	ok, password = util.StartTemporaryInputUI(
		color.GreenString("ModJobMountAllUserStorageStruct.Run: Mount 用户存储空间需要各平台密码"),
		fmt.Sprintf("输入 %s 平台密码", platform.GetPlatformName()),
		"回车确认，ctrl + c 取消",
	)
	if !ok {
		fmt.Println("User canceled input")
		err = fmt.Errorf("ModJobMountAllUserStorageStruct.inputUserLoginInfo: error input")
		return
	}
	return
}

func (m ModJobMountAllUserStorageStruct) getLocalRootStorage() (string, error) {
	// 选定本地挂载根目录
	ok, localRootStorage := util.StartTemporaryInputUI(
		color.GreenString("选定本地挂载根目录"),
		"输入可用本地路径",
		"回车确认，ctrl + c 取消",
	)
	if !ok {
		fmt.Println("User canceled input")
		return "", fmt.Errorf("ModJobMountAllUserStorageStruct.getLocalRootStorage: Error user keyboard input")
	}

	// 对本地挂载根目录进行检验
	if _, err := os.Stat(localRootStorage); os.IsNotExist(err) {
		err := os.Mkdir(localRootStorage, 0755)
		if err != nil {
			return "", fmt.Errorf("ModJobMountAllUserStorageStruct.getLocalRootStorage: Error creating %s", localRootStorage)
		}
	} else {
		// // 检查是否是文件
		info, err := os.Stat(localRootStorage)
		if err != nil {
			return "", fmt.Errorf("ModJobMountAllUserStorageStruct.getLocalRootStorage: Error opening %s", localRootStorage)
		}

		if !info.IsDir() {
			return "", fmt.Errorf("ModJobMountAllUserStorageStruct.getLocalRootStorage: %s is a file, not a directory", localRootStorage)
		}

		files, err := os.ReadDir(localRootStorage)
		if err != nil {
			return "", fmt.Errorf("ModJobMountAllUserStorageStruct.getLocalRootStorage: Error opening %s", localRootStorage)
		}

		if len(files) != 0 {
			return "", fmt.Errorf("ModJobMountAllUserStorageStruct.getLocalRootStorage: %s not empty", localRootStorage)
		}
	}

	return localRootStorage, nil
}

func (m ModJobMountAllUserStorageStruct) Run() {
	// 输入用户信息
	gUsername, gPassword, err := m.inputUserLoginInfoByPlatform(platform_interface.GeminiPlatform{})
	if err != nil {
		fmt.Printf("ModJobMountAllUserStorageStruct.Run: %v", err)
	}

	// 初始化 http 请求
	req := GetAllUserStorageLinkRequest{
		GeminiUserInfo: gemini.GeminiUserInfo{
			Username: gUsername,
			Password: gPassword,
		},
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("ModJobMountAllUserStorageStruct.Run: Error marshalling request: %v", err)
	}

	// 执行请求，拿到 AllUserStorageLink
	serverUrl, err := (util.MainNodeConfReader{}).ReadPubConf(util.PubConfMountAllUserStorageServerUrl{})
	serverUrl = strings.TrimSpace(serverUrl)
	if err != nil {
		fmt.Printf("ModJobMountAllUserStorageStruct.Run: Get server ip error")
		return
	}
	client := &http.Client{Timeout: 60 * time.Second} // 设置 60 秒超时
	httpResp, err := client.Post(util.UrlJoin(serverUrl, "/get/user/storage/link"), "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Printf("ModJobMountAllUserStorageStruct.Run: Error getting response (server_url: %s, fullUrl: %s)", serverUrl, util.UrlJoin(serverUrl, "/mount_all_user_storage_server_url"))
		return
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body) // 读取错误信息
		fmt.Printf("ModJobMountAllUserStorageStruct.Run: Unexpected status: %d, %s", httpResp.StatusCode, string(body))
		return
	}
	var resp GetAllUserStorageLinkResponse
	decoder := json.NewDecoder(httpResp.Body)
	err = decoder.Decode(&resp)
	if err != nil {
		fmt.Printf("ModJobMountAllUserStorageStruct.Run: Error decoding response: %v", err)
		return
	}

	// 打印返回信息
	fmt.Print("\n--------------------------------------------------\n")
	for idx, userMountInfo := range resp.RemoteInfos {
		fmt.Printf("Storage-%d\n", idx)
		fmt.Printf("type: %s, access-server: %s, root-path: %s\n", userMountInfo.UserStorage_.Type, userMountInfo.AccessServer, userMountInfo.UserStorage_.RootStorage)
		for idx2, subPath := range userMountInfo.UserStorage_.SubPaths {
			fmt.Printf("subpath-%d: %s\n", idx2, subPath)
		}
	}
	fmt.Print("\n--------------------------------------------------\n")

	// 用户指定本地挂载点
	localRootStorage, err := m.getLocalRootStorage()
	if err != nil {
		fmt.Printf("ModJobMountAllUserStorageStruct.Run: Error asserting local mount point: %v", err)
	}

	// 调用 SshFs / rclone 进行挂载
	// // 检查安装 SshFs 和 Rclone
	if runtime.GOOS == "linux" && !(BinManagerSshFs{}).CheckInstalled() {
		err := NewBinManager(BinManagerSshFs{}).MakeSureWith()
		if err != nil {
			fmt.Printf("ModJobMountAllUserStorageStruct.Run: Error making sure sshfs: %v", err)
		}
	} else if runtime.GOOS == "windows" && (BinManagerRclone{}).CheckInstalled() {
		err := NewBinManager(BinManagerRclone{}).MakeSureWith()
		if err != nil {
			fmt.Printf("ModJobMountAllUserStorageStruct.Run: Error making sure rclone: %v", err)
		}
	}

	if runtime.GOOS == "linux" {
		for _, userMountInfo := range resp.RemoteInfos {
			localSubPath := path.Join(localRootStorage, userMountInfo.UserStorage_.Type, path.Base(userMountInfo.UserStorage_.RootStorage))
			if err := os.MkdirAll(localSubPath, 0755); err != nil {
				fmt.Printf("ModJobMountAllUserStorageStruct.Run: Error making local mount subpath: %v", err)
			}
			cmd := exec.Command("sshpass", "-p", req.GeminiUserInfo.Password, "sshfs", req.GeminiUserInfo.Username+"@"+userMountInfo.AccessServer+":"+path.Base(userMountInfo.UserStorage_.RootStorage), localSubPath)
			err := cmd.Run()
			if err != nil {
				fmt.Printf("ModJobMountAllUserStorageStruct.Run: Error mounting: %v", err)
			}
		}
	}
}

func (m ModJobMountAllUserStorageStruct) ParseJob(mountAllUserStorageCmd *cobra.Command) *cobra.Command {
	mountAllUserStorageCmd.Run = func(_ *cobra.Command, _ []string) {
		m.Run()
	}
	return mountAllUserStorageCmd
}
