package app

// 本模块为一个

import (
	"fmt"
	"net/http"
	"strings"
	"telego/util"
	"telego/util/gemini"
	"telego/util/platform_interface"
	"telego/util/yamlext"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

type UserMountServer struct{}

var ModJobMountAllUserStorageServer UserMountServer

func (m UserMountServer) JobCmdName() string {
	return "usmnt-server"
}

func (m UserMountServer) printfUserStorageSets(userStorageSets []util.UserOneStorageSet) string {
	var builder strings.Builder

	builder.WriteString("\n--------------------------------------------------\n")
	for idx, userStorage := range userStorageSets {
		builder.WriteString(fmt.Sprintf("Storage-%d\n", idx))
		builder.WriteString(fmt.Sprintf("type: %s, root-storage: %s\n", userStorage.Type, userStorage.RootStorage))
		for idx1, subPath := range userStorage.SubPaths {
			builder.WriteString(fmt.Sprintf("subpath-%d: %s\n", idx1, subPath))
		}
	}
	builder.WriteString("\n--------------------------------------------------\n")

	return builder.String()
}

func (m UserMountServer) doSftp(userStorageSets []util.UserOneStorageSet, username, password string) ([]util.UserMountsInfo, error) {
	SecretConfTypeStorageViewYaml := util.SecretConfTypeStorageViewYaml{}
	SecretConfTypeStorageViewYamlString, err := (util.MainNodeConfReader{}).ReadSecretConf(util.SecretConfTypeStorageViewYaml{})
	if err != nil {
		return nil, fmt.Errorf("UserMountServer.doSftp: Error ReadSecretConf: %v", err)
	}
	err = yamlext.UnmarshalAndValidate([]byte(SecretConfTypeStorageViewYamlString), &SecretConfTypeStorageViewYaml)
	if err != nil {
		return nil, fmt.Errorf("UserMountServer.doSftp: Error UnmarshalAndValidate: %v", err)
	}

	userMountsInfos := make([]util.UserMountsInfo, 0)
	for _, userStorageSet := range userStorageSets {
		mServer, aServer, err := SecretConfTypeStorageViewYaml.GetSftpServerByType(userStorageSet.Type)
		if err != nil {
			return nil, fmt.Errorf("UserMountServer.doSftp: Error getSftpServerByType: %v", err)
		}
		userMountsInfos = append(userMountsInfos, util.UserMountsInfo{
			UserStorage_: userStorageSet,
			ManageServer: mServer,
			AccessServer: aServer,
		})
	}

	err = util.ModSftpgo.CreateUserSpace(SecretConfTypeStorageViewYaml, username, password, userMountsInfos)
	if err != nil {
		return nil, fmt.Errorf("UserMountServer.doSftp: Error CreateUserSpace (userStorageSets: %s): %v", m.printfUserStorageSets(userStorageSets), err)
	}
	return userMountsInfos, nil
}

func (m UserMountServer) handleGetPath(c *gin.Context) {
	var req GetAllUserStorageLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("UserMountServer.handleGetPath: Invalid request payload: %v", err),
		})
		return
	}

	gBaseUrl, err := (util.MainNodeConfReader{}).ReadSecretConf(util.SecretConfTypeGeminiAPIUrl{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("UserMountServer.handleGetPath: Error reading gemini url: %v", err),
		})
		return
	}
	gBaseUrl = strings.TrimSpace(gBaseUrl)

	gServer, err := gemini.NewGeminiServer(gBaseUrl)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("UserMountServer.handleGetPath: Error initializing gemini server: %v", err),
		})
		return
	}

	// 与 Gemini 交互
	userStorageSets, err := platform_interface.GetAllStorageByUser(gServer, req.GeminiUserInfo.Username, req.GeminiUserInfo.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("UserMountServer.handleGetPath: Error getting all storage by user (username: %s, password: %s): %v", req.GeminiUserInfo.Username, req.GeminiUserInfo.Password, err),
		})
		return
	}

	// 返回集群信息，集群存储根目录列表
	userMountInfos, err := m.doSftp(userStorageSets, req.GeminiUserInfo.Username, req.GeminiUserInfo.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("UserMountServer.handleGetPath: Error performing SFTP operations: %v", err),
		})
		return
	}

	// 返回可挂载列表
	c.JSON(http.StatusOK, gin.H{
		"remote_infos": userMountInfos,
	})
}

func (m UserMountServer) listenRequest(port int) {
	r := gin.Default()
	r.POST("/get/user/storage/link", m.handleGetPath)
	if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}

func (m UserMountServer) Run() {
	m.listenRequest(8083)
}

func (m UserMountServer) ParseJob(getAllUserStorageLinkServerCmd *cobra.Command) *cobra.Command {
	getAllUserStorageLinkServerCmd.Run = func(_ *cobra.Command, _ []string) {
		m.Run()
	}

	return getAllUserStorageLinkServerCmd
}
