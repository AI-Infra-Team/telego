package util

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"telego/util/yamlext"

	"github.com/fatih/color"
)

const MainNodeRcloneName = "remote"

// localPath must be a directory
func FetchFromMainNode(remotePath string, localPath string) {

	if !PathIsAbsolute(remotePath) {
		fmt.Println(color.RedString("remotePath should be absolute path"))
		os.Exit(1)
	}

	ConfigMainNodeRcloneIfNeed()

	RcloneSyncDirOrFileToDir(fmt.Sprintf("%s:%s", MainNodeRcloneName, remotePath), localPath)
}

func ReadStrFromMainNode(remotePath string) (string, error) {
	if !PathIsAbsolute(remotePath) {
		fmt.Println(color.RedString("remotePath should be absolute path"))
		os.Exit(1)
	}

	ConfigMainNodeRcloneIfNeed()

	content, err := ModRunCmd.NewBuilder("rclone", "cat", MainNodeRcloneName+":"+remotePath).BlockRun()
	if err != nil {
		return "", fmt.Errorf("read str from main node failed, err: %v, content: %s", err, content)
	}
	content = strings.TrimSpace(content)
	return content, nil
}

// type UploadToMainNodeRes struct {
// 	localPathExist  bool
// 	remotePathExist bool
// 	uploadFail      bool
// 	err             error
// }

// localPath is relative, such as install/bin_rclone
//
// remotePath is root path, such as /teledeploy
//
// localDirContent will be sync to {remotePath}/{localPath}, such as /teledeploy/install/bin_rclone
func UploadToMainNode(localPath string, remotePath string) {

	// if !PathIsAbsolute(remotePath) {
	// 	fmt.Println(color.RedString("remotePath should be absolute path"))
	// 	os.Exit(1)
	// }

	ConfigMainNodeRcloneIfNeed()

	RcloneSyncDirOrFileToDir(localPath, fmt.Sprintf("%s:%s", MainNodeRcloneName, remotePath))
}

var MainNodeFileServerURL = fmt.Sprintf("http://%s:8003", MainNodeIp)

type MainNodeConfWriter struct{}

func (r MainNodeConfWriter) writeConf(path0 string, remotedir string, content string) error {
	// create temp dir
	tempdir, err := os.MkdirTemp("", "mainnode-conf-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %v", err)
	}
	// create temp file
	tempfile, err := os.Create(filepath.Join(tempdir, path0))
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	// write content to temp file
	if _, err := tempfile.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to temp file: %v", err)
	}
	// set file permission to user read-only
	if err := os.Chmod(tempfile.Name(), 0400); err != nil {
		return fmt.Errorf("failed to set temp file permissions: %v", err)
	}
	// copy temp file to remote
	err = RcloneSyncDirOrFileToDir(tempfile.Name(), fmt.Sprintf("%s:%s", MainNodeRcloneName, remotedir))
	if err != nil {
		return fmt.Errorf("failed to copy temp file to remote: %v", err)
	}
	// return error if any
	return nil
}

func (r MainNodeConfWriter) WriteSecretConf(path0 SecretConfType, content string) error {
	return r.writeConf(path0.SecretConfPath(), "/teledeploy_secret/config", content)
}
func (r MainNodeConfWriter) WritePubConf(path0 PubConfType, content string) error {
	return r.writeConf(path0.PubConfPath(), "/teledeploy/config", content)
}

type MainNodeConfReader struct{}

func UrlJoin(path ...string) string {
	res := ""
	for _, p := range path {
		if res == "" {
			res = p
		} else if strings.HasPrefix(p, "/") {
			res += p
		} else {
			res += "/" + p
		}
		if strings.HasSuffix(res, "/") {
			res = strings.TrimRight(res, "/")
		}
	}
	return res
}

type ConfTypeBase interface {
	Template() string
}

type PubConfType interface {
	ConfTypeBase
	PubConfPath() string
}

func NewPubConfType(path string) PubConfType {
	switch path {
	case PubConfTypeImgUploaderUrl{}.PubConfPath():
		return PubConfTypeImgUploaderUrl{}
	default:
		return nil
	}
}

// image_uploader_url

type PubConfTypeImgUploaderUrl struct{}

var _ PubConfType = PubConfTypeImgUploaderUrl{}

func (r PubConfTypeImgUploaderUrl) PubConfPath() string {
	return "img_uploader_url"
}

func (r PubConfTypeImgUploaderUrl) Template() string {
	return "http://127.0.0.1:8002"
}

// mount_all_user_storage_server_url
type PubConfMountAllUserStorageServerUrl struct{}

var _ PubConfType = PubConfMountAllUserStorageServerUrl{}

func (r PubConfMountAllUserStorageServerUrl) PubConfPath() string {
	return "mount_all_user_storage_server_url"
}

func (r PubConfMountAllUserStorageServerUrl) Template() string {
	return "http://127.0.0.1:8002"
}

type SecretConfType interface {
	ConfTypeBase
	SecretConfPath() string
}

func NewSecretConfType(t string) SecretConfType {
	switch t {
	case SecretConfTypeAdminKubeconfig{}.SecretConfPath():
		return SecretConfTypeAdminKubeconfig{}
	case SecretConfTypeImgRepo{}.SecretConfPath():
		return SecretConfTypeImgRepo{}
	case SecretConfTypeSshPrivate{}.SecretConfPath():
		return SecretConfTypeSshPrivate{}
	case SecretConfTypeSshPublic{}.SecretConfPath():
		return SecretConfTypeSshPublic{}
	case SecretConfTypeGeminiAPIUrl{}.SecretConfPath():
		return SecretConfTypeGeminiAPIUrl{}
	case SecretConfTypeStorageViewYaml{}.SecretConfPath():
		return SecretConfTypeStorageViewYaml{}
	default:
		return nil
	}
}

// admin_kubeconfig
type SecretConfTypeAdminKubeconfig struct{}

var _ SecretConfType = SecretConfTypeAdminKubeconfig{}

func (r SecretConfTypeAdminKubeconfig) SecretConfPath() string {
	return "admin_kubeconfig"
}

func (r SecretConfTypeAdminKubeconfig) Template() string {
	return "# Just the kubeconfig content"
}

// img_repo
type SecretConfTypeImgRepo struct{}

var _ SecretConfType = SecretConfTypeImgRepo{}

func (r SecretConfTypeImgRepo) SecretConfPath() string {
	return "img_repo"
}

func (r SecretConfTypeImgRepo) Template() string {
	template := yamlext.GenerateYAMLTemplate(ContainerRegistryConf{
		User:     "Registry default user for cri(containerd)",
		Password: "Registry default user's password for cri(containerd)",
		Tls:      nil, // we don't need tls Node
	})

	return template
}

// ssh_private
type SecretConfTypeSshPrivate struct{}

var _ SecretConfType = SecretConfTypeSshPrivate{}

func (r SecretConfTypeSshPrivate) SecretConfPath() string {
	return "ssh_private"
}

func (r SecretConfTypeSshPrivate) Template() string {
	return "# Just the ssh private key content"
}

// ssh_public
type SecretConfTypeSshPublic struct{}

var _ SecretConfType = SecretConfTypeSshPublic{}

func (r SecretConfTypeSshPublic) SecretConfPath() string {
	return "ssh_public"
}

func (r SecretConfTypeSshPublic) Template() string {
	return "# Just the ssh public key content"
}

// storage_view

type StorageViewYamlModelOneStore struct {
	Type              string `yaml:"type"`
	StoreManageServer string `yaml:"storemanage-server"`
	StoreAccessServer string `yaml:"storeaccess-server"`
	MountPath         string `yaml:"path"`
}

// currently we use sftpgo
type SecretConfTypeStorageViewYaml struct {
	Storages             map[string]StorageViewYamlModelOneStore `yaml:"storages"`
	StoreManageAdmin     string                                  `yaml:"storemanage-admin"`
	StoreManageAdminPass string                                  `yaml:"storemanage-adminpass"`
}

var _ SecretConfType = SecretConfTypeStorageViewYaml{}

func (r SecretConfTypeStorageViewYaml) SecretConfPath() string {
	return "storage_view"
}

func (r SecretConfTypeStorageViewYaml) Template() string {
	return yamlext.GenerateYAMLTemplate(SecretConfTypeStorageViewYaml{
		Storages: map[string]StorageViewYamlModelOneStore{
			"gemini-nm": StorageViewYamlModelOneStore{
				Type:              "sftpgo",
				StoreManageServer: "http://xxxx:xxxx",
				StoreAccessServer: "http://xxxx:xxxx",
				MountPath:         "/gemini-nm",
			},
		},
		StoreManageAdmin:     "j8k2l9m3n4",
		StoreManageAdminPass: "p7q5r2s8t1",
	})
}

func (r SecretConfTypeStorageViewYaml) GetSftpServerByType(sType string) (storeManageServer, storeAccessServer string, err error) {
	for _, storage := range r.Storages {
		if storage.Type == sType {
			storeManageServer = storage.StoreManageServer
			storeAccessServer = storage.StoreAccessServer
			err = nil
			return
		}
	}
	err = fmt.Errorf("ModJobMountAllUserStorageServerStruct.getSftpServerByType: No sftp server found")
	return
}

// Gemini api URL
type SecretConfTypeGeminiAPIUrl struct{}

var _ SecretConfType = SecretConfTypeGeminiAPIUrl{}

func (r SecretConfTypeGeminiAPIUrl) SecretConfPath() string {
	return "gemini_api_url"
}

func (r SecretConfTypeGeminiAPIUrl) Template() string {
	return "http://127.0.0.1:8002"
}

type ConfCacheStruct struct {
	pub    map[string]string
	secret map[string]string
}

var ConfCache *ConfCacheStruct = &ConfCacheStruct{
	pub:    make(map[string]string),
	secret: make(map[string]string),
}

func (c *ConfCacheStruct) tryReadPub(v PubConfType) *string {
	res, ok := c.pub[v.PubConfPath()]
	if !ok {
		return nil
	}
	return &res
}

func (c *ConfCacheStruct) tryReadSecret(v SecretConfType) *string {
	res, ok := c.secret[v.SecretConfPath()]
	if !ok {
		return nil
	}
	return &res
}

func (c *ConfCacheStruct) cachePub(k PubConfType, v string) {
	c.pub[k.PubConfPath()] = v
}

func (c *ConfCacheStruct) cacheSecret(k SecretConfType, v string) {
	c.secret[k.SecretConfPath()] = v
}

func (r MainNodeConfReader) ReadPubConf(path0 PubConfType) (string, error) {
	if !FileServerAccessible() {
		fmt.Println(color.RedString(
			"file server is not accessible, " +
				"please first init file server with 'telego cmd --cmd /update_config/start_mainnode_fileserver'"))
		os.Exit(1)
	}

	cached := ConfCache.tryReadPub(path0)
	if cached != nil {
		return *cached, nil
	}
	res := ""
	err := errors.New("unknown conf path")
	baseurl := UrlJoin(MainNodeFileServerURL, "config")
	res, err = ReadHttpSmallFile(UrlJoin(baseurl, path0.PubConfPath()))
	// cache
	if err == nil {
		ConfCache.cachePub(path0, res)
	} else {
		err = fmt.Errorf("%v, template: %s", err, path0.Template())
	}
	return res, err
}

func (r MainNodeConfReader) ReadSecretConf(path0 SecretConfType) (string, error) {
	if !FileServerAccessible() {
		fmt.Println(color.RedString(
			"file server is not accessible, " +
				"please first init file server with 'telego cmd --cmd /update_config/start_mainnode_fileserver'"))
		os.Exit(1)
	}

	cached := ConfCache.tryReadSecret(path0)
	if cached != nil {
		return *cached, nil
	}

	base := "/teledeploy_secret/config"
	confpath := filepath.Join(base, path0.SecretConfPath())
	if confpath != "" {
		Logger.Debugf("reading secret conf " + path0.SecretConfPath())
		res, err := ReadStrFromMainNode(confpath)
		if err != nil {
			return "", fmt.Errorf("read secret conf %s failed: %v, template: %s", confpath, err, path0.Template())
		}
		// save cache
		ConfCache.cacheSecret(path0, res)
		return res, nil
	} else {
		return "", errors.New("unknown conf path")
	}
}

type cacheFileServerAccessibleStruct struct {
	accessible bool
}

var cacheFileServerAccessible *cacheFileServerAccessibleStruct
var cacheFileServerAccessibleOnce sync.Once

func FileServerAccessible() bool {
	// if cacheFileServerAccessible == nil {
	cacheFileServerAccessibleOnce.Do(func() {
		cacheFileServerAccessible = &cacheFileServerAccessibleStruct{
			accessible: NewCheckURLAccessibilityBuilder().
				SetURL("http://"+MainNodeIp+":8003").
				CheckAccessibility() == nil,
		}
	})

	return cacheFileServerAccessible.accessible
}
