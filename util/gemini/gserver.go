package gemini

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"telego/util"
	"telego/util/platform_interface"
	"telego/util/yamlext"
)

var _ platform_interface.UserStorageInfoGetter = &GeminiServer{}

type GeminiServer struct {
	BaseUrl string

	UserAuthFunc *UserAuthFuncStruct
	WebAPIFunc   *WebAPIFuncStruct
}

func NewGeminiServer(BaseUrl string) (*GeminiServer, error) {
	if !strings.HasPrefix(BaseUrl, "http://") && !strings.HasPrefix(BaseUrl, "https://") {
		return nil, fmt.Errorf("NewGeminiServer: invalid BaseURL: %s", BaseUrl)
	}
	gServer := &GeminiServer{BaseUrl: BaseUrl}
	gServer.UserAuthFunc = &UserAuthFuncStruct{server: gServer}
	gServer.WebAPIFunc = &WebAPIFuncStruct{server: gServer}
	return gServer, nil
}

func (gServer *GeminiServer) httpAct(req Request) (Response, error) {
	// 跳过 TLS 验证
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	// request -> httpRequest
	httpReq, err := req.ToHttpRequest(gServer.BaseUrl)
	if err != nil {
		return nil, fmt.Errorf("httpAct: %w", err)
	}

	// httpRequest -> httpResponse
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("httpAct: failed to send request: %w", err)
	}

	// httpResponse -> Response
	// // 反序列化 Body
	resp := req.GetEmptyResponse()
	err = resp.FromHttpRequest(httpResp)
	if err != nil {
		return nil, fmt.Errorf("httpAct: error get response: %w", err)
	}

	return resp, nil
}

// gServer 功能函数
func (gServer *GeminiServer) GetAllStorageByUser(username, password string) ([]util.UserOneStorageSet, error) {
	// 对 username password 进行鉴权
	passwdLoginReq := &PasswdLoginRequest{
		Header: PasswdLoginRequestHeader{},
		Body: PasswdLoginRequestBody{
			UserName: username,
			Password: password,
		},
	}
	passwdLoginResp, err := gServer.UserAuthFunc.PasswdLogin(passwdLoginReq)
	if err != nil {
		return nil, fmt.Errorf("GeminiServer.GetAllStorageByUser: %v", err)
	}
	if passwdLoginResp.Body.Data.Token == "" {
		return nil, fmt.Errorf("GeminiServer.GetAllStorageByUser: Authorization failed")
	}

	// 获取用户所在 spaceID
	token := passwdLoginResp.Body.Data.Token
	user_joined_space_req := &UserJoinedSpaceRequest{
		Header: UserJoinedSpaceRequestHeader{
			Authorization: "Bearer " + token,
		},
	}
	user_joined_space_resp, err := gServer.WebAPIFunc.UserJoinedSpace(user_joined_space_req)
	if err != nil {
		return nil, fmt.Errorf("GeminiServer.GetAllStorageByUser: %v", err)
	}

	storageRet := make([]util.UserOneStorageSet, 0)
	storageViewYamlString, err := (util.MainNodeConfReader{}).ReadSecretConf(util.SecretConfTypeStorageViewYaml{})
	if err != nil {
		return nil, fmt.Errorf("GeminiServer.GetAllStorageByUser: Error reading storageViewYaml: %v", err)
	}
	storageViewYaml := util.SecretConfTypeStorageViewYaml{}
	err = yamlext.UnmarshalAndValidate([]byte(storageViewYamlString), &storageViewYaml)
	if err != nil {
		return nil, fmt.Errorf("GeminiServer.GetAllStorageByUser: Error unmarshal storageViewYaml: %v", err)
	}

	for _, storage := range storageViewYaml.Storages {
		if storage.Type == "gemini" {
			thisSubPaths := make([]string, 0)
			// 待添加
			thisSubPaths = append(thisSubPaths, "user/"+username)
			thisSubPaths = append(thisSubPaths, "platform/shared"+username)
			for _, spaceInfo := range user_joined_space_resp.Body.Data.SpaceList {
				thisSubPaths = append(thisSubPaths, "space_old/"+spaceInfo.SpaceId)
				thisSubPaths = append(thisSubPaths, "space/"+spaceInfo.SpaceId)
			}

			storageRet = append(storageRet, util.UserOneStorageSet{
				Type:        "gemini",
				RootStorage: storage.MountPath,
				SubPaths:    thisSubPaths,
			})
		}
	}
	return storageRet, nil
}
