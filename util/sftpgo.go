package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
)

type ModSftpgoStruct struct {
}

var ModSftpgo ModSftpgoStruct

func (ModSftpgoStruct) CreateTempSpace(serverAddr, admin, adminPassword,
	tempUser, tempPassword, tempDir string) error {
	type AuthResponse struct {
		AccessToken string `json:"access_token"`
	}

	type VirtualFolder struct {
		Name       string `json:"name"`
		MappedPath string `json:"mapped_path"`
		Filesystem string `json:"filesystem"`
	}

	// Step 1: Authenticate with the SFTPGo server
	loginURL := UrlJoin(serverAddr, "api/v2/token")
	resp, err := HttpAuth(loginURL, admin, adminPassword)
	if err != nil {
		return fmt.Errorf("%s failed to authenticate %s, err: %v", admin, loginURL, err)
	}

	var authResp AuthResponse
	if err := json.Unmarshal(resp, &authResp); err != nil {
		return fmt.Errorf("failed to parse authentication response: %v", err)
	}

	// Step 2: Create the virtual folder
	folderURL := fmt.Sprintf("%s/api/v2/folders", serverAddr)
	folderPayload := map[string]interface{}{
		"id":                0,
		"name":              tempDir,
		"mapped_path":       path.Join("/share", tempDir),
		"description":       "",
		"used_quota_size":   0,
		"used_quota_files":  0,
		"last_quota_update": 0,
		"users":             []string{tempUser},
		"groups":            []string{},
		"filesystem": map[string]interface{}{
			"redacted-secret": "",
			"provider":        0, // https://github.com/sftpgo/sdk/blob/64fc18a344f9c87be4f028ffb7a851fad50976f0/filesystem.go#L20
			// 0: local
		},
	}

	folderData, _ := json.Marshal(folderPayload)
	req, err := http.NewRequest("POST", folderURL, bytes.NewBuffer(folderData))
	if err != nil {
		return fmt.Errorf("failed to create folder request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authResp.AccessToken))
	req.Header.Set("Content-Type", "application/json")

	folderResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create folder: %v", err)
	}
	defer folderResp.Body.Close()

	if folderResp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(folderResp.Body)
		return fmt.Errorf("failed to create folder: %s", string(body))
	}

	// Step 3: Create the temporary user
	userURL := fmt.Sprintf("%s/api/v2/users", serverAddr)
	newUserDirPart := map[string]interface{}{
		"id":                0,
		"name":              tempDir,
		"mapped_path":       path.Join("/share", tempDir),
		"description":       "",
		"used_quota_size":   0,
		"used_quota_files":  0,
		"last_quota_update": 0,
		"users":             []string{tempUser},
		"filesystem": map[string]interface{}{
			"redacted-secret": "",
			"provider":        0, // https://github.com/sftpgo/sdk/blob/64fc18a344f9c87be4f028ffb7a851fad50976f0/filesystem.go#L20
			// 0: local
		},
		"virtual_path": "/",
	}
	userPayload := map[string]interface{}{
		"id":           0,
		"status":       1,
		"username":     tempUser,
		"password":     tempPassword,
		"has_password": true,
		"home_dir":     path.Join("/share", tempDir),
		"uid":          0,
		"gid":          0,
		"max_sessions": 0,
		"quota_size":   0,
		"quota_files":  0,
		"permissions": map[string]interface{}{
			"/": []string{"*"},
		},
		"upload_data_transfer":   0,
		"download_data_transfer": 0,
		"total_data_transfer":    0,
		"created_at":             0,
		"updated_at":             0,
		"filters":                map[string]interface{}{},
		"virtual_folders": []map[string]interface{}{
			newUserDirPart,
		},
		"filesystem": map[string]interface{}{
			"redacted-secret": "",
			"provider":        0, // https://github.com/sftpgo/sdk/blob/64fc18a344f9c87be4f028ffb7a851fad50976f0/filesystem.go#L20
			// 0: local
		},
		"fs-cache":               []interface{}{},
		"group-settings-applied": false,
		"deleted-at":             0,
	}

	userData, _ := json.Marshal(userPayload)
	req, err = http.NewRequest("POST", userURL, bytes.NewBuffer(userData))
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
		return fmt.Errorf("failed to create user: %s", string(body))
	}

	return nil
}
