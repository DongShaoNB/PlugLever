# PlugLever

PlugLever 是一个专为 Minecraft 服主打造的终端（TUI）插件开关管理工具

在调试服务端时，服主通常需要频繁通过重命名文件后缀（如 `.jar` ↔ `.jar.disabled`）来排查插件冲突或临时停用插件。PlugLever 通过终端交互界面接管这一过程，提供单选切换、批量开关、实时搜索、元数据读取与安全重命名等功能，无需在文件管理器或终端命令间来回切换

---

## 开发说明

本项目使用 **Claude Opus 4.6** 开发

---

## 核心特性

- **状态管理**：通过重命名 `.jar` ↔ `.jar.disabled` 实现插件启用与禁用，支持单个切换与批量全部启用/禁用
- **元数据自动解析**：直接读取 `.jar` 内的 `plugin.yml`，在列表中直观展示插件版本号与作者信息
- **智能路径探测**：
  - 支持命令行参数直接指定插件路径：`pluglever /path/to/plugins`
  - 若在服务端根目录运行，自动识别 `./plugins` 目录
  - 若在 `plugins` 目录内运行，直接识别当前目录
  - 若未检测到有效目录，界面内提供交互式路径输入
- **实时搜索过滤**：支持按插件名称、作者进行实时模糊检索
- **安全重命名**：快捷键直接修改文件名，系统自动保留状态后缀并检查重名冲突，避免误覆盖
- **多种排序模式**：支持按名称正倒序、文件大小、启用优先、禁用优先多种规则排序
- **内置多主题**：内置 Default Dark、Catppuccin Mocha、Catppuccin Latte 主题，按键即时切换

---

## 下载与安装

### 1. 从 GitHub Releases 下载（推荐）

前往 [GitHub Releases](https://github.com/DongShaoNB/PlugLever/releases) 下载对应操作系统的最新稳定版二进制文件

### 2. 从 GitHub Actions 下载构建版（尝鲜 / 不稳定）

如果想体验最新的未发布功能或测试修复：

1. 访问项目的 [GitHub Actions](https://github.com/DongShaoNB/PlugLever/actions/workflows/build.yml) 页面。
2. 点击最近一次成功的 Workflow 运行记录
3. 在页面底部的 **Artifacts** 区域下载对应的系统构建包

> [!WARNING]
> GitHub Actions 构建版可能包含未完全验证的代码或潜在 Bug，建议优先使用 Releases 稳定版

---

## 快捷键说明

| 按键              | 功能说明                                                           |
| :---------------- | :----------------------------------------------------------------- |
| `↑` / `k`         | 光标向上移动                                                       |
| `↓` / `j`         | 光标向下移动                                                       |
| `Space` (空格)    | 切换当前选中插件的状态（启用 / 禁用）                              |
| `Ctrl + A`        | 批量**启用**所有插件                                               |
| `Ctrl + D`        | 批量**禁用**所有插件                                               |
| `S`               | 循环切换排序模式（名称 A-Z / Z-A、文件大小、启用优先、禁用优先）   |
| `/` 或 `Ctrl + F` | 进入搜索过滤模式（输入关键词过滤，`Enter` 确认，`Esc` 退出并还原） |
| `N`               | 重命名当前选中的插件（自动保留后缀）                               |
| `T`               | 循环切换界面主题                                                   |
| `R`               | 手动重新扫描并刷新插件列表                                         |
| `Q` 或 `Ctrl + C` | 退出程序                                                           |

---

## 从源码构建

本项目采用 Go 语言编写，依赖 Bubbletea TUI 框架

### 前置要求

- 安装 [Go](https://go.dev/dl/)（推荐 1.22 及以上版本）
- 安装 Git

### 构建步骤

```bash
# 1. 克隆代码仓库
git clone https://github.com/DongShaoNB/PlugLever.git
cd PlugLever

# 2. 下载依赖
go mod download

# 3. 编译二进制文件
# Windows
go build -o pluglever.exe ./cmd/pluglever

# Linux / macOS
go build -o pluglever ./cmd/pluglever
```

### 交叉编译示例

```bash
# Linux (amd64)
GOOS=linux GOARCH=amd64 go build -o pluglever-linux-amd64 ./cmd/pluglever

# Windows (amd64)
GOOS=windows GOARCH=amd64 go build -o pluglever-windows-amd64.exe ./cmd/pluglever

# macOS (Apple Silicon / arm64)
GOOS=darwin GOARCH=arm64 go build -o pluglever-darwin-arm64 ./cmd/pluglever
```

---

## 使用方式

在 Windows 使用可以直接将文件放在服务端根目录或 plugins 文件夹内双击运行

```bash
# 方式 1：进入 Minecraft 服务端根目录或 plugins 目录直接运行
./pluglever

# 方式 2：在任意位置指定插件目录路径运行
./pluglever /home/mcserver/leaf/plugins
# 或 Windows 路径
.\pluglever.exe "F:\MCServer\Leaf\plugins"
```

---

## 问题反馈与交流

- **Bug 反馈 & 功能建议**：欢迎提交 [GitHub Issues](https://github.com/DongShaoNB/PlugLever/issues)
- **QQ 交流群**：`159323818`
