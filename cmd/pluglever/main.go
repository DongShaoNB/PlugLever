// PlugLever - Minecraft 插件开关管理 TUI 工具
//
// 用法:
//
//	pluglever              # 自动探测 plugins 目录
//	pluglever [path]       # 指定插件目录路径
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"PlugLever/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// 执行路径自适应探测
	dir := detectPluginDir()

	// 创建 TUI Model（若 dir 为空则进入路径输入模式）
	model := tui.NewModel(dir)

	// 启动 Bubbletea 程序（全屏模式）
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

// detectPluginDir 实现路径自适应探测机制：
//  1. 优先使用命令行传入的路径参数
//  2. 若当前目录名为 "plugins"，以当前目录为目标
//  3. 若当前目录下存在 "plugins" 子目录，以其为目标
//  4. 均未匹配则返回空字符串，触发 TUI 路径输入模式
func detectPluginDir() string {
	// 1. 检查命令行参数
	if len(os.Args) > 1 {
		path := os.Args[1]
		absPath, err := filepath.Abs(path)
		if err != nil {
			return path // 降级使用原始路径
		}
		return absPath
	}

	// 获取当前工作目录
	cwd, err := os.Getwd()
	if err != nil {
		return "" // 无法获取工作目录，进入路径输入模式
	}

	// 2. 当前目录名为 "plugins" → 直接使用当前目录
	if filepath.Base(cwd) == "plugins" {
		return cwd
	}

	// 3. 当前目录下存在 "plugins" 子目录 → 使用该子目录
	pluginsDir := filepath.Join(cwd, "plugins")
	if info, err := os.Stat(pluginsDir); err == nil && info.IsDir() {
		return pluginsDir
	}

	// 4. 均未匹配 → 返回空字符串，TUI 将进入路径输入模式
	return ""
}
