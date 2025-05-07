package testutil

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	SSH_PORT = 2222 // 非默认 SSH 端口
)

// BuildContext 构建上下文配置
type BuildContext struct {
	HOST_PROJECT_DIR string `yaml:"HOST_PROJECT_DIR"`
}

// GetHostProjectPath 获取主机项目目录路径
func GetHostProjectPath(t *testing.T) string {
	projectRoot := GetProjectRoot(t)
	buildContextPath := filepath.Join(projectRoot, "build_context.yml")

	// 读取 build_context.yml
	data, err := os.ReadFile(buildContextPath)
	if err != nil {
		t.Fatalf("读取 build_context.yml 失败: %v", err)
	}

	var buildContext BuildContext
	if err := yaml.Unmarshal(data, &buildContext); err != nil {
		t.Fatalf("解析 build_context.yml 失败: %v", err)
	}

	if buildContext.HOST_PROJECT_DIR == "" {
		t.Fatal("build_context.yml 中 HOST_PROJECT_DIR 为空")
	}

	return buildContext.HOST_PROJECT_DIR
}

// RunSSHDocker 运行一个用于 SSH 测试的 Docker 容器
func RunSSHDocker(t *testing.T) (string, func()) {
	// projectRoot := GetProjectRoot(t)
	hostProjectPath := GetHostProjectPath(t)

	// 拉取并运行构建镜像，映射项目目录
	cmd := exec.Command("docker", "run", "-d",
		"-p", "2222:22",
		"-v", hostProjectPath+":/telego",
		"telego_build",
		"tail", "-f", "/dev/null")

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("启动容器失败: %v", err)
	}
	containerID := string(output[:12])

	// 等待容器就绪 - 鲁棒的方式
	maxAttempts := 50                       // 最多尝试50次
	waitInterval := 1000 * time.Millisecond // 每次等待1秒
	var ready bool

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		pathCmd := exec.Command("docker", "exec", containerID, "echo", "Container is ready")
		if err := pathCmd.Run(); err == nil {
			ready = true
			t.Logf("容器就绪，第 %d 次尝试成功", attempt)
			break
		}
		t.Logf("等待容器就绪...尝试 %d/%d", attempt, maxAttempts)
		time.Sleep(waitInterval)
	}

	if !ready {
		t.Fatalf("容器启动超时，在 %d 次尝试后仍未就绪", maxAttempts)
	}

	// // remove all systemctl listed by which systemctl
	// t.Log("RunSSHDocker 移除所有的 systemctl")
	// removeSystemctlCmd := exec.Command("docker", "exec", containerID, "bash", "-c",
	// 	"rm -f $(which -a systemctl) || echo 'No systemctl found'")
	// if err := RunCommand(t, removeSystemctlCmd); err != nil {
	// 	t.Logf("移除 systemctl 时出现警告 (可忽略): %v", err)
	// }

	// // 在容器内执行 Python 脚本
	// t.Log("RunSSHDocker 配置systemctl")
	// scriptCmd := exec.Command("docker", "exec", containerID,
	// 	"python3", "/telego/scripts/systemctl_docker.py")
	// if err := RunCommand(t, scriptCmd); err != nil {
	// 	t.Fatalf("执行 systemctl_docker.py 脚本失败: %v", err)
	// }

	// 安装SSH服务器
	t.Log("RunSSHDocker 安装SSH服务器、sudo")
	installSSHCmd := exec.Command("docker", "exec", containerID, "bash", "-c",
		"apt-get update && apt-get install -y openssh-server sudo && mkdir -p /run/sshd")
	if err := RunCommand(t, installSSHCmd); err != nil {
		t.Fatalf("安装SSH服务器失败: %v", err)
	}

	// 添加安装sudo的代码
	// t.Log("RunSSHDocker 安装sudo")
	// installSudoCmd := exec.Command("docker", "exec", containerID, "bash", "-c",
	// 	"apt-get install -y sudo")
	// if err := RunCommand(t, installSudoCmd); err != nil {
	// 	t.Fatalf("安装sudo失败: %v", err)
	// }

	// 配置SSH服务器和创建用户
	t.Log("RunSSHDocker 配置SSH服务器和创建用户")
	configSSHCmd := exec.Command("docker", "exec", containerID, "bash", "-c",
		"echo 'PermitRootLogin yes' >> /etc/ssh/sshd_config && "+
			"echo 'root:password' | chpasswd && "+
			"useradd -m -s /bin/bash abc && "+
			"echo 'abc:abc' | chpasswd && "+
			"mkdir -p /etc/sudoers.d && "+
			"echo 'abc ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/abc && "+
			"chmod 440 /etc/sudoers.d/abc && "+
			"chown root:root /etc/ssh /etc/ssh/sshd_config && "+
			"chmod 755 /etc/ssh && "+
			"chmod 644 /etc/ssh/sshd_config && "+
			"/usr/sbin/sshd")
	if err := RunCommand(t, configSSHCmd); err != nil {
		t.Fatalf("配置SSH服务器失败: %v", err)
	}

	// run /telego/scripts/setup_mock_systemctl.py
	t.Log("RunSSHDocker 配置mock_systemctl")
	setupMockSystemctlCmd := exec.Command("docker", "exec", containerID,
		"python3", "/telego/scripts/setup_mock_systemctl.py")
	if err := RunCommand(t, setupMockSystemctlCmd); err != nil {
		t.Fatalf("配置mock_systemctl失败: %v", err)
	}

	// 检查 PATH 环境变量
	t.Log("RunSSHDocker 检查 PATH 环境变量")
	pathCmd := exec.Command("docker", "exec", containerID, "echo", "$PATH")
	if err := RunCommand(t, pathCmd); err != nil {
		t.Fatalf("检查 PATH 失败: %v", err)
	}

	// 在容器内执行拷贝命令
	t.Log("RunSSHDocker 在容器内执行拷贝命令")
	copyCmd := exec.Command("docker", "exec", containerID,
		"cp", fmt.Sprintf("/telego/dist/%s", GetBinaryName()), "/usr/bin/telego")
	if err := RunCommand(t, copyCmd); err != nil {
		t.Fatalf("复制telego二进制文件失败: %v", err)
	}

	// 启用 SSH 密码认证
	t.Log("RunSSHDocker 启用 SSH 密码认证")
	sshCmd := exec.Command("docker", "exec", containerID,
		"telego", "ssh-passwd-auth", "--enable", "true")

	// 获取命令的输出
	stdout, err := sshCmd.StdoutPipe()
	if err != nil {
		t.Fatalf("获取标准输出失败: %v", err)
	}
	stderr, err := sshCmd.StderrPipe()
	if err != nil {
		t.Fatalf("获取标准错误失败: %v", err)
	}

	// 启动命令
	if err := sshCmd.Start(); err != nil {
		t.Fatalf("启动命令失败: %v", err)
	}

	// 读取输出
	stdoutBytes, _ := io.ReadAll(stdout)
	stderrBytes, _ := io.ReadAll(stderr)

	// 等待命令完成
	if err := sshCmd.Wait(); err != nil {
		t.Fatalf("启用 SSH 密码认证失败: %v\n标准输出: %s\n标准错误: %s",
			err, string(stdoutBytes), string(stderrBytes))
	} else {
		t.Log("启用 SSH 密码认证成功, stdout: %s", string(stdoutBytes))
	}

	// 测试 SSH 访问 (用户 abc，使用密码认证)
	t.Log("RunSSHDocker 测试SSH密码认证")
	// install sshpass first

	// 检查 sshpass 是否安装
	checkCmd := exec.Command("which", "sshpass")
	if err := checkCmd.Run(); err != nil {
		t.Log("sshpass 未安装，尝试安装...")

		// 定义要尝试的包管理器命令
		installCommands := []struct {
			name string
			cmd  *exec.Cmd
		}{
			{"apt-get", exec.Command("apt-get", "install", "-y", "sshpass")},
			{"yum", exec.Command("yum", "install", "-y", "sshpass")},
			{"dnf", exec.Command("dnf", "install", "-y", "sshpass")},
			{"apk", exec.Command("apk", "add", "sshpass")},
			{"brew", exec.Command("brew", "install", "sshpass")},
		}

		// 尝试每个包管理器
		installed := false
		for _, install := range installCommands {
			t.Logf("尝试使用 %s 安装 sshpass...", install.name)
			err := RunCommand(t, install.cmd)
			if err == nil {
				t.Logf("使用 %s 安装 sshpass 成功", install.name)
				installed = true
				break
			} else {
				t.Logf("使用 %s 安装 sshpass 失败, err: %w", install.name, err)
			}
		}

		// 检查是否安装成功
		if !installed {
			t.Fatal("所有安装方法都失败，请手动安装 sshpass 后重试")
		}

		// 再次检查 sshpass 是否可用
		if err := exec.Command("which", "sshpass").Run(); err != nil {
			t.Fatal("sshpass 安装后仍然不可用")
		}
	}

	// 使用sshpass工具自动提供密码
	sshCmd = exec.Command("sshpass", "-p", "abc",
		"ssh", "abc@localhost", "-p", "2222",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null", "echo", "helloworld")
	if err := RunCommand(t, sshCmd); err != nil {
		t.Fatalf("SSH 密码访问失败: %v", err)
	}

	// 输出成功信息
	t.Logf("SSH 密码认证已启用\n标准输出: %s", string(stdoutBytes))

	// 返回容器 ID 和清理函数
	return containerID, func() {
		exec.Command("docker", "rm", "-f", containerID).Run()
	}
}

// RunCommand 运行命令并实时输出
func RunCommand(t *testing.T, cmd *exec.Cmd) error {
	t.Logf("\n>>> 运行命令: %v", cmd.Args)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Logf("run command get stdout err %v", err)
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Logf("run command get stderr err %v", err)
		return err
	}

	if err := cmd.Start(); err != nil {
		t.Logf("run command start err %v", err)
		return err
	}

	stdoutReader := bufio.NewReader(stdout)
	stderrReader := bufio.NewReader(stderr)

	stdoutStr := ""
	stderrStr := ""
	go func() {
		for {
			line, err := stdoutReader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					t.Logf("读取标准输出错误: %v", err)
				}
				break
			}
			t.Log(line)
			stdoutStr += line + "\n"
		}
	}()

	go func() {
		for {
			line, err := stderrReader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					t.Logf("读取标准错误错误: %v", err)
				}
				break
			}
			t.Log(line)
			stderrStr += line + "\n"
		}
	}()

	err = cmd.Wait()
	if err != nil {
		t.Logf("run command wait err %v, stdout: %s, stderr: %s", err, stdoutStr, stderrStr)
		return fmt.Errorf("命令执行失败: %w, stdout: %s, stderr: %s",
			err, stdoutStr, stderrStr)
	}
	return nil
}
