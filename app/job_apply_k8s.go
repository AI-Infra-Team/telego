package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"telego/util"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type ApplyJob struct {
	Project        string
	K8sDirs        []string
	K8sNamespaces  []string
	HelmDirs       []string
	HelmNamespaces []string
	ClusterContext string
}

type ModJobApplyStruct struct{}

var ModJobApply ModJobApplyStruct

func (ModJobApplyStruct) JobCmdName() string {
	return "apply-k8s"
}

func (ModJobApplyStruct) ParseJob(applyCmd *cobra.Command) *cobra.Command {
	job := &ApplyJob{}

	// 绑定命令行标志到结构体字段
	applyCmd.Flags().StringVar(&job.Project, "project", "", "Sub project dir in user specified workspace")
	applyCmd.Flags().StringArrayVar(&job.K8sDirs, "k8s", []string{}, "Path to k8s yaml dirs")
	applyCmd.Flags().StringArrayVar(&job.K8sNamespaces, "k8s-ns", []string{}, "Helm namespace")
	applyCmd.Flags().StringArrayVar(&job.HelmDirs, "helm", []string{}, "Path to helm yaml dirs")
	applyCmd.Flags().StringArrayVar(&job.HelmNamespaces, "helm-ns", []string{}, "Helm namespace")
	applyCmd.Flags().StringVar(&job.ClusterContext, "cluster-context", "", "Cluster context")

	applyCmd.Run = func(_ *cobra.Command, _ []string) {
		if job.Project == "" {
			fmt.Println(color.RedString("No project provided"))
			os.Exit(1)
		}
		ModJobApply.applyLocal(*job)
	}
	// err := applyCmd.Execute()
	// if err != nil {
	// 	return nil,nil
	// }

	return applyCmd
}

func (_ ModJobApplyStruct) applyLocal(job ApplyJob) {
	fmt.Println(color.BlueString("Applying k8s %s, k8ss:%v, helms:%v %v, cluster_context:%s",
		job.Project,
		job.K8sDirs,
		job.HelmDirs,
		job.HelmNamespaces,
		job.ClusterContext))
	if len(job.HelmDirs) != len(job.HelmNamespaces) {
		fmt.Println(color.RedString("helm:%v, helm-ns%v suppose to be aligned"))
		return
	}
	os.Chdir(filepath.Join(ConfigLoad().ProjectDir, job.Project))

	errs := []error{}
	for i, k8s := range job.K8sDirs {
		k8sNs := job.K8sNamespaces[i]
		cmds := []string{"kubectl", "apply", "-f", k8s, "--context", job.ClusterContext}
		if k8sNs != "" {
			cmds = append(cmds, "--namespace", k8sNs)
		}
		_, err := util.ModRunCmd.ShowProgress(cmds[0], cmds[1:]...).BlockRun()
		if err != nil {
			fmt.Println(color.RedString("Error in k8s dir %s: %v", k8s, err))
			errs = append(errs, err)
		}
	}

	if len(job.HelmDirs) != 0 {
		if err := NewBinManager(BinManagerHelm{}).MakeSureWith(); err != nil {
			fmt.Println(color.RedString("Error in compatible with helm: %v", err))
			os.Exit(1)
		}
	}

	for i, helmDir := range job.HelmDirs {
		config := ""
		if strings.Contains(helmDir, ",") {
			// split helm dir and overwrite config with ,
			split := strings.Split(helmDir, ",")
			helmDir = split[0]
			config = split[1]
		}
		helmNs := job.HelmNamespaces[i]
		// helm := filepath.Base(helmDir)
		standardProjectName := strings.ReplaceAll(job.Project, "_", "-")
		helmCmds := []string{"helm", "install", standardProjectName, helmDir, "--kube-context", job.ClusterContext}
		if helmNs != "" {
			helmCmds = append(helmCmds, "--namespace", helmNs)
		}
		// config is a file path
		if config != "" {
			helmCmds = append(helmCmds, "-f", config)
		}

		_, err := util.ModRunCmd.ShowProgress(helmCmds[0], helmCmds[1:]...).BlockRun()
		if err != nil {
			fmt.Println(color.RedString("helm install failed, cmd:%v,dir:%s, will retry with upgrade", helmCmds, helmDir))

			helmCmds[1] = "upgrade"
			_, err := util.ModRunCmd.ShowProgress(helmCmds[0], helmCmds[1:]...).BlockRun()
			if err != nil {
				fmt.Println(color.RedString("helm install/upgrade failed, cmd:%v,dir:%s", helmCmds, helmDir))
				// return
				errs = append(errs, err)
			}
		}
	}

	if len(errs) != 0 {
		fmt.Println(color.RedString("Apply failed with, errs: %v", errs))
		os.Exit(1)
	} else {
		fmt.Println(color.GreenString("Applyed %s", job.Project))
	}
}

func (m ModJobApplyStruct) NewApplyCmd(
	prj string,
	k8ss map[string]DeploymentK8s,
	helms map[string]DeploymentHelm,
	clusterContextName string,
) []string {
	cmds := []string{"telego", m.JobCmdName(), "--project", prj}

	for _, k8s := range k8ss {
		cmds = append(cmds, "--k8s", *k8s.K8sDir)
		if k8s.Namespace != nil && *k8s.Namespace != "" {
			cmds = append(cmds, "--k8s-ns", *k8s.Namespace)
		} else {
			cmds = append(cmds, "--k8s-ns", "\"\"")
		}
	}
	for _, helm := range helms {
		if helm.OverwriteConfig != nil && *helm.OverwriteConfig != "" {
			cmds = append(cmds, "--helm", *helm.HelmDir+","+*helm.OverwriteConfig)
		} else {
			cmds = append(cmds, "--helm", *helm.HelmDir)
		}
		if helm.Namespace != nil && *helm.Namespace != "" {
			cmds = append(cmds, "--helm-ns", *helm.Namespace)
		} else {
			cmds = append(cmds, "--helm-ns", "\"\"")
		}
	}
	cmds = append(cmds, "--cluster-context", clusterContextName)
	return cmds
}

// in app entry
func (_ ModJobApplyStruct) ApplyLocal(k8sprj string, k8sdp *Deployment, clusterContextName string) {
	// cmds := []string{}
	cmds := ModJobApply.NewApplyCmd(k8sprj, k8sdp.K8s, k8sdp.Helms, clusterContextName)
	util.ModRunCmd.ShowProgress(cmds[0], cmds[1:]...).BlockRun()
	// cmd += NewApplyCmd(binpack, bin, binBin[bin])

	// util.Logger.Debugf("apply cmds split: %s", cmds)

}
