package tui

import (
	"fmt"
	"sort"
	"strings"

	"PlugLever/internal/config"
	"PlugLever/internal/manager"
	"PlugLever/internal/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Mode 表示 TUI 当前的交互模式
type Mode int

const (
	ModeNormal    Mode = iota // 普通浏览模式
	ModeSearch                // 搜索过滤模式
	ModeRename                // 重命名模式
	ModePathInput             // 路径输入模式（首次启动无法自动探测路径时）
	ModeDetail                // 插件详情模态弹窗
	ModeHelp                  // 快捷键帮助模态弹窗
)

// SortMode 表示插件列表的排序方式
type SortMode int

const (
	SortNameAsc    SortMode = iota // 文件名 A→Z（默认）
	SortNameDesc                   // 文件名 Z→A
	SortSizeAsc                    // 文件大小 小→大
	SortSizeDesc                   // 文件大小 大→小
	SortStatusOn                   // 启用优先
	SortStatusOff                  // 禁用优先
	sortModeCount                  // 哨兵值，用于循环计数
)

// SortModeLabel 返回排序模式的中文显示标签
func (s SortMode) Label() string {
	switch s {
	case SortNameAsc:
		return "名称 A→Z"
	case SortNameDesc:
		return "名称 Z→A"
	case SortSizeAsc:
		return "大小 ↑"
	case SortSizeDesc:
		return "大小 ↓"
	case SortStatusOn:
		return "启用优先"
	case SortStatusOff:
		return "禁用优先"
	default:
		return "名称 A→Z"
	}
}

// UndoItem 记录单个文件的路径变更，用于支持撤销
type UndoItem struct {
	OldPath string
	NewPath string
}

// UndoAction 记录一次完整的原子操作
type UndoAction struct {
	Description string
	Items       []UndoItem
}

// Model 是 Bubbletea 的核心数据模型，管理所有 TUI 状态
type Model struct {
	plugins    []manager.Plugin // 全量插件列表
	filtered   []int            // 过滤后的索引映射（指向 plugins 切片的下标）
	cursor     int              // 光标在 filtered 列表中的位置
	viewTop    int              // 分页视窗顶部在 filtered 中的位置
	pluginDir  string           // 插件目录绝对路径
	themeIndex int              // 当前主题索引
	sortMode   SortMode         // 当前排序方式

	mode Mode // 当前交互模式

	marked    map[string]bool // 多选标记（Key 为 Plugin.FilePath）
	undoStack []UndoAction    // 撤销历史栈

	searchInput textinput.Model // 搜索输入框组件
	renameInput textinput.Model // 重命名输入框组件
	pathInput   textinput.Model // 路径输入框组件

	statusMsg  string // 底部状态栏消息（操作反馈/错误提示）
	termWidth  int    // 终端宽度（字符数）
	termHeight int    // 终端高度（行数）
	ready      bool   // 是否已接收到终端尺寸信息
}

// NewModel 创建并初始化 TUI Model。
// 若 dir 为空字符串，则进入路径输入模式供用户手动输入。
// 可选参数 initialInput 可传入命令行指定但校验失败的原始路径与错误提示。
func NewModel(dir string, initialInput ...string) Model {
	// 读取用户持久化的首选项配置
	cfg := config.Load()
	themeIdx := cfg.ThemeIndex
	if themeIdx < 0 || themeIdx >= len(theme.Themes) {
		themeIdx = 0
	}
	sortM := SortMode(cfg.SortMode)
	if sortM < 0 || sortM >= sortModeCount {
		sortM = SortNameAsc
	}

	// 初始化搜索输入框
	si := textinput.New()
	si.Placeholder = "输入关键词过滤插件（支持空格分隔多词检索）..."
	si.CharLimit = 100

	// 初始化重命名输入框
	ri := textinput.New()
	ri.Placeholder = "输入新文件名..."
	ri.CharLimit = 200

	// 初始化路径输入框
	pi := textinput.New()
	pi.Placeholder = "例如: /srv/minecraft/plugins 或 C:\\Server\\plugins"
	pi.CharLimit = 500
	pi.Width = 60

	m := Model{
		themeIndex:  themeIdx,
		sortMode:    sortM,
		marked:      make(map[string]bool),
		searchInput: si,
		renameInput: ri,
		pathInput:   pi,
		termWidth:   80, // 默认值，会被 WindowSizeMsg 更新
		termHeight:  24, // 默认值，会被 WindowSizeMsg 更新
	}

	if dir == "" {
		// 未检测到有效路径 → 进入路径输入模式
		m.mode = ModePathInput
		m.pathInput.Focus()
		if len(initialInput) > 0 && initialInput[0] != "" {
			m.pathInput.SetValue(initialInput[0])
			m.statusMsg = "[错误] 指定的目录不存在或无法访问，请核对并按回车确认"
		}
	} else {
		// 有有效路径 → 直接扫描加载插件列表
		m.pluginDir = dir
		m.mode = ModeNormal
		m = m.loadPlugins()
	}

	return m
}

// PluginUpdateMsg 异步检查返回的插件更新消息
type PluginUpdateMsg struct {
	FilePath      string
	SpigotID      string
	UpdateVersion string
}

// checkUpdatesCmd 为当前插件列表中所有具备 SpigotID 的插件创建异步更新检测任务
func (m Model) checkUpdatesCmd() tea.Cmd {
	var cmds []tea.Cmd
	for _, p := range m.plugins {
		if p.SpigotID != "" {
			item := p
			cmds = append(cmds, func() tea.Msg {
				newVer, hasUpdate := manager.CheckSpigotUpdate(item.Website, item.Version)
				if hasUpdate && newVer != "" {
					return PluginUpdateMsg{
						FilePath:      item.FilePath,
						SpigotID:      item.SpigotID,
						UpdateVersion: newVer,
					}
				}
				return nil
			})
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// Init 实现 tea.Model 接口，返回初始化命令（启动光标闪烁与后台异步更新检测）
func (m Model) Init() tea.Cmd {
	if m.mode == ModePathInput {
		return textinput.Blink
	}
	return m.checkUpdatesCmd()
}

// saveConfig 保存当前配置到磁盘
func (m *Model) saveConfig() {
	_ = config.Save(config.Config{
		ThemeIndex: m.themeIndex,
		SortMode:   int(m.sortMode),
	})
}

// pushUndo 向历史栈压入操作记录，最多保留 50 条
func (m *Model) pushUndo(action UndoAction) {
	if len(action.Items) == 0 {
		return
	}
	m.undoStack = append(m.undoStack, action)
	if len(m.undoStack) > 50 {
		m.undoStack = m.undoStack[len(m.undoStack)-50:]
	}
}

// popUndo 弹出最近一条操作记录
func (m *Model) popUndo() (UndoAction, bool) {
	if len(m.undoStack) == 0 {
		return UndoAction{}, false
	}
	idx := len(m.undoStack) - 1
	action := m.undoStack[idx]
	m.undoStack = m.undoStack[:idx]
	return action, true
}

// toggleMark 切换指定插件的多选标记
func (m *Model) toggleMark(filePath string) {
	if m.marked == nil {
		m.marked = make(map[string]bool)
	}
	if m.marked[filePath] {
		delete(m.marked, filePath)
	} else {
		m.marked[filePath] = true
	}
}

// clearMarks 清除所有多选标记
func (m *Model) clearMarks() {
	m.marked = make(map[string]bool)
}

// isMarked 检查插件是否被标记
func (m Model) isMarked(filePath string) bool {
	return m.marked != nil && m.marked[filePath]
}

// markedCount 返回当前被标记的插件数量
func (m Model) markedCount() int {
	return len(m.marked)
}

// loadPlugins 扫描插件目录并刷新内存中的插件列表。
// 若 statusMsg 已有内容（如重命名成功提示），则不予覆盖。
func (m Model) loadPlugins() Model {
	plugins, err := manager.ScanPlugins(m.pluginDir)
	if err != nil {
		m.statusMsg = "[错误] 扫描失败: " + err.Error()
		m.plugins = nil
		m.filtered = nil
		return m
	}

	// 保留此前已获取的更新状态，防止重命名或局部状态刷新后角标丢失
	oldUpdates := make(map[string]string)
	for _, oldP := range m.plugins {
		if oldP.UpdateVersion != "" {
			if oldP.SpigotID != "" {
				oldUpdates[oldP.SpigotID] = oldP.UpdateVersion
			}
			oldUpdates[oldP.FilePath] = oldP.UpdateVersion
		}
	}
	if len(oldUpdates) > 0 {
		for i := range plugins {
			if ver, ok := oldUpdates[plugins[i].SpigotID]; ok && ver != "" {
				plugins[i].UpdateVersion = ver
			} else if ver, ok := oldUpdates[plugins[i].FilePath]; ok && ver != "" {
				plugins[i].UpdateVersion = ver
			}
		}
	}

	m.plugins = plugins
	m.sortPlugins()
	m.resetFilter()

	// 仅在无自定义反馈时显示加载提示
	if m.statusMsg == "" {
		m.statusMsg = fmt.Sprintf("已加载 %d 个插件", len(plugins))
	}
	return m
}

// sortPlugins 根据当前 sortMode 对插件列表进行排序
func (m *Model) sortPlugins() {
	switch m.sortMode {
	case SortNameAsc:
		sort.Slice(m.plugins, func(i, j int) bool {
			return strings.ToLower(m.plugins[i].Name) < strings.ToLower(m.plugins[j].Name)
		})
	case SortNameDesc:
		sort.Slice(m.plugins, func(i, j int) bool {
			return strings.ToLower(m.plugins[i].Name) > strings.ToLower(m.plugins[j].Name)
		})
	case SortSizeAsc:
		sort.Slice(m.plugins, func(i, j int) bool {
			return m.plugins[i].FileSizeBytes < m.plugins[j].FileSizeBytes
		})
	case SortSizeDesc:
		sort.Slice(m.plugins, func(i, j int) bool {
			return m.plugins[i].FileSizeBytes > m.plugins[j].FileSizeBytes
		})
	case SortStatusOn:
		sort.SliceStable(m.plugins, func(i, j int) bool {
			if m.plugins[i].Enabled != m.plugins[j].Enabled {
				return m.plugins[i].Enabled // 启用的排前面
			}
			return strings.ToLower(m.plugins[i].Name) < strings.ToLower(m.plugins[j].Name)
		})
	case SortStatusOff:
		sort.SliceStable(m.plugins, func(i, j int) bool {
			if m.plugins[i].Enabled != m.plugins[j].Enabled {
				return !m.plugins[i].Enabled // 禁用的排前面
			}
			return strings.ToLower(m.plugins[i].Name) < strings.ToLower(m.plugins[j].Name)
		})
	}
}

// resetFilter 重置过滤列表为全量显示
func (m *Model) resetFilter() {
	m.filtered = make([]int, len(m.plugins))
	for i := range m.plugins {
		m.filtered[i] = i
	}
	if m.cursor >= len(m.filtered) {
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
		} else {
			m.cursor = 0
		}
	}
	m.viewTop = 0
}

// applyFilter 根据搜索关键词过滤插件列表（支持空格分隔多词，匹配文件名、真实名、作者、描述与版本）
func (m *Model) applyFilter(keyword string) {
	words := strings.Fields(strings.ToLower(keyword))
	if len(words) == 0 {
		m.resetFilter()
		return
	}

	m.filtered = nil
	for i, p := range m.plugins {
		nameLower := strings.ToLower(p.Name)
		rawNameLower := strings.ToLower(p.RawName)
		authorLower := strings.ToLower(p.Author)
		descLower := strings.ToLower(p.Description)
		verLower := strings.ToLower(p.Version)

		allMatch := true
		for _, w := range words {
			if !strings.Contains(nameLower, w) &&
				!strings.Contains(rawNameLower, w) &&
				!strings.Contains(authorLower, w) &&
				!strings.Contains(descLower, w) &&
				!strings.Contains(verLower, w) {
				allMatch = false
				break
			}
		}

		if allMatch {
			m.filtered = append(m.filtered, i)
		}
	}

	// 确保光标不越界
	if m.cursor >= len(m.filtered) {
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
		} else {
			m.cursor = 0
		}
	}
	m.viewTop = 0
}

// selectedIndex 获取当前光标所选插件在 plugins 切片中的索引。
// 若列表为空或光标越界则返回 -1。
func (m Model) selectedIndex() int {
	if len(m.filtered) == 0 || m.cursor < 0 || m.cursor >= len(m.filtered) {
		return -1
	}
	return m.filtered[m.cursor]
}

// visibleRows 计算当前终端尺寸下可显示的插件行数。
// 严密核算 Header 与 Footer 的实际占用行数，杜绝终端向上滚动抖动。
func (m Model) visibleRows() int {
	if m.termWidth <= 0 || m.termHeight <= 0 {
		return 1
	}

	// Header 行数：
	// 空行(1) + 标题(1) + 路径(1) + 统计行(1) + 分隔线(1) = 5
	headerLines := 5
	if m.termWidth < 65 {
		headerLines += 1 // 较窄时统计行折行
	}

	// Footer 行数：
	// 分隔线(1) + 帮助/输入提示(1) + 状态行(1) + 末尾换行(1) = 4
	footerLines := 4
	if m.mode == ModeSearch || m.mode == ModeRename {
		footerLines += 1 // 搜索/重命名额外增加输入行
	}
	if m.termWidth < 80 {
		footerLines += 1 // 较窄时帮助栏折行
	}

	overhead := headerLines + footerLines
	rows := m.termHeight - overhead
	if rows < 1 {
		rows = 1
	}
	return rows
}

// ensureCursorVisible 调整视窗位置以确保光标始终在可视范围内
func (m *Model) ensureCursorVisible() {
	visible := m.visibleRows()
	if m.cursor < m.viewTop {
		m.viewTop = m.cursor
	}
	if m.cursor >= m.viewTop+visible {
		m.viewTop = m.cursor - visible + 1
	}
	if m.viewTop < 0 {
		m.viewTop = 0
	}
}

// countEnabled 统计全量插件列表中启用状态的插件数量
func (m Model) countEnabled() int {
	count := 0
	for _, p := range m.plugins {
		if p.Enabled {
			count++
		}
	}
	return count
}
