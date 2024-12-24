package app

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"telego/app/config"
	"telego/util"

	"github.com/fatih/color"
)

func (i *MenuItem) EnterItemUploadTemplate() DispatchExecRes {
	return DispatchExecRes{
		Exit: true,
		ExitWithDelegate: func() {
			fmt.Println(color.BlueString("Uploading template..."))

			err := os.Chdir(util.WorkspaceDir())
			if err != nil {
				fmt.Println(color.RedString("Upload template error: %s", err))
			}

			localPath := path.Join(path.Dir(filepath.ToSlash(config.Load().ProjectDir)), "teleyard-template")

			err = os.RemoveAll("teleyard-template.zip")
			if err != nil {
				fmt.Println(color.RedString("Upload template error: %s", err))
			}

			err = util.ZipDirectory(localPath, "teleyard-template.zip")
			if err != nil {
				fmt.Println(color.RedString("Upload template error: %s", err))
			}

			util.UploadToMainNode("teleyard-template.zip", "/teledeploy/")
			fmt.Println(color.GreenString("Upload template success"))
		},
	}
}
