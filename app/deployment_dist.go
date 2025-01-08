package app

import (
	"fmt"
	"telego/util"
)

// https://qcnoe3hd7k5c.feishu.cn/wiki/Y9SkwEPmqiTov1knR8KctyJ0nJf#share-DtWBdCrDfoz9UGxyqazc7i9PnbJ
type DeploymentDistConfYaml struct {
	Type         string                       `yaml:"type"`
	Conf         map[string]map[string]string `yaml:"conf"`
	Distribution map[string][]string          `yaml:"distribution"`
	StateBackup  string                       `yaml:"state_backup"`
	Install      string                       `yaml:"install"`
	StateRestore string                       `yaml:"state_restore"`
	EntryPoint   string                       `yaml:"entrypoint"`
	NodeIps      map[string]string            `yaml:"node_ips,omitempty"`
}

type DeploymentDistConf struct {
	Type         DeploymentDistConfType       // type->typename
	Conf         map[string]map[string]string // conf->servce1->confkey->confvalue
	Distribution map[string][]string          // distribution->nodename->[servce1,service2...]
	StateBackup  string
	StateRestore string
	EntryPoint   string
	Install      string
	NodeIps      map[string]string
}

type DeploymentDistConfConvArg struct {
	ClusterName string
}

var _ util.Conv[util.Empty, DeploymentDistConfYaml] = DeploymentDistConf{}

func (d DeploymentDistConf) To(util.Empty) (DeploymentDistConfYaml, error) {
	return DeploymentDistConfYaml{
		Conf:         d.Conf,
		Distribution: d.Distribution,
		EntryPoint:   d.EntryPoint,
		NodeIps:      d.NodeIps,
		StateBackup:  d.StateBackup,
		StateRestore: d.StateRestore,
		Install:      d.Install,
		Type:         d.Type.DeploymentDistConfName(),
	}, nil
}

var _ util.Conv[DeploymentDistConfConvArg, DeploymentDistConf] = DeploymentDistConfYaml{}

func (d DeploymentDistConfYaml) To(convarg DeploymentDistConfConvArg) (DeploymentDistConf, error) {
	// check type
	ty := NewDeploymentDistConfType(d.Type)
	if ty == nil {
		return DeploymentDistConf{}, fmt.Errorf("unknown type: %s", d.Type)
	}

	// check distribution (node->[service,...] )

	serviceSet, nodes2ip, err := func() (map[string]bool, map[string]string, error) {
		// 1. no repeated service
		// 2. no repeated node
		// 3. nodes are all valid
		serviceSet := map[string]bool{}
		nodeSet := map[string]bool{}
		for node, services := range d.Distribution {
			if _, ok := nodeSet[node]; ok {
				return nil, nil, fmt.Errorf("node %s repeated in distribution", node)
			}
			nodeSet[node] = true
			for _, service := range services {
				if _, ok := serviceSet[service]; ok {
					return nil, nil, fmt.Errorf("service %s repeated in distribution", service)
				}
				serviceSet[service] = true
			}
		}
		nodes2ip, err := util.KubeNodeName2Ip(convarg.ClusterName)
		if err != nil {
			return nil, nil, fmt.Errorf("get kube client error: %v", err)
		}
		// distribution nodes must be cluster nodes
		for node, _ := range d.Distribution {
			if _, ok := nodes2ip[node]; !ok {
				return nil, nil, fmt.Errorf("node %s not found in kube cluster", node)
			}
		}
		// remove nodes not in distribution
		{
			deletedNodes := []string{}
			for node, _ := range nodes2ip {
				if _, ok := d.Distribution[node]; !ok {
					deletedNodes = append(deletedNodes, node)
				}
			}
			for _, node := range deletedNodes {
				delete(nodes2ip, node)
			}
		}

		return serviceSet, nodes2ip, nil
	}()
	if err != nil {
		return DeploymentDistConf{}, err
	}

	// check conf
	for service, _ := range d.Conf {
		if service == "global" {
			continue
		}
		if _, ok := serviceSet[service]; !ok {
			return DeploymentDistConf{}, fmt.Errorf("service %s in conf not found in distribution", service)
		}
	}

	return DeploymentDistConf{
		Type:         ty,
		Conf:         d.Conf,
		Distribution: d.Distribution,
		StateBackup:  d.StateBackup,
		StateRestore: d.StateRestore,
		EntryPoint:   d.EntryPoint,
		NodeIps:      nodes2ip,
		Install:      d.Install,
	}, nil
}

type DeploymentDistConfType interface {
	DeploymentDistConfName() string
}

type DeploymentDistConfTypeRawMetal struct{}

func (DeploymentDistConfTypeRawMetal) DeploymentDistConfName() string {
	return "raw_metal"
}

func NewDeploymentDistConfType(name string) DeploymentDistConfType {
	switch name {
	case DeploymentDistConfTypeRawMetal{}.DeploymentDistConfName():
		return DeploymentDistConfTypeRawMetal{}
	default:
		return nil
	}
}
