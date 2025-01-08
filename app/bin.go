package app

import (
	"path"
	"telego/util"
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
	if !w.b.CheckInstalled() {
		specInstall := w.b.SpecInstallFunc()
		if specInstall != nil {
			return specInstall()
		}
		return ModJobInstall.InstallLocal("bin_" + w.b.BinName())
	}
	return nil
}

func BinInstallDir(project string) string {
	return path.Join(util.WorkspaceDir(), project, "install")
}
