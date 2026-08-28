// Package tui 实现基于 Bubbletea 的终端交互界面。
package tui

// 各交互模式下底部帮助栏的提示文本

// helpNormal 普通浏览模式的快捷键说明
const helpNormal = "↑↓/jk 移动 │ Space 切换 │ Ctrl+A 全启用 │ Ctrl+D 全禁用 │ S 排序 │ / 搜索 │ N 重命名 │ T 主题 │ R 刷新 │ Q 退出"

// helpSearch 搜索过滤模式的帮助文本
const helpSearch = "输入关键词实时过滤 │ Enter 确认 │ Esc 退出搜索"

// helpRename 重命名模式的帮助文本
const helpRename = "输入新文件名（自动保留后缀）│ Enter 确认 │ Esc 取消"

// helpPathInput 路径输入模式的帮助文本
const helpPathInput = "输入插件目录的完整路径 │ Enter 确认 │ Ctrl+C 退出"
