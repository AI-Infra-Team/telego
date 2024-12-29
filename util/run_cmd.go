package util

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/thoas/go-funk"
)

type ModRunCmdStruct struct {
}

var ModRunCmd ModRunCmdStruct

type CmdBuilder struct {
	Cmd          *exec.Cmd
	outputBuffer bytes.Buffer
	errWriters   []io.Writer
	outWriters   []io.Writer
	showProgress bool
}

func (b *CmdBuilder) Output() string {
	return b.outputBuffer.String()
}

func (b *CmdBuilder) Cmds() []string {
	return append([]string{b.Cmd.Path}, b.Cmd.Args...)
}

func (b *CmdBuilder) ShowProgress() *CmdBuilder {
	b.showProgress = true
	b.errWriters = append(b.errWriters, os.Stderr)
	b.outWriters = append(b.outWriters, os.Stdout)
	return b
}

func (b *CmdBuilder) SetDir(dir string) *CmdBuilder {
	b.Cmd.Dir = dir
	return b
}

func (b *CmdBuilder) SetEnv(envs ...string) *CmdBuilder {
	env := os.Environ()
	env = append(env, envs...)
	b.Cmd.Env = env
	return b
}

func (b *CmdBuilder) AsyncRun() (*exec.Cmd, error) {

	errWriter := io.MultiWriter(b.errWriters...)
	outWriter := io.MultiWriter(b.outWriters...)

	if b.showProgress {
		b.Cmd.Stdout = os.Stdout
		b.Cmd.Stderr = os.Stderr
	} else {
		b.Cmd.Stdout = outWriter
		b.Cmd.Stderr = errWriter
	}

	// 启动命令
	if err := b.Cmd.Start(); err != nil {
		return b.Cmd, fmt.Errorf("error starting command: %v", err)
	}
	return b.Cmd, nil
}

func (b *CmdBuilder) BlockRun() (string, error) {

	errWriter := io.MultiWriter(b.errWriters...)
	outWriter := io.MultiWriter(b.outWriters...)

	if b.showProgress {
		b.Cmd.Stdout = os.Stdout
		b.Cmd.Stderr = os.Stderr
	} else {
		b.Cmd.Stdout = outWriter
		b.Cmd.Stderr = errWriter
	}

	// 启动命令
	if err := b.Cmd.Start(); err != nil {
		return b.outputBuffer.String(), fmt.Errorf("error starting command: %v", err)
	}

	// 等待命令执行完成
	if err := b.Cmd.Wait(); err != nil {
		return b.outputBuffer.String(), fmt.Errorf("error waiting for command: %v", err)
	}

	return b.outputBuffer.String(), nil
}

func (b *CmdBuilder) PrintCmd() *CmdBuilder {
	fmt.Printf("%s %v\n", b.Cmd.Path, b.Cmd.Args)
	return b
}

func (b *CmdBuilder) WithRoot() *CmdBuilder {
	if IsWindows() || IsRoot() {
		return b
	}
	if b.Cmd.Args[0] == "sudo" {
		return b
	}
	nb := ModRunCmd.NewBuilder("sudo", b.Cmd.Args...)

	nb.SetDir(b.Cmd.Dir)
	nb.SetEnv(b.Cmd.Env...)
	nb.errWriters = b.errWriters

	return nb
}

func (m ModRunCmdStruct) ShowProgress(name string, args ...string) *CmdBuilder {
	b := m.NewBuilder(name, args...).ShowProgress()
	return b
}

// runCommand 用于执行 shell 命令并返回输出
func (m ModRunCmdStruct) NewBuilder(name string, args ...string) *CmdBuilder {
	// output, err := cmd.CombinedOutput()
	// if err != nil {
	// 	return "", fmt.Errorf("命令执行错误: %v, 输出: %s", err, string(output))
	// }
	// return string(output), nil
	cmd := exec.Command(name, args...)

	b := CmdBuilder{
		Cmd: cmd,
	}
	b.errWriters = []io.Writer{
		&b.outputBuffer,
	}
	b.outWriters = []io.Writer{
		&b.outputBuffer,
	}
	return &b
}

func (m ModRunCmdStruct) RequireRootRunCmd(name string, args ...string) (string, error) {
	// is root
	if IsWindows() || IsRoot() {
		return ModRunCmd.NewBuilder(name, args...).BlockRun()
	}

	args = append([]string{name}, args...)
	return ModRunCmd.NewBuilder("sudo", args...).BlockRun()
}

func (m ModRunCmdStruct) CopyDirContentOrFileTo(srcDirOrFile, destDir string) error {
	_, err := ModRunCmd.NewBuilder("rclone", "copy", "-P", srcDirOrFile, destDir).BlockRun()
	return err
}

func (m ModRunCmdStruct) SplitCmdline(cmdline string) []string {
	return funk.Map(strings.Split(cmdline, " "), func(slice string) string {
		if slice == "\"\"" || slice == "''" {
			return ""
		}
		return slice
	}).([]string)
}

type Cmdline struct {
	Cmdline string
}

func (c Cmdline) toCmds() []string {
	return ModRunCmd.SplitCmdline(c.Cmdline)
}

type CmdModels struct {
}

func (m CmdModels) InstallTelegoWithPy() string {
	return fmt.Sprintf("python3 -c \"import urllib.request, os; script = urllib.request.urlopen('http://%s:8003/bin_telego/install.py').read(); exec(script.decode());\"", MainNodeIp)
}

func (m ModRunCmdStruct) CmdModels() CmdModels {
	return CmdModels{}
}

// 在timeout时间范围内条件不满足，返回error
// 满足了条件，继续
func RunCmdWithTimeoutCheck(
	cmdStr []string, timeout time.Duration, conditionMet func(output string) bool) (*bytes.Buffer, *exec.Cmd, error) {
	cmd := exec.Command(cmdStr[0], cmdStr[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start command: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var accumulatedOutput *bytes.Buffer = &bytes.Buffer{}
	done := make(chan struct{})

	// 协程不断监控输出，如果 conditionMet == true，则取消超时机制
	go func() {
		scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
		for scanner.Scan() {
			line := scanner.Text()
			accumulatedOutput.WriteString(line + "\n")
			if conditionMet(accumulatedOutput.String()) {
				cancel()
				break
			}
		}
		close(done)
	}()

	select {
	case <-ctx.Done(): // ctx.timeout 时触发
		if ctx.Err() == context.DeadlineExceeded {
			_ = cmd.Process.Kill() // 超时强制结束子进程
			return nil, nil, fmt.Errorf("timeout exceeded, process killed")
		}
	case <-done: // close(done) 时触发
		// 等待 cmd 任务结束
	}

	// if err := cmd.Wait(); err != nil {
	// 	return accumulatedOutput.String(), fmt.Errorf("process exited with error: %w", err)
	// }

	return accumulatedOutput, cmd, nil
}
