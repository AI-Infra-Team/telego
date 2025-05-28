package app

import (
	"fmt"
	"os"
	"path/filepath"
	"telego/util"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"k8s.io/client-go/util/homedir"
)

func FetchAdminKubeconfig() {
	fmt.Println(color.BlueString("Fetching admin kubeconfig ..."))
	//mkdir
	os.MkdirAll(filepath.Join(homedir.HomeDir(), ".kube"), 0700)
	conf, err := util.MainNodeConfReader{}.ReadSecretConf(util.SecretConfTypeAdminKubeconfig{})
	if err != nil {
		fmt.Println(color.RedString("FetchAdminKubeconfig Error1: %s", err))
		os.Exit(1)
	}
	// open and write to file
	f, err := os.OpenFile(filepath.Join(homedir.HomeDir(), ".kube/config"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Println(color.RedString("FetchAdminKubeconfig Error2: %s", err))
		os.Exit(1)
	}
	defer f.Close()
	_, err = f.WriteString(conf)
	if err != nil {
		fmt.Println(color.RedString("FetchAdminKubeconfig Error3: %s", err))
		os.Exit(1)
	}
	// util.FetchFromMainNode("/teledeploy_secret/kubeconfig/config", filepath.Join(homedir.HomeDir(), ".kube"))
	fmt.Println(color.GreenString("Fetched admin kubeconfig"))
}

type ModJobFetchAdminKubeconfigStruct struct{}

var ModJobFetchAdminKubeconfig ModJobFetchAdminKubeconfigStruct

func (_ ModJobFetchAdminKubeconfigStruct) JobCmdName() string {
	return "fetch-admin-kubeconfig"
}

func (_ ModJobFetchAdminKubeconfigStruct) ParseJob(applyCmd *cobra.Command) *cobra.Command {

	applyCmd.Run = func(_ *cobra.Command, _ []string) {
		FetchAdminKubeconfig()
	}
	// err := applyCmd.Execute()
	// if err != nil {
	// 	return nil,nil
	// }

	return applyCmd
}
