package app

import (
	"fmt"
	"os"
	"telego/util"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type ModJobRcloneStruct struct{}

var ModJobRclone ModJobRcloneStruct

// Usage:
// telego rclone --mount {localPath} {remoteUser@remoteHostIP:remotePath}

type rcloneJob struct {
	jobType   string // mount / sync
	mountArgv *rcloneMountArgv
	// syncArgv rcloneSyncArgv
}

type rcloneMountArgv struct {
	localPath  string
	remoteHost string
	remotePath string
}

func (m ModJobRcloneStruct) JobCmdName() string {
	return "rclone"
}

func (m ModJobRcloneStruct) ParseJob(rcloneCmd *cobra.Command) *cobra.Command {
	job := &rcloneJob{
		mountArgv: &rcloneMountArgv{},
	}

	// 读入参数
	rcloneCmd.Flags().StringVar(&job.jobType, "mode", "", "rclone - sub operation") // --mode mount/sync
	rcloneCmd.Flags().StringVar(&job.mountArgv.localPath, "localpath", "", "rclone mount - local mountPath")
	rcloneCmd.Flags().StringVar(&job.mountArgv.remoteHost, "remotehost", "", "rclone mount - remote host")
	rcloneCmd.Flags().StringVar(&job.mountArgv.remotePath, "remotepath", "", "rclone mount - remote path")

	rcloneCmd.Run = func(_ *cobra.Command, _ []string) {
		switch job.jobType {
		case "mount":
			doMount(job.mountArgv)
		default:
			fmt.Println(color.RedString("unsupported rclone sub operation"))
			os.Exit(1)
		}
	}

	return rcloneCmd
}

func doMount(argv *rcloneMountArgv) error {
	if argv.localPath == "" || argv.remoteHost == "" || argv.remotePath == "" {
		return fmt.Errorf("rclone mount argument empty")
	}
	_, err := util.RunCmdWithTimeoutCheck(
		fmt.Sprintf("rclone --debug mount %s:%s %s", argv.remoteHost, argv.remotePath, argv.localPath),
		1*time.Second,
		func(output string) bool {
			return len(output) < 50
		},
	)
	return err
}
