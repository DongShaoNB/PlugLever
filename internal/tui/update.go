package tui

import (
	"path/filepath"

	"PlugLever/internal/manager"
	"PlugLever/internal/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Update 实现 tea.Model 接口，处理所有消息并返回更新后的模型。
// 使用状态机模式：根据当前 Mode 分发到对应的处理函数。
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// 终端尺寸变更消息（首次启动与窗口调整时触发）
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		m.ready = true
		return m, nil

	// 键盘事件消息
	case tea.KeyMsg:
		switch m.mode {
		case ModePathInput:
			return m.updatePathInput(msg)
		case ModeSearch:
			return m.updateSearch(msg)
		case ModeRename:
			return m.updateRename(msg)
		default:
			return m.updateNormal(msg)
		}
	}

	return m, nil
}

// updatePathInput 处理路径输入模式下的键盘事件
func (m Model) updatePathInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "enter":
		path := m.pathInput.Value()
		if path == "" {
			m.statusMsg = "[错误] 路径不能为空"
			return m, nil
		}
		// 将用户输入的路径转为绝对路径
		absPath, err := filepath.Abs(path)
		if err != nil {
			m.statusMsg = "[错误] 无效路径: " + err.Error()
			return m, nil
		}
		m.pluginDir = absPath
		m = m.loadPlugins()
		m.mode = ModeNormal
		return m, nil

	default:
		// 其他按键交给 textinput 组件处理
		var cmd tea.Cmd
		m.pathInput, cmd = m.pathInput.Update(msg)
		return m, cmd
	}
}

// updateNormal 处理普通浏览模式下的键盘事件
func (m Model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {

	// ---- 退出 ----
	case "ctrl+c", "q", "Q":
		return m, tea.Quit

	// ---- 光标上移 ----
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.ensureCursorVisible()
		}

	// ---- 光标下移 ----
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.ensureCursorVisible()
		}

	// ---- 空格：切换当前插件状态 ----
	case " ":
		idx := m.selectedIndex()
		if idx >= 0 {
			p := &m.plugins[idx]
			if err := manager.TogglePlugin(p); err != nil {
				m.statusMsg = "[错误] " + err.Error()
			} else {
				m.statusMsg = "" // 切换成功不显示提示
			}
		}

	// ---- T/t：切换主题 ----
	case "t", "T":
		m.themeIndex = theme.NextThemeIndex(m.themeIndex)
		m.statusMsg = "主题已切换: " + theme.Themes[m.themeIndex].Name

	// ---- Ctrl+A：批量启用所有插件 ----
	case "ctrl+a":
		count, err := manager.EnableAll(m.plugins)
		if err != nil {
			m.statusMsg = "[错误] " + err.Error()
		} else {
			m.statusMsg = "" // 批量启用不显示提示
			_ = count
		}

	// ---- Ctrl+D：批量禁用所有插件 ----
	case "ctrl+d":
		count, err := manager.DisableAll(m.plugins)
		if err != nil {
			m.statusMsg = "[错误] " + err.Error()
		} else {
			m.statusMsg = "" // 批量禁用不显示提示
			_ = count
		}

	// ---- S/s：切换排序方式 ----
	case "s", "S":
		m.sortMode = SortMode((int(m.sortMode) + 1) % int(sortModeCount))
		m.sortPlugins()
		m.resetFilter()
		m.statusMsg = "排序方式: " + m.sortMode.Label()

	// ---- / 或 Ctrl+F：进入搜索模式 ----
	case "/", "ctrl+f":
		m.mode = ModeSearch
		m.searchInput.SetValue("")
		m.statusMsg = ""
		cmd := m.searchInput.Focus()
		return m, cmd

	// ---- N/n：进入重命名模式 ----
	case "n", "N":
		idx := m.selectedIndex()
		if idx >= 0 {
			m.mode = ModeRename
			m.renameInput.SetValue(m.plugins[idx].Name) // 预填当前主文件名
			m.statusMsg = ""
			cmd := m.renameInput.Focus()
			return m, cmd
		} else {
			m.statusMsg = "[错误] 没有可操作的插件"
		}

	// ---- R/r：手动刷新插件列表 ----
	case "r", "R":
		m = m.loadPlugins()
	}

	return m, nil
}

// updateSearch 处理搜索过滤模式下的键盘事件
func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	// Esc：退出搜索并还原完整列表
	case "esc":
		m.mode = ModeNormal
		m.searchInput.Blur()
		m.resetFilter()
		m.statusMsg = ""
		return m, nil

	// Enter：退出搜索但保留当前过滤结果
	case "enter":
		m.mode = ModeNormal
		m.searchInput.Blur()
		m.statusMsg = ""
		return m, nil

	default:
		// 将输入交给 textinput 处理，然后实时过滤
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.applyFilter(m.searchInput.Value())
		return m, cmd
	}
}

// updateRename 处理重命名模式下的键盘事件
func (m Model) updateRename(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	// Esc：取消重命名
	case "esc":
		m.mode = ModeNormal
		m.renameInput.Blur()
		m.statusMsg = ""
		return m, nil

	// Enter：执行重命名
	case "enter":
		idx := m.selectedIndex()
		if idx >= 0 {
			newName := m.renameInput.Value()
			if err := manager.RenamePlugin(&m.plugins[idx], newName); err != nil {
				m.statusMsg = "[错误] " + err.Error()
			} else {
				m.statusMsg = "已重命名为: " + m.plugins[idx].Name
				// 重命名后刷新列表（文件名变更可能影响排序）
				m = m.loadPlugins()
			}
		}
		m.mode = ModeNormal
		m.renameInput.Blur()
		return m, nil

	default:
		// 将输入交给 textinput 处理
		var cmd tea.Cmd
		m.renameInput, cmd = m.renameInput.Update(msg)
		return m, cmd
	}
}

// 确保 textinput 包被正确引用（Focus 返回的 Cmd 类型来自 bubbles）
var _ = textinput.Blink
