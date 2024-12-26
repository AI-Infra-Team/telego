package util

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

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
		return "", err
	}
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

type MainNodeConfReader struct{}

var MainNodeFileServerURL = fmt.Sprintf("http://%s:8003", MainNodeIp)

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

type PubConfTypeImgUploaderUrl struct{}

var _ PubConfType = PubConfTypeImgUploaderUrl{}

func (r PubConfTypeImgUploaderUrl) PubConfPath() string {
	return "img_uploader_url"
}

func (r PubConfTypeImgUploaderUrl) Template() string {
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
	default:
		return nil
	}
}

type SecretConfTypeAdminKubeconfig struct{}

var _ SecretConfType = SecretConfTypeAdminKubeconfig{}

func (r SecretConfTypeAdminKubeconfig) SecretConfPath() string {
	return "admin_kubeconfig"
}

func (r SecretConfTypeAdminKubeconfig) Template() string {
	return "# Just the kubeconfig content"
}

type SecretConfTypeImgRepo struct{}

var _ SecretConfType = SecretConfTypeImgRepo{}

func (r SecretConfTypeImgRepo) SecretConfPath() string {
	return "img_repo"
}

func (r SecretConfTypeImgRepo) Template() string {
	template := GenerateYAMLTemplate(ContainerRegistryConf{
		User:     "Registry default user for cri(containerd)",
		Password: "Registry default user's password for cri(containerd)",
		Tls:      nil, // we don't need tls Node
	})

	return template
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
	cached := ConfCache.tryReadSecret(path0)
	if cached != nil {
		return *cached, nil
	}

	base := "/teledeploy_secret/config"
	confpath := path.Join(base, path0.SecretConfPath())
	if confpath != "" {
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

func FileServerAccessible() bool {
	if cacheFileServerAccessible == nil {
		cacheFileServerAccessible = &cacheFileServerAccessibleStruct{
			accessible: NewCheckURLAccessibilityBuilder().
				SetURL("http://"+MainNodeIp+":8003").
				CheckAccessibility() == nil,
		}
	}

	return cacheFileServerAccessible.accessible
}
