package app

import (
	"path"
	"strings"
	"telego/app/config"
	"telego/util"
)

func (selected *MenuItem) IsDeploySubPrj(selectedParent string) bool {
	return selectedParent == "deploy" && (strings.HasPrefix(selected.Name, "bin_") || strings.HasPrefix(selected.Name, "k8s_"))
}

func (i *MenuItem) LoadDeploymentYml() {
	if i.IsDeploySubPrj("deploy") {
		if i.Deployment == nil {
			// curDir0 := util.CurDir()
			// os.Chdir(config.Load().ProjectDir)
			dep, err := LoadDeploymentYml(
				path.Join(config.Load().ProjectDir, i.Name),
			)
			if err != nil {
				// os.Chdir(curDir0)
				// allowNext = false
				if !strings.Contains(i.Comment, "[yml加载失败]") {
					i.Comment = i.Comment + "[yml加载失败]"
				}
				util.Logger.Warnf("%s yml加载失败: %s", i.Name, err)
				// allowNext = false
			} else {
				// os.Chdir(curDir0)
				i.Deployment = dep
			}
		}
	}
}
