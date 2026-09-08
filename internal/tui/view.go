package tui

import (
	"fmt"
	"strings"

	"PlugLever/internal/manager"
	"PlugLever/internal/theme"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
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

	// 插件详情弹窗模式
	if m.mode == ModeDetail {
		return m.viewDetail()
	}

	// 帮助菜单弹窗模式
	if m.mode == ModeHelp {
		return m.viewHelp()
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

	// 标题行
	title := theme.TitleStyle(t).Width(m.termWidth).Render("  PlugLever - Minecraft Plugin Manager")
	b.WriteString(title)
	b.WriteString("\n")

	// 路径行（做最大宽度截断，防止超长路径导致折行抖动）
	maxPathLen := m.termWidth - 14
	if maxPathLen < 10 {
		maxPathLen = 10
	}
	pathText := fmt.Sprintf("  当前目录: %s", truncateStringWidth(m.pluginDir, maxPathLen))
	pathLine := theme.SecondaryStyle(t).Width(m.termWidth).Render(pathText)
	b.WriteString(pathLine)
	b.WriteString("\n")

	// 统计行（含多选统计）
	enabled := m.countEnabled()
	disabled := len(m.plugins) - enabled
	statsText := fmt.Sprintf("  总计: %d │ 启用: %d │ 禁用: %d", len(m.plugins), enabled, disabled)
	if m.markedCount() > 0 {
		statsText += fmt.Sprintf(" │ 已选: %d", m.markedCount())
	}
	statsText += fmt.Sprintf(" │ 排序: %s │ 主题: %s", m.sortMode.Label(), t.Name)
	b.WriteString(theme.SecondaryStyle(t).Width(m.termWidth).Render(statsText))
	b.WriteString("\n")

	// 分隔线
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
		if len(m.plugins) == 0 {
			b.WriteString(theme.SecondaryStyle(t).Render("  （目录中没有找到插件文件）"))
		} else {
			b.WriteString(theme.SecondaryStyle(t).Render("  （没有匹配的插件，按 Esc 可清空搜索）"))
		}
		b.WriteString("\n")
	} else {
		visible := m.visibleRows()
		end := m.viewTop + visible
		if end > len(m.filtered) {
			end = len(m.filtered)
		}
		versionColWidth := m.maxVersionColWidth()

		for i := m.viewTop; i < end; i++ {
			idx := m.filtered[i]
			plugin := m.plugins[idx]
			isSelected := (i == m.cursor)
			isMarked := m.isMarked(plugin.FilePath)

			line := m.renderPluginLine(plugin, isSelected, isMarked, t, versionColWidth)
			b.WriteString(line)
			b.WriteString("\n")
		}

		// 填充空行以稳定视窗高度，防止跳动
		rendered := end - m.viewTop
		for i := rendered; i < visible; i++ {
			b.WriteString("\n")
		}
	}

	// ========== Footer 区域 ==========
	b.WriteString(separator)
	b.WriteString("\n")

	helpStyle := theme.HelpStyle(t).Width(m.termWidth - 2)
	switch m.mode {
	case ModeSearch:
		searchLabel := theme.AccentStyle(t).Render("  搜索: ")
		matchTip := theme.SecondaryStyle(t).Render(fmt.Sprintf(" (匹配 %d/%d)", len(m.filtered), len(m.plugins)))
		b.WriteString(searchLabel + m.searchInput.View() + matchTip)
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

	// 状态消息行（做宽度硬截断防折行抖动）
	if m.statusMsg != "" {
		maxMsgLen := m.termWidth - 4
		if maxMsgLen < 10 {
			maxMsgLen = 10
		}
		msgText := truncateStringWidth(m.statusMsg, maxMsgLen)
		if strings.HasPrefix(m.statusMsg, "[错误]") {
			b.WriteString(theme.ErrorStyle(t).Render("  " + msgText))
		} else if strings.Contains(m.statusMsg, "⚠️") {
			b.WriteString(theme.SearchHighlightStyle(t).Render("  " + msgText))
		} else {
			b.WriteString(theme.AccentStyle(t).Render("  " + msgText))
		}
	}
	b.WriteString("\n")

	return b.String()
}

// formatVersionCol 完整格式化版本号标签。若存在可用更新，则附加醒目的新版本提示。
func formatVersionCol(ver, updateVer string) string {
	if ver == "" || ver == "Unknown" {
		if updateVer != "" {
			return fmt.Sprintf(" [➜ v%s]", cleanVersionPrefix(updateVer))
		}
		return ""
	}
	vTrim := strings.TrimSpace(ver)
	prefix := "v"
	if strings.HasPrefix(strings.ToLower(vTrim), "v") {
		prefix = ""
	}

	if updateVer != "" {
		uTrim := cleanVersionPrefix(updateVer)
		return fmt.Sprintf(" [%s%s ➜ v%s]", prefix, vTrim, uTrim)
	}
	return fmt.Sprintf(" [%s%s]", prefix, vTrim)
}

func cleanVersionPrefix(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	return strings.TrimSpace(v)
}

// maxVersionColWidth 动态计算全量列表中版本列的最大视觉宽度，确保所有版本（含新版角标）完全展示且右侧文件大小列对齐
func (m Model) maxVersionColWidth() int {
	maxW := 0
	for _, p := range m.plugins {
		col := formatVersionCol(p.Version, p.UpdateVersion)
		w := runewidth.StringWidth(col)
		if w > maxW {
			maxW = w
		}
	}
	return maxW
}

// renderPluginLine 渲染单行插件信息。
// 采用严格的终端显示列宽对齐与超长文本平滑截断，确保中文/全角/Emoji 下无折行且列严格对齐。
func (m Model) renderPluginLine(p manager.Plugin, isSelected bool, isMarked bool, t theme.Theme, versionColWidth int) string {
	cursorText := "  "
	if isSelected {
		cursorText = "▸ "
	}

	markText := "    "
	if isMarked {
		markText = "[✓] "
	} else if m.markedCount() > 0 {
		markText = "[ ] "
	}

	statusText := "[OFF]"
	if p.Enabled {
		statusText = "[ON] "
	}

	prefixPlain := "  " + cursorText + markText + statusText + " "
	prefixWidth := runewidth.StringWidth(prefixPlain)

	// 右侧版本与文件大小列（版本号完整原样展示，不截断）
	sizeStr := manager.FormatFileSize(p.FileSizeBytes)
	sizeCol := fmt.Sprintf("%10s", sizeStr)
	sizeColWidth := runewidth.StringWidth(sizeCol)

	versionCol := formatVersionCol(p.Version, p.UpdateVersion)
	verW := runewidth.StringWidth(versionCol)
	if verW < versionColWidth {
		versionCol = versionCol + strings.Repeat(" ", versionColWidth-verW)
	}

	rightColWidth := versionColWidth + sizeColWidth + 2
	availableNameWidth := m.termWidth - prefixWidth - rightColWidth
	if availableNameWidth < 12 {
		availableNameWidth = 12
	}

	displayName := truncateStringWidth(p.Name, availableNameWidth)
	namePadding := availableNameWidth - runewidth.StringWidth(displayName)
	if namePadding < 0 {
		namePadding = 0
	}
	nameCol := displayName + strings.Repeat(" ", namePadding)

	if isSelected {
		bg := lipgloss.Color(t.CursorColor)

		cursorPart := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Accent)).Background(bg).Bold(true).Render("  " + cursorText)
		markPart := lipgloss.NewStyle().Foreground(lipgloss.Color(t.SearchHighlight)).Background(bg).Bold(true).Render(markText)

		var statusPart string
		if p.Enabled {
			statusPart = lipgloss.NewStyle().Foreground(lipgloss.Color(t.OnColor)).Background(bg).Bold(true).Render(statusText)
		} else {
			statusPart = lipgloss.NewStyle().Foreground(lipgloss.Color(t.ErrorColor)).Background(bg).Render(statusText)
		}

		nameColor := t.OnColor
		if !p.Enabled {
			nameColor = t.ErrorColor
		}
		namePart := lipgloss.NewStyle().Foreground(lipgloss.Color(nameColor)).Background(bg).Bold(true).Render(" " + nameCol)

		var verPart string
		if p.UpdateVersion != "" {
			verPart = lipgloss.NewStyle().Foreground(lipgloss.Color(t.SearchHighlight)).Background(bg).Bold(true).Render(versionCol)
		} else {
			verPart = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgSecondary)).Background(bg).Render(versionCol)
		}
		sizePart := lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgSecondary)).Background(bg).Render(sizeCol)

		rowContent := cursorPart + markPart + statusPart + namePart + verPart + sizePart
		rowWidth := lipgloss.Width(rowContent)
		if rowWidth < m.termWidth {
			padding := strings.Repeat(" ", m.termWidth-rowWidth)
			rowContent += lipgloss.NewStyle().Background(bg).Render(padding)
		}
		return rowContent
	}

	// 非选中行
	cursorPart := "  " + cursorText
	markPart := theme.SecondaryStyle(t).Render(markText)
	var statusPart string
	if p.Enabled {
		statusPart = theme.StatusOnStyle(t).Render(statusText)
	} else {
		statusPart = theme.StatusOffStyle(t).Render(statusText)
	}

	var namePart string
	if p.Enabled {
		namePart = lipgloss.NewStyle().Foreground(lipgloss.Color(t.OnColor)).Bold(true).Render(" " + nameCol)
	} else {
		namePart = lipgloss.NewStyle().Foreground(lipgloss.Color(t.ErrorColor)).Render(" " + nameCol)
	}

	var verPart string
	if p.UpdateVersion != "" {
		verPart = lipgloss.NewStyle().Foreground(lipgloss.Color(t.SearchHighlight)).Bold(true).Render(versionCol)
	} else {
		verPart = theme.SecondaryStyle(t).Render(versionCol)
	}
	sizePart := theme.SecondaryStyle(t).Render(sizeCol)

	return cursorPart + markPart + statusPart + namePart + verPart + sizePart
}

// viewDetail 渲染插件详细信息模态框
func (m Model) viewDetail() string {
	t := theme.Themes[m.themeIndex]
	idx := m.selectedIndex()
	if idx < 0 || idx >= len(m.plugins) {
		return m.viewMain()
	}
	p := m.plugins[idx]

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(theme.TitleStyle(t).Render("  插件详情 (Plugin Inspector)"))
	b.WriteString("\n\n")

	statusBadge := theme.StatusOffStyle(t).Render("[OFF] 已禁用")
	if p.Enabled {
		statusBadge = theme.StatusOnStyle(t).Render("[ON] 已启用")
	}

	depStr := "无强依赖"
	if len(p.Dependencies) > 0 {
		depStr = strings.Join(p.Dependencies, ", ")
	}
	softDepStr := "无软依赖"
	if len(p.SoftDependencies) > 0 {
		softDepStr = strings.Join(p.SoftDependencies, ", ")
	}

	descStr := p.Description
	if descStr == "" {
		descStr = "（未提供描述）"
	}
	webStr := p.Website
	if webStr == "" {
		webStr = "（未提供官网）"
	}
	modStr := "未知"
	if !p.ModTime.IsZero() {
		modStr = p.ModTime.Format("2006-01-02 15:04:05")
	}

	boxWidth := m.termWidth - 6
	if boxWidth > 74 {
		boxWidth = 74
	}
	if boxWidth < 40 {
		boxWidth = 40
	}

	versionDisplay := p.Version
	if p.UpdateVersion != "" {
		versionDisplay = fmt.Sprintf("%s ➜ %s (发现新版本！)", p.Version, p.UpdateVersion)
	}

	content := fmt.Sprintf(
		"文件名:          %s\n"+
			"注册名:          %s\n"+
			"当前状态:        %s\n"+
			"版本号:          %s\n"+
			"插件作者:        %s\n"+
			"强依赖 (depend):    %s\n"+
			"软依赖 (softdepend):%s\n"+
			"插件主页:        %s\n"+
			"文件大小:        %s (%d 字节)\n"+
			"修改时间:        %s\n"+
			"完整路径:        %s\n\n"+
			"功能简介:\n  %s",
		p.FileName,
		p.RawName,
		statusBadge,
		versionDisplay,
		p.Author,
		depStr,
		softDepStr,
		webStr,
		manager.FormatFileSize(p.FileSizeBytes), p.FileSizeBytes,
		modStr,
		truncateStringWidth(p.FilePath, boxWidth-14),
		truncateStringWidth(descStr, (boxWidth-4)*2),
	)

	padV := 1
	if m.termHeight < 28 {
		padV = 0 // 小高度屏幕紧凑展示防溢出
	}

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Accent)).
		Padding(padV, 2).
		Width(boxWidth).
		Render(content)

	b.WriteString("  " + strings.ReplaceAll(card, "\n", "\n  "))
	b.WriteString("\n\n")
	b.WriteString(theme.HelpStyle(t).Render("  " + helpDetail))
	b.WriteString("\n")

	return b.String()
}

// viewHelp 渲染快捷键帮助模态框
func (m Model) viewHelp() string {
	t := theme.Themes[m.themeIndex]

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(theme.TitleStyle(t).Render("  PlugLever 快捷键完整指南"))
	b.WriteString("\n\n")

	helpContent := `【浏览与快速导航】
  ↑ / ↓ 或 j / k       光标上下移动单行
  Home / End 或 g / G  快速跳至首行 / 末行
  PageUp / PageDown    视窗向上 / 向下快速翻页

【状态管理与安全操作】
  Space (空格)         切换当前插件或所有已标记插件的状态
  v 或 x               切换当前行的多选标记（单项标记）
  a 或 A               快速全选 / 取消全选当前过滤列表的标记
  Ctrl + A             批量启用插件（若存在标记则仅启用标记项，否则全量）
  Ctrl + D             批量禁用插件（若存在标记则仅禁用标记项，否则全量）
  u 或 Ctrl + Z        撤销上一步操作（重命名 / 切换 / 批量修改）
  N                    安全重命名（自动清洗并保留对应后缀）
  R                    重新扫描并刷新插件目录列表

【检索与视图】
  / 或 Ctrl + F        进入关键词实时搜索过滤（支持多词空格检索，↑↓预览）
  Esc                  清空多选标记 / 清空搜索过滤 / 退出当前弹窗
  i 或 Enter           查看选中插件的完整元数据、依赖与软依赖详情
  S                    循环切换排序模式（名称、文件大小、启用优先）
  T                    循环切换界面配色主题并自动保存偏好
  Q 或 Ctrl + C        退出程序`

	boxWidth := m.termWidth - 6
	if boxWidth > 74 {
		boxWidth = 74
	}
	if boxWidth < 45 {
		boxWidth = 45
	}

	padV := 1
	if m.termHeight < 30 {
		padV = 0 // 小高度屏幕紧凑展示防溢出
	}

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Accent)).
		Padding(padV, 2).
		Width(boxWidth).
		Render(helpContent)

	b.WriteString("  " + strings.ReplaceAll(card, "\n", "\n  "))
	b.WriteString("\n\n")
	b.WriteString(theme.HelpStyle(t).Render("  " + helpHelp))
	b.WriteString("\n")

	return b.String()
}

// truncateStringWidth 根据终端实际显示列宽（视觉宽度）安全截断过长字符串，并以省略号结尾。
// 准确处理中文字符、全角标点与 Emoji，杜绝终端列宽计算偏差。
func truncateStringWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= maxWidth {
		return s
	}

	ellipsis := "…"
	ellipsisWidth := runewidth.StringWidth(ellipsis)
	if maxWidth <= ellipsisWidth {
		cur := 0
		var res []rune
		for _, r := range s {
			w := runewidth.RuneWidth(r)
			if cur+w > maxWidth {
				break
			}
			res = append(res, r)
			cur += w
		}
		return string(res)
	}

	targetWidth := maxWidth - ellipsisWidth
	cur := 0
	var res []rune
	for _, r := range s {
		w := runewidth.RuneWidth(r)
		if cur+w > targetWidth {
			break
		}
		res = append(res, r)
		cur += w
	}
	return string(res) + ellipsis
}

// truncateString 兼容旧版调用，底层重定向至基于视觉宽度的 truncateStringWidth
func truncateString(s string, maxLen int) string {
	return truncateStringWidth(s, maxLen)
}
