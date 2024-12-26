package app

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"telego/app/config"
	"telego/util"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// 菜单节点定义
type MenuItem struct {
	Name       string      `yaml:"name"`
	Comment    string      `yaml:"comment,omitempty"`
	Children   []*MenuItem `yaml:"children,omitempty"`
	SpecTags   []string    `yaml:"spectags,omitempty"`
	Deployment *Deployment `json:"deployment,omitempty"`
}

// skip the root node self
// func (node *MenuItem) FindByPath(prefixes []*MenuItem, path_slices ...string) (*MenuItem, []*MenuItem) {

// 	if len(path_slices) == 0 {
// 		return nil, prefixes
// 	}
// 	for _, child := range node.Children {
// 		if child.Name == path_slices[0] {
// 			if 0 == len(path_slices)-1 {
// 				return child, prefixes
// 			} else {
// 				prefixes = append(prefixes, child)
// 				return child.FindByPath(prefixes, path_slices[1:]...)
// 			}
// 		}
// 	}
// 	return nil, prefixes
// }

func (i *MenuItem) containsChildName(name string) bool {
	// for each child
	for _, child := range i.Children {
		if child.Name == name {
			return true
		}
	}
	return false
}

func (i *MenuItem) FindChild(name string) *MenuItem {
	for _, child := range i.Children {
		if child.Name == name {
			return child
		}
	}
	return nil
}

func (i *MenuItem) isPy() bool {
	return strings.HasSuffix(i.Name, ".py")
}

func (i *MenuItem) isUnlimit() bool {
	return strings.Compare(i.Comment, "未限定脚本") == 0
}

func (i *MenuItem) getEmbedScript() string {
	// find child name is embed_script
	for _, child := range i.Children {
		if child.Name == "embed_script" {
			return child.Comment
		}
	}
	return ""
}

var initMenuTreeWg sync.WaitGroup

// 初始化菜单树ji
func InitMenuTree(loadSubPrjs bool) *MenuItem {
	// 解析 YAML 数据
	var menu MenuItem
	err := yaml.Unmarshal([]byte(MenuTreeData), &menu)
	if err != nil {
		fmt.Println("Error decoding YAML:", err)
		// 退出程序 ,直接panic
		os.Exit(1)
	}
	// if util.HasNetwork() {
	// 	// remove deploy-templete
	// 	menu.Children = funk.Filter(menu.Children, func(item *MenuItem) bool {
	// 		return item.Name != "deploy-templete-upload"
	// 	}).([]*MenuItem)
	// }
	deploy := menu.FindChild("deploy")

	var waitGoFuncStart sync.WaitGroup
	waitGoFuncStart.Add(1)
	go func() {
		initMenuTreeWg.Add(1)
		defer initMenuTreeWg.Done()
		waitGoFuncStart.Done()

		if deploy != nil && loadSubPrjs {
			deploy.LoadSubPrjs()
			initMenuTreeWg.Add(len(deploy.Children))
			for _, c := range deploy.Children {
				util.Logger.Debug("async load deployment yml: " + c.Name)
				go func() {
					defer initMenuTreeWg.Done()
					c.LoadDeploymentYml()
					if c.Deployment != nil {
						c.Comment = c.Comment
						util.Logger.Debug("async load deployment yml success: " + c.Name)
					} else {
						util.Logger.Warn("async load deployment yml failed: " + c.Name)
					}
				}()
			}
		}
	}()
	waitGoFuncStart.Wait()
	deploy_temp := menu.FindChild("deploy-templete")
	for _, c := range deploy_temp.Children {
		util.Logger.Debug("one deploy-templete: " + c.Name)
	}

	return &menu
}

func Main() {
	go util.HasNetwork()
	workdir := util.WorkspaceDir()

	// 设置日志级别
	util.Logger.SetLevel(logrus.DebugLevel)
	// time stamp
	curtime := time.Now().Format("2006-01-02-15h04m05s")
	file, err := os.OpenFile(path.Join(workdir, fmt.Sprintf("%s.log", curtime)), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Printf("Error opening log file: %v\n", err)
		return
	}
	defer file.Close()
	util.Logger.SetOutput(file)

	util.SaveEntryDir()

	_, err = os.Getwd()
	if err != nil {
		// fmt.Println("Error:", err)
		util.Logger.Error("Error:", err)
		return
	}

	// run faster than at mounted path
	err = os.Chdir(workdir)
	if err != nil {
		util.Logger.Error("Error:", err)
		return
	}

	err = NewBinManager(BinManagerRclone{}).MakeSureWith()
	if err != nil {
		fmt.Println(color.RedString("Rclone install failed, err: v%", err))
		os.Exit(1)
	}

	parseFail := false
	var rootCmd = &cobra.Command{
		// Use: "cmdmode",
		RunE: func(cmd *cobra.Command, args []string) error {
			parseFail = true
			return nil
		},
	}

	jobmods := []JobModInterface{
		ModJobInstall,
		ModJobApply,
		ModJobCmd,
		ModJobSsh,
		ModJobStartFileserver,
		ModJobDistributeDeploy,
		ModJobImgRepo,
		ModJobImgUploader,
		ModJobImgPrepare,
		ModJobCreateNewUser,
		ModJobFetchAdminKubeconfig,
		ModJobConfigExporter,
		ModJobRclone,
		ModJobSshFs,
	}
	for _, mod := range jobmods {
		rootCmd.AddCommand(mod.ParseJob(&cobra.Command{
			Use: mod.JobCmdName(),
		}))
	}

	rootCmd.Execute()
	if !parseFail {
		return
	}

	if util.FileServerAccessible() {
		checkAndUpgrade()
		if err := NewBinManager(BinManagerKubectl{}).MakeSureWith(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	config.Load()

	// Define table columns
	columns := []table.Column{
		{Title: "选项", Width: 30},
		{Title: "注释", Width: 50},
	}

	// Create the initial model with the table and input field
	rootMenu := InitMenuTree(true)

	// Initialize table with columns, rows, and focus settings
	t := table.New(
		table.WithColumns(columns),
		// table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7), // Set the height of the table to limit visible rows and allow scrolling
	)
	m := MenuViewModel{
		Table:   t,
		Stack:   []*MenuItem{rootMenu}, // 根菜单
		Current: rootMenu,
		Height:  12,
		Shared:  &MenuViewModelShared{},
	}

	// Define table rows (cities in this case)
	rows := FilterRows(&m, "") // Initially show all rows
	m.Table.SetRows(rows)

	// Set custom table styles (headers, selected rows, etc.)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	// Run the program
	_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()

	// util.RunPy()

	if m.Shared.EndDelegate != nil {
		m.Shared.EndDelegate()
	}

	// for _, v := range buffered {
	// 	fmt.Println(v)
	// }

	if err != nil {
		// fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
