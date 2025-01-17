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

// func (m ModJobGetAllUserStorageLinkServerStruct) SyncStoreManageServerVirtualPath(userStorages []util.UserStorage) error {
// 	ExpandUserStorageListWithCustom := func(userStorage []util.UserStorage) []util.UserStorage {
// 		return userStorage
// 	}

// 	collectMountInfo := func(userStorage []util.UserStorage) (CollectMountInfo, error) {
// 		configstr, err := util.MainNodeConfReader{}.ReadSecretConf(util.SecretConfTypeStorageViewYaml{})
// 		if err != nil {
// 			return CollectMountInfo{}, err
// 		}

// 		config := util.SecretConfTypeStorageViewYaml{}
// 		err = yamlext.UnmarshalAndValidate([]byte(configstr), &config)
// 		if err != nil {
// 			return CollectMountInfo{}, err
// 		}

// 		return CollectMountInfo{
// 			ManageAdmin:     config.StoreManageAdmin,
// 			ManageAdminPass: config.StoreManageAdminPass,
// 			EachStoreInfo: funk.Map(userStorage, func(userStorage util.UserStorage) util.UserMountsInfo {
// 				return util.UserMountsInfo{
// 					UserStorage_: userStorage,
// 					MountPath:    config.Storages[userStorage.RootStorage].MountPath,
// 					ManageServer: config.Storages[userStorage.RootStorage].StoreManageServer,
// 					AccessServer: config.Storages[userStorage.RootStorage].StoreAccessServer,
// 				}
// 			}).([]util.UserMountsInfo),
// 		}, nil
// 	}
// 	// 0. 拿到 []UserStorage, 代表用户可以访问的多个存储集群以及子路径
// 	// gemeni-nm:
// 	//
// 	//	subpath1
// 	//	subpath2
// 	//	subpath3
// 	//
// 	// gemini-sh:
// 	//
// 	//	subpath1
// 	//	subpath2
// 	//	subpath3

// 	// 1. 读取配置扩展用户可以访问的目录集
// 	userStorages = ExpandUserStorageListWithCustom(userStorages)

// 	// 2. 根据配置确认每个存储的挂载路径，虚拟路径，管理员，管理员密码，管理server访问路径，数据访问路径
// 	collectedMountInfo, err := collectMountInfo(userStorages)
// 	if err != nil {
// 		return err
// 	}

// 	// 3. 将合并后的目录集写入到 storemanage-server 的虚拟路径中
// }
