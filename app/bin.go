package app

import (
	"fmt"
	"path/filepath"
	"telego/util"

	"github.com/fatih/color"
)

type BinManager interface {
	CheckInstalled() bool
	// without bin_ prefix
	BinName() string
	SpecInstallFunc() func() error
}

type BinMangerWrapper struct {
	b BinManager
}

func NewBinManager(b BinManager) BinMangerWrapper {
	return BinMangerWrapper{b}
}

// call at beginning
func (w BinMangerWrapper) MakeSureWith() error {
	util.PrintStep("make sure with bin", "checking "+w.b.BinName())
	if !w.b.CheckInstalled() {
		util.PrintStep("make sure with bin", "installing "+w.b.BinName())
		specInstall := w.b.SpecInstallFunc()
		printres := func(res error) {
			if res != nil {
				fmt.Println(color.RedString("install %s failed, err: %v", w.b.BinName(), res))
			} else {
				util.PrintStep("make sure with bin", "install "+w.b.BinName()+" success")
			}
		}
		if specInstall != nil {
			res := specInstall()
			printres(res)
			return res
		}
		res := ModJobInstall.InstallLocal("bin_" + w.b.BinName())
		printres(res)
		return res
	}
	fmt.Println(color.GreenString(w.b.BinName() + " already installed"))
	return nil
}

func BinInstallDir(project string) string {
	return filepath.Join(util.WorkspaceDir(), project, "install")
}
