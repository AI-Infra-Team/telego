package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
)

func getMostRecentLog(logDir string, filter func(string) bool) string {
	files, err := filepath.Glob(filepath.Join(logDir, "*.log"))
	if err != nil {
		errMsg := color.RedString("Error listing log files: %v", err)
		fmt.Println(errMsg)
		os.Exit(1)
	}
	mostRecentFile := ""
	mostRecentTime := time.Time{}
	for _, file := range files {
		if !filter(file) {
			continue
		}
		info, err := os.Stat(file)
		if err != nil {
			errMsg := color.RedString("Error getting file info: %v", err)
			fmt.Println(errMsg)
			os.Exit(1)
		}

		if info.ModTime().After(mostRecentTime) {
			mostRecentTime = info.ModTime()
			mostRecentFile = file
		}
	}
	return mostRecentFile
}

func GetMostRecentRemoteLog() string {
	f := getMostRecentLog(LogDir(), func(file string) bool {
		return strings.HasPrefix(file, "remote_")
	})
	content, err := os.ReadFile(filepath.Join(LogDir(), f))
	if err != nil {
		fmt.Println(color.RedString("Error reading log file: %v", err))
		os.Exit(1)
	}
	return string(content)
}

func GetMostRecentLog() string {

	f := getMostRecentLog(LogDir(), func(file string) bool {
		return !strings.HasPrefix(file, "remote_")
	})
	content, err := os.ReadFile(filepath.Join(LogDir(), f))
	if err != nil {
		fmt.Println(color.RedString("Error reading log file: %v", err))
		os.Exit(1)
	}
	return string(content)
}
