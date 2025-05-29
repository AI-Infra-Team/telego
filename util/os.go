package util

import (
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"encoding/base64"

	"github.com/fatih/color"
	"k8s.io/client-go/util/homedir"
)

// SystemType 系统类型接口
type SystemType interface {
	GetArchCmd() string
	GetTypeName() string
}

// LinuxSystem Linux 系统类型
type LinuxSystem struct{}

func (s LinuxSystem) GetArchCmd() string {
	return "uname -m"
}

func (s LinuxSystem) GetTypeName() string {
	return "Linux"
}

// WindowsSystem Windows 系统类型
type WindowsSystem struct{}

func (s WindowsSystem) GetArchCmd() string {
	return "echo %PROCESSOR_ARCHITECTURE%"
}

func (s WindowsSystem) GetTypeName() string {
	return "Windows"
}

// DarwinSystem macOS 系统类型
type DarwinSystem struct{}

func (s DarwinSystem) GetArchCmd() string {
	return "uname -m"
}

func (s DarwinSystem) GetTypeName() string {
	return "macOS"
}

type UnknownSystem struct{}

func (s UnknownSystem) GetArchCmd() string {
	return ""
}

func (s UnknownSystem) GetTypeName() string {
	return "Unknown"
}

func IsWindows() bool {
	return runtime.GOOS == "windows"
}

func scanDirWithFilter(root string, filter func(entry os.DirEntry) bool) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("error reading directory: %w", err)
	}

	var results []string
	for _, entry := range entries {
		if filter(entry) { // 根据回调函数进行过滤
			results = append(results, filepath.Join(root, entry.Name()))
		}
	}

	return results, nil
}

var entryDir = ""

func GetEntryDir() string {
	return entryDir
}

func SaveEntryDir() string {
	entryDir0, err := filepath.Abs(CurDir())
	if err != nil {
		entryDir = homedir.HomeDir()
		fmt.Println(color.RedString("get entry dir failed %+v, \nwill set entryDir to home dir %s", err, homedir.HomeDir()))
	} else {
		entryDir = entryDir0
		fmt.Println(color.GreenString("get entry dir success %s", entryDir))
	}
	return entryDir
}

func CurDir() string {
	curDir, err := os.Getwd()
	if err != nil {
		fmt.Println(color.RedString("get current dir failed %s", err))
		os.Exit(1)
	}
	return curDir
}

type CachedHasNetwork struct {
	v bool
}

var cachedHasNetwork *CachedHasNetwork = nil

const ArchArm64 = "arm64"
const ArchAmd64 = "amd64"

func GetCurrentArch() string {
	arch := runtime.GOARCH
	switch arch {
	case "arm64", "aarch64":
		return ArchArm64
	case "amd64", "x86_64":
		return ArchAmd64
	default:
		return ArchAmd64 // Default to amd64 if unknown
	}
}

func HasNetwork() bool {
	if cachedHasNetwork != nil {
		return cachedHasNetwork.v
	}

	// 检查网络连接
	client := &http.Client{
		Timeout: 5 * time.Second, // 设置超时时间
	}
	resp, err := client.Get("https://www.baidu.com")
	if err != nil {
		cachedHasNetwork = &CachedHasNetwork{v: false}
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		cachedHasNetwork = &CachedHasNetwork{v: true}
		return true
	}

	cachedHasNetwork = &CachedHasNetwork{v: false}
	return false
}

func IsRoot() bool {
	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("Error retrieving user:", err)
		return false
	}

	return currentUser.Uid == "0"
}

// CurrentTimeString returns the current time as a formatted string
// suitable for filenames (YYYY-MM-DD-HHMMSS format)
func CurrentTimeString() string {
	return time.Now().Format("2006-01-02-150405")
}

// IsLinux returns true if the current OS is Linux
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// IsMacOS returns true if the current OS is macOS
func IsMacOS() bool {
	return runtime.GOOS == "darwin"
}

// WriteFileWithContent writes content to a file with root privileges if needed
func WriteFileWithContent(path string, content string) (string, error) {
	// Encode content to base64
	encodedContent := base64.StdEncoding.EncodeToString([]byte(content))

	// Use telego command to execute the job
	output, err := ModRunCmd.RequireRootRunCmd("telego", "decode-base64-to-file",
		"--base64", encodedContent,
		"--path", path,
		"--mode", "0644")
	return output, err
}

// GetCurrentUser 获取当前进程的用户名
func GetCurrentUser() string {
	currentUser, err := user.Current()
	if err != nil {
		Logger.Warnf("Error retrieving current user: %v", err)
		return "adminuser"
	}
	return currentUser.Username
}

// FileNode 表示文件树中的一个节点
type FileNode struct {
	Name     string               // 文件或目录名
	Path     string               // 完整路径
	IsDir    bool                 // 是否为目录
	IsLink   bool                 // 是否为符号链接
	LinkTo   string               // 如果是链接，指向的目标路径
	Children map[string]*FileNode // 子文件/目录
}

// FileTreeStruct 表示文件树结构
type FileTreeStruct struct {
	Root         *FileNode       // 根节点
	MaxDepth     int             // 最大递归深度
	VisitedPaths map[string]bool // 已访问的路径，防止循环引用
}

// GetFileTree 获取指定目录的文件树
// depth: 递归深度，0 表示不限制深度
// startPath: 起始目录路径，默认为当前目录
func GetFileTree(depth int, startPath ...string) (*FileTreeStruct, error) {
	// 确定起始路径
	path := CurDir()
	if len(startPath) > 0 && startPath[0] != "" {
		path = startPath[0]
	}

	// 获取绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("获取绝对路径失败: %w", err)
	}

	// 创建文件树结构
	tree := &FileTreeStruct{
		MaxDepth:     depth,
		VisitedPaths: make(map[string]bool),
	}

	// 检查路径是否存在
	info, err := os.Lstat(absPath)
	if err != nil {
		return nil, fmt.Errorf("获取路径信息失败: %w", err)
	}

	// 创建根节点
	isLink := info.Mode()&os.ModeSymlink != 0
	linkTo := ""
	if isLink {
		linkTo, err = os.Readlink(absPath)
		if err != nil {
			return nil, fmt.Errorf("读取链接目标失败: %w", err)
		}

		// 如果是相对路径，转换为绝对路径
		if !filepath.IsAbs(linkTo) {
			linkTo = filepath.Join(filepath.Dir(absPath), linkTo)
		}
	}

	isDir := info.IsDir()
	// 如果是链接，检查目标是否为目录
	if isLink {
		targetInfo, err := os.Stat(absPath) // 使用Stat会跟随链接
		if err == nil {
			isDir = targetInfo.IsDir()
		}
	}

	root := &FileNode{
		Name:     filepath.Base(absPath),
		Path:     absPath,
		IsDir:    isDir,
		IsLink:   isLink,
		LinkTo:   linkTo,
		Children: make(map[string]*FileNode),
	}
	tree.Root = root

	// 如果是目录，递归构建文件树
	if isDir {
		err = tree.buildTree(root, 0)
		if err != nil {
			return nil, fmt.Errorf("构建文件树失败: %w", err)
		}
	}

	return tree, nil
}

// buildTree 递归构建文件树
func (ft *FileTreeStruct) buildTree(node *FileNode, currentDepth int) error {
	// 检查是否超过最大深度
	if ft.MaxDepth > 0 && currentDepth >= ft.MaxDepth {
		return nil
	}

	// 标记当前路径为已访问
	ft.VisitedPaths[node.Path] = true

	// 获取目录内容
	var readPath string
	if node.IsLink {
		readPath = node.LinkTo
	} else {
		readPath = node.Path
	}

	entries, err := os.ReadDir(readPath)
	if err != nil {
		return fmt.Errorf("读取目录失败 %s: %w", readPath, err)
	}

	// 遍历目录内容
	for _, entry := range entries {
		childName := entry.Name()
		childPath := filepath.Join(node.Path, childName)

		// 获取文件信息，不跟随链接
		info, err := os.Lstat(childPath)
		if err != nil {
			// 忽略无法访问的文件
			continue
		}

		isLink := info.Mode()&os.ModeSymlink != 0
		linkTo := ""
		if isLink {
			linkTo, err = os.Readlink(childPath)
			if err != nil {
				// 忽略无法读取的链接
				continue
			}

			// 如果是相对路径，转换为绝对路径
			if !filepath.IsAbs(linkTo) {
				linkTo = filepath.Join(filepath.Dir(childPath), linkTo)
			}
		}

		isDir := entry.IsDir()
		// 如果是链接，尝试判断目标是否为目录
		if isLink {
			targetInfo, err := os.Stat(childPath) // 使用Stat跟随链接
			if err == nil {
				isDir = targetInfo.IsDir()
			}
		}

		childNode := &FileNode{
			Name:     childName,
			Path:     childPath,
			IsDir:    isDir,
			IsLink:   isLink,
			LinkTo:   linkTo,
			Children: make(map[string]*FileNode),
		}
		node.Children[childName] = childNode

		// 如果是目录并且还没被访问过，递归处理
		if isDir {
			// 如果是链接，检查目标路径是否已访问，防止循环
			pathToCheck := childNode.Path
			if isLink {
				pathToCheck = linkTo
			}

			if !ft.VisitedPaths[pathToCheck] {
				err = ft.buildTree(childNode, currentDepth+1)
				if err != nil {
					// 忽略子目录的错误，继续处理其他条目
					continue
				}
			}
		}
	}

	return nil
}

// GetDebugStr 获取文件树的字符串表示，便于调试或记录
func (ft *FileTreeStruct) GetDebugStr() string {
	var builder strings.Builder
	getNodeString(ft.Root, 0, &builder)
	return builder.String()
}

// 辅助函数用于递归构建节点字符串
func getNodeString(node *FileNode, depth int, builder *strings.Builder) {
	indent := strings.Repeat("  ", depth)
	nodeType := "文件"
	if node.IsDir {
		nodeType = "目录"
	}
	if node.IsLink {
		if node.IsDir {
			nodeType = fmt.Sprintf("链接(->目录: %s)", node.LinkTo)
		} else {
			nodeType = fmt.Sprintf("链接(->文件: %s)", node.LinkTo)
		}
	}

	builder.WriteString(fmt.Sprintf("%s%s [%s]\n", indent, node.Name, nodeType))

	// 按字母顺序排序子节点，使输出更有序
	var names []string
	for name := range node.Children {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		getNodeString(node.Children[name], depth+1, builder)
	}
}

// PrintTree 打印文件树结构，便于调试
func (ft *FileTreeStruct) PrintTree() {
	fmt.Print(ft.GetDebugStr())
}

func SafeCopyOverwrite(src, dst string) error {
	dstDir := filepath.Dir(dst)
	err := os.MkdirAll(dstDir, 0755)
	if err != nil {
		return fmt.Errorf("create dst dir failed %s", err)
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer srcFile.Close()

	// Create destination file (will overwrite if exists)
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer dstFile.Close()

	// Copy file contents
	_, err = srcFile.WriteTo(dstFile)
	if err != nil {
		return fmt.Errorf("failed to copy file contents from %s to %s: %w", src, dst, err)
	}

	// Sync to ensure data is written to disk
	err = dstFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync destination file %s: %w", dst, err)
	}

	return nil
}
