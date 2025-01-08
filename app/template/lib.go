package template

import (
	"fmt"
	"os"
	"path"
	"strings"
	"telego/util"

	"github.com/fatih/color"
)

func GenSpecTemp(thepath string) string {
	if util.PathIsAbsolute(thepath) {
		fmt.Println(color.RedString("genSpecTemp input path should be relative path %s", thepath))
		os.Exit(1)
	}

	// 1. git clone project to current dir
	if util.HasNetwork() {
		util.ModRunCmd.NewBuilder("git", "clone", "https://github.com/AI-Infra-Team/teleyard-template")
		os.Chdir("teleyard-template")
		util.ModRunCmd.NewBuilder("git", "pull")
		os.Chdir("..")
	} else {
		util.DownloadFile(fmt.Sprintf("http://%s:8003/teleyard-template.zip", util.MainNodeIp), "teleyard-template.zip")
		util.UnzipFile("teleyard-template.zip", "teleyard-template")
	}

	pathBase := path.Base(thepath)
	// pathDir := path.Dir(thepath)
	targetDir := path.Join(ConfigLoad().ProjectDir, pathBase)
	// fmt.Printf("%s -> %s\n", path.Join("teleyard-template", thepath), targetDir)
	// os.Exit(1)
	err := util.ModRunCmd.CopyDirContentOrFileTo(path.Join("teleyard-template", thepath), targetDir)
	if err != nil {
		fmt.Println(color.RedString("copy template failed %s", err))
	}
	entries, err := os.ReadDir(targetDir)
	if err == nil {
		for _, entry := range entries {
			if strings.Contains(entry.Name(), "upload.py") || strings.Contains(entry.Name(), "prepare.py") || strings.Contains(entry.Name(), "run.py") {
				os.Remove(path.Join(targetDir, entry.Name()))
			}
		}
	}

	return targetDir
}
