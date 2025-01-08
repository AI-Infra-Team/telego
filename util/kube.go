package util

import (
	"context"
	"errors"
	"fmt"
	"os"
	"telego/util/yamlext"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// host: https://xxx
// user:
// password:
// # tls: { cert_file, ca_file }
type ContainerRegistryConfTls struct {
	CertFile string `json:"cert_file" yaml:"cert_file"` // 客户端证书文件路径
	CAFile   string `json:"ca_file" yaml:"ca_file"`     // CA 证书文件路径
}

// 配置容器注册表的信息
type ContainerRegistryConf struct {
	// Host     string                    `json:"host" yaml:"host"`          // 注册表地址
	User                      string                    `json:"user" yaml:"user"`         // 用户名
	Password                  string                    `json:"password" yaml:"password"` // 密码
	UploaderStoreAddr         string                    `json:"uploader_store_addr" yaml:"uploader_store_addr"`
	UploaderStoreAdmin        string                    `json:"uploader_store_admin" yaml:"uploader_store_admin"`
	UploaderStoreAdminPw      string                    `json:"uploader_store_admin_pw" yaml:"uploader_store_admin_pw"`
	UploaderStoreTransferAddr string                    `json:"uploader_store_transfer_addr" yaml:"uploader_store_transfer_addr"`
	Tls                       *ContainerRegistryConfTls `json:"tls,omitempty" yaml:"tls,omitempty"` // TLS 配置（可选）
}

// 根据 Cluster 名称获取对应的 Context 名称
func GetKubeContextByCluster(clusterName string) string {
	// 获取默认的 kubeconfig 文件路径
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = clientcmd.RecommendedHomeFile // 默认为 ~/.kube/config
	}

	// 加载 kubeconfig 文件
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		// 加载失败返回空字符串
		fmt.Printf("加载 kubeconfig 文件失败: %v\n", err)
		return ""
	}

	// 遍历 Contexts，查找匹配的 Cluster
	for contextName, context := range config.Contexts {
		if context.Cluster == clusterName {
			return contextName
		}
	}

	// 未找到匹配的 Cluster
	return ""
}

func KubeList() []string {
	// 获取默认的 kubeconfig 文件路径
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = clientcmd.RecommendedHomeFile // 默认为 ~/.kube/config
	}

	res := []string{}
	// 加载 kubeconfig 文件
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		// log.Fatalf("加载 kubeconfig 文件失败: %v", err)
		return res
	}

	for _, context := range config.Contexts {
		res = append(res, context.Cluster)
	}
	return res

	// // 打印当前使用的 Context
	// fmt.Println("当前使用的 Context:", config.CurrentContext)
}

// 根据集群名称获取 Kubernetes 客户端句柄
func KubeClusterClient(clusterName string) (*kubernetes.Clientset, error) {
	// // 获取默认的 kubeconfig 文件路径
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = clientcmd.RecommendedHomeFile // 默认为 ~/.kube/config
	}

	// 使用 BuildConfigFromFlags 加载 kubeconfig 配置
	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("无法加载 kubeconfig 文件: %w", err)
	}

	// 设置集群的上下文（如果需要）
	// restConfig = clusterName // 设置当前集群上下文

	// 创建 Kubernetes 客户端
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("无法创建客户端: %w", err)
	}

	return clientset, nil
}

type KubeNodeName2IpHookType func(cluster string) (map[string]string, error)

var kubeNodeName2IpHook KubeNodeName2IpHookType

func KubeNodeName2IpSetHook(
	KubeNodeName2IpHook0 KubeNodeName2IpHookType,
) {
	kubeNodeName2IpHook = KubeNodeName2IpHook0
}

// KubeNodeName2Ip 获取集群中所有节点的名称和 IP 地址的映射
func KubeNodeName2Ip(cluster string) (map[string]string, error) {
	if kubeNodeName2IpHook != nil {
		return kubeNodeName2IpHook(cluster)
	}

	// 获取 Kubernetes 客户端
	clientset, err := KubeClusterClient(cluster)
	if err != nil {
		return nil, fmt.Errorf("无法获取 Kubernetes 客户端: %w", err)
	}

	// 获取集群中的所有节点
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("无法获取节点信息: %w", err)
	}

	// 创建一个 map 映射节点名称到 IP 地址
	nodeIpMap := make(map[string]string)

	// 遍历节点，提取节点名称和 IP 地址
	for _, node := range nodes.Items {
		for _, address := range node.Status.Addresses {
			if address.Type == "InternalIP" {
				nodeIpMap[node.Name] = address.Address
				break
			}
		}
	}

	return nodeIpMap, nil
}

// namespace: tele-deployment
//
// secret: tele-deploy-secret
//
// return content decoded
func KubeSecret(cluster string, key string) (string, error) {
	// Get Kubernetes client for the cluster
	clientset, err := KubeClusterClient(cluster)
	if err != nil {
		return "", fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// Namespace and secret name
	namespace := "tele-deployment"
	secretName := "tele-deploy-secret"

	// Get the secret
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, v1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s/%s: %w", namespace, secretName, err)
	}

	// Check if the key exists in the secret data
	encodedValue, exists := secret.Data[key]
	if !exists {
		return "", errors.New("key not found in secret")
	}

	// Decode the value
	// decodedValue, err := base64.StdEncoding.DecodeString(string(encodedValue))
	// if err != nil {
	// 	return "", fmt.Errorf("failed to decode value for key %s, err: %w, content: %s", key, err, string(encodedValue))
	// }

	return string(encodedValue), nil
}

// namespace: tele-deployment
//
// secret: tele-deploy-secret
//
// key: nodes_conf
//
// yaml value: sshUser
//
// KubeSecretSshUser retrieves the sshUser from the nodes_conf in the Kubernetes secret.
func KubeSecretSshUser(cluster string) (string, error) {
	// Get the nodes_conf data from the secret
	nodesConfYml, err := KubeSecret(cluster, "nodes_conf")
	if err != nil {
		return "", fmt.Errorf("failed to get nodes_conf from secret: %w", err)
	}

	// Parse the YAML content
	var config map[string]interface{}
	err = yamlext.UnmarshalAndValidate([]byte(nodesConfYml), &config)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	// Extract the sshUser field from the parsed YAML data
	sshUser, ok := config["sshUser"].(string)
	if !ok {
		return "", errors.New("sshUser not found in the nodes_conf YAML")
	}

	return sshUser, nil
}

// namespace: tele-deployent
//
// secret: tele-deploy-secret
//
// key: nodes_conf
//
// yaml value: sshSecret
func KubeSecretSshSecret(cluster string) (string, error) {
	panic("not impl")
	// return "", nil
}

func KubeGetMasterIP(cluster string) (string, error) {
	clientset, err := KubeClusterClient(cluster)
	if err != nil {
		return "", fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list nodes: %w", err)
	}

	for _, node := range nodes.Items {
		if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok || node.Labels["node-role.kubernetes.io/master"] == "true" {
			for _, address := range node.Status.Addresses {
				if address.Type == corev1.NodeInternalIP {
					return address.Address, nil
				}
			}
		}
	}

	return "", fmt.Errorf("master node not found in cluster")
}
