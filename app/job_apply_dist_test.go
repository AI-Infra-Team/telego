package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"telego/util"
	"telego/util/yamlext"
	"testing"

	"github.com/fatih/color"
)

func TestApplyDist(t *testing.T) {
	testPrjDir := util.ModTestUtil.SetupForTest()
	// Mock the node to IP mapping function
	hookTrigger := false
	util.KubeNodeName2IpSetHook(func(cluster string) (map[string]string, error) {
		hookTrigger = true
		return map[string]string{
			"lab1": "192.168.1.1",
			"lab2": "192.168.1.2",
			"lab3": "192.168.1.3",
		}, nil
	})

	testOnePrjDir := filepath.Join(testPrjDir, "dist_test_prj")
	fmt.Println(color.GreenString("testOnePrjDir %s", testOnePrjDir))
	os.MkdirAll(testOnePrjDir, 0755)
	// Prepare the test project with deployment.yml
	deploymentf, err := os.Create(filepath.Join(testOnePrjDir, "deployment.yml"))
	if err != nil {
		t.Error(err)
	}
	// https://qcnoe3hd7k5c.feishu.cn/wiki/Y9SkwEPmqiTov1knR8KctyJ0nJf#share-WuNMdOqLJoFpV2x7PMxc7QqpnJc
	deploymentYaml := `
local_values:
  
dist:
  waverless-test:
    type: raw_metal
    conf:
      global: {port: 2500}
      1: {tag: "[meta, master]"}
      2: {tag: "[meta, worker]"}
      3: {tag: '[meta, worker]'}
    distribution:
      lab1: [1]
      lab2: [2]
      lab3: [3]
    state_backup: |
      mkdir -p backup
      mv apps backup
      mv files backup
      mv kv_store_engine* backup
    state_restore: |
      mv backup/* .
    entrypoint: |
      telego install --bin-prj waverless
      cat >> gen_nodes_config.py << EOF
      import os
      with open("nodes_config.yaml", "w") as f:
          f.write("nodes:\\n")
          for unique_id in DIST_UNIQUE_ID_LIST:
              ip = os.getenv(f"DIST_CONF_{unique_id}_NODE_IP")
              port = os.getenv(f"DIST_CONF_{unique_id}_port")
              spec = os.getenv(f"DIST_CONF_{unique_id}_spec")
              f.write(f"  {unique_id}:\\n")
              f.write(f"    addr: {ip}:{port}\\n")
              f.write(f"    spec: {spec}\\n")
      EOF
      python3 gen_nodes_config.py
      waverless_entry $DIST_UNIQUE_ID
`
	_, err = deploymentf.WriteString(deploymentYaml)
	if err != nil {
		t.Error(err)
	}
	deploymentf.Close()

	ModJobApplyDist.ApplyDistLocal("dist_test_prj", "whatever")

	if !hookTrigger {
		t.Error("hookTrigger should be true")
	}

	// read and parse {testOnePrjDir}/configmap.yaml
	readAndParseYaml := func(path string) (interface{}, error) {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		yamlContent, err := io.ReadAll(f)
		if err != nil {
			return nil, err
		}
		var res interface{}
		err = yamlext.UnmarshalAndValidate(yamlContent, &res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}
	readAndParseYaml(filepath.Join(testOnePrjDir, "configmap.yaml"))
	// read and parse {testOnePrjDir}/daemonset-waverless-test.yaml
	readAndParseYaml(filepath.Join(testOnePrjDir, "daemonset-waverless-test.yaml"))
}
