package app

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"telego/util"

	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"k8s.io/client-go/util/homedir"
)

// - main node FileServerAccessible
// - main node fileserver rclone and kubectl project uploaded
// - k8s集群
//   - kubeconfig secret 配置
//   - 主k8s集群
//     - k8s secret 是否配置全
// - 镜像上传服务
//   - 对应secret配置是否已配置
//   - 镜像仓库
//   - 镜像上传server(需要加一个 health接口，返回其子项的检查)
//     - 对应public 配置是否已配置

type UiBackendJob struct {
	Port string
}

type ModJobUiBackendStruct struct{}

var ModJobUiBackend ModJobUiBackendStruct

func (m ModJobUiBackendStruct) JobCmdName() string {
	return "ui-backend"
}

func (_ ModJobUiBackendStruct) ParseJob(uiBackendCmd *cobra.Command) *cobra.Command {
	job := &UiBackendJob{}

	uiBackendCmd.Flags().StringVar(&job.Port, "port", "8080", "Port to run UI backend server")

	uiBackendCmd.Run = func(_ *cobra.Command, _ []string) {
		fmt.Println(color.BlueString("UI Backend job running on port %s", job.Port))
		ModJobUiBackend.StartServer(*job)
	}

	return uiBackendCmd
}

// InitializationStep 表示初始化流程中的一个步骤
type InitializationStep struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Status      string               `json:"status"` // "pending", "running", "completed", "error"
	Error       string               `json:"error,omitempty"`
	Progress    int                  `json:"progress"` // 0-100
	StartTime   *time.Time           `json:"startTime,omitempty"`
	EndTime     *time.Time           `json:"endTime,omitempty"`
	Children    []InitializationStep `json:"children,omitempty"` // 子步骤
}

// InitializationStatus 表示整个初始化流程的状态
type InitializationStatus struct {
	Steps           []InitializationStep `json:"steps"`
	OverallStatus   string               `json:"overallStatus"`   // "pending", "running", "completed", "error"
	OverallProgress int                  `json:"overallProgress"` // 0-100
	StartTime       *time.Time           `json:"startTime,omitempty"`
	EndTime         *time.Time           `json:"endTime,omitempty"`
}

func (_ ModJobUiBackendStruct) StartServer(job UiBackendJob) {
	// 设置gin为发布模式
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	// 启用CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// API路由
	api := r.Group("/api")
	{
		api.GET("/initialization/status", ModJobUiBackend.getInitializationStatus)
		api.POST("/initialization/start", ModJobUiBackend.startInitialization)
		api.POST("/initialization/retry/:stepId", ModJobUiBackend.retryStep)
	}

	// 静态文件服务 (用于Vue前端)
	r.Static("/static", "./ui/dist")
	r.StaticFile("/", "./ui/dist/index.html")

	fmt.Println(color.GreenString("UI Backend server starting on port %s", job.Port))
	fmt.Println(color.BlueString("Web UI available at: http://localhost:%s", job.Port))

	err := r.Run(":" + job.Port)
	if err != nil {
		fmt.Println(color.RedString("Failed to start UI Backend server: %v", err))
		os.Exit(1)
	}
}

func (_ ModJobUiBackendStruct) getInitializationStatus(c *gin.Context) {
	status := ModJobUiBackend.checkInitializationStatus()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}

func (_ ModJobUiBackendStruct) startInitialization(c *gin.Context) {
	// 启动初始化流程
	go ModJobUiBackend.runInitializationProcess()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Initialization process started",
	})
}

func (_ ModJobUiBackendStruct) retryStep(c *gin.Context) {
	stepId := c.Param("stepId")

	// 重试特定步骤
	go ModJobUiBackend.retryInitializationStep(stepId)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Retrying step: %s", stepId),
	})
}

func (_ ModJobUiBackendStruct) checkInitializationStatus() InitializationStatus {
	// 主节点检查
	mainNodeStep := InitializationStep{
		ID:          "main_node",
		Name:        "主节点检查",
		Description: "检查主节点相关配置和服务",
		Status:      ModJobUiBackend.getGroupStatus([]string{"check_fileserver", "check_fileserver_tools"}),
		Progress:    ModJobUiBackend.getGroupProgress([]string{"check_fileserver", "check_fileserver_tools"}),
		Children: []InitializationStep{
			func() InitializationStep {
				status, errorMsg := processCheckResult(ModJobUiBackend.checkFileserverStatus)
				return InitializationStep{
					ID:          "check_fileserver",
					Name:        "文件服务器可访问性检查",
					Description: "检查主节点文件服务器是否可访问, 配置文档：https://qcnoe3hd7k5c.feishu.cn/wiki/PKetwap1EiBiylkea1mc4i8Tn4g#share-FOcgdIzpho6BGXxg6rBcEGVGnRP",
					Status:      status,
					Error:       errorMsg,
					Progress:    ModJobUiBackend.getStepProgress("check_fileserver"),
				}
			}(),
			func() InitializationStep {
				status, errorMsg := processCheckResult(ModJobUiBackend.checkFileserverToolsStatus)
				return InitializationStep{
					ID:          "check_fileserver_tools",
					Name:        "文件服务器工具检查",
					Description: "检查主节点文件服务器上 rclone 和 kubectl 项目是否已上传，配置文档：https://qcnoe3hd7k5c.feishu.cn/wiki/CS9XwVa5ViQoqTkuyDXcrL0Dnkd",
					Status:      status,
					Error:       errorMsg,
					Progress:    ModJobUiBackend.getStepProgress("check_fileserver_tools"),
				}
			}(),
		},
	}

	// K8s 集群检查
	k8sClusterStep := InitializationStep{
		ID:          "k8s_cluster",
		Name:        "K8s 集群",
		Description: "检查 Kubernetes 集群相关配置",
		Status:      ModJobUiBackend.getGroupStatus([]string{"check_kubeconfig", "k8s_main_cluster"}),
		Progress:    ModJobUiBackend.getGroupProgress([]string{"check_kubeconfig", "k8s_main_cluster"}),
		Children: []InitializationStep{
			{
				ID:          "check_kubeconfig",
				Name:        "Kubeconfig Secret 配置",
				Description: "检查 kubeconfig secret 配置是否正确",
				Status:      ModJobUiBackend.checkKubeconfigStatus(),
				Progress:    ModJobUiBackend.getStepProgress("check_kubeconfig"),
			},
			{
				ID:          "k8s_main_cluster",
				Name:        "主 K8s 集群",
				Description: "检查主 K8s 集群连接和配置",
				Status:      ModJobUiBackend.getGroupStatus([]string{"check_k8s_cluster"}),
				Progress:    ModJobUiBackend.getGroupProgress([]string{"check_k8s_cluster"}),
				Children: []InitializationStep{
					{
						ID:          "check_k8s_cluster",
						Name:        "K8s Secret 配置检查",
						Description: "检查 K8s secret 是否配置完整",
						Status:      ModJobUiBackend.checkK8sClusterStatus(),
						Progress:    ModJobUiBackend.getStepProgress("check_k8s_cluster"),
					},
				},
			},
		},
	}

	// 镜像上传服务检查
	imageServiceStep := InitializationStep{
		ID:          "image_service",
		Name:        "镜像上传服务",
		Description: "检查镜像上传相关服务和配置",
		Status:      ModJobUiBackend.getGroupStatus([]string{"check_image_secret", "check_image_registry", "image_upload_server"}),
		Progress:    ModJobUiBackend.getGroupProgress([]string{"check_image_secret", "check_image_registry", "image_upload_server"}),
		Children: []InitializationStep{
			{
				ID:          "check_image_secret",
				Name:        "镜像服务 Secret 配置",
				Description: "检查镜像服务对应的 secret 配置是否已配置",
				Status:      ModJobUiBackend.checkImageSecretStatus(),
				Progress:    ModJobUiBackend.getStepProgress("check_image_secret"),
			},
			{
				ID:          "check_image_registry",
				Name:        "镜像仓库",
				Description: "检查镜像仓库配置和连接",
				Status:      ModJobUiBackend.checkImageRegistryStatus(),
				Progress:    ModJobUiBackend.getStepProgress("check_image_registry"),
			},
			{
				ID:          "image_upload_server",
				Name:        "镜像上传服务器",
				Description: "检查镜像上传服务器和健康检查接口",
				Status:      ModJobUiBackend.getGroupStatus([]string{"check_image_upload_service"}),
				Progress:    ModJobUiBackend.getGroupProgress([]string{"check_image_upload_service"}),
				Children: []InitializationStep{
					{
						ID:          "check_image_upload_service",
						Name:        "Public 配置检查",
						Description: "检查镜像上传服务对应的 public 配置是否已配置",
						Status:      ModJobUiBackend.checkImageUploadServiceStatus(),
						Progress:    ModJobUiBackend.getStepProgress("check_image_upload_service"),
					},
				},
			},
		},
	}

	// 本地工具和配置检查
	localToolsStep := InitializationStep{
		ID:          "local_tools",
		Name:        "本地工具和配置",
		Description: "检查本地工具安装和配置",
		Status:      ModJobUiBackend.getGroupStatus([]string{"check_rclone", "check_kubectl", "check_ssh_config", "check_workspace"}),
		Progress:    ModJobUiBackend.getGroupProgress([]string{"check_rclone", "check_kubectl", "check_ssh_config", "check_workspace"}),
		Children: []InitializationStep{
			func() InitializationStep {
				status, errorMsg := processCheckResult(ModJobUiBackend.checkRcloneStatus)
				return InitializationStep{
					ID:          "check_rclone",
					Name:        "Rclone 检查",
					Description: "检查 Rclone 是否安装并配置正确",
					Status:      status,
					Error:       errorMsg,
					Progress:    ModJobUiBackend.getStepProgress("check_rclone"),
				}
			}(),
			func() InitializationStep {
				status, errorMsg := processCheckResult(ModJobUiBackend.checkKubectlStatus)
				return InitializationStep{
					ID:          "check_kubectl",
					Name:        "Kubectl 检查",
					Description: "检查 Kubectl 是否安装并可用",
					Status:      status,
					Error:       errorMsg,
					Progress:    ModJobUiBackend.getStepProgress("check_kubectl"),
				}
			}(),
			func() InitializationStep {
				status, errorMsg := processCheckResult(ModJobUiBackend.checkSshConfigStatus)
				return InitializationStep{
					ID:          "check_ssh_config",
					Name:        "SSH 配置检查",
					Description: "检查 SSH 密钥配置是否正确",
					Status:      status,
					Error:       errorMsg,
					Progress:    ModJobUiBackend.getStepProgress("check_ssh_config"),
				}
			}(),
			{
				ID:          "check_workspace",
				Name:        "工作空间检查",
				Description: "检查工作空间目录和权限",
				Status:      ModJobUiBackend.checkWorkspaceStatus(),
				Progress:    ModJobUiBackend.getStepProgress("check_workspace"),
			},
		},
	}

	// 组织所有根步骤
	steps := []InitializationStep{
		mainNodeStep,
		k8sClusterStep,
		imageServiceStep,
		localToolsStep,
	}

	// 计算整体状态和进度
	overallStatus := "completed"
	totalProgress := 0
	hasError := false
	hasRunning := false
	hasPending := false

	// 递归计算所有步骤的状态和进度
	var calculateStats func([]InitializationStep)
	calculateStats = func(stepList []InitializationStep) {
		for _, step := range stepList {
			totalProgress += step.Progress
			switch step.Status {
			case "error":
				hasError = true
			case "running":
				hasRunning = true
			case "pending":
				hasPending = true
			}
			// 递归处理子步骤
			if len(step.Children) > 0 {
				calculateStats(step.Children)
			}
		}
	}

	calculateStats(steps)

	if hasError {
		overallStatus = "error"
	} else if hasRunning {
		overallStatus = "running"
	} else if hasPending {
		overallStatus = "pending"
	}

	// 计算总步骤数（包括子步骤）
	var countSteps func([]InitializationStep) int
	countSteps = func(stepList []InitializationStep) int {
		count := 0
		for _, step := range stepList {
			count++
			if len(step.Children) > 0 {
				count += countSteps(step.Children)
			}
		}
		return count
	}

	totalStepCount := countSteps(steps)
	overallProgress := totalProgress / totalStepCount

	return InitializationStatus{
		Steps:           steps,
		OverallStatus:   overallStatus,
		OverallProgress: overallProgress,
	}
}

func (_ ModJobUiBackendStruct) checkRcloneStatus() string {
	manager := BinManagerRclone{}
	if manager.CheckInstalled() {
		return "completed"
	}
	return "Rclone 工具未安装，请运行安装命令或手动安装 Rclone"
}

func (_ ModJobUiBackendStruct) checkSshConfigStatus() string {
	homeDir := homedir.HomeDir()
	ed25519FilePath := filepath.Join(homeDir, ".ssh", "id_ed25519")
	ed25519PubFilePath := filepath.Join(homeDir, ".ssh", "id_ed25519.pub")

	if _, err := os.Stat(ed25519FilePath); err == nil {
		if _, err := os.Stat(ed25519PubFilePath); err == nil {
			return "completed"
		}
		return fmt.Sprintf("SSH 公钥文件缺失: %s", ed25519PubFilePath)
	}
	return fmt.Sprintf("SSH 私钥文件缺失: %s，请生成 SSH 密钥对", ed25519FilePath)
}

func (_ ModJobUiBackendStruct) checkWorkspaceStatus() string {
	workdir := util.WorkspaceDir()
	if _, err := os.Stat(workdir); err == nil {
		// 检查是否有写权限
		testFile := filepath.Join(workdir, ".telego_test")
		if file, err := os.Create(testFile); err == nil {
			file.Close()
			os.Remove(testFile)
			return "completed"
		}
	}
	return "error"
}

func (_ ModJobUiBackendStruct) getStepProgress(stepId string) int {
	// 这里可以实现更精细的进度追踪
	// 目前简单返回基于状态的进度
	result := ""
	switch stepId {
	case "check_fileserver":
		result = ModJobUiBackend.checkFileserverStatus()
	case "check_rclone":
		result = ModJobUiBackend.checkRcloneStatus()
	case "check_kubectl":
		result = ModJobUiBackend.checkKubectlStatus()
	case "check_fileserver_tools":
		result = ModJobUiBackend.checkFileserverToolsStatus()
	case "check_kubeconfig":
		result = ModJobUiBackend.checkKubeconfigStatus()
	case "check_k8s_cluster":
		result = ModJobUiBackend.checkK8sClusterStatus()
	case "check_image_secret":
		result = ModJobUiBackend.checkImageSecretStatus()
	case "check_image_registry":
		result = ModJobUiBackend.checkImageRegistryStatus()
	case "check_image_upload_service":
		result = ModJobUiBackend.checkImageUploadServiceStatus()
	case "check_ssh_config":
		result = ModJobUiBackend.checkSshConfigStatus()
	case "check_workspace":
		result = ModJobUiBackend.checkWorkspaceStatus()
	}

	if result == "completed" {
		return 100
	} else if result == "running" {
		return 50
	} else {
		return 0 // error 或 pending 的情况
	}
}

func (_ ModJobUiBackendStruct) runInitializationProcess() {
	// 这里实现实际的初始化流程
	fmt.Println(color.BlueString("Starting initialization process..."))

	steps := []string{
		"check_fileserver",
		"check_rclone",
		"check_kubectl",
		"check_fileserver_tools",
		"check_kubeconfig",
		"check_k8s_cluster",
		"check_image_registry",
		"check_image_upload_service",
		"check_ssh_config",
		"check_workspace",
	}

	for _, stepId := range steps {
		fmt.Printf("Processing step: %s\n", stepId)
		ModJobUiBackend.executeInitializationStep(stepId)
		time.Sleep(1 * time.Second) // 模拟处理时间
	}

	fmt.Println(color.GreenString("Initialization process completed"))
}

func (_ ModJobUiBackendStruct) executeInitializationStep(stepId string) {
	switch stepId {
	case "check_rclone":
		// 安装 Rclone
		manager := BinManagerRclone{}
		if !manager.CheckInstalled() {
			binManager := NewBinManager(manager)
			err := binManager.MakeSureWith()
			if err != nil {
				fmt.Printf("Failed to install rclone: %v\n", err)
			}
		}
	case "check_kubectl":
		// 安装 Kubectl
		manager := BinManagerKubectl{}
		if !manager.CheckInstalled() {
			binManager := NewBinManager(manager)
			err := binManager.MakeSureWith()
			if err != nil {
				fmt.Printf("Failed to install kubectl: %v\n", err)
			}
		}
	case "check_fileserver_tools":
		// 检查并上传文件服务器工具
		fmt.Println("Checking fileserver tools...")
		// TODO: 实现文件服务器工具上传逻辑
	case "check_kubeconfig":
		// 检查 kubeconfig 配置
		fmt.Println("Checking kubeconfig...")
		// TODO: 实现 kubeconfig 配置检查逻辑
	case "check_k8s_cluster":
		// 检查 K8s 集群连接
		fmt.Println("Checking K8s cluster connection...")
		// TODO: 实现 K8s 集群连接检查逻辑
	case "check_image_secret":
		// 检查镜像服务 Secret 配置
		fmt.Println("Checking image service secret...")
		// TODO: 实现镜像服务 secret 配置检查逻辑
	case "check_image_registry":
		// 检查镜像仓库
		fmt.Println("Checking image registry...")
		// TODO: 实现镜像仓库检查逻辑
	case "check_image_upload_service":
		// 检查镜像上传服务
		fmt.Println("Checking image upload service...")
		// TODO: 实现镜像上传服务检查逻辑
	case "check_ssh_config":
		// 尝试配置SSH密钥
		// 这里可以调用相关的SSH配置函数
		fmt.Println("Checking SSH configuration...")
	case "check_workspace":
		// 初始化工作空间
		util.InitOwnedDir()
	}
}

func (_ ModJobUiBackendStruct) retryInitializationStep(stepId string) {
	fmt.Printf("Retrying step: %s\n", stepId)
	ModJobUiBackend.executeInitializationStep(stepId)
}

func (_ ModJobUiBackendStruct) checkKubectlStatus() string {
	manager := BinManagerKubectl{}
	if manager.CheckInstalled() {
		return "completed"
	}
	return "Kubectl 工具未安装，请运行安装命令或手动安装 Kubectl"
}

func (_ ModJobUiBackendStruct) checkFileserverStatus() string {
	if util.FileServerAccessible() {
		return "completed"
	}
	return fmt.Sprintf("无法访问主节点文件服务器 (http://%s:8003), 请检查网络连接或确保文件服务器已启动", util.MainNodeIp)
}

func (_ ModJobUiBackendStruct) checkFileserverToolsStatus() string {
	// 检查本地 /teledeploy 目录下是否有 rclone 和 kubectl 项目

	// 需要检查的文件列表
	filesToCheck := []string{
		// rclone
		"/teledeploy/bin_rclone/rclone_amd64",
		"/teledeploy/bin_rclone/rclone_arm64",
		"/teledeploy/bin_rclone/rclone.exe",
		// kubectl
		"/teledeploy/bin_kubectl/kubectl.exe",
		"/teledeploy/bin_kubectl/kubectl_arm64",
		"/teledeploy/bin_kubectl/kubectl_amd64",
	}

	// 检查每个文件是否存在
	missingFiles := []string{}
	for _, file := range filesToCheck {
		if _, err := os.Stat(file); err != nil {
			if os.IsNotExist(err) {
				missingFiles = append(missingFiles, file)
			}
		}
	}

	// 根据检查结果返回状态或错误信息
	if len(missingFiles) > 0 {
		errorMsg := fmt.Sprintf("缺失以下工具文件: %v. 请确保已经正确上传 bin_rclone 和 bin_kubectl 项目到主节点", missingFiles)
		util.Logger.Debugf("Missing fileserver tools: %v", missingFiles)
		return errorMsg
	}

	return "completed"
}

func (_ ModJobUiBackendStruct) checkKubeconfigStatus() string {
	homeDir := homedir.HomeDir()
	kubeconfigPath := filepath.Join(homeDir, ".kube", "config")

	if _, err := os.Stat(kubeconfigPath); err == nil {
		return "completed"
	}
	return "error"
}

func (_ ModJobUiBackendStruct) checkK8sClusterStatus() string {
	// 检查 K8s 集群连接
	// 这里需要实现具体的集群连接检查逻辑
	// TODO: 实现 K8s 集群连接检查
	return "pending"
}

func (_ ModJobUiBackendStruct) checkImageRegistryStatus() string {
	// 检查镜像仓库配置
	// 这里需要实现具体的镜像仓库检查逻辑
	// TODO: 实现镜像仓库检查逻辑
	return "pending"
}

func (_ ModJobUiBackendStruct) checkImageUploadServiceStatus() string {
	// 检查镜像上传服务
	// 这里需要实现具体的镜像上传服务检查逻辑，包括 health 接口
	// TODO: 实现镜像上传服务检查逻辑
	return "pending"
}

func (_ ModJobUiBackendStruct) getGroupStatus(stepIds []string) string {
	completedCount := 0
	errorCount := 0
	runningCount := 0

	for _, stepId := range stepIds {
		result := ""
		switch stepId {
		case "check_fileserver":
			result = ModJobUiBackend.checkFileserverStatus()
		case "check_fileserver_tools":
			result = ModJobUiBackend.checkFileserverToolsStatus()
		case "check_rclone":
			result = ModJobUiBackend.checkRcloneStatus()
		case "check_kubectl":
			result = ModJobUiBackend.checkKubectlStatus()
		case "check_kubeconfig":
			result = ModJobUiBackend.checkKubeconfigStatus()
		case "check_k8s_cluster":
			result = ModJobUiBackend.checkK8sClusterStatus()
		case "check_image_secret":
			result = ModJobUiBackend.checkImageSecretStatus()
		case "check_image_registry":
			result = ModJobUiBackend.checkImageRegistryStatus()
		case "check_image_upload_service":
			result = ModJobUiBackend.checkImageUploadServiceStatus()
		case "check_ssh_config":
			result = ModJobUiBackend.checkSshConfigStatus()
		case "check_workspace":
			result = ModJobUiBackend.checkWorkspaceStatus()
		// 对于组合步骤，递归检查
		case "k8s_main_cluster":
			result = ModJobUiBackend.getGroupStatus([]string{"check_k8s_cluster"})
		case "image_upload_server":
			result = ModJobUiBackend.getGroupStatus([]string{"check_image_upload_service"})
		default:
			result = "pending"
		}

		// 根据返回值判断状态
		if result == "completed" {
			completedCount++
		} else if result == "running" {
			runningCount++
		} else if result == "pending" {
			// pending 状态不计入错误
		} else {
			// 其他情况都视为错误
			errorCount++
		}
	}

	// 如果有错误，整体状态为错误
	if errorCount > 0 {
		return "error"
	}
	// 如果有运行中的，整体状态为运行中
	if runningCount > 0 {
		return "running"
	}
	// 如果全部完成，整体状态为完成
	if completedCount == len(stepIds) {
		return "completed"
	}
	// 否则为待处理
	return "pending"
}

func (_ ModJobUiBackendStruct) getGroupProgress(stepIds []string) int {
	totalProgress := 0
	for _, stepId := range stepIds {
		totalProgress += ModJobUiBackend.getStepProgress(stepId)
	}
	return totalProgress / len(stepIds)
}

func (_ ModJobUiBackendStruct) checkImageSecretStatus() string {
	// 检查镜像服务对应的 secret 配置是否已配置
	// 这里需要实现具体的检查逻辑，暂时返回 pending
	// TODO: 实现镜像服务 secret 配置检查逻辑
	return "pending"
}

func processCheckResult(checkFunc func() string) (status string, errorMsg string) {
	result := checkFunc()
	if result == "completed" {
		return "completed", ""
	}
	return "error", result
}
