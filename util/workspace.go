package util

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/fatih/color"
)

var fakeWorkspace *string

func SetFakeWorkspace(dir string) {
	fakeWorkspace = &dir
}

func InitOwnedDir(){
	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("Error getting current user:", err)
		os.Exit(1)
	}
	// if root, return
	if currentUser.Uid == "0" {
		return
	}

	dirs := []string{
		"/teledeploy_secret",
		"/teledeploy",
	}
	
	// On Windows, adjust directory paths to use home directory
	if runtime.GOOS == "windows" {
		dirs = []string{
			filepath.Join(currentUser.HomeDir, "teledeploy_secret"),
			filepath.Join(currentUser.HomeDir, "teledeploy"),
		}
	}
	
	for _, dir := range dirs {
		if runtime.GOOS == "windows" {
			// Windows: Use os.MkdirAll directly, no need for chown/chmod
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				fmt.Println(color.RedString("Error creating directory %s, err: %v", dir, err))
				os.Exit(1)
			}
		} else {
			// Linux/Unix: Use the existing command approach
			cmds:=[][]string{
				{"mkdir","-p",dir},
				{"chown",currentUser.Uid+":"+currentUser.Gid,"-R",dir},
				{"chmod","700",dir},
			}
			for _, cmd := range cmds {
				_,err:=ModRunCmd.NewBuilder(
					cmd[0],cmd[1:]...).
					// fmt.Sprintf(
					// "mkdir -p %s && chown -R %s:%s %s && chmod 700 %s", 
					// dir, currentUser.Uid, currentUser.Gid, dir, dir)
					
					WithRoot().
					ShowProgress().
					BlockRun()
				if err != nil {
					fmt.Println(color.RedString("Error creating and owning directory, err: %v", err))
					os.Exit(1)
				}
			}
		}
	}
}

func WorkspaceDir() string {
	if fakeWorkspace != nil {
		return *fakeWorkspace
	}

	var dirPath string
	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("Error getting current user:", err)
		return ""
	}

	// Determine directory based on OS
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		// For Windows and macOS, use user's home directory
		dirPath = filepath.Join(currentUser.HomeDir, "teledeploy_secret")
	} else {
		// For Linux, use the root path /teledeploy_secret
		dirPath = "/teledeploy_secret"
	}

	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		// Create directory
		err := makeDirAll(dirPath)
		if err != nil {
			fmt.Println("Error creating directory:", err)
			return ""
		}
	}

	// Call function to handle chown based on OS and permissions
	err = chownDirectory(dirPath, currentUser)
	if err != nil {
		fmt.Println("Error in chown operation:", err)
		return ""
	}

	return dirPath
}

// chownDirectory handles ownership change based on the current OS and user permissions
func chownDirectory(dirPath string, currentUser *user.User) error {
	// On Windows, skip the chown operation
	if runtime.GOOS == "windows" {
		// fmt.Println("Skipping chown on Windows")
		return nil
	}

	// On Linux/macOS, check if user is root
	uid := syscall.Getuid()
	if uid != 0 {
		// If not root, use sudo to perform chown
		// fmt.Println("Not running as root, attempting to use sudo for chown.")
		return sudoChown(dirPath, currentUser)
	}

	// If root, perform chown directly
	gid := syscall.Getgid()
	// fmt.Println("Running as root, performing direct chown.")
	return os.Chown(dirPath, uid, gid)
}

// sudoChown runs the `sudo chown` command to change ownership of the directory
func sudoChown(dirPath string, currentUser *user.User) error {
	cmd := exec.Command("sudo", "chown", fmt.Sprintf("%s:%s", currentUser.Uid, currentUser.Gid), dirPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func sudoMkdir(dirPath string) error {
	cmd := exec.Command("sudo", "mkdir", "-p", dirPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func makeDirAll(dirPath string) error {
	// if windows
	if runtime.GOOS == "windows" {
		// create dir in C:\Users\Public\teledeploy_secret
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			fmt.Println("Error creating directory:", err)
			return err
		}
	} else {
		// if cur user is root
		uid := syscall.Getuid()
		if uid == 0 {
			err := os.MkdirAll(dirPath, 0755)
			if err != nil {
				fmt.Println("Error creating directory:", err)
				return err
			}
		} else {
			// fmt.Println("Not running as root, attempting to use sudo for chown.")
			currentUser, err := user.Current()
			if err != nil {
				fmt.Println("Error getting current user:", err)
				return err
			}

			err = sudoMkdir(dirPath)
			if err != nil {
				fmt.Println("Error sudo creating directory:", err)
				return err
			}

			err = sudoChown(dirPath, currentUser)
			if err != nil {
				fmt.Println("Error sudo chown:", err)
				return err
			}
		}
	}

	return nil
}
