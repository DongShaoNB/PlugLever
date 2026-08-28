package tui

import (
	"fmt"
	"sort"
	"strings"

	"PlugLever/internal/manager"

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
func NewModel(dir string) Model {
	// 初始化搜索输入框
	si := textinput.New()
	si.Placeholder = "输入关键词过滤插件..."
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
		themeIndex:  0,
		searchInput: si,
		renameInput: ri,
		pathInput:   pi,
		termWidth:   80,  // 默认值，会被 WindowSizeMsg 更新
		termHeight:  24,  // 默认值，会被 WindowSizeMsg 更新
	}

	if dir == "" {
		// 未检测到有效路径 → 进入路径输入模式
		m.mode = ModePathInput
		m.pathInput.Focus()
	} else {
		// 有有效路径 → 直接扫描加载插件列表
		m.pluginDir = dir
		m.mode = ModeNormal
		m = m.loadPlugins()
	}

	return m
}

// Init 实现 tea.Model 接口，返回初始化命令（启动光标闪烁等）
func (m Model) Init() tea.Cmd {
	if m.mode == ModePathInput {
		return textinput.Blink
	}
	return nil
}

// loadPlugins 扫描插件目录并刷新内存中的插件列表。
// 使用值接收者模式：返回修改后的 Model 副本。
func (m Model) loadPlugins() Model {
	plugins, err := manager.ScanPlugins(m.pluginDir)
	if err != nil {
		m.statusMsg = "[错误] 扫描失败: " + err.Error()
		m.plugins = nil
		m.filtered = nil
		return m
	}

	m.plugins = plugins
	m.sortPlugins()
	m.resetFilter()
	m.statusMsg = fmt.Sprintf("已加载 %d 个插件", len(plugins))
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

// resetFilter 重置过滤列表为全量显示（退出搜索或刷新后调用）
func (m *Model) resetFilter() {
	m.filtered = make([]int, len(m.plugins))
	for i := range m.plugins {
		m.filtered[i] = i
	}
	m.cursor = 0
	m.viewTop = 0
}

// applyFilter 根据搜索关键词过滤插件列表（匹配名称和作者）
func (m *Model) applyFilter(keyword string) {
	if keyword == "" {
		m.resetFilter()
		return
	}

	keyword = strings.ToLower(keyword)
	m.filtered = nil
	for i, p := range m.plugins {
		nameLower := strings.ToLower(p.Name)
		authorLower := strings.ToLower(p.Author)
		if strings.Contains(nameLower, keyword) || strings.Contains(authorLower, keyword) {
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
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return -1
	}
	return m.filtered[m.cursor]
}

// visibleRows 计算当前终端尺寸下可显示的插件行数。
// 动态估算 Header/Footer 行数，考虑窄终端下文本换行的影响。
func (m Model) visibleRows() int {
	if m.termWidth <= 0 || m.termHeight <= 0 {
		return 1
	}

	// Header 固定行：空行(1) + 标题(1) + 路径(1) + 分隔线(1) = 4
	headerFixed := 4
	// 统计行可能换行：约 60 个可见字符宽度
	statsLines := ceilDiv(60, m.termWidth)

	// Footer 固定行：分隔线(1) + 状态消息(1) = 2
	footerFixed := 2
	// 帮助文本可能换行：根据当前模式选择对应文本
	helpLen := 100 // helpNormal 约 100 个可见字符
	if m.mode == ModeSearch {
		helpLen = 40
	} else if m.mode == ModeRename {
		helpLen = 45
	}
	helpLines := ceilDiv(helpLen, m.termWidth)

	// 搜索/重命名模式额外增加输入行
	extraLines := 0
	if m.mode == ModeSearch || m.mode == ModeRename {
		extraLines = 1
	}

	overhead := headerFixed + statsLines + footerFixed + helpLines + extraLines
	rows := m.termHeight - overhead
	if rows < 1 {
		rows = 1
	}
	return rows
}

// ceilDiv 向上取整除法
func ceilDiv(a, b int) int {
	if b <= 0 {
		return 1
	}
	return (a + b - 1) / b
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
