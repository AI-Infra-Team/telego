package app

import (
	"fmt"

	"github.com/spf13/cobra"
)

type ModJobCreateNewUserStruct struct{}

type NewUserInfoStruct struct {
	// apiserverIP string
	// username    string
	// namespace   string
	// ...
}

var ModJobCreateNewUser ModJobCreateNewUserStruct

func getUserInfo() (*NewUserInfoStruct, error) {
	// TODO
	return nil, nil
}

func CreateNewUser() error {
	if userInfo, err := getUserInfo(); err != nil {
		return err
	} else {
		fmt.Print(userInfo)
		// TODO
		// 参考 teleyard-template/update_config/access_control 中的 {bash 脚本 + 集群 RBAC 设置} 进行创建
	}
	return nil
}

func (ModJobCreateNewUserStruct) JobCmdName() string {
	return "create-new-user-kubeconfig"
}

func (ModJobCreateNewUserStruct) ParseJob(createUserCmd *cobra.Command) *cobra.Command {
	createUserCmd.Run = func(_ *cobra.Command, _ []string) {
		CreateNewUser()
	}
	return createUserCmd
}
