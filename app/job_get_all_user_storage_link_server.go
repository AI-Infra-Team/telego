package app

// 本模块为一个

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

type ModJobGetAllUserStorageLinkServerStruct struct{}

var ModJobGetAllUserStorageLinkServer ModJobGetAllUserStorageLinkServerStruct

func (m ModJobGetAllUserStorageLinkServerStruct) JobCmdName() string {
	return "get-all-user-storage-server"
}

func (m ModJobGetAllUserStorageLinkServerStruct) handleGetPath(c *gin.Context) {
	userName := c.DefaultQuery("username", "")
	if userName == "" {
		c.JSON(400, gin.H{
			"error": "username is required",
		})
		return
	}

	// 处理

	// 未完成
	c.JSON(200, GetAllUserStorageLinkResponse{
		RemoteLinks: []string{"path1", "path2", "path3"},
	})
}

func (m ModJobGetAllUserStorageLinkServerStruct) listenRequest(port int) {
	r := gin.Default()
	r.GET("/getalluserstoragelink", m.handleGetPath)
	r.Run(fmt.Sprintf(":%d", port))
}

func (m ModJobGetAllUserStorageLinkServerStruct) Run() {
	go m.listenRequest(8083)
}

func (m ModJobGetAllUserStorageLinkServerStruct) ParseJob(getAllUserStorageLinkServerCmd *cobra.Command) *cobra.Command {
	getAllUserStorageLinkServerCmd.Run = func(_ *cobra.Command, _ []string) {
		m.Run()
	}

	return getAllUserStorageLinkServerCmd
}
