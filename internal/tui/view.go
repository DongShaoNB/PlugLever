package tui

import (
	"fmt"
	"strings"

	"PlugLever/internal/manager"
	"PlugLever/internal/theme"

	"github.com/charmbracelet/lipgloss"
)

// View 实现 tea.Model 接口，返回当前帧的渲染字符串。
func (m Model) View() string {
	// 等待接收终端尺寸后再渲染
	if !m.ready {
		return "\n  正在初始化...\n"
	}

	// 路径输入模式使用独立的渲染逻辑
	if m.mode == ModePathInput {
		return m.viewPathInput()
	}

	return m.viewMain()
}

// viewPathInput 渲染路径输入模式的界面
func (m Model) viewPathInput() string {
	t := theme.Themes[m.themeIndex]
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(theme.TitleStyle(t).Render("  PlugLever - Minecraft Plugin Manager"))
	b.WriteString("\n\n")
	b.WriteString(theme.NormalTextStyle(t).Render("  未检测到 plugins 目录，请手动输入插件目录路径："))
	b.WriteString("\n\n")
	b.WriteString("  > " + m.pathInput.View())
	b.WriteString("\n\n")
	b.WriteString(theme.HelpStyle(t).Render("  " + helpPathInput))
	b.WriteString("\n")

	// 显示错误/状态消息
	if m.statusMsg != "" {
		b.WriteString("\n")
		if strings.HasPrefix(m.statusMsg, "[错误]") {
			b.WriteString(theme.ErrorStyle(t).Render("  " + m.statusMsg))
		} else {
			b.WriteString(theme.AccentStyle(t).Render("  " + m.statusMsg))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// viewMain 渲染主界面（Header + Plugin List + Footer）
func (m Model) viewMain() string {
	t := theme.Themes[m.themeIndex]
	var b strings.Builder

	// ========== Header 区域 ==========
	b.WriteString("\n")

	// 标题行（限制宽度自动换行）
	title := theme.TitleStyle(t).Width(m.termWidth).Render("  PlugLever - Minecraft Plugin Manager")
	b.WriteString(title)
	b.WriteString("\n")

	// 路径行（限制宽度自动换行）
	pathLine := theme.SecondaryStyle(t).Width(m.termWidth).Render(fmt.Sprintf("  当前目录: %s", m.pluginDir))
	b.WriteString(pathLine)
	b.WriteString("\n")

	// 统计行（中文标签，限制宽度自动换行）
	enabled := m.countEnabled()
	disabled := len(m.plugins) - enabled
	statsText := fmt.Sprintf("  总计: %d │ 启用: %d │ 禁用: %d │ 排序: %s │ 主题: %s",
		len(m.plugins), enabled, disabled, m.sortMode.Label(), t.Name)
	b.WriteString(theme.SecondaryStyle(t).Width(m.termWidth).Render(statsText))
	b.WriteString("\n")

	// 分隔线（不超过终端宽度）
	sepWidth := m.termWidth - 4
	if sepWidth < 10 {
		sepWidth = 10
	}
	if sepWidth > m.termWidth {
		sepWidth = m.termWidth
	}
	separator := theme.BorderStyle(t).Render("  " + strings.Repeat("─", sepWidth))
	b.WriteString(separator)
	b.WriteString("\n")

	// ========== Plugin List 区域 ==========
	if len(m.filtered) == 0 {
		// 空列表提示
		if len(m.plugins) == 0 {
			b.WriteString(theme.SecondaryStyle(t).Render("  （目录中没有找到插件文件）"))
		} else {
			b.WriteString(theme.SecondaryStyle(t).Render("  （没有匹配的插件）"))
		}
		b.WriteString("\n")
	} else {
		// 计算可视范围
		visible := m.visibleRows()
		end := m.viewTop + visible
		if end > len(m.filtered) {
			end = len(m.filtered)
		}

		// 渲染每一行插件
		for i := m.viewTop; i < end; i++ {
			idx := m.filtered[i]
			plugin := m.plugins[idx]
			isSelected := (i == m.cursor)

			line := m.renderPluginLine(plugin, isSelected, t)
			b.WriteString(line)
			b.WriteString("\n")
		}

		// 若列表项不足以填满可视区域，填充空行保持布局稳定
		rendered := end - m.viewTop
		for i := rendered; i < visible; i++ {
			b.WriteString("\n")
		}
	}

	// ========== Footer 区域 ==========
	b.WriteString(separator)
	b.WriteString("\n")

	// 模式相关的帮助/输入行（使用 Width 限制宽度实现自动换行）
	helpStyle := theme.HelpStyle(t).Width(m.termWidth - 2)
	switch m.mode {
	case ModeSearch:
		searchLabel := theme.AccentStyle(t).Render("  搜索: ")
		b.WriteString(searchLabel + m.searchInput.View())
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  " + helpSearch))
	case ModeRename:
		renameLabel := theme.AccentStyle(t).Render("  重命名: ")
		b.WriteString(renameLabel + m.renameInput.View())
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  " + helpRename))
	default:
		b.WriteString(helpStyle.Render("  " + helpNormal))
	}
	b.WriteString("\n")

	// 状态消息行
	if m.statusMsg != "" {
		if strings.HasPrefix(m.statusMsg, "[错误]") {
			b.WriteString(theme.ErrorStyle(t).Render("  " + m.statusMsg))
		} else {
			b.WriteString(theme.AccentStyle(t).Render("  " + m.statusMsg))
		}
	}
	b.WriteString("\n")

	return b.String()
}

// renderPluginLine 渲染单行插件信息。
// 选中行通过为每个组件单独设置 Background 实现完整高亮，
// 避免嵌套 ANSI 转义码导致背景色中断。
func (m Model) renderPluginLine(p manager.Plugin, isSelected bool, t theme.Theme) string {
	// 纯文本片段（用于计算可见宽度）
	cursorText := "    "
	if isSelected {
		cursorText = "  ▸ "
	}

	statusText := "[OFF]"
	if p.Enabled {
		statusText = "[ON] "
	}

	metaText := fmt.Sprintf(" [v%s]", p.Version)
	sizeText := " " + manager.FormatFileSize(p.FileSizeBytes)

	if isSelected {
		// 选中行：为每个组件分别设置前景 + 背景，确保高亮连续无断裂
		bg := lipgloss.Color(t.CursorColor)

		cursorPart := lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Accent)).
			Background(bg).Bold(true).
			Render(cursorText)

		var statusPart string
		if p.Enabled {
			statusPart = lipgloss.NewStyle().
				Foreground(lipgloss.Color(t.OnColor)).
				Background(bg).Bold(true).
				Render(statusText)
		} else {
			statusPart = lipgloss.NewStyle().
				Foreground(lipgloss.Color(t.ErrorColor)).
				Background(bg).
				Render(statusText)
		}

		// 插件名称：启用=绿色，禁用=红色
		var nameColor string
		if p.Enabled {
			nameColor = t.OnColor
		} else {
			nameColor = t.ErrorColor
		}
		namePart := lipgloss.NewStyle().
			Foreground(lipgloss.Color(nameColor)).
			Background(bg).Bold(true).
			Render(" " + p.Name)

		metaPart := lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.FgSecondary)).
			Background(bg).
			Render(metaText)

		sizePart := lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.FgSecondary)).
			Background(bg).
			Render(sizeText)

		content := cursorPart + statusPart + namePart + metaPart + sizePart

		// 用背景色填充剩余宽度，使高亮条延伸到行尾
		visibleWidth := lipgloss.Width(content)
		if visibleWidth < m.termWidth {
			padding := strings.Repeat(" ", m.termWidth-visibleWidth)
			content += lipgloss.NewStyle().Background(bg).Render(padding)
		}

		return content
	}

	// 非选中行：仅设置前景色
	var statusPart string
	if p.Enabled {
		statusPart = theme.StatusOnStyle(t).Render(statusText)
	} else {
		statusPart = theme.StatusOffStyle(t).Render(statusText)
	}

	// 插件名称：启用=绿色，禁用=红色
	var namePart string
	if p.Enabled {
		namePart = lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.OnColor)).Bold(true).
			Render(" " + p.Name)
	} else {
		namePart = lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.ErrorColor)).
			Render(" " + p.Name)
	}
	metaPart := theme.SecondaryStyle(t).Render(metaText)
	sizePart := theme.SecondaryStyle(t).Render(sizeText)

	return cursorText + statusPart + namePart + metaPart + sizePart
}
