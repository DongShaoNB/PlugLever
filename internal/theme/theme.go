// Package theme 定义 TUI 界面的配色方案与 Lipgloss 动态样式生成。
package theme

import "github.com/charmbracelet/lipgloss"

// Theme 定义了 TUI 界面的完整配色方案
type Theme struct {
	Name            string // 主题名称
	BgPrimary       string // 主背景色
	FgPrimary       string // 主前景色（正文文字）
	FgSecondary     string // 次要前景色（元数据、辅助文字）
	Accent          string // 强调色（标题、高亮）
	OnColor         string // 启用状态颜色（绿色系）
	OffColor        string // 禁用状态颜色（灰色系）
	CursorColor     string // 光标行背景高亮色
	HelpText        string // 帮助栏文字色
	ErrorColor      string // 错误消息色
	SearchHighlight string // 搜索匹配高亮色
	Border          string // 边框/分隔线颜色
}

// ---- 预设主题 ----

// DefaultDark 经典现代化终端黑灰高亮风格（强调对比与清晰度）
var DefaultDark = Theme{
	Name:            "Default Dark",
	BgPrimary:       "#1a1a2e",
	FgPrimary:       "#e0e0e0",
	FgSecondary:     "#888888",
	Accent:          "#00d4aa",
	OnColor:         "#00ff87",
	OffColor:        "#666666",
	CursorColor:     "#2a2a4e",
	HelpText:        "#777777",
	ErrorColor:      "#ff6b6b",
	SearchHighlight: "#ffdd57",
	Border:          "#444444",
}

// CatppuccinMocha 深色低饱和温和配色（Catppuccin 社区配色方案）
var CatppuccinMocha = Theme{
	Name:            "Catppuccin Mocha",
	BgPrimary:       "#1e1e2e",
	FgPrimary:       "#cdd6f4",
	FgSecondary:     "#a6adc8",
	Accent:          "#cba6f7",
	OnColor:         "#a6e3a1",
	OffColor:        "#6c7086",
	CursorColor:     "#313244",
	HelpText:        "#7f849c",
	ErrorColor:      "#f38ba8",
	SearchHighlight: "#f9e2af",
	Border:          "#45475a",
}

// CatppuccinLatte 高对比浅色配色（适配浅色终端背景）
var CatppuccinLatte = Theme{
	Name:            "Catppuccin Latte",
	BgPrimary:       "#eff1f5",
	FgPrimary:       "#4c4f69",
	FgSecondary:     "#6c6f85",
	Accent:          "#8839ef",
	OnColor:         "#40a02b",
	OffColor:        "#9ca0b0",
	CursorColor:     "#e6e9ef",
	HelpText:        "#8c8fa1",
	ErrorColor:      "#d20f39",
	SearchHighlight: "#df8e1d",
	Border:          "#bcc0cc",
}

// Themes 全局预设主题列表，按 T 键循环切换
var Themes = []Theme{
	DefaultDark,
	CatppuccinMocha,
	CatppuccinLatte,
}

// NextThemeIndex 返回下一个主题的索引（循环轮换）
func NextThemeIndex(current int) int {
	return (current + 1) % len(Themes)
}

// ---- Lipgloss 动态样式生成 ----

// TitleStyle 标题样式（强调色 + 粗体）
func TitleStyle(t Theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Accent)).
		Bold(true)
}

// StatusOnStyle 启用状态 [ON] 的样式（绿色 + 粗体）
func StatusOnStyle(t Theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.OnColor)).
		Bold(true)
}

// StatusOffStyle 禁用状态 [OFF] 的样式（红色）
func StatusOffStyle(t Theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.ErrorColor))
}

// NormalTextStyle 普通正文样式
func NormalTextStyle(t Theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.FgPrimary))
}

// SecondaryStyle 次要文字样式（元数据、路径、文件大小等）
func SecondaryStyle(t Theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.FgSecondary))
}

// HelpStyle 帮助栏样式
func HelpStyle(t Theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.HelpText))
}

// ErrorStyle 错误消息样式（红色 + 粗体）
func ErrorStyle(t Theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.ErrorColor)).
		Bold(true)
}

// AccentStyle 强调色样式（用于状态消息等）
func AccentStyle(t Theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Accent))
}

// BorderStyle 分隔线/边框样式
func BorderStyle(t Theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Border))
}

// SearchHighlightStyle 搜索高亮与高警示文字样式
func SearchHighlightStyle(t Theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.SearchHighlight)).
		Bold(true)
}

// WarningStyle 警告消息样式（黄色/高亮 + 粗体）
func WarningStyle(t Theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.SearchHighlight)).
		Bold(true)
}

// CursorLineStyle 光标所在行的高亮背景样式
func CursorLineStyle(t Theme, width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(t.CursorColor)).
		Width(width)
}

