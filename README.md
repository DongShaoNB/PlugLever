# PlugLever

PlugLever 是一个专为 Minecraft 服主打造的终端（TUI）插件开关管理工具

在调试服务端时，服主通常需要频繁通过重命名文件后缀（如 `.jar` ↔ `.jar.disabled`）来排查插件冲突或临时停用插件。PlugLever 通过终端交互界面接管这一过程，提供单选切换、批量开关、实时搜索、元数据读取与安全重命名等功能，无需在文件管理器或终端命令间来回切换

---

## 开发说明

本项目使用 **Claude Opus 4.6 + Gemini 3.8 Flash** 开发

---

## 核心特性

- **状态管理 & 大小写安全防冲突**：通过重命名 `.jar` ↔ `.jar.disabled` 实现插件秒级开关，内置同名冲突检查与大小写安全中转，杜绝覆盖。
- **强弱依赖解析与双向预警**：自动解析 `depend` 强依赖与 `softdepend` 软依赖。禁用被依赖项或启用缺失依赖项时，状态栏主动给出双向预警。
- **多平台元数据适配**：全面支持 Bukkit / Spigot / Paper (`plugin.yml`)、现代 Paper 1.19.3+ (`paper-plugin.yml`)、BungeeCord (`bungee.yml`) 及 Velocity (`velocity-plugin.json`)。
- **智能路径探测与错误回填**：
  - 支持命令行参数直接指定插件路径：`pluglever /path/to/plugins`（路径无效时自动回填并提示）
  - 忽略大小写自动识别服务端根目录 `./plugins` 或当前目录
  - 未检测到有效目录时进入交互式路径输入模式
- **插件详情模态框 (Inspector)**：快捷键直接弹出浮层，完整查看插件注册名、版本、作者、强依赖与软依赖项、主页、描述、修改时间与绝对路径。
- **多选标记与批量操作**：支持按键标记多选插件集合，支持 `a` 键一键全选/反选，一键批量切换/启用/禁用。
- **撤销机制 (Undo)**：操作记录自动入栈，误触或批量操作支持快捷键一键撤销。
- **多词组合搜索过滤**：支持按空格分隔多关键词进行实时检索（匹配文件名、真实注册名、作者、描述或版本），支持在搜索状态下同步移动光标。
- **SpigotMC 更新异步检测**：自动提取 `plugin.yml` 中的 Spigot 资源地址，后台静默异步查询官方 Update API，发现新版本时在列表与详情浮层以醒目角标标出（如 `[v1.7.2 ➜ v1.7.3]`），全程无感且遇错静默跳过。
- **优雅对齐与防闪烁**：基于字符视觉显示宽度（runewidth）的表格列对齐与平滑截断，中英文混合排版不折行、不抖动。
- **多主题与偏好持久化**：内置 Default Dark、Catppuccin Mocha、Catppuccin Latte 主题，配置自动保存。

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

| 按键                        | 功能说明                                                                 |
| :-------------------------- | :----------------------------------------------------------------------- |
| `↑` / `k`                   | 光标向上移动单行                                                         |
| `↓` / `j`                   | 光标向下移动单行                                                         |
| `Home` / `End` 或 `g` / `G` | 快速跳至列表首行 / 末行                                                  |
| `PageUp` / `PageDown`       | 列表向上 / 向下翻页                                                      |
| `Space` (空格)              | 切换当前插件状态（若存在多选标记，则批量切换所有已标记项）               |
| `v` 或 `x`                  | 切换当前插件的多选标记（`[✓]`）                                          |
| `a` 或 `A`                  | 快速**全选 / 取消全选**当前过滤列表的多选标记                            |
| `Ctrl + A`                  | 批量**启用**插件（存在标记项时仅启用标记插件，否则全量启用）             |
| `Ctrl + D`                  | 批量**禁用**插件（存在标记项时仅禁用标记插件，否则全量禁用）             |
| `u` 或 `Ctrl + Z`           | **撤销**上一步状态修改、重命名或批量操作                                 |
| `i` 或 `Enter`              | 查看当前选中插件的**详细元数据、强依赖与软依赖弹窗**                     |
| `/` 或 `Ctrl + F`           | 进入实时搜索过滤（支持空格多词分词，`↑↓` 同步导航预览，`Esc` 清除）      |
| `Esc`                       | 清除多选标记 / 清除当前搜索过滤 / 关闭当前打开的弹窗                     |
| `N`                         | 重命名当前选中的插件（自动清洗非法字符并保留对应后缀）                   |
| `S`                         | 循环切换排序模式（名称 A-Z / Z-A、文件大小、启用优先、禁用优先）         |
| `T`                         | 循环切换界面配色主题并自动保存偏好                                       |
| `R`                         | 重新并发扫描并刷新插件列表                                               |
| `?`                         | 打开快捷键完整帮助指南弹窗                                               |
| `Q` 或 `Ctrl + C`           | 退出程序                                                                 |

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
