package app

import (
	"strings"
	"telego/util"
)

// prefix: deploy/k8s_{project}/apply, cur: {cluster_name}
func (selected *MenuItem) EnterItemtagExeApply(prefixNodes []*MenuItem) DispatchExecRes {
	if len(prefixNodes) < 3 {
		util.Logger.Warnf("invalid tag exe apply command %v", selected)
		return DispatchExecRes{}
	}
	stack := prefixNodes

	return DispatchExecRes{
		Exit: true,
		ExitWithDelegate: func() {
			// parent := m.ParentNode()
			// current := m.Current
			var findK8s *MenuItem = nil
			var findDist *MenuItem = nil
			// find node begin with bin_
			for i := len(stack) - 1; i >= 0; i-- {
				if strings.HasPrefix(stack[i].Name, "k8s_") {
					findK8s = stack[i]
					break
				} else if strings.HasPrefix(stack[i].Name, "dist_") {
					findDist = stack[i]
					break
				}
			}

			if findK8s != nil {
				if findK8s.Deployment == nil {
					// fmt.Println(color.RedString("deployment is nil for %s", find.Name))
					util.Logger.Warnf("deployment is nil for %s", findK8s.Name)
				} else {
					ModJobApply.ApplyLocal(findK8s.Name, findK8s.Deployment, util.GetKubeContextByCluster(selected.Name))
				}
			} else if findDist != nil {
				cmds := ModJobApplyDist.NewApplyDistCmd(findDist.Name, util.GetKubeContextByCluster(selected.Name))
				_, err := util.ModRunCmd.NewBuilder(cmds[0], cmds[1:]...).ShowProgress().BlockRun()
				if err != nil {
					util.Logger.Errorf("apply dist %s error: %v", findDist.Name, err)
				} else {
					util.Logger.Infof("apply dist %s success", findDist.Name)
				}
				// ModJobApplyDist.ApplyDistLocal(findDist.Name, util.GetKubeContextByCluster(selected.Name))
			} else {
				util.Logger.Warnf("can not find k8s_ prefix project")
			}
		},
	}

}
