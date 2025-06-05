package app

import (
	"fmt"
	"os"
	"strings"
	"telego/util"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type SshPasswdAuthJob struct {
	Enable bool
}

type ModJobSshPasswdAuthStruct struct{}

var ModJobSshPasswdAuth ModJobSshPasswdAuthStruct

func debugSshConfig(withbackup string) {
	withLineNumber := func(content []byte) string {
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			// fmt.Printf("%d: %s\n", i+1, line)
			lines[i] = fmt.Sprintf("%d: %s", i+1, line)
		}
		return strings.Join(lines, "\n")
	}

	if len(withbackup) > 0 {
		// debug old file and new file
		fmt.Println(color.RedString("********** Old file **********"))
		oldFile, err := os.ReadFile(withbackup)
		if err != nil {
			fmt.Println(color.RedString("Failed to read old file: %w", err))
		}

		fmt.Println(withLineNumber(oldFile))

	}
	fmt.Println(color.RedString("********** New file **********"))
	newFile, err := os.ReadFile(sshdConfigPath)
	if err != nil {
		fmt.Println(color.RedString("Failed to read new file: %w", err))
	}

	fmt.Println(withLineNumber(newFile))
}
func (m ModJobSshPasswdAuthStruct) ConfigureSshPasswdAuth(enable bool) (string, error) {
	action := "Enabling"
	if !enable {
		action = "Disabling"
	}
	util.PrintStep("JobSshPasswdAuth ConfigureSshPasswdAuth", fmt.Sprintf("%s SSH password authentication", action))

	// Only support Linux platforms
	if !util.IsLinux() {
		fmt.Println(color.RedString("This feature is only supported on Linux systems"))
		return "", fmt.Errorf("unsupported operating system")
	}

	err := restartSshService()
	if err != nil {
		fmt.Println(color.RedString("Current ssh configuration maybe wrong, err: %w", err))
		debugSshConfig("")
		return "", fmt.Errorf("failed to restart SSH service: %w", err)
	}

	// 1. Update SSH config to allow/disallow password authentication
	backupFile, err := configureSshdConfig(enable)
	if err != nil {
		return "", fmt.Errorf("failed to configure SSH server: %w", err)
	}

	// 2. Restart SSH service
	err = restartSshService()
	if err != nil {
		debugSshConfig(backupFile)
		return backupFile, fmt.Errorf("failed to restart SSH service: %w", err)
	}

	return backupFile, nil
}

var sshdConfigPath = "/etc/ssh/sshd_config"

func configureSshdConfig(enable bool) (string, error) {

	// Check if file exists
	_, err := os.Stat(sshdConfigPath)
	if err != nil {
		return "", fmt.Errorf("SSH config file not found: %w", err)
	}

	// Read current config
	content, err := os.ReadFile(sshdConfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to read SSH config: %w", err)
	}

	// Create backup
	backupPath := sshdConfigPath + ".bak." + util.CurrentTimeString()
	output, err := util.WriteFileWithContent(backupPath, string(content))
	if err != nil {
		return "", fmt.Errorf("failed to create backup, err:%w, output:%s", err, output)
	}

	util.PrintStep("JobSshPasswdAuth ConfigureSshdConfig", fmt.Sprintf("Created backup at %s", backupPath))

	// Update configuration settings
	config := string(content)

	// Set value based on enable parameter
	value := "yes"
	if !enable {
		value = "no"
	}

	// Update PasswordAuthentication
	config = updateSshConfigSetting(config, "PasswordAuthentication", value)

	// // Update ChallengeResponseAuthentication if present
	// config = updateSshConfigSetting(config, "ChallengeResponseAuthentication", "yes")

	// // Update UsePAM if present
	// config = updateSshConfigSetting(config, "UsePAM", "yes")

	// Write updated config
	output, err = util.WriteFileWithContent(sshdConfigPath, config)
	if err != nil {
		return backupPath, fmt.Errorf("failed to write updated SSH config: %w, output: %s", err, output)
	}

	return backupPath, nil
}

func updateSshConfigSetting(config, setting, value string) string {
	lines := strings.Split(config, "\n")
	updated := false

	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), setting+" ") ||
			strings.HasPrefix(strings.TrimSpace(line), "#"+setting+" ") {
			lines[i] = setting + " " + value
			updated = true
			break
		}
	}

	// Add setting if not found
	if !updated {
		lines = append(lines, setting+" "+value)
	}

	return strings.Join(lines, "\n")
}

func restartSshService() error {
	util.PrintStep("JobSshPasswdAuth RestartSshService", "Restarting SSH service")

	var err1, err2, err3 error
	var output1, output2, output3 string

	// Try systemctl first (systemd)
	// try 5 times
	for i := 0; i < 5; i++ {
		output1, err1 = util.ModRunCmd.NewBuilder("systemctl", "restart", "sshd").WithRoot().ShowProgress().BlockRun()

		if err1 == nil {
			return nil
		}

		// Try service command (older systems)
		output2, err2 = util.ModRunCmd.NewBuilder("service", "sshd", "restart").WithRoot().ShowProgress().BlockRun()
		if err2 == nil {
			return nil
		}

		// Try ssh instead of sshd (some distributions)
		output3, err3 = util.ModRunCmd.NewBuilder("service", "ssh", "restart").WithRoot().ShowProgress().BlockRun()
		if err3 == nil {
			return nil
		}
	}

	return fmt.Errorf("failed to restart SSH service: err1: %w, err2: %w, err3: %w, output1: %s, output2: %s, output3: %s", err1, err2, err3, output1, output2, output3)
}

// Job entry point
func JobSshPasswdAuth(j SshPasswdAuthJob) error {
	// Only support Linux platforms
	if !util.IsLinux() {
		return fmt.Errorf("SSH password authentication is only supported on Linux systems")
	}

	_, err := ModJobSshPasswdAuth.ConfigureSshPasswdAuth(j.Enable)
	return err
}

func (m ModJobSshPasswdAuthStruct) NewCmd(job SshPasswdAuthJob) []string {
	return []string{"telego", m.JobCmdName(), "--enable", fmt.Sprintf("%t", job.Enable)}
}

func (m ModJobSshPasswdAuthStruct) JobCmdName() string {
	return "ssh-passwd-auth"
}

func (m ModJobSshPasswdAuthStruct) ParseJob(sshPasswdAuthCmd *cobra.Command) *cobra.Command {
	job := &SshPasswdAuthJob{}

	// Add the enable flag and make it required
	sshPasswdAuthCmd.Flags().BoolVar(&job.Enable, "enable", false, "Required. Set to true to enable SSH password authentication or false to disable it")
	sshPasswdAuthCmd.MarkFlagRequired("enable")

	sshPasswdAuthCmd.Run = func(_ *cobra.Command, _ []string) {
		err := JobSshPasswdAuth(*job)
		if err != nil {
			fmt.Println(color.RedString("SSH password authentication configuration failed: %v", err))
			os.Exit(1)
		}
		if job.Enable {
			fmt.Println(color.GreenString("SSH password authentication enabled successfully"))
		} else {
			fmt.Println(color.GreenString("SSH password authentication disabled successfully"))
		}
	}

	return sshPasswdAuthCmd
}
