package app

import (
	"fmt"
	"strings"
	"telego/app/template"
	"telego/util"

	"github.com/fatih/color"
	"github.com/thoas/go-funk"
)

type DispatchEnterNextRes struct {
	NextLevel bool
	PrevLevel bool
	Exit      bool
}

func (i *MenuItem) DispatchEnterNext(prefixNodes []*MenuItem) DispatchEnterNextRes {
	util.Logger.Debugf("DispatchEnterNext {%v}, prefixNodes {%v}", i, prefixNodes)
	var parentNode *MenuItem = nil
	if len(prefixNodes) >= 1 {
		parentNode = prefixNodes[len(prefixNodes)-1]
	}
	if parentNode != nil {
		if len(prefixNodes) >= 2 && prefixNodes[len(prefixNodes)-2].Name == "deploy-templete" {
			if i.Name == "install" {
				if strings.HasPrefix(parentNode.Name, "bin_") {
					util.Logger.Debugf("DispatchEnterNext: install bin")
					return i.EnterItemBinInstall(prefixNodes)
					// if len(selected.Children) > 0 {
					// 	// 进入下一级菜单
					// 	m.Stack = append(m.Stack, selected)
					// 	m.Current = selected
					// 	m.Filter = ""
					// }
				}
			}
		} else if i.Name == "deploy" {
			WaitInitMenuTree()
		} else if i.Name == "cancel" {
			util.Logger.Debugf("DispatchEnterNext: cancel")
			return i.EnterItemCancel(prefixNodes)
		} else if len(prefixNodes) >= 2 && prefixNodes[len(prefixNodes)-2].Name == "deploy" {
			if i.Name == "apply" {
				util.Logger.Debugf("DispatchEnterNext: apply")
				return i.EnterItemOpeApply(prefixNodes)
			}
		}
	}

	util.Logger.Debugf("DispatchEnterNext: not match, this %v, parentNode %v, len(prefixNodes) %v", i, parentNode, len(prefixNodes))
	return DispatchEnterNextRes{}
}

type DispatchExecRes struct {
	Exit             bool
	ExitWithDelegate func()
}

func (i *MenuItem) DispatchExec(prefixNodes []*MenuItem) DispatchExecRes {
	currentPath := strings.Join(funk.Map(prefixNodes, func(n *MenuItem) string {
		return n.Name
	}).([]string), "/")
	util.Logger.Debugf("DispatchExec {%v}, prefixes: {%v}", i, currentPath)
	var parentNode *MenuItem = nil
	if len(prefixNodes) > 1 {
		parentNode = prefixNodes[len(prefixNodes)-1]
	}
	// current := prefixNodes[len(prefixNodes)-2]
	// stack := prefixNodes[:len(prefixNodes)-2]

	if i.Name == "deploy-templete-upload" {
		return i.EnterItemUploadTemplate()
	} else if parentNode != nil { // deployments.yml 模板项目
		if len(prefixNodes) >= 2 && prefixNodes[len(prefixNodes)-2].Name == "deploy-templete" {
			if i.Name == "generate" {
				util.Logger.Debugf("DispatchExec match generate")
				return DispatchExecRes{
					Exit: true,
					ExitWithDelegate: func() {
						tempDir := template.GenSpecTemp(strings.ReplaceAll(currentPath, "主菜单/", ""))
						fmt.Println(color.GreenString("生成模板成功,请根据具体情况复制并修改名称及参数 %s ", tempDir))
					},
				}
			} else if i.Name == "install" {
			}

		} else if i.Name == "start_mainnode_fileserver" {
			return ModJobStartFileserver.ExecDelegate(
				StartFileserverJob{Mode: StartFileserverModeCaller}.ModeString())
		} else if parentNode.Name == "ssh_config" {
			return ModJobSsh.ExecDelegate(i.Name)
			// if i.Name == "1.gen_or_get_key" {

			// } else if i.Name == "2.update_key_to_cluster" {

			// }
		} else if i.Name == "run" {
			util.Logger.Debugf("DispatchExec match run")
			return i.EnterItemRun(prefixNodes)
			// } else if strings.Join([]string{m.CurrentPath(0), selected.Name}, "/") == "主菜单/upload_template" {
			// 	m.SecondLevelView = NewTemporaryInputViewModel(
			// 		color.GreenString("请输入模板在目录中的相对路径, 当前目录 %s", LoadConfig().ProjectDir),
			// 		"", "(回车确认，ctrl+c取消)")
			// 	m.Shared.EndDelegate = func() {
			// 		userinput := m.SecondLevelView.(*TemporaryInputViewModel).textInput.Value()
			// 		UploadToMainNode(userinput, "/teledeploy/template/")
			// 	}
		} else if parentNode.Name == "deploy" && strings.HasPrefix(i.Name, "dist_") {
			if i.Name == "dist_k3s" {
				return ModJobDistributeDeploy.ExecDelegate(DistributeDeployJob{
					Deployer: NewDistributeDeployer("k3s"),
					Mode:     DistDeployModeDeployerAll,
				})
			}
		} else if len(prefixNodes) >= 2 && prefixNodes[len(prefixNodes)-2].Name == "deploy" {
			if i.Name == "prepare" {
				util.Logger.Debugf("DispatchExec match deploy.prepare")
				return i.EnterItemOpePrepare(prefixNodes)
			} else if i.Name == "upload" {
				util.Logger.Debugf("DispatchExec match deploy.upload")
				return i.EnterItemOpeUpload(prefixNodes)
			}
		} else if i.isExeInstall() {
			util.Logger.Debugf("DispatchExec match install")
			return i.EnterItemtagExeInstall(prefixNodes)
		} else if i.isExeApply() {
			util.Logger.Debugf("DispatchExec match apply")
			return i.EnterItemtagExeApply(prefixNodes)
		} else if i.Name == "fetch_admin_kubeconfig" {
			util.Logger.Debugf("DispatchExec match fetch_admin_kubeconfig")
			return i.EnterItemFetchAdminKubeconfig(prefixNodes)
		} else if i.Name == "create_new_user_config" {
			util.Logger.Debugf("DispatchExec match create_new_user_kubeconfig")
			return i.EnterItemCreateNewUser(prefixNodes)
			// util.Logger.Warnf("can not find any handler for %s", i)
		} else {
			util.Logger.Warnf("can not find any handler for %+v", i)
		}
		// else if len(m.Stack) >= 3 && m.Stack[len(m.Stack)-3].Name == "deploy" {
		// 	if m.ParentNode().Name == "apply" {
		// 		if strings.HasPrefix(m.ParentNode().Name, "bin_") {
		// 		} else if strings.HasPrefix(m.ParentNode().Name, "k8s_") {
		// 		}
		// 	}
		// }

		// 生成模板 或 选择项目并执行
		// if selected.Name == "gen_temp" {

		// } else {

		// }
	} else {
		util.Logger.Warnf("can not find any handler for %+v", i)
	}

	return DispatchExecRes{}
}
