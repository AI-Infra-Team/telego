package util

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

var Logger = logrus.New()

func PrintStep(tag string, step string) {
	fmt.Println(color.New().Add(color.BgBlue).Add(color.FgHiWhite).Sprintf("\n[%s] %s", tag, step))
}

var createdLogDir = false

func LogDir() string {
	logDir := filepath.Join(WorkspaceDir(), "logs")
	if !createdLogDir {
		PrintStep("log", "create log dir")
		err := os.MkdirAll(logDir, 0755)
		if err != nil || func() bool { _, err = os.Stat(logDir); return err != nil }() {
			fmt.Println("LogDir: MkdirAll error")
			os.Exit(1)
		}
		createdLogDir = true
	}
	return logDir
}

// return log file, defer close at outter function
func SetupFileLog() *os.File {

	Logger.SetLevel(logrus.DebugLevel)
	// time stamp
	curtime := time.Now().Format("2006-01-02-15h04m05s")
	file, err := os.OpenFile(path.Join(LogDir(), fmt.Sprintf("%s.log", curtime)), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		fmt.Printf("Error opening log file: %v\n", err)
		return nil
	}
	Logger.SetOutput(file)
	return file
}
