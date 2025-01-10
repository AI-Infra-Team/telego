package app

import (
	"fmt"
	"os"
	"strings"
	"telego/app/config"
	"telego/util"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/thoas/go-funk"
)

type CmdJob struct {
	CmdPath string
}

type ModJobCmdStruct struct{}

var ModJobCmd ModJobCmdStruct

func (_ ModJobCmdStruct) JobCmdName() string {
	return "cmd"
}

func (_ ModJobCmdStruct) ParseJob(CmdCmd *cobra.Command) *cobra.Command {
	job := &CmdJob{}

	// 绑定命令行标志到结构体字段
	CmdCmd.Flags().StringVar(&job.CmdPath, "cmd", "", "Sub project dir in user specified workspace")

	CmdCmd.Run = func(_ *cobra.Command, _ []string) {
		ModJobCmd.CmdLocal(*job)
	}
	// err := CmdCmd.Execute()
	// if err != nil {
	// 	return nil,nil
	// }

	return CmdCmd
}

func (m ModJobCmdStruct) NewCmd(path string) []string {
	return []string{"telego", "cmd", "--cmd", path}
}

func (_ ModJobCmdStruct) CmdLocal(job CmdJob) {
	if job.CmdPath == "" {
		fmt.Println(color.RedString("No cmd path provided"))
		os.Exit(1)
	}
	// invalidChars := "\\/:*?\"<>|"
	invalidChars := []string{"\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	for _, c := range invalidChars {
		if strings.Contains(job.CmdPath, c) {
			fmt.Println(color.RedString("Invalid char '%s' in path %s", c, job.CmdPath))
			os.Exit(1)
		}
	}

	if strings.HasPrefix(job.CmdPath, "/") {
		fmt.Println(color.RedString("Path must be relative, shouldn't start with '/'"))
		os.Exit(1)
	}

	// avoid nil ptr for conf

	loadedConfig := config.MayFailLoad(util.WorkspaceDir())
	fmt.Println("loadedConfig:", loadedConfig)
	rootMenu := InitMenuTree(loadedConfig)
	cmds := strings.Split(job.CmdPath, "/")
	prefixes := []*MenuItem{}
	for i, cmd := range cmds {
		if cmd == "" {
			continue
		}
		if rootMenu.Name == "deploy" {
			WaitInitMenuTree()
		}
		found := rootMenu.FindChild(cmd)
		if found == nil {
			fmt.Println(color.RedString(
				"Command not found: %s, looking for cmd slice: %s, existing cmds: %v",
				job.CmdPath, cmd, funk.Map(rootMenu.Children,
					func(c *MenuItem) string { return c.Name })))
			os.Exit(1)
		}
		if i < len(cmds)-1 {
			found.DispatchEnterNext(prefixes)
		}
		prefixes = append(prefixes, found)
		rootMenu = found
	}

	res := prefixes[len(prefixes)-1].DispatchExec(prefixes[:len(prefixes)-1])
	if res.Exit && res.ExitWithDelegate != nil {
		res.ExitWithDelegate()
	} else {
		fmt.Println(color.RedString("unexecutable cmd path %s", job.CmdPath))
		os.Exit(1)
	}
}

// func NewCmdCmd(

// ) []string {
// 	cmds := []string{"telego", "Cmd", "--project", prj}

// 	for _, k8s := range k8ss {
// 		cmds = append(cmds, "--k8s", *k8s.K8sDir)
// 		if k8s.Namespace != nil && *k8s.Namespace != "" {
// 			cmds = append(cmds, "--k8s-ns", *k8s.Namespace)
// 		} else {
// 			cmds = append(cmds, "--k8s-ns", "\"\"")
// 		}
// 	}
// 	for _, helm := range helms {
// 		cmds = append(cmds, "--helm", *helm.HelmDir)
// 		if helm.Namespace != nil && *helm.Namespace != "" {
// 			cmds = append(cmds, "--helm-ns", *helm.Namespace)
// 		} else {
// 			cmds = append(cmds, "--helm-ns", "\"\"")
// 		}
// 	}
// 	cmds = append(cmds, "--cluster-context", clusterContextName)
// 	return cmds
// }

// // in app entry
// func CmdLocal(k8sprj string, k8sdp *Deployment, clusterContextName string) {
// 	// cmds := []string{}
// 	cmds := NewCmdCmd(k8sprj, k8sdp.K8s, k8sdp.Helms, clusterContextName)
// 	util.ModRunCmd.RunCommandShowProgress(cmds[0], cmds[1:]...)
// 	// cmd += NewCmdCmd(binpack, bin, binBin[bin])

// 	// util.Logger.Debugf("Cmd cmds split: %s", cmds)

// }
