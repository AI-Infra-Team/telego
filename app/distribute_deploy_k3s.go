package app

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"telego/util"
	clusterconf "telego/util/cluster_conf"

	"github.com/barweiss/go-tuple"
	"github.com/fatih/color"
	"github.com/thoas/go-funk"
	"gopkg.in/yaml.v3"
)

type DistributeDeployerK3s struct{}

func (d DistributeDeployerK3s) BindInterface() {
	_ = DistributeDeployer(DistributeDeployerK3s{})
}

func (d DistributeDeployerK3s) CheckMaster(conf clusterconf.ClusterConfYmlModel) ([]clusterconf.NodeInfo, error) {
	// k3s kubectl get nodes --selector='node-role.kubernetes.io/master'
	hosts := funk.Map(
		conf.Nodes,
		func(nodename string, info clusterconf.ClusterConfYmlModelNode) string {
			return fmt.Sprintf("%s@%s", conf.Global.SshUser, info.Ip)
		},
	).([]string)

	remoteResults, _ := util.StartRemoteCmds(
		hosts,
		"k3s kubectl get nodes --selector='node-role.kubernetes.io/master'",
		"",
	)

	// find one with result like
	// NAME        STATUS   ROLES                  AGE    VERSION
	// {nodename}           control-plane,master
	find_ := funk.Find(remoteResults, func(result string) bool {
		return strings.Contains(result, "master")
	})
	if find_ == nil {
		return []clusterconf.NodeInfo{}, nil
	}

	find := find_.(string)
	nodenames := funk.Map(
		funk.Filter(
			strings.Split(find, "\n"),
			func(line string) bool {
				return strings.Contains(line, "master") && strings.Contains(line, "Ready")
			},
		),
		func(line string) string {
			util.Logger.Debugf("contain master info line %s", line)
			return strings.Split(line, " ")[0]
		},
	)

	var notfounderr error
	nodeinfos := funk.Map(
		nodenames,
		func(nodename string) clusterconf.NodeInfo {
			info, ok := conf.Nodes[nodename]
			if !ok {
				notfounderr = fmt.Errorf("node %s not found in conf", nodename)
				return clusterconf.NodeInfo{}
			}
			return clusterconf.NodeInfo{
				Name: nodename,
				Ip:   info.Ip,
			}
		},
	).([]clusterconf.NodeInfo)
	if notfounderr != nil {
		return nil, notfounderr
	}

	return nodeinfos, nil
}

// return worker nodes
func (d DistributeDeployerK3s) CheckWorker(conf clusterconf.ClusterConfYmlModel) ([]clusterconf.NodeInfo, error) {
	getAllNodes := func() ([]clusterconf.NodeInfo, error) {
		// k3s kubectl get nodes --selector='node-role.kubernetes.io/master'
		hosts := funk.Map(
			conf.Nodes,
			func(nodename string, info clusterconf.ClusterConfYmlModelNode) string {
				return fmt.Sprintf("%s@%s", conf.Global.SshUser, info.Ip)
			},
		).([]string)

		remoteResults, _ := util.StartRemoteCmds(
			hosts,
			"k3s kubectl get nodes",
			"",
		)

		// find one with result like
		// NAME        STATUS   ROLES                  AGE    VERSION
		// {nodename}           control-plane,master
		find_ := funk.Find(remoteResults, func(result string) bool {
			return strings.Contains(result, "Ready")
		})
		if find_ == nil {
			return []clusterconf.NodeInfo{}, nil
		}

		find := find_.(string)
		nodenames := funk.Map(
			funk.Filter(
				strings.Split(find, "\n"),
				func(line string) bool {
					return !strings.Contains(line, "NotReady") && strings.Contains(line, "Ready")
				},
			),
			func(line string) string {
				return strings.Split(line, " ")[0]
			},
		)

		var notfounderr error
		nodeinfos := funk.Map(
			nodenames,
			func(nodename string) clusterconf.NodeInfo {
				info, ok := conf.Nodes[nodename]
				if !ok {
					notfounderr = fmt.Errorf("node %s not found in conf", nodename)
					return clusterconf.NodeInfo{}
				}
				return clusterconf.NodeInfo{
					Name: nodename,
					Ip:   info.Ip,
				}
			},
		).([]clusterconf.NodeInfo)
		if notfounderr != nil {
			return nil, notfounderr
		}

		return nodeinfos, nil
	}
	allNodes, err := getAllNodes()
	if err != nil {
		return nil, err
	}
	util.Logger.Debugf("all nodes of k3s %v", allNodes)
	return funk.Filter(
		allNodes,
		func(nodeinfo clusterconf.NodeInfo) bool {
			find_ := funk.Find(funk.Map(conf.Nodes, func(nodename string, info clusterconf.ClusterConfYmlModelNode) tuple.T2[string, clusterconf.ClusterConfYmlModelNode] {
				return tuple.New2(nodename, info)
			}), func(nodenameInfo tuple.T2[string, clusterconf.ClusterConfYmlModelNode]) bool {
				return nodeinfo.Name == nodenameInfo.V1
			})
			if find_ == nil {
				return true
			}
			find := find_.(tuple.T2[string, clusterconf.ClusterConfYmlModelNode])
			return funk.Contains(find.V2.Tags, func(tag string) bool {
				return tag == d.Name()+"_worker"
			})
		},
	).([]clusterconf.NodeInfo), nil
}

func (d DistributeDeployerK3s) ThisNodeGeneralInstall() error {
	util.PrintStep("ThisNodeGeneralInstall", "installing k3s binary...")
	curUser, err := user.Current()
	if err != nil {
		return err
	}

	err = NewBinManager(BinManagerK3s{}).MakeSureWith()
	if err != nil {
		return err
	}

	_, err = util.ModRunCmd.NewBuilder("cp", "/usr/bin/k3s", "/usr/local/bin/k3s").WithRoot().BlockRun()
	if err != nil {
		return err
	}

	// chmod 555
	_, err = util.ModRunCmd.NewBuilder("chmod", "555", "/usr/local/bin/k3s").WithRoot().BlockRun()
	if err != nil {
		return err
	}

	util.PrintStep("ThisNodeGeneralInstall", "preparing binded resources...")
	download := func(fileName string, targetDir string) {
		fp := filepath.Join(targetDir, fileName)
		if _, err := os.Stat(fp); err != nil {
			util.DownloadFile(fmt.Sprintf("http://%s:8003/bin_k3s/"+fileName, util.MainNodeIp), fp)
		} else {
			fmt.Println(color.GreenString("File %s already exists, skip downloading", fp))
		}
	}
	download("install.sh", "/tmp/k3s")
	download(
		fmt.Sprintf("k3s-airgap-images-%s.tar.gz", util.GetCurrentArch()),
		"/tmp/k3s",
	)

	util.PrintStep("ThisNodeGeneralInstall", "preparing images...")

	_, err = util.ModRunCmd.NewBuilder("mkdir", "-p", "/var/lib/rancher/k3s/agent/images/").ShowProgress().BlockRun()
	if err != nil {
		return err
	}
	_, err = util.ModRunCmd.NewBuilder("chown", "-R", curUser.Name, "/var/lib/rancher/k3s/agent/images/").ShowProgress().BlockRun()
	if err != nil {
		return err
	}
	_, err = util.ModRunCmd.NewBuilder("cp", fmt.Sprintf("/tmp/k3s/k3s-airgap-images-%s.tar.gz", util.GetCurrentArch()), "/var/lib/rancher/k3s/agent/images/").ShowProgress().BlockRun()
	if err != nil {
		return err
	}

	return nil
}

type WorkerSetupCtx struct {
	GeneralSetupCtx

	Token  string `json:"token"`
	Server string `json:"server"` // format like https://192.168.1.1:6443

}

type MasterSetupCtx struct {
	GeneralSetupCtx
}

type GeneralSetupCtx struct {
	Registry *util.ContainerRegistryConf `json:"registry,omitempty"`
}

func (d DistributeDeployerK3s) PrepareWorkerSetupCtxBase64(masters []clusterconf.NodeInfo, conf clusterconf.ClusterConfYmlModel) (string, error) {
	util.PrintStep("PrepareWorkerSetupCtxBase64", "getting token from master node")
	masterHost := fmt.Sprintf("%s@%s", conf.Global.SshUser, masters[0].Ip)
	res, _ := util.StartRemoteCmds(
		[]string{masterHost},
		"cat /var/lib/rancher/k3s/server/token",
		"",
	)
	if res[0] == "" {
		return "", fmt.Errorf("get k3s token failed")
	}

	util.Logger.Debugf("token from master %s: %s", masters[0].Ip, res[0])
	jsonObj := WorkerSetupCtx{
		GeneralSetupCtx: GeneralSetupCtx{
			Registry: conf.Global.Registry,
		},
		Token: strings.ReplaceAll(res[0], "\n", ""),
		// k3s require https address
		Server: fmt.Sprintf("https://%s:6443", masters[0].Ip),
	}

	jsonBytes, err := json.Marshal(jsonObj)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(jsonBytes), nil
}

func (d DistributeDeployerK3s) PrepareMasterSetupCtxBase64(masters []clusterconf.NodeInfo, conf clusterconf.ClusterConfYmlModel) (string, error) {
	ctx := MasterSetupCtx{
		GeneralSetupCtx: GeneralSetupCtx{
			Registry: conf.Global.Registry,
		},
	}
	// encode
	jsonBytes, err := json.Marshal(ctx)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(jsonBytes), nil
}

func (d DistributeDeployerK3s) Registry(registryConf util.ContainerRegistryConf) error {
	util.PrintStep("Registry", "updating registry...")
	regiPath := "/etc/rancher/k3s/registries.yaml"
	registryHost := util.ImgRepoAddressNoPrefix()
	configHost := &map[string]interface{}{
		"auth": map[string]interface{}{
			"username": registryConf.User,
			"password": registryConf.Password,
		},
	}
	yamlObj := map[string]interface{}{
		"mirrors": map[string]interface{}{
			registryHost: map[string]interface{}{
				"endpoint": []string{util.ImgRepoAddressWithPrefix},
			},
		},
		"configs": map[string]interface{}{
			registryHost: configHost,
		},
	}

	if strings.HasPrefix(util.ImgRepoAddressWithPrefix, "http://") {
		(*configHost)["tls"] = map[string]interface{}{
			"insecure_skip_verify": true,
		}
	}

	model, err := yaml.Marshal(yamlObj)
	if err != nil {
		return err
	}

	err = os.RemoveAll(regiPath)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(regiPath), 0755)
	if err != nil {
		return err
	}

	// temporary skip the tls mode
	file, err := os.OpenFile(regiPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(model)
	return err
}

func (d DistributeDeployerK3s) ThisNodeWorkerInstall(ctxb64 string) error {
	util.PrintStep("ThisNodeWorkerInstall", "start install worker...")
	ctx := WorkerSetupCtx{}
	ctxJsonBytes, err := base64.StdEncoding.DecodeString(ctxb64)
	if err != nil {
		return err
	}
	err = json.Unmarshal(ctxJsonBytes, &ctx)
	if err != nil {
		return err
	}

	if ctx.Registry != nil {
		err := d.Registry(*ctx.Registry)
		if err != nil {
			return err
		}
	} else {
		fmt.Println("no registry config")
	}

	_, err = util.ModRunCmd.
		NewBuilder("bash", "./install.sh").
		WithRoot().
		SetEnv(
			"INSTALL_K3S_SKIP_DOWNLOAD=true",
			fmt.Sprintf("K3S_URL=%s", ctx.Server),
			fmt.Sprintf("K3S_TOKEN=%s", ctx.Token)).
		SetDir("/tmp/k3s/").
		ShowProgress().
		BlockRun()

	if err != nil {
		return err
	}

	_, err = util.ModRunCmd.
		NewBuilder("systemctl", "restart", "k3s-agent").
		WithRoot().
		ShowProgress().
		BlockRun()

	if err != nil {
		return err
	}

	return nil
}

func (d DistributeDeployerK3s) ThisNodeMasterInstall(ctxb64 string) error {
	util.PrintStep("ThisNodeMasterInstall", "start install master...")
	ctx := MasterSetupCtx{}
	jsonbytes, err := base64.StdEncoding.DecodeString(ctxb64)
	if err != nil {
		return err
	}
	err = json.Unmarshal(jsonbytes, &ctx)
	if err != nil {
		return err
	}

	if ctx.Registry != nil {
		err = d.Registry(*ctx.Registry)
		if err != nil {
			return err
		}
	} else {
		fmt.Println("no registry config")
	}

	_, err = util.ModRunCmd.
		NewBuilder("bash", "./install.sh").
		WithRoot().
		SetEnv("INSTALL_K3S_SKIP_DOWNLOAD=true").
		SetDir("/tmp/k3s/").
		ShowProgress().
		BlockRun()
	if err != nil {
		return err
	}

	_, err = util.ModRunCmd.
		NewBuilder("systemctl", "restart", "k3s").
		WithRoot().
		ShowProgress().
		BlockRun()

	if err != nil {
		return err
	}

	return nil
}

func (d DistributeDeployerK3s) Name() string {
	return "k3s"
}
