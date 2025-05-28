package test3_main_node_config

import (
	"os/exec"
	"path/filepath"
	"telego/test/testutil"
	"telego/util"
	"testing"
)

func TestSSHKeyGeneration(t *testing.T) {
	projectRoot := testutil.GetProjectRoot(t)

	// copy compiled telego to /usr/bin/telego
	distPath := filepath.Join(projectRoot, "dist/telego_linux_"+util.GetCurrentArch())
	_, err := util.ModRunCmd.NewBuilder("cp", distPath, "/usr/bin/telego").BlockRun()
	if err != nil {
		t.Fatalf("复制telego失败: %v", err)
	}

	// 启动 SSH 测试容器
	_, _ = testutil.RunSSHDocker(t)
	// we don't need to cleanup here, because we will use it in later tests
	// defer cleanup()

	// os.Setenv("SSH_PW", util.MainNodeUser)
	// passthrough password input
	err = util.WriteAdminUserConfig(&util.AdminUserConfig{
		Username: util.MainNodeUser,
		Password: util.MainNodeUser,
	})
	if err != nil {
		t.Fatalf("写入管理员用户配置失败: %v", err)
	}

	// // suppose to be failed
	// cmd := exec.Command("telego", "cmd", "--cmd", "/update_config/ssh_config/1.gen_or_get_key")
	// cmd.Dir = projectRoot

	// if err := testutil.RunCommand(t, cmd); err != nil {
	// 	t.Logf("生成 SSH 密钥失败，只有初始化fileserver后才会成功: %v", err)
	// } else {
	// 	t.Fatalf("生成 SSH 密钥成功，理论上未初始化main node file server，不应该成功")
	// }

	// test remote cmd
	cmd := testutil.NewPtyCommand(t, "go", "test", "./test/test3_main_node_config/remote_cmd_test.go", "-v")
	cmd.Dir = projectRoot
	if err = testutil.RunCommand(t, cmd); err != nil {
		// debug telego log
		t.Fatalf("测试remote cmd 失败: %v", err)
	}

	// init fileserver
	cmd = testutil.NewPtyCommand(t, "telego", "cmd", "--cmd", "/update_config/start_mainnode_fileserver")
	cmd.Dir = projectRoot
	if err = testutil.RunCommand(t, cmd); err != nil {
		// debug telego log
		t.Fatalf("初始化main node file server失败: %v", err)
	}

	// upload telego
	t.Logf("upload telego to main node")
	_, err = util.ModRunCmd.NewBuilder("python3", filepath.Join(projectRoot, "3.upload.py")).ShowProgress().BlockRun()
	if err != nil {
		t.Fatalf("上传telego到 main node 失败: %v", err)
	}

	t.Logf("telego log for start main node file server:\n %s", testutil.GetMostRecentLog(t))

	// 重新生成 SSH 密钥
	cmd = testutil.NewPtyCommand(t, "telego", "cmd", "--cmd", "/update_config/ssh_config/1.gen_or_get_key")
	cmd.Dir = projectRoot
	if err := testutil.RunCommand(t, cmd); err != nil {
		// debug telego log
		t.Fatalf("重新生成 SSH 密钥失败: %v", err)
	}

	t.Logf("telego log for gen or get key:\n %s", testutil.GetMostRecentLog(t))

	t.Logf("debug authorized_keys:")
	sshCmd := exec.Command("sshpass", "-p", util.MainNodeUser, "ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-p", "2222", util.MainNodeUser+"@"+util.MainNodeIp, "cat", "~/.ssh/authorized_keys")
	if err := testutil.RunCommand(t, sshCmd); err != nil {
		t.Fatalf("SSH 连接 abc 测试失败: %v", err)
	}

	// 测试 SSH 连接
	t.Logf("test ssh no pw access")
	sshCmd = exec.Command("ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-p", "2222", util.MainNodeUser+"@"+util.MainNodeIp, "echo", "test")
	if err := testutil.RunCommand(t, sshCmd); err != nil {
		t.Fatalf("SSH 连接 abc 测试失败: %v", err)
	}

	t.Log("SSH 密钥生成和连接测试成功")
}
