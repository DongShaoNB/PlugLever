// PlugLever - Minecraft 插件开关管理 TUI 工具
//
// 用法:
//
//	pluglever              # 自动探测 plugins 目录
//	pluglever [path]       # 指定插件目录路径
//	pluglever -v           # 显示版本信息
//	pluglever -h           # 显示帮助信息
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"PlugLever/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

var version = "1.0.0"

func printUsage() {
	fmt.Printf("PlugLever v%s - 专为 Minecraft 服主打造的插件开关管理 TUI 工具\n\n", version)
	fmt.Println("用法:")
	fmt.Println("  pluglever              自动探测当前或子目录的 plugins 目录")
	fmt.Println("  pluglever <路径>        指定插件目录路径并直接启动")
	fmt.Println("  pluglever -v, --version 显示当前程序版本")
	fmt.Println("  pluglever -h, --help    显示帮助信息")
	fmt.Println()
	fmt.Println("常用快捷键:")
	fmt.Println("  ↑ / ↓ / j / k          光标上下移动")
	fmt.Println("  Space (空格)           切换插件启用/禁用状态")
	fmt.Println("  v / x                  切换多选标记")
	fmt.Println("  Ctrl + A / Ctrl + D    批量启用 / 批量禁用")
	fmt.Println("  u / Ctrl + Z           撤销上一步操作")
	fmt.Println("  i / Enter              查看选中插件元数据与依赖详情")
	fmt.Println("  / 或 Ctrl + F          关键词实时搜索过滤")
	fmt.Println("  N                      安全重命名")
	fmt.Println("  ?                      打开快捷键完整帮助面板")
	fmt.Println("  Q 或 Ctrl + C          退出程序")
}

func main() {
	var showVer bool
	var showHelp bool

	flag.BoolVar(&showVer, "v", false, "显示版本信息")
	flag.BoolVar(&showVer, "version", false, "显示版本信息")
	flag.BoolVar(&showHelp, "h", false, "显示帮助信息")
	flag.BoolVar(&showHelp, "help", false, "显示帮助信息")

	flag.Usage = printUsage
	flag.Parse()

	if showVer {
		fmt.Printf("PlugLever v%s\n", version)
		os.Exit(0)
	}

	if showHelp {
		printUsage()
		os.Exit(0)
	}

	// 执行路径自适应探测
	args := flag.Args()
	dir, invalidArg := detectPluginDir(args)

	// 创建 TUI Model（若 dir 为空则进入路径输入模式，若命令行指定了非法路径则回填）
	var model tui.Model
	if invalidArg != "" {
		model = tui.NewModel("", invalidArg)
	} else {
		model = tui.NewModel(dir)
	}

	// 启动 Bubbletea 程序（全屏模式）
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

// detectPluginDir 实现路径自适应探测机制：
//  1. 优先使用命令行传入的路径参数
//  2. 若当前目录名为 "plugins"（忽略大小写），以当前目录为目标
//  3. 若当前目录下存在 "plugins" 子目录（忽略大小写），以其为目标
//  4. 均未匹配则返回空字符串，触发 TUI 路径输入模式
// 返回 (有效路径, 无效命令行参数)
func detectPluginDir(args []string) (string, string) {
	// 1. 检查命令行参数
	if len(args) > 0 {
		path := args[0]
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path
		}
		if info, err := os.Stat(absPath); err == nil && info.IsDir() {
			return absPath, ""
		}
		// 指定路径不存在或非目录，返回供 TUI 输入框回填提示
		return "", path
	}

	// 获取当前工作目录
	cwd, err := os.Getwd()
	if err != nil {
		return "", "" // 无法获取工作目录，进入路径输入模式
	}

	// 2. 当前目录名为 "plugins"（忽略大小写）→ 直接使用当前目录
	if strings.EqualFold(filepath.Base(cwd), "plugins") {
		return cwd, ""
	}

	// 3. 当前目录下存在 "plugins" 子目录（忽略大小写，兼容 Linux 下的大小写命名差异）
	if entries, err := os.ReadDir(cwd); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && strings.EqualFold(entry.Name(), "plugins") {
				return filepath.Join(cwd, entry.Name()), ""
			}
		}
	}

	// 4. 均未匹配 → 返回空字符串，TUI 将进入路径输入模式
	return "", ""
}
