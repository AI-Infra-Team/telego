package app

import (
	"github.com/spf13/cobra"
)

type ModJobMountAllUserStorageStruct struct{}

var ModJobMountAllUserStorage ModJobMountAllUserStorageStruct

func (m ModJobMountAllUserStorageStruct) JobCmdName() string {
	return "mount-all-user-storage"
}

// 请求结构体
type GetAllUserStorageLinkRequest struct {
	UserName string `json:"username"` // Telego 用户名
}

// 返回结构体
type GetAllUserStorageLinkResponse struct {
	RemoteLinks []string `json:"remote_links"` // 供 SshFs 进行挂载的远程链接
}

func (m ModJobMountAllUserStorageStruct) Run() {
	// 填入 GetAllUserStorageLinkRequest

	// 进行 http Get 请求

	// 接收结果，调用 ssh / rclone 进行挂载
}

func (m ModJobMountAllUserStorageStruct) ParseJob(mountAllUserStorageCmd *cobra.Command) *cobra.Command {
	mountAllUserStorageCmd.Run = func(_ *cobra.Command, _ []string) {
		m.Run()
	}
	return mountAllUserStorageCmd
}
