package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"telego/util"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/thoas/go-funk"
)

type InstallJob struct {
	BinPrj string
	Bin    string // left empty to install all
	// BinMeta DeploymentBinDetails
}

type ModJobInstallStruct struct{}

var ModJobInstall ModJobInstallStruct

func (m ModJobInstallStruct) JobCmdName() string {
	return "install"
}

func (_ ModJobInstallStruct) ParseJob(installCmd *cobra.Command) *cobra.Command {
	job := &InstallJob{}
	// Bind command line flags to struct fields
	installCmd.Flags().StringVar(&job.BinPrj, "bin-prj", "", "Path to install")
	installCmd.Flags().StringVar(&job.Bin, "bin", "", "Path to binary")
	// bool job.BinMeta.NoDefaultInstaller
	// installCmd.Flags().BoolVar(&job.BinMeta.NoDefaultInstaller, "no-default-installer", false, "No default installer")
	// installCmd.Flags().StringVar(&job.BinMeta.WinInstaller, "win-installer", "", "Windows installer")
	// installCmd.Flags().StringVar(&job.BinMeta.Appimage, "appimage", "", "Appimage")

	installCmd.Run = func(_ *cobra.Command, _ []string) {
		fmt.Println(color.BlueString("Install job running %s %s", job.BinPrj, job.Bin))
		// if job.Bin == "" {
		// 	fmt.Println(color.RedString("No bin provided"))
		// 	os.Exit(1)
		// }
		if job.BinPrj == "" {
			fmt.Println(color.RedString("No bin provided"))
			os.Exit(1)
		}
		ModJobInstall.InstallLocalByJob(*job)
	}

	// err := installCmd.Execute()
	// if err != nil {
	// 	return
	// }

	return installCmd

	// return JobParse{
	// 	Cmd: installCmd,
	// 	Cb: func() {

	// 		os.Exit(0)
	// 	},
	// }
}

func (_ ModJobInstallStruct) getBinDeploymentFromMainNode(job InstallJob) (*Deployment, error) {
	url := fmt.Sprintf("http://%s:8003/%s/deployment.yml", util.MainNodeIp, job.BinPrj)
	ymlData, err := util.HttpGetUrlContent(url)
	if err != nil {
		util.Logger.Warnf("getBinDeploymentFromMainNode Failed to fetch file from %s: %s", url, err)
		return nil, err
	}
	return LoadDeploymentYmlByContent(job.BinPrj, "", ymlData)
}

func (_ ModJobInstallStruct) InstallLocalByJob(job InstallJob) {
	// Internal helper structure to organize installation-related methods
	installer := struct {
		job InstallJob

		// Unified meta fetching, prioritize local, return deployment info and local flag
		getBinDeploymentUnified func() (*Deployment, bool, error)

		// Install default binary for specified arch and system, determine install location based on metadata source
		installDefaultBin func(binname string, arch string, isLocal bool, installTempDir string) error

		// Install non-default binary files (custom installers)
		installCustomBin func(binname string, bininfo DeploymentBinDetails, isLocal bool, installTempDir string) error
	}{
		job: job,
	}

	// Encapsulate unified meta fetching method
	installer.getBinDeploymentUnified = func() (*Deployment, bool, error) {
		// Try local first
		localDir := filepath.Join(ConfigLoad().ProjectDir, job.BinPrj)
		localPath := filepath.Join(localDir, "deployment.yml")
		if _, err := os.Stat(localPath); err == nil {
			util.Logger.Debugf("Found local deployment.yml at %s", localPath)
			deployment, err := LoadDeploymentYml(job.BinPrj, localDir)
			if err == nil {
				return deployment, true, nil // Return local flag true
			}
			util.Logger.Warnf("Failed to load local deployment.yml: %s", err)
		}

		// Local fetch failed, fetch from main node
		util.Logger.Debugf("Local deployment.yml not found or failed to load, fetching from main node")
		deployment, err := ModJobInstall.getBinDeploymentFromMainNode(job)
		return deployment, false, err // Return local flag false
	}

	// Encapsulate default installer method
	installer.installDefaultBin = func(binname string, arch string, isLocal bool, installTempDir string) error {
		util.PrintStep("install", fmt.Sprintf("%s/%s with default installer (arch: %s, local: %v)", job.BinPrj, binname, arch, isLocal))

		var baseUrl string
		if isLocal {
			// Local installation, use local file path
			localBinDir := filepath.Join(ConfigLoad().ProjectDir, job.BinPrj, "teledeploy")
			if util.IsWindows() {
				binPath := filepath.Join(localBinDir, binname+".exe")
				if _, err := os.Stat(binPath); err == nil {
					// Local file exists, copy directly

					// return util.SafeCopyOverwrite(binPath, filepath.Join(installTempDir, binname+".exe"))
					return util.InstallWindowsPreparedBin(binPath, binname)
				}
			} else {
				binPath := filepath.Join(localBinDir, binname+"_"+arch)
				if _, err := os.Stat(binPath); err == nil {
					// Local file exists, copy directly
					// return util.SafeCopyOverwrite(binPath, filepath.Join(installTempDir, binname))
					return util.InstallLinuxPreparedBin(binPath, binname)
				}
			}
			// Local file doesn't exist, fallback to remote
			util.Logger.Warnf("Local binary not found, falling back to remote download")
			baseUrl = fmt.Sprintf("http://%s:8003/%s", util.MainNodeIp, job.BinPrj)
		} else {
			// Remote installation, use HTTP download
			baseUrl = fmt.Sprintf("http://%s:8003/%s", util.MainNodeIp, job.BinPrj)
		}

		if util.IsWindows() {
			return util.InstallWindowsBin(
				fmt.Sprintf("%s/%s.exe", baseUrl, binname),
				installTempDir,
				binname,
			)
		} else {
			return util.InstallLinuxBin(
				fmt.Sprintf("%s/%s_%s", baseUrl, binname, arch),
				installTempDir,
				binname,
			)
		}
	}

	// Encapsulate custom installer method
	installer.installCustomBin = func(binname string, bininfo DeploymentBinDetails, isLocal bool, installTempDir string) error {
		// Python script installer
		if pyinstall, err := bininfo.PyInstaller(job.BinPrj); err == nil {
			util.PrintStep("install", fmt.Sprintf("%s/%s with pyscript", job.BinPrj, binname))
			err := pyinstall.Run()
			if err != nil {
				return fmt.Errorf("failed to install with pyscript %s: %v", binname, err)
			}
			return nil
		}

		// Windows installer
		if util.IsWindows() {
			if bininfo.WinInstaller != "" {
				util.PrintStep("install", fmt.Sprintf("%s/%s with win_installer", job.BinPrj, binname))

				if isLocal {
					// Local installation
					localInstallerPath := filepath.Join(util.WorkspaceDir(), job.BinPrj, bininfo.WinInstaller)
					if _, err := os.Stat(localInstallerPath); err == nil {
						// Local file exists, copy directly
						err := util.SafeCopyOverwrite(localInstallerPath, filepath.Join(installTempDir, bininfo.WinInstaller))
						if err != nil {
							return fmt.Errorf("failed to copy local installer: %v", err)
						}
					} else {
						// Local file doesn't exist, fallback to remote download
						installerUrl := fmt.Sprintf("http://%s:8003/%s/%s", util.MainNodeIp, job.BinPrj, bininfo.WinInstaller)
						util.DownloadFile(installerUrl, filepath.Join(installTempDir, bininfo.WinInstaller))
					}
				} else {
					// Remote download
					installerUrl := fmt.Sprintf("http://%s:8003/%s/%s", util.MainNodeIp, job.BinPrj, bininfo.WinInstaller)
					util.DownloadFile(installerUrl, filepath.Join(installTempDir, bininfo.WinInstaller))
				}

				if strings.HasSuffix(bininfo.WinInstaller, ".msi") {
					_, err := util.ModRunCmd.ShowProgress("msiexec", "/i", strings.ReplaceAll(filepath.Join(installTempDir, bininfo.WinInstaller), "/", "\\\\"), "/quiet", "/norestart").BlockRun()
					if err != nil {
						return fmt.Errorf("failed to install %s: %v", binname, err)
					}
				} else {
					_, err := util.ModRunCmd.ShowProgress(filepath.Join(installTempDir, bininfo.WinInstaller)).BlockRun()
					if err != nil {
						return fmt.Errorf("failed to install %s: %v", binname, err)
					}
				}
				return nil
			} else {
				return fmt.Errorf("at least one of 'non no_default_installer' or 'win_installer' should be provided for windows")
			}
		}

		// Linux AppImage installer
		if bininfo.Appimage != "" {
			util.PrintStep("install", fmt.Sprintf("%s/%s with appimage", job.BinPrj, binname))
			return fmt.Errorf("appimage not supported for now")
		}

		return fmt.Errorf("at least one of 'non no_default_installer' or 'appimage' should be provided for linux")
	}

	// Main installation process starts
	fmt.Println(color.BlueString("Fetching %s meta", job.BinPrj))
	dplymnt, isLocal, err := installer.getBinDeploymentUnified()
	if err != nil {
		fmt.Println(color.RedString("Failed to fetch meta: %s", err.Error()))
		os.Exit(1)
	}

	if isLocal {
		fmt.Println(color.GreenString("Using local deployment configuration"))
	} else {
		fmt.Println(color.GreenString("Using remote deployment configuration"))
	}

	// Filter binaries to install
	bins := map[string]DeploymentBinDetails{}
	if job.Bin == "" {
		for binname, bin := range dplymnt.Bin {
			bins[binname] = bin
		}
	} else {
		for binname, bin := range dplymnt.Bin {
			if binname == job.Bin {
				bins[binname] = bin
				break
			}
		}
	}

	if len(bins) == 0 {
		fmt.Println(color.RedString("Failed to find bin %s in meta", job.Bin))
		return
	}

	// Execute installation
	util.PrintStep("install", fmt.Sprintf("%s / %v", job.BinPrj, bins))
	os.Chdir(util.WorkspaceDir())

	for binname, bininfo := range bins {
		util.PrintStep("install", fmt.Sprintf("one sub bin %s / %s with config %v", job.BinPrj, binname, bininfo))
		installTempDir := filepath.Join("install_"+job.BinPrj, binname)
		os.MkdirAll(installTempDir, 0755)
		defer os.RemoveAll(installTempDir)

		var installErr error

		// Choose installation method based on configuration
		if !bininfo.NoDefaultInstaller {
			// Use default installer
			arch := util.GetCurrentArch()
			installErr = installer.installDefaultBin(binname, arch, isLocal, installTempDir)
		} else {
			// Use custom installer
			installErr = installer.installCustomBin(binname, bininfo, isLocal, installTempDir)
		}

		if installErr != nil {
			fmt.Println(color.RedString("Failed to install %s: %s", binname, installErr.Error()))
			os.Exit(1)
		} else {
			fmt.Println(color.GreenString("Installed %s / %s", job.BinPrj, binname))
		}
	}
}

func NewInstallCmd(binPack, bin string) []string {
	cmds := []string{"telego", "install",
		"--bin", bin,
		"--bin-prj", binPack}
	// "--no-default-installer %v "+
	// "--win-installer", meta.WinInstaller,
	// "--appimage", meta.Appimage}

	// if meta.NoDefaultInstaller {
	// 	// cmd += "--no-default-installer "
	// 	cmds = append(cmds, "--no-default-installer")
	// }
	return cmds
}

// in app entry
func (_ ModJobInstallStruct) InstallLocal(binpack string) error {
	fmt.Println(color.BlueString("install %s local", binpack))
	cmds := NewInstallCmd(binpack, "")
	_, err := util.ModRunCmd.ShowProgress(cmds[0], cmds[1:]...).BlockRun()
	return err
}

// in app entry
func (_ ModJobInstallStruct) InstallToNodes(binpack string, cluster string, nodes []string) {
	fmt.Println(color.BlueString("install %s to remote", binpack))

	user, err := util.KubeSecretSshUser(cluster)
	if err != nil {
		util.Logger.Warn("failed to get ssh user from secret: " + err.Error())
		fmt.Println(color.RedString("failed to get ssh user from secret: " + err.Error()))
		os.Exit(1)
	}
	name2Ip, err := util.KubeNodeName2Ip(cluster)
	if err != nil {
		util.Logger.Warn("failed to get node ip: " + err.Error())
		fmt.Println(color.RedString("failed to get node ip: " + err.Error()))
		os.Exit(1)
	}
	hosts := funk.Map(nodes, func(node string) string {
		return user + "@" + name2Ip[node]
	}).([]string)

	cmd := fmt.Sprintf("python3 -c \"import urllib.request, os; script = urllib.request.urlopen('http://%s:8003/bin_telego/install.py').read(); exec(script.decode());\" ", util.MainNodeIp)
	cmd += "&& " + CmdsToCmd(NewInstallCmd(binpack, ""))

	util.Logger.Debugf("install cmd: %s", cmd)
	util.StartRemoteCmds(hosts, cmd, "")
}
