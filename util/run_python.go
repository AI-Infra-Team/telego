package util

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
)

func EncodeRemoteRunPy(scriptContent string) string {
	// 将脚本内容编码为 Base64
	scriptContentB64 := base64.StdEncoding.EncodeToString([]byte(scriptContent))

	// 生成 Python 命令，将 Base64 内容解码并执行
	return fmt.Sprintf(
		"python3 -c \"import base64; "+
			"script = base64.b64decode('%s').decode('utf-8'); "+
			"exec(script)\"",
		scriptContentB64,
	)
}

type ScriptDiscript struct {
	RequireRoot bool
}

type commitedRunPyModel struct {
	scriptPath     string
	script         string
	content_base64 string
	descript       *ScriptDiscript
}

var commitedRunPy *commitedRunPyModel = nil

func CommitRunPy(scriptPath string, script string, content_base64 string, dicript *ScriptDiscript) {
	commitedRunPy = &commitedRunPyModel{
		scriptPath:     scriptPath,
		script:         script,
		content_base64: content_base64,
		descript:       dicript,
	}
}

func pyCmdHead() string {
	cmd := "python3"
	if runtime.GOOS == "windows" {
		cmd = "python"
	}
	return cmd
}

func RunPy() {
	if commitedRunPy == nil {
		return
	}
	scriptPath := commitedRunPy.scriptPath
	script := commitedRunPy.script
	content_base64 := commitedRunPy.content_base64
	dicript := commitedRunPy.descript

	if dicript == nil {
		dicript = &ScriptDiscript{
			RequireRoot: false,
		}
	}
	logDir := LogDir()
	err := makeDirAll(logDir)
	if err != nil {
		fmt.Printf("tea run cmd failed: %v\n", err)
		os.Exit(1)
	}

	cmd := []string{"python3"}
	// if linux, permission
	if runtime.GOOS == "linux" {
		if dicript.RequireRoot {
			cmd = []string{"sudo"}
		}
	} else if runtime.GOOS == "windows" {
		cmd = []string{"python"}
	}
	// cmd = append(cmd, "-c")

	timestamp := time.Now().Format("20060102_150405")
	timestamp_file_name := filepath.Join(logDir, fmt.Sprintf("run_%s.log", timestamp))

	tempfileContent := fmt.Sprintf(`
import sys
class MultiStream:
	def __init__(self, *streams):
		self.streams = streams

	def write(self, data):
		for stream in self.streams:
			stream.write(data)

	def flush(self):
		for stream in self.streams:
			stream.flush()
def color(string):
	return f"\033[1;32m{string}\033[0m"
scriptPy="%s/%s"
outputFile="%s"
outputFileOpen=open(outputFile, "w")
sys.stdout = MultiStream(sys.stdout, outputFileOpen)
sys.stderr = MultiStream(sys.stderr, outputFileOpen)

# print(color(f"run python script {scriptPy},\n   output is at {outputFile}"))
# open log file
content_base64 = "%v"
import base64,time
content=base64.b64decode(content_base64).decode(encoding='utf-8')
exec(content)
# print(color(f"finished running {scriptPy},\n   output is at {outputFile}"))
	`, scriptPath, script, PathWinStyleToLinux(timestamp_file_name), content_base64)
	tempFile, err := os.CreateTemp("", fmt.Sprintf("%s_*.py",
		strings.ReplaceAll(
			strings.ReplaceAll(path.Join(scriptPath, script), "/", "_"),
			"\\", "_")))
	if err != nil {
		fmt.Println("Create temp failed")
		os.Exit(1)
	}
	tempFile.WriteString(tempfileContent)
	tempFile.Close()

	cmd = append(cmd, tempFile.Name())

	run := exec.Command(cmd[0], cmd[1:]...)
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr

	err = run.Run()
	if err != nil {
		fmt.Println(color.New(color.FgRed).Sprintf("run python script %s/%s failed,\n  output will be at %s", scriptPath, script, timestamp_file_name))
		fmt.Println(color.New(color.FgRed).Sprintf("script is at %s\n", tempFile.Name()))
	} else {
		os.Remove(tempFile.Name())
		fmt.Println(color.New(color.FgGreen).Sprintf("run python script %s/%s success,\n  output will be at %s", scriptPath, script, timestamp_file_name))
	}
}
