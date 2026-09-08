// Package tui 实现基于 Bubbletea 的终端交互界面。
package tui

// 各交互模式下底部帮助栏的提示文本

// helpNormal 普通浏览模式的快捷键说明
const helpNormal = "↑↓/jk 移动 │ Space 切换 │ v 标记 │ a 全选 │ Ctrl+A 启用 │ Ctrl+D 禁用 │ S 排序 │ / 搜索 │ N 重命名 │ i 详情 │ u 撤销 │ ? 帮助 │ Q 退出"

// helpSearch 搜索过滤模式的帮助文本
const helpSearch = "↑↓ 预览 │ 输入关键词过滤 │ Enter 保留结果 │ Esc 清除退出"

// helpRename 重命名模式的帮助文本
const helpRename = "输入新文件名（自动保留后缀）│ Enter 确认 │ Esc 取消"

// helpPathInput 路径输入模式的帮助文本
const helpPathInput = "输入插件目录的完整路径 │ Enter 确认 │ Ctrl+C 退出"

// helpDetail 详情弹窗的帮助文本
const helpDetail = "Esc / Enter / i / q 关闭详情"

// helpHelp 帮助弹窗的帮助文本
const helpHelp = "Esc / Enter / ? / q 关闭帮助"
