package test3_main_node_config

import (
	"os"
	"telego/test/testutil"
	"telego/util"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestImgRepoSetup(t *testing.T) {
	projectRoot := testutil.GetProjectRoot(t)

	// prepare main node docker with secret config
	// create ImgRepoConfig
	ymlmodel := util.ContainerRegistryConf{
		User:                      "testadmin",
		Password:                  "testpassword",
		UploaderStoreAddr:         "http://127.0.0.1:5000",
		UploaderStoreAdmin:        "testadmin",
		UploaderStoreAdminPw:      "testpassword",
		UploaderStoreTransferAddr: "http://127.0.0.1:8000",
		Tls:                       nil,
	}
	// marshal to yaml
	yml, err := yaml.Marshal(ymlmodel)
	if err != nil {
		t.Fatalf("marshal to yaml failed: %v", err)
	}
	// write to file
	err = os.WriteFile("/tmp/img_repo", yml, 0644)
	if err != nil {
		t.Fatalf("write to file failed: %v", err)
	}

	// rclone config to main node
	cmd := testutil.NewPtyCommand(t, "rclone", "copy", "/tmp/img_repo", "remote:/teledeploy_secret/config")
	cmd.Dir = projectRoot
	testutil.RunCommand(t, cmd)
	if testutil.GetPtyExitCode(t, cmd) != 0 {
		t.Fatalf("rclone to main node failed")
	}

	// telego start img repo
	cmd = testutil.NewPtyCommand(t, "telego", "img-repo")
	cmd.Dir = projectRoot
	testutil.RunCommand(t, cmd)
	if testutil.GetPtyExitCode(t, cmd) != 0 {
		t.Fatalf("telego start img repo failed")
	}

	// docker login
	cmd = testutil.NewPtyCommand(t, "docker", "login", "127.0.0.1:5000", "-u", "testadmin", "-p", "testpassword")
	cmd.Dir = projectRoot
	testutil.RunCommand(t, cmd)
	if testutil.GetPtyExitCode(t, cmd) != 0 {
		t.Fatalf("docker login failed")
	}

	t.Log("img repo started")
}
