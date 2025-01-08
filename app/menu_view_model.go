package app

import (
	"fmt"
	"strings"
	"telego/util"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/thoas/go-funk"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type MenuViewModelShared struct {
	EndDelegate func()
}

type MenuViewModel struct {
	Table           table.Model
	Filter          string
	Input           tea.Model   // Input field for filtering
	Stack           []*MenuItem // 用于跟踪当前的菜单层级
	Current         *MenuItem   // 当前菜单项
	Height          int
	SecondLevelView tea.Model // interface
	Shared          *MenuViewModelShared
	// filtered []*MenuItem // 筛选后的当前层级菜单
}

func (m *MenuViewModel) CurrentPath(backOffset int) string {
	path := []string{}
	// iter stack 0..len-2
	for i := 0; i < len(m.Stack)-backOffset; i++ {
		// path += m.Stack[i].Name + "/"
		path = append(path, m.Stack[i].Name)
	}
	return strings.Join(path, "/")
}

// return maybe nil
func (m *MenuViewModel) ParentNode() *MenuItem {
	if len(m.Stack) > 1 {
		return m.Stack[len(m.Stack)-1]
	} else {
		return nil
	}
}

func (m MenuViewModel) Init() tea.Cmd { return nil }

// func (m *model) Resize(msg tea.Msg) tea.Model {
// 	// Capture the terminal size change and update the terminal height
// 	if msize, ok := msg.(tea.WindowSizeMsg); ok {
// 		// m.Table.SetHeight(msize.Height - 5)
// 		m.height = msize.Height
// 	}

//		return m
//	}

func (m MenuViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.SecondLevelView != nil {
		return m.SecondLevelView.Update(msg)
	}
	selected_str := m.Table.Rows()[m.Table.Cursor()][0]
	var selected *MenuItem = nil
	for _, child := range m.Current.Children {
		if selected_str == child.Name {
			selected = child
			break
		}
	}
	skipRowDiffResetTableOffset := false

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if len(m.Stack) > 1 {
				// 返回上一级菜单
				m.Stack = m.Stack[:len(m.Stack)-1]
				m.Current = m.Stack[len(m.Stack)-1]
				m.Filter = ""
				// m.input.Init()
			} else {
				return m, tea.Quit
			}
		case "enter":
			// Handle row selection or actions
			prefixes := m.Stack
			// prefixes = append(prefixes, m.Current)
			util.Logger.Debugf("prefixes %s", strings.Join(
				funk.Map(prefixes, func(p *MenuItem) string { return p.Name }).([]string),
				"/"))

			if selected != nil {
				childerenConn := ""
				for child := range selected.Children {
					childerenConn += selected.Children[child].Name + ","
				}
				util.Logger.Debugf("menu stack %s/%s [%s]", m.CurrentPath(0), selected.Name, childerenConn)
				if selected.Children != nil && len(selected.Children) > 0 {
					allowNext := true
					if selected.IsDeploySubPrj(m.Current.Name) {
						if selected.Deployment == nil {
							allowNext = false
						}
					}

					// 进入下一级菜单
					if allowNext {
						m.Stack = append(m.Stack, selected)
						m.Current = selected
						m.Filter = ""
					}

				} else {
					res := selected.DispatchEnterNext(prefixes)
					if res.Exit {
						return m, tea.Quit
					} else if res.NextLevel {
						m.Stack = append(m.Stack, selected)
						m.Current = selected
						m.Filter = ""
					} else if res.PrevLevel {
						m.Stack = m.Stack[:len(m.Stack)-1]
						m.Current = m.Stack[len(m.Stack)-1]
						m.Filter = ""
					} else {
						res := selected.DispatchExec(prefixes)
						if res.Exit {
							if res.ExitWithDelegate != nil {
								m.Shared.EndDelegate = res.ExitWithDelegate
							}
							return m, tea.Quit
						}
					}
				}
			}
		case "backspace":
			if len(m.Filter) > 0 {
				m.Filter = m.Filter[:len(m.Filter)-1]
			}
		case "up":
		case "down":
		case "ctrl+l":
			if m.Current != nil && selected != nil && m.Current.isMultiSelect() {
				if selected.isSelected() {
					selected.Name = selected.Name[:len(selected.Name)-2]
					selected.unsetSelectedTag()
				} else {
					selected.Name += " ✓"
					selected.setSelectedTag()
				}
				skipRowDiffResetTableOffset = true
			}
		default:
			if len(msg.String()) == 1 {
				// string must be charactor can be displayed
				if msg.String()[0] >= 32 {
					// return m, nil
					m.Filter += msg.String()
				}

			}
			// Collect input for filtering

		}
	case tea.WindowSizeMsg:
		m.Height = msg.Height
		// return m, tea.Printf("%dx%d", msg.Width, msg.Height)
	}

	// Filter rows based on the current input
	filteredRows := FilterRows(&m, m.Filter)
	if FilterRowDiff(m.Table.Rows(), filteredRows) {
		m.Table.SetRows(filteredRows)
		if !skipRowDiffResetTableOffset {
			m.Table.SetCursor(0)
			m.Table.MoveUp(100000)
		}
	}

	// Update the table
	m.Table.SetHeight(m.Height - 7)
	m.Table, cmd = m.Table.Update(msg)
	return m, cmd
}

func (m MenuViewModel) View() string {
	if m.SecondLevelView != nil {
		return m.SecondLevelView.View()
	}
	// Render the input box and the table's view with the help view (scrolling and help instructions)
	inputView := lipgloss.NewStyle().Width(30).Render(fmt.Sprintf("Filter: %s", m.Filter))
	connStackStrs := []string{}
	for stackOne := range m.Stack {
		connStackStrs = append(connStackStrs, m.Stack[stackOne].Name)
	}
	connStackStr := strings.Join(connStackStrs, " > ")
	return inputView + "\n\n" + baseStyle.Render(m.Table.View()) + "\n  " +
		connStackStr + color.HiBlackString("\n  键入筛选 | ↑ 上 | ↓ 下 | ctrl+c 回退 | ctrl+l 选中/取消选中 | 项目路径 %s", ConfigLoad().ProjectDir) + "\n"
	// m.Table.HelpView() + "\n"
}

func FilterRows(m *MenuViewModel, filter string) []table.Row {
	// Original rows to filter from
	// generate rows from m.Current.Children
	rows := []table.Row{}
	for c := range m.Current.Children {
		// if m.Current.isMultiSelect() {
		// 	if m.Current.Children[c].isSelected() {
		// 		rows = append(rows, table.Row{m.Current.Children[c].Name + " ✔", m.Current.Children[c].Comment})
		// 		continue
		// 	}
		// }
		rows = append(rows, table.Row{m.Current.Children[c].Name, m.Current.Children[c].Comment})
	}

	// Filter rows based on the filter string
	var filtered []table.Row
	for _, row := range rows {
		if strings.Contains(row[0], "deployment") {
			continue
		}
		if strings.Contains(row[0], "embed_script") {
			continue
		}
		if strings.Contains(strings.ToLower(row[0]), strings.ToLower(filter)) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func FilterRowDiff(new []table.Row, old []table.Row) bool {
	if len(new) != len(old) {
		return true
	}
	for i := 0; i < len(new); i++ {
		if new[i][0] != old[i][0] {
			return true
		}
	}
	return false
}
