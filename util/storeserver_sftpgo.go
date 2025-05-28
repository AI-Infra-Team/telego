package util

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/barweiss/go-tuple"
	"github.com/fatih/color"
	"github.com/thoas/go-funk"
)

type SftpgoAuthResponse struct {
	AccessToken string `json:"access_token"`
}

type SftpgoVirtualFolder struct {
	Name       string `json:"name"`
	MappedPath string `json:"mapped_path"`
	Filesystem string `json:"filesystem"`
}

type ModSftpgoStruct struct {
}

var ModSftpgo ModSftpgoStruct

type UserMountsInfo struct {
	UserStorage_ UserOneStorageSet
	ManageServer string
	AccessServer string
}

type SftpgoFileSystem struct {
	RedactedSecret string `json:"redacted-secret"`
	Provider       int    `json:"provider"` // 0: local
}

type SftpgoFolderPayload struct {
	ID              int              `json:"id"`
	Name            string           `json:"name"`
	MappedPath      string           `json:"mapped_path"`
	Description     string           `json:"description"`
	UsedQuotaSize   int64            `json:"used_quota_size"`
	UsedQuotaFiles  int              `json:"used_quota_files"`
	LastQuotaUpdate int64            `json:"last_quota_update"`
	Users           []string         `json:"users"`
	Groups          []string         `json:"groups"`
	FileSystem      SftpgoFileSystem `json:"filesystem"`
}

type SftpgoUserDirPart struct {
	ID              int              `json:"id"`
	Name            string           `json:"name"`
	MappedPath      string           `json:"mapped_path"`
	Description     string           `json:"description"`
	UsedQuotaSize   int64            `json:"used_quota_size"`
	UsedQuotaFiles  int              `json:"used_quota_files"`
	LastQuotaUpdate int64            `json:"last_quota_update"`
	Users           []string         `json:"users"`
	FileSystem      SftpgoFileSystem `json:"filesystem"`
	VirtualPath     string           `json:"virtual_path"`
}

type SftpgoUserPayload struct {
	ID                   int                    `json:"id"`
	Status               int                    `json:"status"`
	Username             string                 `json:"username"`
	Password             string                 `json:"password"`
	HasPassword          bool                   `json:"has_password"`
	HomeDir              string                 `json:"home_dir"`
	UID                  int                    `json:"uid"`
	GID                  int                    `json:"gid"`
	MaxSessions          int                    `json:"max_sessions"`
	QuotaSize            int64                  `json:"quota_size"`
	QuotaFiles           int                    `json:"quota_files"`
	Permissions          map[string][]string    `json:"permissions"`
	UploadDataTransfer   int64                  `json:"upload_data_transfer"`
	DownloadDataTransfer int64                  `json:"download_data_transfer"`
	TotalDataTransfer    int64                  `json:"total_data_transfer"`
	CreatedAt            int64                  `json:"created_at"`
	UpdatedAt            int64                  `json:"updated_at"`
	Filters              map[string]interface{} `json:"filters"`
	VirtualFolders       []SftpgoUserDirPart    `json:"virtual_folders"`
	FileSystem           SftpgoFileSystem       `json:"filesystem"`
	FSCache              []interface{}          `json:"fs-cache"`
	GroupSettingsApplied bool                   `json:"group-settings-applied"`
	DeletedAt            int64                  `json:"deleted-at"`
}

func (ModSftpgoStruct) sftpgoAuth(serverAddr, admin, adminPassword string) (SftpgoAuthResponse, error) {

	fmt.Println(color.GreenString("authenticating %s", admin))
	// Step 1: Authenticate with the SFTPGo server
	loginURL := UrlJoin(serverAddr, "api/v2/token")
	resp, err := HttpAuth(loginURL, admin, adminPassword)
	if err != nil {
		return SftpgoAuthResponse{}, fmt.Errorf("%s failed to authenticate %s, err: %v", admin, loginURL, err)
	}

	var authResp SftpgoAuthResponse
	if err := json.Unmarshal(resp, &authResp); err != nil {
		return SftpgoAuthResponse{}, fmt.Errorf("failed to parse authentication response: %v", err)
	}

	return authResp, nil
}

func (ModSftpgoStruct) sftpgoRegisterHostDir(userName, storeName, server, mountPath string, authResp SftpgoAuthResponse) (SftpgoFolderPayload, error) {
	folderURL := fmt.Sprintf("%s/api/v2/folders", server)
	folderPayload := SftpgoFolderPayload{
		ID:             0,
		Name:           userName + "@" + storeName + ":" + strings.ReplaceAll(mountPath, "/", ":"),
		MappedPath:     mountPath, //filepath.Join("/share", tempDir), // the host path
		Description:    "",
		UsedQuotaSize:  0,
		UsedQuotaFiles: 0,
		Users:          []string{userName},
		Groups:         []string{},
		FileSystem: SftpgoFileSystem{
			RedactedSecret: "",
			Provider:       0, // 0: local
		},
	}

	folderData, _ := json.Marshal(folderPayload)
	req, err := http.NewRequest("POST", folderURL, bytes.NewBuffer(folderData))
	if err != nil {
		return SftpgoFolderPayload{}, fmt.Errorf("failed to create folder request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authResp.AccessToken))
	req.Header.Set("Content-Type", "application/json")

	folderResp, err := http.DefaultClient.Do(req)
	if err != nil {

		return SftpgoFolderPayload{}, fmt.Errorf("failed to create folder: %v", err)

	}
	defer folderResp.Body.Close()

	if folderResp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(folderResp.Body)
		if strings.Contains(string(body), "duplicated key not allowed") {
			// already exists
		} else {
			return SftpgoFolderPayload{}, fmt.Errorf("failed to create folder: %s", string(body))
		}
	}

	return folderPayload, nil
}

func (m ModSftpgoStruct) sftpgoCreateUser(server, userName, userPassword string,
	folders []SftpgoFolderPayload, virtualFolders []string, authResp SftpgoAuthResponse) error {

	fmt.Println(color.GreenString("creating user %s", userName))
	userURL := fmt.Sprintf("%s/api/v2/users", server)

	folderidx := 0
	newUserDirParts := funk.Map(folders, func(folder SftpgoFolderPayload) SftpgoUserDirPart {
		ret := SftpgoUserDirPart{
			ID:              0,
			Name:            folder.Name,
			MappedPath:      folder.MappedPath,
			Description:     "",
			UsedQuotaSize:   0,
			UsedQuotaFiles:  0,
			LastQuotaUpdate: 0,
			Users:           []string{userName},
			FileSystem: SftpgoFileSystem{
				RedactedSecret: "",
				Provider:       0, // https://github.com/sftpgo/sdk/blob/64fc18a344f9c87be4f028ffb7a851fad50976f0/filesystem.go#L20
				// 0: local
			},
			VirtualPath: virtualFolders[folderidx],
		}
		folderidx++
		return ret
	}).([]SftpgoUserDirPart)

	fmt.Println(color.GreenString("newUserDirParts %+v", newUserDirParts))

	userPayload := SftpgoUserPayload{
		ID:          0,
		Status:      1,
		Username:    userName,
		Password:    userPassword,
		HasPassword: true,
		HomeDir:     "", //filepath.Join(store.RootStorage, store.SubPaths[0]),
		UID:         0,
		GID:         0,
		MaxSessions: 0,
		QuotaSize:   0,
		QuotaFiles:  0,
		Permissions: map[string][]string{
			"/": []string{"*"},
		},
		UploadDataTransfer:   0,
		DownloadDataTransfer: 0,
		TotalDataTransfer:    0,
		CreatedAt:            0,
		UpdatedAt:            0,
		Filters:              map[string]interface{}{},
		VirtualFolders:       newUserDirParts,
		FileSystem: SftpgoFileSystem{
			RedactedSecret: "",
			Provider:       0, // https://github.com/sftpgo/sdk/blob/64fc18a344f9c87be4f028ffb7a851fad50976f0/filesystem.go#L20
			// 0: local
		},
		FSCache:              []interface{}{},
		GroupSettingsApplied: false,
		DeletedAt:            0,
	}

	userData, _ := json.Marshal(userPayload)
	req, err := http.NewRequest("POST", userURL, bytes.NewBuffer(userData))
	if err != nil {
		return fmt.Errorf("failed to create user request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authResp.AccessToken))
	req.Header.Set("Content-Type", "application/json")

	userResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(userResp.Body)
		if !strings.Contains(string(body), "duplicated key not allowed") {
			return fmt.Errorf("failed to create user: %s", string(body))
		} else {
			// update user
			fmt.Println(color.GreenString("user already exists, updating user"))
			userURL := fmt.Sprintf("%s/api/v2/users/%s", server, userName)
			userData, _ := json.Marshal(userPayload)
			req, err := http.NewRequest("PUT", userURL, bytes.NewBuffer(userData))
			if err != nil {
				return fmt.Errorf("failed to update user request: %v", err)
			}
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authResp.AccessToken))
			req.Header.Set("Content-Type", "application/json")

			userResp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("failed to update user: %v", err)
			}
			defer userResp.Body.Close()
			if userResp.StatusCode != http.StatusOK {
				return fmt.Errorf("failed to update user: %s", string(body))
			}
		}

	}

	return nil
}

// serveraddr -> lock
// to make sure the sftpgo server will not go crazy
var createAdminFoldersLocks = sync.Map{}

type OneStoreRootFoldersType struct {
	sshAddr     string
	rootFolders []string
}

var eachServerRootFoldersAndSsh = map[string]OneStoreRootFoldersType{}

// the dirs not exist on host will also be created
// this function should not read config
// the config should already be read
func (m ModSftpgoStruct) CreateUserSpace(config SecretConfTypeStorageViewYaml,
	UserName string,
	UserPassword string,
	userMountsInfos []UserMountsInfo) error {

	admin := config.StoreManageAdmin
	adminPassword := config.StoreManageAdminPass

	fmt.Println(color.GreenString("create user space for %s", UserName))

	// create admin folders for check subpath exist
	if UserName != config.StoreManageAdmin+"_specforcheck" {
		eachServerRootFoldersAndSsh = map[string]OneStoreRootFoldersType{}
		admin := admin + "_specforcheck"
		// collect each server root folders
		for _, store := range config.Storages {
			if _, ok := eachServerRootFoldersAndSsh[store.StoreManageServer]; !ok {
				eachServerRootFoldersAndSsh[store.StoreManageServer] = OneStoreRootFoldersType{
					sshAddr:     store.StoreAccessServer,
					rootFolders: []string{store.MountPath},
				}
			} else {
				rf := eachServerRootFoldersAndSsh[store.StoreManageServer].rootFolders
				rf = append(rf, store.MountPath)
				eachServerRootFoldersAndSsh[store.StoreManageServer] = OneStoreRootFoldersType{
					sshAddr:     store.StoreAccessServer,
					rootFolders: rf,
				}
			}
		}
		fmt.Println(color.GreenString("eachServerRootFoldersAndSsh %+v", eachServerRootFoldersAndSsh))
		fmt.Println()

		// create admin
		fmt.Println(color.GreenString("create admin %s's folders", admin))
		for server, rootFolders := range eachServerRootFoldersAndSsh {
			lock_, _ := createAdminFoldersLocks.LoadOrStore(server, &sync.Mutex{})
			lock := lock_.(*sync.Mutex)
			lock.Lock()
			userOneStorage := UserOneStorageSet{
				RootStorage: "/",
				SubPaths: funk.Map(rootFolders.rootFolders, func(folder string) string {
					return strings.TrimPrefix(folder, "/")
				}).([]string),
			}
			fmt.Println(color.GreenString("create admin %s with one server folders %+v", server, userOneStorage))
			err := m.CreateUserSpace(config, admin, adminPassword, []UserMountsInfo{
				{
					AccessServer: eachServerRootFoldersAndSsh[server].sshAddr,
					ManageServer: server,
					UserStorage_: userOneStorage,
				},
			})
			if err != nil {
				err = fmt.Errorf("failed to create user space for admin %s: %v", server, err)
				fmt.Println(color.RedString(err.Error()))
				return err
			} else {
				fmt.Println(color.GreenString("create admin %s with folders success", server))
			}
			fmt.Println()
			// config rclone in sync mode
			// use base64 ssh addr as name
			sshAddr := eachServerRootFoldersAndSsh[server].sshAddr
			fmt.Println(color.GreenString("configuringrclone for admin %s", sshAddr))
			err = NewRcloneConfiger(RcloneConfigTypeSsh{}, base64.RawURLEncoding.EncodeToString([]byte(sshAddr)), sshAddr).
				WithUser(admin, adminPassword).
				DoConfig()
			if err != nil {
				fmt.Println(color.RedString("failed to config rclone for admin %s", sshAddr))
				return err
			}

			fmt.Println(color.GreenString("configuring rclone for admin %s success", sshAddr))
			lock.Unlock()
		}

	}

	// collect each server user folders
	type RootStorageStr string
	type eachServerFoldersType struct {
		folders  []tuple.T2[SftpgoFolderPayload, RootStorageStr]
		authResp SftpgoAuthResponse
	}
	eachServerFolders := map[string]eachServerFoldersType{}
	for _, userMountsInfo := range userMountsInfos {
		// auth
		authResp, err := m.sftpgoAuth(userMountsInfo.ManageServer, admin, adminPassword)
		if err != nil {
			return err
		}

		fmt.Println(color.GreenString("auth success %s", userMountsInfo.ManageServer))

		// register host dir
		rootFolder := userMountsInfo.UserStorage_.RootStorage
		for _, subpath := range userMountsInfo.UserStorage_.SubPaths {
			fmt.Println()
			fmt.Println(color.GreenString("register host dir %s for user %s", filepath.Join(rootFolder, subpath), UserName))
			checkSubPathExist := func(subpath string) bool {
				fmt.Println(color.GreenString("check subpath exist %s", filepath.Join(rootFolder, subpath)))

				if UserName != config.StoreManageAdmin+"_specforcheck" && !funk.Contains(eachServerRootFoldersAndSsh[userMountsInfo.ManageServer].rootFolders, rootFolder) {
					fmt.Println(color.RedString("root folder (%s) not in eachServerRootFoldersAndSsh (%+v)",
						rootFolder, eachServerRootFoldersAndSsh[userMountsInfo.ManageServer].rootFolders))
					return false
				}
				sshAddr := eachServerRootFoldersAndSsh[userMountsInfo.ManageServer].sshAddr
				// use rclone to check
				res := RcloneCheckDirExist(base64.RawURLEncoding.EncodeToString([]byte(sshAddr)) + ":" + filepath.Join(rootFolder, subpath))
				if !res {
					fmt.Println(color.RedString("subpath not exist"))
				} else {
					fmt.Println(color.GreenString("check subpath exist %s result %+v", filepath.Join(rootFolder, subpath), res))
				}

				return res
			}

			if UserName != config.StoreManageAdmin+"_specforcheck" && !checkSubPathExist(subpath) {
				fmt.Println(color.GreenString("subpath not exist, skip"))
				continue
			}
			folder, err := m.sftpgoRegisterHostDir(UserName,
				userMountsInfo.UserStorage_.RootStorage,
				userMountsInfo.ManageServer,
				filepath.Join(userMountsInfo.UserStorage_.RootStorage, subpath),
				authResp)
			if err != nil {
				return err
			}
			if _, ok := eachServerFolders[userMountsInfo.ManageServer]; !ok {
				eachServerFolders[userMountsInfo.ManageServer] = eachServerFoldersType{
					folders:  []tuple.T2[SftpgoFolderPayload, RootStorageStr]{},
					authResp: authResp,
				}
			}
			value := eachServerFolders[userMountsInfo.ManageServer]
			value.folders = append(value.folders, tuple.New2(folder, RootStorageStr(userMountsInfo.UserStorage_.RootStorage)))
			eachServerFolders[userMountsInfo.ManageServer] = value

			fmt.Println(color.GreenString("register host dir success %s", folder.Name))
		}
		fmt.Println()
	}

	// create user
	for server, folders_userStorage_authResp := range eachServerFolders {
		folders := funk.Map(folders_userStorage_authResp.folders, func(folder tuple.T2[SftpgoFolderPayload, RootStorageStr]) SftpgoFolderPayload {
			return folder.V1
		}).([]SftpgoFolderPayload)
		virtualFolders := funk.Map(folders_userStorage_authResp.folders, func(folder tuple.T2[SftpgoFolderPayload, RootStorageStr]) string {
			return filepath.Join(string(folder.V2), filepath.Base(folder.V1.MappedPath))
		}).([]string)

		authResp := folders_userStorage_authResp.authResp

		fmt.Println(color.GreenString("creating user %s with folders %+v", UserName, folders))
		err := m.sftpgoCreateUser(server,
			UserName,
			UserPassword,
			folders,
			virtualFolders,
			authResp)
		if err != nil {
			return err
		}
		fmt.Println(color.GreenString("created user %s with folders %+v success", UserName, folders))
	}
	return nil
}

func (m ModSftpgoStruct) CreateTempSpace(serverAddr, admin, adminPassword,
	tempUser, tempPassword, tempDir string) error {

	authResp, err := m.sftpgoAuth(serverAddr, admin, adminPassword)
	if err != nil {
		return err
	}

	// Step 2: Create the virtual folder
	folderPayload, err := m.sftpgoRegisterHostDir(tempUser, tempDir, serverAddr, filepath.Join("/share", tempDir), authResp)
	if err != nil {
		return err
	}

	// Step 3: Create the temporary user
	err = m.sftpgoCreateUser(serverAddr,
		tempUser,
		tempPassword,
		[]SftpgoFolderPayload{folderPayload},
		[]string{"/"},
		authResp,
	)

	return nil
}
