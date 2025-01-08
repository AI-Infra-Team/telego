package app

import (
	"telego/app/config"
	"telego/util"
)

func ConfigLoad() config.Config {
	return config.Load(util.WorkspaceDir(), util.StartTemporaryInputUI)
}
