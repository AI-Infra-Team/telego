package app

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"telego/util"
	clusterconf "telego/util/cluster_conf"
	"telego/util/prjerr"
	"telego/util/yamlext"

	"github.com/barweiss/go-tuple"
	"github.com/fatih/color"
	"github.com/thoas/go-funk"
)

// some interfaces:
// - check_master: find the master nodes
// - check_worker: find the worker nodes
// - general_install: install the general part for master and worker
// - master_install: install the master part
// - worker_install: install the worker part
type DistributeDeployer interface {
	// check_master: find the master nodes
	CheckMaster(conf clusterconf.ClusterConfYmlModel) ([]clusterconf.NodeInfo, error)
	// check_worker: find the worker nodes
	CheckWorker(conf clusterconf.ClusterConfYmlModel) ([]clusterconf.NodeInfo, error)
	// maybe the token to join master or registry info
	PrepareWorkerSetupCtxBase64(masters []clusterconf.NodeInfo, conf clusterconf.ClusterConfYmlModel) (string, error)
	PrepareMasterSetupCtxBase64(masters []clusterconf.NodeInfo, conf clusterconf.ClusterConfYmlModel) (string, error)
	// install the general part with master & worker
	ThisNodeGeneralInstall() error
	// install the master part
	ThisNodeMasterInstall(ctxb64 string) error
	// install the worker part with ctx
	ThisNodeWorkerInstall(ctxb64 string) error
	Name() string
}

func NewDistributeDeployer(name string) DistributeDeployer {
	switch name {
	case "k3s":
		return DistributeDeployerK3s{}
	default:
		return nil
	}
}

type ModDistributeDeployStruct struct {
}

var ModDistributeDeploy ModDistributeDeployStruct

func (m ModDistributeDeployStruct) SetupAll(d DistributeDeployer) {
	// read cluster conf
	ok, yamlFilePath := util.StartTemporaryInputUI(color.GreenString(
		"初始集群配置需要初始集群配置文件 cluster_config.yml"),
		"此处键入 yaml 配置路径",
		"回车确认，ctrl+c取消，参照https://github.com/340Lab/serverless_benchmark_plus/blob/main/middlewares/cluster_config.yml")
	if !ok {
		fmt.Println("User canceled config cluster")
		os.Exit(1)
	}
	// load yaml
	// 读取 YAML 文件
	data, err := ioutil.ReadFile(yamlFilePath)
	if err != nil {
		fmt.Println(color.RedString("读取配置文件失败: %v", err))
		os.Exit(1)
	}

	// 解析 YAML
	util.PrintStep("DistributeDeploySetupAll", "reading yaml file...")
	var clusterConf clusterconf.ClusterConfYmlModel
	err = yamlext.UnmarshalAndValidate(data, &clusterConf)
	if err != nil {
		fmt.Println(color.RedString("解析 YAML 文件失败: %v", err))
		os.Exit(1)
	}

	// 打印解析后的内容
	fmt.Printf("解析后的集群配置: %+v\n", clusterConf)

	masters := funk.Map(
		// turn to array first because filter can't take map as parameter
		funk.Filter(
			funk.Map(
				clusterConf.Nodes,
				func(nodename string, node clusterconf.ClusterConfYmlModelNode) tuple.T2[string, clusterconf.ClusterConfYmlModelNode] {
					return tuple.New2(nodename, node)
				},
			),
			func(nodename_node tuple.T2[string, clusterconf.ClusterConfYmlModelNode]) bool {
				return funk.Contains(nodename_node.V2.Tags, func(tag string) bool { return tag == d.Name()+"_master" })
			}),

		func(nodename_node tuple.T2[string, clusterconf.ClusterConfYmlModelNode]) string {
			return nodename_node.V1
		}).([]string)

	util.PrintStep("DistributeDeploySetupAll", "setting up masters...")
	oldMasters, err := m.setupMasters(d, masters, clusterConf)
	mastersSame := false
	if err != nil {
		if err.Error() == prjerr.DistributeDeployMasterIsAlreadySetup().Error() {
			fmt.Println(color.YellowString("masters alredy exists:"))
			for _, m := range oldMasters {
				fmt.Println(color.YellowString("- %s", m))
			}
			// sort old Masters & masters
			if len(oldMasters) == len(masters) {
				mastersSame = true
				sort.Strings(oldMasters)
				sort.Strings(masters)
				for i, m := range oldMasters {
					if m != masters[i] {
						mastersSame = false
						break
					}
				}
			}
			if !mastersSame {
				fmt.Println(color.RedString("master distribution changed, fix manually: %v", err))
				return
			}
		} else {
			fmt.Println(color.RedString("setup master failed: %v", err))
			return
		}
	}

	workers := funk.Map(
		funk.Filter(
			funk.Map(
				clusterConf.Nodes,
				func(nodename string, node clusterconf.ClusterConfYmlModelNode) tuple.T2[string, clusterconf.ClusterConfYmlModelNode] {
					return tuple.New2(nodename, node)
				},
			),
			func(nodename_node tuple.T2[string, clusterconf.ClusterConfYmlModelNode]) bool {
				return funk.Contains(nodename_node.V2.Tags, func(tag string) bool { return tag == d.Name()+"_worker" })
			}),
		func(nodename_node tuple.T2[string, clusterconf.ClusterConfYmlModelNode]) string {
			return nodename_node.V1
		}).([]string)
	util.PrintStep("DistributeDeploySetupAll", "setting up workers...")
	m.setupWorkers(d, workers, clusterConf)

}

// unsafe: targetMasters Must Be In conf
// return the old masters
func (m ModDistributeDeployStruct) setupMasters(d DistributeDeployer, targetMasters []string, conf clusterconf.ClusterConfYmlModel) ([]string, error) {
	if len(targetMasters) == 0 {
		return nil, fmt.Errorf("targetMasters is empty")
	}
	var err error
	nodes := funk.Map(targetMasters, func(node string) clusterconf.NodeInfo {
		res, ok := conf.Nodes[node]
		if !ok {
			err = fmt.Errorf("node %s not found in conf", node)
			return clusterconf.NodeInfo{}
		}
		return clusterconf.NodeInfo{
			Ip:   res.Ip,
			Name: node,
		}
	}).([]clusterconf.NodeInfo)
	util.PrintStep("DistributeDeploySetupMaster", "checking master...")
	masternodes, err := d.CheckMaster(conf)
	if err != nil {
		fmt.Println(color.RedString("check master failed: %s", err))
		return []string{}, err
	}

	oldMasterSameAsTargetMasters := true
	if len(masternodes) != len(targetMasters) {
		oldMasterSameAsTargetMasters = false
	} else {
		for _, old := range masternodes {
			if !funk.ContainsString(targetMasters, old.Name) {
				oldMasterSameAsTargetMasters = false
			}
		}
	}

	if len(masternodes) == 0 || oldMasterSameAsTargetMasters {
		fmt.Println(color.BlueString("no master node found, start to create..."))
		fmt.Println("target nodes:")

		ctxb64, err := d.PrepareMasterSetupCtxBase64(nodes, conf)
		if err != nil {
			fmt.Println(color.RedString("setupWorkers prepareWorkerSetupCtxBase64 failed: %s", err))
			return []string{}, err
		}

		// print the nodes will be created
		for _, node := range nodes {
			fmt.Println(color.BlueString("- %+v", node))
		}

		hosts := funk.Map(nodes, func(node clusterconf.NodeInfo) string {
			return fmt.Sprintf("%s@%s", conf.Global.SshUser, node.Ip)
		}).([]string)

		util.PrintStep("DistributeDeploySetupMaster", "installing masters...")
		remoteResults := util.StartRemoteCmds(
			hosts,
			util.ModRunCmd.CmdModels().InstallTelegoWithPy()+" && "+
				strings.Join(ModJobDistributeDeploy.NewCmd(DistributeDeployJob{
					Deployer: DistributeDeployerK3s{},
					Mode:     DistDeployModeThisNodeMaster,
				}, ctxb64), " "),
			"",
		)

		for i := 0; i < len(remoteResults); i++ {
			fmt.Println()
			fmt.Println(color.BlueString("host %s end with result: %s", hosts[i], remoteResults[i]))
		}

		return []string{}, nil
	} else {
		err := prjerr.DistributeDeployMasterIsAlreadySetup()
		fmt.Println(color.YellowString("%v", err))

		return funk.Map(masternodes, func(n clusterconf.NodeInfo) string {
			return n.Name
		}).([]string), err
	}
}

// unsafe: targetWorkers Must Be In conf
func (m ModDistributeDeployStruct) setupWorkers(d DistributeDeployer, targetWorkers []string, conf clusterconf.ClusterConfYmlModel) error {
	util.PrintStep("DistributeDeploySetupWorker", "checking workers...")
	workernodes, err := d.CheckWorker(conf)
	if err != nil {
		fmt.Println(color.RedString("check worker failed: %v", err))
		return err
	}
	// fmt.Println("old workers", workernodes)
	targetWorkersInfo := funk.Map(
		// funk.Filter(targetWorkers, func(nodename string) bool {
		// 	return !funk.Contains(workernodes, func(node clusterconf.NodeInfo) bool {
		// 		return node.Name == nodename
		// 	})
		// }).([]string),
		targetWorkers,
		func(nodename string) clusterconf.NodeInfo {
			res, ok := conf.Nodes[nodename]
			if !ok {
				panic(fmt.Sprintf("node %s not found in conf", nodename))
			}
			return clusterconf.NodeInfo{
				Name: nodename,
				Ip:   res.Ip,
			}
		},
	).([]clusterconf.NodeInfo)

	masters := funk.Map(
		funk.Filter(
			funk.Map(conf.Nodes, func(name string, conf clusterconf.ClusterConfYmlModelNode) tuple.T2[string, clusterconf.ClusterConfYmlModelNode] {
				return tuple.New2(name, conf)
			}),
			func(nodename_info tuple.T2[string, clusterconf.ClusterConfYmlModelNode]) bool {
				return funk.ContainsString(nodename_info.V2.Tags, d.Name()+"_master")
			},
		),
		func(nodename_info tuple.T2[string, clusterconf.ClusterConfYmlModelNode]) clusterconf.NodeInfo {
			return clusterconf.NodeInfo{
				Name: nodename_info.V1,
				Ip:   nodename_info.V2.Ip,
			}
		},
	).([]clusterconf.NodeInfo)
	// prepareWorkerSetupCtx
	util.PrintStep("DistributeDeploySetupWorker", "preparing worker setup context...")
	ctxb64, err := d.PrepareWorkerSetupCtxBase64(masters, conf)
	if err != nil {
		fmt.Println(color.RedString("setupWorkers prepareWorkerSetupCtxBase64 failed: %s", err))
		return err
	}

	// startup newworkers
	if len(targetWorkersInfo) == 0 {
		fmt.Println(color.BlueString("no new worker to be created/updated"))
	} else {
		fmt.Println(color.BlueString("workers to be created/updated:"))
		for _, node := range targetWorkersInfo {
			fmt.Println(color.BlueString("- %+v", node))
		}
		newWorkerHosts := funk.Map(targetWorkersInfo, func(node clusterconf.NodeInfo) string {
			return fmt.Sprintf("%s@%s", conf.Global.SshUser, node.Ip)
		}).([]string)

		util.PrintStep("DistributeDeploySetupWorker", "installing workers...")
		util.StartRemoteCmds(
			newWorkerHosts,
			util.ModRunCmd.CmdModels().InstallTelegoWithPy()+" && "+
				strings.Join(ModJobDistributeDeploy.NewCmd(DistributeDeployJob{
					Deployer: d,
					Mode:     DistDeployModeThisNodeWorker,
				}, ctxb64), " "),
			"",
		)
	}

	removeWorkers := funk.Filter(workernodes, func(node clusterconf.NodeInfo) bool {
		return !funk.Contains(targetWorkers, func(nodename string) bool {
			return node.Name == nodename
		})
	})
	if len(removeWorkers.([]clusterconf.NodeInfo)) > 0 {
		fmt.Println(color.YellowString("Remove workers is not supported yet, workers to be removed:"))
		for _, node := range removeWorkers.([]clusterconf.NodeInfo) {
			fmt.Println(color.YellowString("- %+v", node))
		}
	}

	return nil
}
