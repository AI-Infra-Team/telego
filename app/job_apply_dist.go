package app

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"telego/util"
	"text/template"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/thoas/go-funk"
	"gopkg.in/yaml.v3"
)

type ApplyDistJob struct {
	ClusterContext string
}

type ModJobApplyDistStruct struct{}

var ModJobApplyDist ModJobApplyDistStruct

func (ModJobApplyDistStruct) JobCmdName() string {
	return "apply-dist"
}

func (ModJobApplyDistStruct) ParseJob(ApplyDistCmd *cobra.Command) *cobra.Command {
	prjname := ""
	kubecontext := ""

	ApplyDistCmd.Flags().StringVar(&prjname, "project", "", "Sub project dir in user specified workspace")
	ApplyDistCmd.Flags().StringVar(&kubecontext, "kube-context", "", "k8s cluster context name")

	ApplyDistCmd.Run = func(cmd *cobra.Command, _ []string) {
		ModJobApplyDist.ApplyDistLocal(prjname, kubecontext)
	}

	return ApplyDistCmd
}

func (m ModJobApplyDistStruct) NewApplyDistCmd(distprj string, kubecontext string) []string {

	return []string{"telego", m.JobCmdName(), "--project", distprj, "--kube-context", kubecontext}
}

func (m ModJobApplyDistStruct) DistConfigMapName(distprj string) string {
	return strings.ReplaceAll(distprj, "_", "-")
}

func (m ModJobApplyDistStruct) ApplyDistLocal(prjname string, kubecontext string) {
	prjdir := ConfigLoad().ProjectDir
	distprjdir := path.Join(prjdir, prjname)

	if _, err := os.Stat(distprjdir); err != nil {
		fmt.Println(color.RedString("Error: project %s not found in prjdir %s", prjname, prjdir))
		os.Exit(1)
	}

	if !strings.HasPrefix(prjname, "dist_") {
		fmt.Println(color.RedString("Error: project %s is not a dist project", prjname))
		os.Exit(1)
	}

	util.PrintStep("ApplyDistLocal", "load raw project deployment.yml at "+distprjdir)
	tempYamlDir := func() string { // load raw project deployment.yml
		d, err := LoadDeploymentYml(prjname, distprjdir)
		if err != nil {
			fmt.Println(color.RedString("Error: %s", err))
		}

		util.PrintStep("ApplyDistLocal", "verify deployment.yml at "+distprjdir)
		for distname, distyml := range d.Dist {
			// use To interface to do verify
			dist, err := distyml.To(DeploymentDistConfConvArg{
				ClusterName: kubecontext,
			})
			if err != nil {
				fmt.Println(color.RedString("distyml -> dist error: %s", err))
				os.Exit(1)
			}
			distyml, err = dist.To(util.Empty{})
			if err != nil {
				fmt.Println(color.RedString("dist -> distyml error: %s", err))
				os.Exit(1)
			}
			// write back
			d.Dist[distname] = distyml
		}

		// temp yaml dir
		// The generated Kubernetes manifests will be stored in this directory
		// Final project structure will look like:
		// {tempYamlDir}/
		//   ├── configmap.yaml          // Stores distribution configurations
		//   └── daemonset-{distname}.yaml   // DaemonSet for each node group
		tempYamlDir := path.Join(util.WorkspaceDir(), prjname)
		// kubectl delete existing daemonsets
		_, err = util.ModRunCmd.NewBuilder("kubectl", "delete", "-f", tempYamlDir, "--context", kubecontext, "--namespace", "tele-deployment").ShowProgress().BlockRun()
		if err != nil {
			fmt.Println(color.YellowString("Error: %s", err))
		}
		os.RemoveAll(tempYamlDir)
		os.MkdirAll(tempYamlDir, 0755)
		// generate configmap.yaml
		util.PrintStep("ApplyDistLocal", "generate configmap.yaml at "+tempYamlDir)
		{
			const configMapTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.Name}}
  {{- if .Namespace}}
  namespace: {{.Namespace}}
  {{- end}}
data:
  {{- range .Configs}}
  {{.Key}}: |
    {{.Value}}
  {{- end}}
`

			type ConfigEntry struct {
				Key   string
				Value string
			}
			type ConfigMapData struct {
				Name      string
				Namespace string
				Configs   []ConfigEntry
			}

			// 构造 ConfigMapData
			configMap := ConfigMapData{
				Name: m.DistConfigMapName(prjname),
				Configs: funk.Map(d.Dist, func(distname string, dist DeploymentDistConfYaml) ConfigEntry {
					// 序列化 dist 到字符串，这取决于 DeploymentDistConfYaml 的具体结构
					// 这里只是简单地将 dist 转换为字符串表示
					// value := fmt.Sprintf("%+v", dist)
					fmt.Printf("dist %v\n", dist)
					distyaml, err := yaml.Marshal(dist)
					fmt.Printf("distyaml %s\n", string(distyaml))
					if err != nil {
						fmt.Println(color.RedString("Error: %s", err))
						os.Exit(1)
					}
					return ConfigEntry{
						Key:   distname,
						Value: strings.ReplaceAll(string(distyaml), "\n", "\n    "),
					}
				}).([]ConfigEntry),
			}

			// 渲染模板
			var result bytes.Buffer
			{
				tmpl, err := template.New("configMapTemplate").Parse(configMapTemplate)
				if err != nil {
					fmt.Println(color.RedString("Error: %s", err))
					os.Exit(1)
				}

				err = tmpl.Execute(&result, configMap)
				if err != nil {
					fmt.Println(color.RedString("Error: %s", err))
					os.Exit(1)
				}
			}

			// 将生成的 ConfigMap 写入文件
			err = os.WriteFile(path.Join(tempYamlDir, "configmap.yaml"), result.Bytes(), 0644)
			if err != nil {
				fmt.Println(color.RedString("Error creating configmap.yaml: %s", err))
				os.Exit(1)
			}
		}

		for dname, d := range d.Dist {
			util.PrintStep("ApplyDistLocal", fmt.Sprintf("generate %s daemonset.yaml at %s", dname, tempYamlDir))
			err := m.gendaemonset(prjname, dname, d, tempYamlDir)
			if err != nil {
				fmt.Println(color.RedString("Error: %s", err))
				os.Exit(1)
			}
		}
		fmt.Println(color.GreenString("Success: DaemonSets generated successfully"))
		return tempYamlDir
	}()

	util.PrintStep("ApplyDistLocal", "apply dist")

	_, err := util.ModRunCmd.NewBuilder("kubectl", "apply", "-f", tempYamlDir, "--context", kubecontext, "--namespace", "tele-deployment").ShowProgress().BlockRun()
	if err != nil {
		fmt.Println(color.RedString("Error: %s", err))
		os.Exit(1)
	}
	fmt.Println(color.GreenString("Success: DaemonSets applied successfully"))
}

// generate each dist daemonset
// gendaemonset generates a DaemonSet for a given distribution
// gendaemonset generates a DaemonSet YAML using a template
func (m ModJobApplyDistStruct) gendaemonset(
	prjName string, dname string, d DeploymentDistConfYaml, tempYamlDir string) error {
	daemonSetYamlPath := filepath.Join(tempYamlDir, fmt.Sprintf("daemonset-%s.yaml", dname))

	// 聚合所有 unique_id 生成 UniqueIDList
	uniqueIDList := funk.Flatten(funk.Values(d.Distribution)).([]string)

	// 模板数据
	data := map[string]interface{}{
		"DaemonSetName": strings.ReplaceAll(prjName+"-"+dname, "_", "-"),
		"ConfigMapName": m.DistConfigMapName(prjName),
		"ProjectName":   prjName,
		"DistName":      dname,
		"UniqueIDList":  strings.Join(uniqueIDList, ","),
		"NodeNames":     strings.Join(funk.Keys(d.NodeIps).([]string), ","),
		"InstanceIndices": func() []int {
			maxint := funk.MaxInt(
				// each array length
				funk.Map(funk.Values(d.Distribution), func(instances []string) int {
					return len(instances)
				}).([]int),
			)
			// a increacing array
			arr := make([]int, maxint)
			for i := 0; i < maxint; i++ {
				arr[i] = i
			}
			fmt.Printf("InstanceIndices %v\n", arr)
			return arr
		}(),
		"Distribution": d.Distribution,
		"InitImage":    path.Join(util.ImgRepoAddressNoPrefix, "teleinfra/python:3.12.5"),
		"RuntimeImage": path.Join(util.ImgRepoAddressNoPrefix, "teleinfra/openssh:9.1"),
		"MainNodeIP":   util.MainNodeIp,
		"InstallCurlCmd": strings.ReplaceAll(
			util.ModContainerCmd{}.WithHostCurl("/host-usr"), "\n", "\n          "),
		"SshUser": util.MainNodeUser,
	}

	// YAML 模板
	// Rememeber to use 2 spaces for indentation rather than tab
	daemonSetTemplate := `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: {{ .DaemonSetName }}-daemonset
spec:
  selector:
    matchLabels:
      app: {{ .DaemonSetName }}-daemon
  template:
    metadata:
      labels:
        app: {{ .DaemonSetName }}-daemon
    spec:
      securityContext: # for /dev/tty permission
        runAsUser: 0
        runAsGroup: 0
        fsGroup: 0
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: kubernetes.io/hostname
                operator: In
                values: [{{ .NodeNames }}]
      volumes:
      - name: teledeploy-secret
        hostPath:
          path: /teledeploy_secret/{{ .ProjectName }}
          type: DirectoryOrCreate
      - name: dist-config
        configMap:
          name: {{ .ConfigMapName }}
      - name: host-usr
        hostPath:
          path: /usr
          type: Directory
      - name: host-bin
        hostPath:
          path: /bin
          type: Directory
      - name: workdir
        emptyDir: {}
      initContainers:
      - name: install-telego
        image: {{ $.InitImage }}
        command: ["sh", "-c"]
        args:
        - |
          MAIN_NODE_IP={{ $.MainNodeIP }}
          python -c "import urllib.request, os; script = urllib.request.urlopen('http://${MAIN_NODE_IP}:8003/bin_telego/install.py').read(); exec(script.decode());"
          cp /usr/bin/telego /workdir/
        volumeMounts:
        - name: workdir
          mountPath: /workdir
      containers:
      {{- range $idx := .InstanceIndices }}
      - name: {{ $.DaemonSetName }}-container-{{$idx}}
        image: {{ $.RuntimeImage}}
        volumeMounts:
        - name: teledeploy-secret
          mountPath: /teledeploy_secret/{{ $.ProjectName }}
        - name: dist-config
          mountPath: /etc/dist-config
        - name: workdir
          mountPath: /workdir
        workingDir: /teledeploy_secret/{{ $.ProjectName }}
        env:
        - name: SSH_PW # from secret
          valueFrom:
            secretKeyRef:
              name: tele-deploy-secret
              key: SSH_PW
        - name: DIST_NODE
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: HOST_IP
          valueFrom:
            fieldRef:
              fieldPath: status.hostIP
        - name: DIST_INSTANCE_IDX
          value: "{{ $idx }}"
        - name: DIST_UNIQUE_ID_LIST
          value: "{{ $.UniqueIDList }}"
        command: ["bash", "-c"]
        args:
        - |
          # Error handling function
          function handle_error() {
            local msg=$1
            echo "[ERROR] ${msg}"
            echo "Entering infinite sleep for debugging..."
            while true; do sleep 3600; done
          }

          distdir=/teledeploy_secret/{{ $.ProjectName }}/{{ $.DistName }}_{{ $idx }}
          mkdir -p "$distdir"
          cd "$distdir" || handle_error "Failed to change directory"

          # Copy telego binary from workdir
          echo ">>> install telego in container"
          cp /workdir/telego /usr/bin/
          chmod +x /usr/bin/telego

          NODE_NAME=$DIST_NODE
          if [ -z "$NODE_NAME" ]; then
            handle_error "DIST_NODE environment variable is not set"
          fi

          echo ">>> getting dist specific configs"
          # Get SSH secret and setup SSH key
          telego config-exporter --dist {{ $.ProjectName }}:{{ $.DistName }} \
                            --dist-node "$NODE_NAME" \
                            --dist-instance-idx "$DIST_INSTANCE_IDX" \
                            --dist-config /etc/dist-config/{{ $.DistName }} \
                            --secret ssh_private:as.SSH_PRIVATE \
                            --saveas ./export.sh || handle_error "Failed to run config-exporter"

          echo ">>> sourcing exported environment variables"
          # Source the exported environment variables
          if [ -f "./export.sh" ]; then
            source ./export.sh
          else
            handle_error "export.sh not found"
          fi

          echo ">>> setting up ssh key"
          # Setup SSH key
          mkdir -p ~/.ssh
          echo -e "$SSH_PRIVATE" > ~/.ssh/id_ed25519
          chmod 600 ~/.ssh/id_ed25519

          # Install telego on host
          echo ">>> installing telego on host"
          ssh -o StrictHostKeyChecking=no "{{ $.SshUser }}@$HOST_IP" \
            "python3 -c \"import urllib.request, os; script = urllib.request.urlopen('http://{{ $.MainNodeIP }}:8003/bin_telego/install.py').read(); exec(script.decode());\""

          # Continue only if this instance is needed on this node
          if [ -z "$DIST_UNIQUE_ID" ]; then
            echo "No instance assigned to this container. Sleeping..."
            sleep infinity
          fi

          echo "Starting setup for $DIST_UNIQUE_ID"
          if [ -z "$DIST_UNIQUE_ID" ] || [ -z "$SSH_PW" ]; then
            handle_error "Required environment variables are not set"
          fi
      
          # Inject environment variables into script
          function inject_env_vars() {
            local script=$1
            cp "$script" "$script.bak"
            {
              echo "export DIST_UNIQUE_ID=$DIST_UNIQUE_ID"
              echo "export MAIN_NODE_IP=$MAIN_NODE_IP"
              echo "export HOST_IP=$HOST_IP"
              echo "export DIST_INSTANCE_IDX=$DIST_INSTANCE_IDX"
              echo "export DIST_NODE=$DIST_NODE"
              echo "export DIST_UNIQUE_ID_LIST=$DIST_UNIQUE_ID_LIST"
              cat export.sh "$script.bak"
            } > "$script"
            rm -f "$script.bak"
          }

          echo ">>> injecting environment variables into host running scripts"
          inject_env_vars install.sh
          inject_env_vars entrypoint.sh

          echo ">>> running prepare scripts"
          for script in backup.sh install.sh restore.sh; do
            if [ -f "./$script" ]; then
              echo "Running $script..."
              case "$script" in
                install.sh)
                  ssh -o StrictHostKeyChecking=no "{{ $.SshUser }}@$HOST_IP" \
                    "bash $distdir/$script" || handle_error "Failed to execute $script"
                  ;;
                *)
                  bash "./$script" || handle_error "Failed to execute $script"
                  ;;
              esac
            else
              handle_error "Missing $script"
            fi
          done
      
          # Execute entrypoint.sh on host
          echo ">>> executing entrypoint.sh on host"
          ssh -o StrictHostKeyChecking=no "{{ $.SshUser }}@$HOST_IP" \
            "cd $distdir && bash ./entrypoint.sh" || handle_error "Failed to execute entrypoint.sh on host"
      {{- end }}
`

	// 渲染模板
	tmpl, err := template.New("daemonSetTemplate").Parse(daemonSetTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(daemonSetYamlPath)
	if err != nil {
		return fmt.Errorf("failed to create YAML file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
