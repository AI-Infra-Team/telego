package app

// import (
// 	"fmt"
// 	"os"
// 	"os/user"
// 	"telego/util"
// 	"time"

// 	"github.com/spf13/cobra"
// )

// type ModJobSshFsStruct struct{}

// var ModJobSshFs ModJobSshFsStruct

// // Usage:
// // telego rclone --mount {localPath} {remoteUser@remoteHostIP:remotePath}

// type sshFsJob struct {
// 	remotePath string
// 	localPath  string
// }

// func (m ModJobSshFsStruct) JobCmdName() string {
// 	return "sshfs"
// }

// func (m ModJobSshFsStruct) ParseJob(sshFsCmd *cobra.Command) *cobra.Command {
// 	job := &sshFsJob{}

// 	// 读入参数
// 	sshFsCmd.Flags().StringVar(&job.remotePath, "remotepath", "", "sshfs mount - remote path")
// 	sshFsCmd.Flags().StringVar(&job.localPath, "localpath", "", "sshfs mount - local mountPath")

// 	sshFsCmd.Run = func(_ *cobra.Command, _ []string) {
// 		m.doMount(job)
// 	}

// 	return sshFsCmd
// }

// func (m ModJobSshFsStruct) doMount(job *sshFsJob) error {
// 	if job.localPath == "" || job.remotePath == "" {
// 		return fmt.Errorf("sshfs mount argument empty")
// 	}
// 	// 判断 localPath 是否存在
// 	if _, err := os.Stat(job.localPath); os.IsNotExist(err) {
// 		return fmt.Errorf("local path %s does not exist", job.localPath)
// 	}

// 	// command := []string{"sshfs", job.remotePath, job.localPath, "-o", "reconnect"}
// 	// if _, err := util.ModRunCmd.NewBuilder(command[0], command[1:]...).WithRoot().ShowProgress().BlockRun(); err != nil {
// 	// 	return fmt.Errorf("%v exec error", command)
// 	// }
// 	currentUser, _ := user.Current()
// 	addRoot := ""
// 	if currentUser.Uid != "0" {
// 		addRoot = "sudo "
// 	}
// 	command := addRoot + fmt.Sprintf("sshfs %s %s -o reconnect", job.remotePath, job.localPath)
// 	util.RunCmdWithTimeoutCheck(command, 1*time.Minute, func(output string) bool { return true })
// 	return nil
// }
