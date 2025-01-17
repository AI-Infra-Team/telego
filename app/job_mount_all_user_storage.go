package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"telego/util"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type ModJobMountAllUserStorageStruct struct{}

var ModJobMountAllUserStorage ModJobMountAllUserStorageStruct

func (m ModJobMountAllUserStorageStruct) JobCmdName() string {
	return "mount-all-user-storage"
}

// 请求结构体
type GetAllUserStorageLinkRequest struct {
	UserName string `json:"username"` // 第三方平台用户名
	PassWord string `json:"password"` // 第三方平台密码
}

// 返回结构体
type GetAllUserStorageLinkResponse struct {
	RemoteInfos []struct {
		Link    string `json:"link"`    // 远程链接
		Storage string `json:"storage"` // 存储信息
	} `json:"remote_infos"` // 用于挂载的远程链接信息
}

func (m ModJobMountAllUserStorageStruct) Run() {
	var userName, passWord string

	// 登录验证
	userInfoFile := util.WorkspaceDir() + "/config/userinfo"
	if _, err := os.Stat(userInfoFile); err == nil {
		// // 文件存在，尝试读取文件中的内容
		content, err := os.ReadFile(userInfoFile)
		if err != nil {
			fmt.Println("ModJobMountAllUserStorageStruct.Run: Error reading userinfo file:", err)
			return
		}

		// // 解析文件内容，查找 username 和 password
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line) // 去掉行首尾的空格
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "username:") {
				userName = strings.TrimSpace(strings.TrimPrefix(line, "username:"))
			} else if strings.HasPrefix(line, "password:") {
				passWord = strings.TrimSpace(strings.TrimPrefix(line, "password:"))
			}
		}

		// // 如果从文件中读取了有效的 username 和 password，则直接使用
		if userName != "" && passWord != "" {
			fmt.Println("ModJobMountAllUserStorageStruct.Run: Found credentials in userinfo file.")
		} else {
			fmt.Println("ModJobMountAllUserStorageStruct.Run: No valid credentials found in userinfo file, using UI for input.")
		}
	} else {
		// // 文件不存在，使用 UI 获取输入
		var ok bool

		ok, userName = util.StartTemporaryInputUI(
			color.GreenString("ModJobMountAllUserStorageStruct.Run: Mount 用户存储空间需要鉴权，userName"),
			"输入 userName",
			"回车确认，ctrl + c 取消",
		)
		if !ok {
			fmt.Println("User canceled input")
			return
		}

		ok, passWord = util.StartTemporaryInputUI(
			color.GreenString("ModJobMountAllUserStorageStruct.Run: Mount 用户存储空间需要鉴权，passWord"),
			"输入 passWord",
			"回车确认，ctrl + c 取消",
		)
		if !ok {
			fmt.Println("User canceled input")
			return
		}

		if userName == "" || passWord == "" {
			fmt.Println("ModJobMountAllUserStorageStruct.Run: Input username or password empty")
			return
		}
	}

	// 初始化 http 请求
	req := GetAllUserStorageLinkRequest{
		UserName: userName,
		PassWord: passWord,
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("ModJobMountAllUserStorageStruct.Run: Error marshalling request: %v", err)
	}

	// 执行请求，拿到 AllUserStorageLink
	serverUrl, err := (util.MainNodeConfReader{}).ReadPubConf(util.PubConfUserStorageServerUrl{})
	if err != nil {
		fmt.Printf("ModJobMountAllUserStorageStruct.Run: Get server ip error")
	}
	httpResp, err := http.Post(path.Join(serverUrl, "/getalluserstoragelink"), "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Printf("ModJobMountAllUserStorageStruct.Run: Error Getting response")
	}
	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body) // 读取错误信息
		fmt.Printf("ModJobMountAllUserStorageStruct.Run: unexpected status: %d, %s", httpResp.StatusCode, string(body))
		return
	}
	var resp GetAllUserStorageLinkResponse
	defer httpResp.Body.Close()
	decoder := json.NewDecoder(httpResp.Body)
	err = decoder.Decode(&resp)
	if err != nil {
		fmt.Printf("ModJobMountAllUserStorageStruct.Run: Error decoding response: %v\n", err)
		return
	}

	// 选定本地挂载根目录
	ok, localRootStorage := util.StartTemporaryInputUI(
		color.GreenString("选定本地挂载根目录"),
		"输入可用本地路径",
		"回车确认，ctrl + c 取消",
	)
	if !ok {
		fmt.Println("User canceled input")
		return
	}

	// 对本地挂载根目录进行检验
	if _, err := os.Stat(localRootStorage); os.IsNotExist(err) {
		err := os.Mkdir(localRootStorage, 0755)
		if err != nil {
			fmt.Printf("ModJobMountAllUserStorageStruct.Run: Error creating %s", localRootStorage)
			return
		}
	} else {
		// // 检查是否是文件
		info, err := os.Stat(localRootStorage)
		if err != nil {
			fmt.Printf("ModJobMountAllUserStorageStruct.Run: Error opening %s\n", localRootStorage)
			return
		}

		if !info.IsDir() {
			fmt.Printf("ModJobMountAllUserStorageStruct.Run: %s is a file, not a directory\n", localRootStorage)
			return
		}

		files, err := os.ReadDir(localRootStorage)
		if err != nil {
			fmt.Printf("ModJobMountAllUserStorageStruct.Run: Error opening %s", localRootStorage)
			return
		}

		if len(files) != 0 {
			fmt.Printf("ModJobMountAllUserStorageStruct.Run: %s not empty", localRootStorage)
			return
		}
	}

	// 调用 SshFs / rclone 进行挂载
	// // 检查安装 SshFs 和 Rclone
	if runtime.GOOS == "linux" && !(BinManagerSshFs{}).CheckInstalled() {
		NewBinManager(BinManagerSshFs{}).MakeSureWith()
	} else if runtime.GOOS == "windows" && (BinManagerRclone{}).CheckInstalled() {
		NewBinManager(BinManagerRclone{}).MakeSureWith()
	}

	// // 分系统挂载
	for _, elem := range resp.RemoteInfos {
		if runtime.GOOS == "linux" {
			err := ModJobSshFs.doMount(&sshFsArgv{
				remotePath: elem.Link,
				localPath:  path.Join(localRootStorage, elem.Storage),
			})
			if err != nil {
				fmt.Printf("ModJobMountAllUserStorageStruct.Run: linux mount %s to %s error", elem.Link, path.Join(localRootStorage, elem.Storage))
			}
		} else if runtime.GOOS == "windows" {
			err := ModJobRclone.doMount(&rcloneMountArgv{
				remotePath: elem.Link,
				localPath:  path.Join(localRootStorage, elem.Storage),
			})
			if err != nil {
				fmt.Printf("ModJobMountAllUserStorageStruct.Run: windows mount %s to %s error", elem.Link, path.Join(localRootStorage, elem.Storage))
			}
		}
	}

	// 写入配置文件
	err = os.WriteFile(userInfoFile, []byte("username: "+userName+"\npassword: "+passWord+"\n"), 0644)
	if err != nil {
		fmt.Println("ModJobMountAllUserStorageStruct.Run: Error writing to userinfo file:", err)
		return
	}

	fmt.Println("ModJobMountAllUserStorageStruct.Run: Configuration saved successfully.")
}

func (m ModJobMountAllUserStorageStruct) ParseJob(mountAllUserStorageCmd *cobra.Command) *cobra.Command {
	mountAllUserStorageCmd.Run = func(_ *cobra.Command, _ []string) {
		m.Run()
	}
	return mountAllUserStorageCmd
}
