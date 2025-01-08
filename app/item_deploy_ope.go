package app

import (
	"fmt"
	"strings"
	"telego/util"

	"github.com/fatih/color"
)

// prefix: deploy/k8s_{project}, cur: prepare
func (i *MenuItem) EnterItemOpePrepare(prefixNodes []*MenuItem) DispatchExecRes {
	if len(prefixNodes) < 2 {
		util.Logger.Warnf("invalid ope prepare command %v", i)
		return DispatchExecRes{}
	}
	parentNode := prefixNodes[len(prefixNodes)-1]
	return DispatchExecRes{
		Exit: true,
		ExitWithDelegate: func() {
			// fmt.Println("deploymentPrepare", m.ParentNode().Name)
			err := DeploymentPrepare(parentNode.Name, parentNode.Deployment)
			if err != nil {
				fmt.Println(color.RedString("prepare '%s' failed, err: %v", parentNode.Name, err.Error()))
			} else {
				fmt.Println(color.GreenString("prepare '%s' success", parentNode.Name))
			}
		},
	}
}

// prefix: deploy/k8s_{project}, cur: upload
func (i *MenuItem) EnterItemOpeUpload(prefixNodes []*MenuItem) DispatchExecRes {
	if len(prefixNodes) < 2 {
		util.Logger.Warnf("invalid ope upload command %v", i)
		return DispatchExecRes{}
	}
	parentNode := prefixNodes[len(prefixNodes)-1]

	return DispatchExecRes{
		Exit: true,
		ExitWithDelegate: func() {
			// fmt.Println("deploymentPrepare", m.ParentNode().Name)
			err := DeploymentUpload(parentNode.Name, parentNode.Deployment)
			if err != nil {
				fmt.Println(color.RedString("upload '%s' failed, err: %v", parentNode.Name, err.Error()))
			} else {
				fmt.Println(color.GreenString("upload '%s' success", parentNode.Name))
			}
		},
	}
}

func (i *MenuItem) EnterItemOpeApply(prefixNodes []*MenuItem) DispatchEnterNextRes {
	if len(prefixNodes) < 1 {
		util.Logger.Warnf("EnterItemOpeApply prefixNodes < 1")
		return DispatchEnterNextRes{}
	}
	parentNode := prefixNodes[len(prefixNodes)-1]

	if strings.HasPrefix(parentNode.Name, "bin_") {
		return i.EnterItemBinInstall(prefixNodes)
	}

	if strings.HasPrefix(parentNode.Name, "k8s_") || strings.HasPrefix(parentNode.Name, "dist_") {
		clusters := util.KubeList()
		if len(clusters) == 0 {
			i.Children = []*MenuItem{(&MenuItem{
				Name:     "no_kube_config",
				Comment:  "update_config/fetch_{xxx}_kubeconfig 获取集群配置",
				Children: []*MenuItem{},
			}).setExeApplyTag()}
		} else {
			for _, cluster := range clusters {
				i.Children = append(i.Children, (&MenuItem{
					Name:     cluster,
					Comment:  "选择该目标集群进行部署",
					Children: []*MenuItem{},
				}).setExeApplyTag())
			}
		}

	}

	if len(i.Children) > 0 {
		return DispatchEnterNextRes{
			NextLevel: true,
		}
	} else {
		return DispatchEnterNextRes{}
	}
}
