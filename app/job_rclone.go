package app

import (
	"fmt"
	"os"
	"os/exec"
	"telego/util"

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
	remotePath string
	localPath  string
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
	rcloneCmd.Flags().StringVar(&job.mountArgv.remotePath, "remotepath", "", "rclone mount - remote path")
	rcloneCmd.Flags().StringVar(&job.mountArgv.localPath, "localpath", "", "rclone mount - local mountPath")

	rcloneCmd.Run = func(_ *cobra.Command, _ []string) {
		switch job.jobType {
		case "mount":
			err := m.doMount(job.mountArgv)
			if err != nil {
				fmt.Println(color.RedString("rclone mount failed %v", err))
				os.Exit(1)
			}
		default:
			fmt.Println(color.RedString("unsupported rclone sub operation"))
			os.Exit(1)
		}
	}

	return rcloneCmd
}

type JobRcloneType interface {
	JobRcloneTypeDummyInterface()
}

type JobRcloneTypeMount struct {
	rcloneMountArgv
}

func (JobRcloneTypeMount) JobRcloneTypeDummyInterface() {}

func (ModJobRcloneStruct) NewCmd(t JobRcloneType) []string {
	switch t := t.(type) {
	case JobRcloneTypeMount:
		return []string{"telego", "rclone", "--mode", "mount", "--remotepath", t.remotePath, "--localpath", t.localPath}
	default:
		panic(color.RedString("unknown rclone type %v", t))
	}
}

func (ModJobRcloneStruct) mountCmd(argv *rcloneMountArgv) []string {
	return []string{"rclone", "mount", argv.remotePath, argv.localPath,
		"--cache-dir", "D:/rclone_cache",
		"--vfs-cache-max-size", "10G",
		"--vfs-cache-poll-interval", "10s",
		"--vfs-cache-mode", "full",
		"--debug-fuse",
	}
}

func (m ModJobRcloneStruct) asyncMount(argv *rcloneMountArgv) (*exec.Cmd, error) {
	cmds := m.mountCmd(argv)
	cmd, err := util.ModRunCmd.NewBuilder(cmds[0], cmds[1:]...).ShowProgress().AsyncRun()
	return cmd, err
}

func (m ModJobRcloneStruct) doMount(argv *rcloneMountArgv) error {
	tag := "Job rclone mount"
	util.PrintStep(tag, fmt.Sprintf("remotepath:%s, localpath:%s", argv.remotePath, argv.localPath))
	if argv.localPath == "" || argv.remotePath == "" {
		return fmt.Errorf("rclone mount argument empty")
	}
	// if _, err := os.Stat(argv.localPath); os.IsNotExist(err) {
	// 	return fmt.Errorf("local path %s does not exist", argv.localPath)
	// }

	// // if is not root, append sudo
	// if !util.IsWindows() && !util.IsRoot() {
	// 	cmds = append([]string{"sudo"}, cmds...)
	// }
	cmds := m.mountCmd(argv)
	fmt.Printf("rclone cmds %v\n", cmds)
	_, err := util.ModRunCmd.NewBuilder(cmds[0], cmds[1:]...).ShowProgress().BlockRun()
	// var err error
	// var cmd *exec.Cmd
	// var output string
	// var cmdbuilder *util.CmdBuilder
	// for i := 0; i < 100; i++ {
	// 	cmdbuilder = util.ModRunCmd.NewBuilder(cmds[0], cmds[1:]...)
	// 	cmd, err = cmdbuilder.AsyncRun()
	// 	if err == nil {
	// 		time.Sleep(3 * time.Second)
	// 		output = cmdbuilder.Output()
	// 		if len(output) < 50 {
	// 			fmt.Printf("rclone output ok %v %v\n", output, len(output))
	// 			break
	// 		} else {
	// 			fmt.Println(color.YellowString("rclone mount invalid output %s, len:%v, will retry", output, len(output)))
	// 			cmd.Process.Kill()
	// 			// wait killed
	// 			cmd.Process.Wait()
	// 			for {
	// 				_, err := os.Stat(argv.localPath)
	// 				if os.IsNotExist(err) {
	// 					break
	// 				}
	// 				time.Sleep(100 * time.Millisecond)
	// 			}
	// 		}
	// 	} else {
	// 		fmt.Println(color.YellowString("rclone mount failed: %v, will retry", err))
	// 		cmd.Process.Kill()
	// 		// wait killed
	// 		cmd.Process.Wait()
	// 		for {
	// 			_, err := os.Stat(argv.localPath)
	// 			if os.IsNotExist(err) {
	// 				break
	// 			}
	// 			time.Sleep(100 * time.Millisecond)
	// 		}
	// 	}
	// }
	if err != nil {
		return fmt.Errorf("rclone mount failed: %w", err)
	}
	util.PrintStep(tag, "Blocking mount rclone")
	// err = cmd.Wait()
	// output = cmdbuilder.Output()
	// if err != nil {
	// 	return fmt.Errorf("rclone mount wait failed, output: %s, err: %w", output, err)
	// }
	return nil
}
