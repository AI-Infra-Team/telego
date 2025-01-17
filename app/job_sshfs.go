package app

import (
	"fmt"
	"os/user"
	"telego/util"
	"time"

	"github.com/spf13/cobra"
)

type ModJobSshFsStruct struct{}

var ModJobSshFs ModJobSshFsStruct

// Usage:
// telego sshfs --remotepath {} --localpath {}

type sshFsArgv struct {
	remotePath string
	localPath  string
}

func (m ModJobSshFsStruct) JobCmdName() string {
	return "sshfs"
}

func (m ModJobSshFsStruct) ParseJob(sshFsCmd *cobra.Command) *cobra.Command {
	job := &sshFsArgv{}

	// 读入参数
	sshFsCmd.Flags().StringVar(&job.remotePath, "remotepath", "", "sshfs mount - remote path")
	sshFsCmd.Flags().StringVar(&job.localPath, "localpath", "", "sshfs mount - local mountPath")

	sshFsCmd.Run = func(_ *cobra.Command, _ []string) {
		err := m.doMount(job)
		if err != nil {
			fmt.Println("ParseJob error: ", err)
		}
	}

	return sshFsCmd
}

func (m ModJobSshFsStruct) doMount(job *sshFsArgv) error {
	if job.localPath == "" || job.remotePath == "" {
		return fmt.Errorf("doMount: sshfs mount argument empty")
	}

	command := []string{"sshfs", job.remotePath, job.localPath, "-o", "reconnect"}
	currentUser, _ := user.Current()
	if currentUser.Uid != "0" {
		command = append([]string{"sudo"}, command...)
	}

	_, _, err := util.RunCmdWithTimeoutCheck(command, 1*time.Minute, func(output string) bool { return true })
	if err != nil {
		return fmt.Errorf("doMount: error mount %s to %s", job.remotePath, job.localPath)
	}
	return nil
}
