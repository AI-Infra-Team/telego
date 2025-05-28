package app

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"telego/util"
)

// DeploymentBinDetails represents the details for a binary in the bin field.
type DeploymentBinDetails struct {
	NoDefaultInstaller bool    `yaml:"no_default_installer,omitempty"`
	WinInstaller       string  `yaml:"win_installer,omitempty"`
	Appimage           string  `yaml:"appimage,omitempty"`
	PyInstaller0       *string `yaml:"py_installer,omitempty"`
}

func (b *DeploymentBinDetails) PyInstaller(prjname string) (DeploymentBinDetailsPyInstaller, error) {
	if b.PyInstaller0 == nil || *b.PyInstaller0 == "" {
		return DeploymentBinDetailsPyInstaller{}, fmt.Errorf("py_installer is not set")
	}

	splits := strings.Fields( //*
		*b.PyInstaller0)
	return DeploymentBinDetailsPyInstaller{
		Script:  splits[0],
		Args:    splits[1:],
		PrjName: prjname,
	}, nil
}

type DeploymentBinDetailsPyInstaller struct {
	Script  string
	Args    []string
	PrjName string
}

// start the bin install process
func (d *DeploymentBinDetailsPyInstaller) Run() error {

	installdir := BinInstallDir(d.PrjName)
	err := os.MkdirAll(installdir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create install dir: %w", err)
	}
	err = os.Chdir(installdir)
	if err != nil {
		return fmt.Errorf("failed to change dir to install dir: %w", err)
	}
	err = util.DownloadFile(
		util.UrlJoin(util.MainNodeFileServerURL, d.PrjName, d.Script), filepath.Join(installdir, path.Base(d.Script)))
	if err != nil {
		return fmt.Errorf("failed to download file from %s: %w", d.Script, err)
	}

	cmds := []string{"python3", filepath.Base(d.Script)}
	cmds = append(cmds, d.Args...)
	if util.IsWindows() {
		cmds[0] = "python"
	}
	_, err = util.ModRunCmd.ShowProgress(cmds[0], cmds[1:]...).BlockRun()
	if err != nil {
		return fmt.Errorf("failed to run script: %w", err)
	}
	return nil
}
