package app

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"telego/util"
	"telego/util/yamlext"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/thoas/go-funk"
)

// 菜单节点定义
type MenuItemYaml struct {
	Name     string          `yaml:"name"`
	Comment  string          `yaml:"comment,omitempty"`
	Children []*MenuItemYaml `yaml:"children,omitempty"`
}

type MenuItem struct {
	Name       string
	Comment    string
	Children   []*MenuItem
	SpecTags   []string
	Deployment *Deployment
}

var _ util.Conv[util.Empty, *MenuItem] = &MenuItemYaml{}

func (m *MenuItemYaml) To(arg util.Empty) (*MenuItem, error) {
	children := make([]*MenuItem, 0)
	for _, child := range m.Children {
		childItem, err := child.To(arg)
		if err != nil {
			return nil, err
		}
		children = append(children, childItem)
	}
	return &MenuItem{
		Name:     m.Name,
		Comment:  m.Comment,
		Children: children,
	}, nil
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

func WaitInitMenuTree() {
	initMenuTreeWg.Wait()
}

// 初始化菜单树ji
func InitMenuTree(loadSubPrjs bool) *MenuItem {
	// 解析 YAML 数据
	var menuYaml MenuItemYaml
	err := yamlext.UnmarshalAndValidate([]byte(MenuTreeData), &menuYaml)
	if err != nil {
		fmt.Println("Error decoding MenuItem:", err)
		// 退出程序 ,直接panic
		os.Exit(1)
	}
	menu, err := menuYaml.To(util.Empty{})
	if err != nil {
		fmt.Println("Error decoding MenuItem:", err)
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

	return menu
}

func Main() {
	go util.HasNetwork()
	workdir := util.WorkspaceDir()

	// 设置日志级别
	logfile := util.SetupFileLog()
	if logfile == nil {
		util.Logger.Error("Error setup log file")
		os.Exit(1)
	}
	defer logfile.Close()

	util.SaveEntryDir()

	_, err := os.Getwd()
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

	parseFail := false
	var rootCmd = &cobra.Command{
		// Use: "cmdmode",
		RunE: func(cmd *cobra.Command, args []string) error {
			parseFail = true
			return nil
		},
	}

	for _, mod := range jobmods {
		// fmt.Println("with job:", mod.JobCmdName())
		cmd := mod.ParseJob(&cobra.Command{
			Use: mod.JobCmdName(),
		})
		cmdRunInner := cmd.Run
		cmd.Run = func(cmd *cobra.Command, args []string) {
			util.PrintStep("telego start", "starting "+mod.JobCmdName())
			if funk.Contains(PreinitSkipInstallRcloneJobs, mod.JobCmdName()) {
				util.PrintStep(mod.JobCmdName(), "skip install rclone")
			} else {
				err = NewBinManager(BinManagerRclone{}).MakeSureWith()
				if err != nil {
					fmt.Println(color.RedString("Rclone install failed, err: v%", err))
					os.Exit(1)
				}
			}
			cmdRunInner(cmd, args)
		}
		rootCmd.AddCommand(cmd)
	}

	rootCmd.Execute()
	if !parseFail {
		return
	}

	util.PrintStep("telego start", "starting tui menu")

	if util.FileServerAccessible() {
		checkAndUpgrade()
		err = NewBinManager(BinManagerRclone{}).MakeSureWith()
		if err != nil {
			fmt.Println(color.RedString("Rclone install failed, err: v%", err))
			os.Exit(1)
		}
		if err := NewBinManager(BinManagerKubectl{}).MakeSureWith(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	ConfigLoad()

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
