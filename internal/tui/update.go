package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"PlugLever/internal/manager"
	"PlugLever/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

// Update 实现 tea.Model 接口，分发不同交互模式下的消息处理
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// 终端尺寸变更消息
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		m.ready = true
		m.ensureCursorVisible()
		return m, nil

	// 后台 SpigotMC 更新检测消息
	case PluginUpdateMsg:
		for i := range m.plugins {
			if m.plugins[i].FilePath == msg.FilePath || (m.plugins[i].SpigotID != "" && m.plugins[i].SpigotID == msg.SpigotID) {
				m.plugins[i].UpdateVersion = msg.UpdateVersion
				break
			}
		}
		return m, nil

	// 键盘事件消息
	case tea.KeyMsg:
		switch m.mode {
		case ModePathInput:
			return m.updatePathInput(msg)
		case ModeSearch:
			return m.updateSearch(msg)
		case ModeRename:
			return m.updateRename(msg)
		case ModeDetail:
			return m.updateDetail(msg)
		case ModeHelp:
			return m.updateHelp(msg)
		default:
			return m.updateNormal(msg)
		}
	}

	return m, nil
}

// updatePathInput 处理路径输入模式
func (m Model) updatePathInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "enter":
		path := strings.TrimSpace(m.pathInput.Value())
		if path == "" {
			m.statusMsg = "[错误] 路径不能为空"
			return m, nil
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			m.statusMsg = "[错误] 无效路径: " + err.Error()
			return m, nil
		}
		info, err := os.Stat(absPath)
		if err != nil || !info.IsDir() {
			m.statusMsg = "[错误] 目录不存在或无法访问"
			return m, nil
		}

		m.pluginDir = absPath
		m.statusMsg = ""
		m = m.loadPlugins()
		m.mode = ModeNormal
		return m, m.checkUpdatesCmd()

	default:
		var cmd tea.Cmd
		m.pathInput, cmd = m.pathInput.Update(msg)
		return m, cmd
	}
}

// updateNormal 处理普通浏览模式
func (m Model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {

	// ---- 退出程序 ----
	case "ctrl+c", "q", "Q":
		return m, tea.Quit

	// ---- 导航：上移 / 下移 ----
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.ensureCursorVisible()
		}
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.ensureCursorVisible()
		}

	// ---- 导航：跳至顶部 / 底部 ----
	case "home", "g":
		if len(m.filtered) > 0 {
			m.cursor = 0
			m.ensureCursorVisible()
		}
	case "end", "G":
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
			m.ensureCursorVisible()
		}

	// ---- 导航：翻页 ----
	case "pgup":
		m.cursor -= m.visibleRows()
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.ensureCursorVisible()
	case "pgdown":
		m.cursor += m.visibleRows()
		if m.cursor >= len(m.filtered) {
			m.cursor = len(m.filtered) - 1
		}
		m.ensureCursorVisible()

	// ---- 多选标记：v 或 x ----
	// ---- 多选标记：v 或 x ----
	case "v", "x":
		idx := m.selectedIndex()
		if idx >= 0 {
			m.toggleMark(m.plugins[idx].FilePath)
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				m.ensureCursorVisible()
			}
		}

	// ---- 全选 / 取消全选标记：a 或 A ----
	case "a", "A":
		if len(m.filtered) == 0 {
			return m, nil
		}
		allMarked := true
		for _, idx := range m.filtered {
			if !m.isMarked(m.plugins[idx].FilePath) {
				allMarked = false
				break
			}
		}

		if allMarked {
			for _, idx := range m.filtered {
				delete(m.marked, m.plugins[idx].FilePath)
			}
			m.statusMsg = "已取消当前列表的所有标记"
		} else {
			if m.marked == nil {
				m.marked = make(map[string]bool)
			}
			for _, idx := range m.filtered {
				m.marked[m.plugins[idx].FilePath] = true
			}
			m.statusMsg = fmt.Sprintf("已全选标记当前 %d 个插件", len(m.filtered))
		}

	// ---- 空格：切换状态（单选或标记批量） ----
	case " ":
		if m.markedCount() > 0 {
			// 若存在标记，批量切换所有被标记的插件
			var items []UndoItem
			var allWarns []string
			count := 0
			failCount := 0
			var lastErr error

			for i := range m.plugins {
				if m.isMarked(m.plugins[i].FilePath) {
					oldPath := m.plugins[i].FilePath
					wasEnabled := m.plugins[i].Enabled
					deps, missingDeps, err := manager.TogglePlugin(&m.plugins[i], m.plugins)
					if err == nil {
						items = append(items, UndoItem{OldPath: oldPath, NewPath: m.plugins[i].FilePath})
						count++
						if wasEnabled && len(deps) > 0 {
							allWarns = append(allWarns, fmt.Sprintf("%s(被 %s 依赖)", m.plugins[i].Name, strings.Join(deps, ", ")))
						}
						if !wasEnabled && len(missingDeps) > 0 {
							allWarns = append(allWarns, fmt.Sprintf("%s(缺依赖 %s)", m.plugins[i].Name, strings.Join(missingDeps, ", ")))
						}
					} else {
						failCount++
						lastErr = err
					}
				}
			}

			if count > 0 {
				m.pushUndo(UndoAction{
					Description: fmt.Sprintf("切换 %d 个标记插件状态", count),
					Items:       items,
				})
			}
			m.clearMarks()

			if failCount > 0 {
				m.statusMsg = fmt.Sprintf("已切换 %d 个插件，%d 个失败（%s）", count, failCount, lastErr.Error())
			} else if len(allWarns) > 0 {
				m.statusMsg = fmt.Sprintf("已切换 %d 个插件。⚠️ 依赖告警: %s", count, strings.Join(allWarns, " | "))
			} else {
				m.statusMsg = fmt.Sprintf("已成功切换 %d 个标记插件的状态", count)
			}
		} else {
			// 切换单个当前光标所选插件
			idx := m.selectedIndex()
			if idx >= 0 {
				p := &m.plugins[idx]
				oldPath := p.FilePath
				wasEnabled := p.Enabled
				deps, missingDeps, err := manager.TogglePlugin(p, m.plugins)
				if err != nil {
					m.statusMsg = "[错误] " + err.Error()
				} else {
					m.pushUndo(UndoAction{
						Description: fmt.Sprintf("切换插件 %s 状态", p.Name),
						Items:       []UndoItem{{OldPath: oldPath, NewPath: p.FilePath}},
					})

					if wasEnabled && len(deps) > 0 {
						m.statusMsg = fmt.Sprintf("已禁用 '%s' ⚠️ 依赖告警: 以下启用插件依赖它: %s", p.Name, strings.Join(deps, ", "))
					} else if !wasEnabled && len(missingDeps) > 0 {
						m.statusMsg = fmt.Sprintf("已启用 '%s' ⚠️ 依赖告警: 缺少或未启用强依赖: %s", p.Name, strings.Join(missingDeps, ", "))
					} else if p.Enabled {
						m.statusMsg = fmt.Sprintf("已启用插件: %s", p.Name)
					} else {
						m.statusMsg = fmt.Sprintf("已禁用插件: %s", p.Name)
					}
				}
			}
		}

	// ---- Ctrl+A：批量启用（支持标记集合或全量） ----
	case "ctrl+a":
		if m.markedCount() > 0 {
			var items []UndoItem
			var allWarns []string
			count := 0
			failCount := 0
			var lastErr error

			for i := range m.plugins {
				if m.isMarked(m.plugins[i].FilePath) && !m.plugins[i].Enabled {
					oldPath := m.plugins[i].FilePath
					_, missingDeps, err := manager.TogglePlugin(&m.plugins[i], m.plugins)
					if err == nil {
						items = append(items, UndoItem{OldPath: oldPath, NewPath: m.plugins[i].FilePath})
						count++
						if len(missingDeps) > 0 {
							allWarns = append(allWarns, fmt.Sprintf("%s(缺依赖 %s)", m.plugins[i].Name, strings.Join(missingDeps, ", ")))
						}
					} else {
						failCount++
						lastErr = err
					}
				}
			}
			m.clearMarks()
			if count > 0 {
				m.pushUndo(UndoAction{Description: fmt.Sprintf("启用 %d 个标记插件", count), Items: items})
			}

			if failCount > 0 {
				m.statusMsg = fmt.Sprintf("已启用 %d 个标记插件，%d 个失败（%s）", count, failCount, lastErr.Error())
			} else if count > 0 {
				if len(allWarns) > 0 {
					m.statusMsg = fmt.Sprintf("已成功启用 %d 个标记插件。⚠️ 依赖告警: %s", count, strings.Join(allWarns, " | "))
				} else {
					m.statusMsg = fmt.Sprintf("已成功启用 %d 个标记插件", count)
				}
			} else {
				m.statusMsg = "所选插件已处于启用状态"
			}
		} else {
			var items []UndoItem
			var allWarns []string
			count := 0
			failCount := 0
			var lastErr error

			for i := range m.plugins {
				if !m.plugins[i].Enabled {
					oldPath := m.plugins[i].FilePath
					_, missingDeps, err := manager.TogglePlugin(&m.plugins[i], m.plugins)
					if err == nil {
						items = append(items, UndoItem{OldPath: oldPath, NewPath: m.plugins[i].FilePath})
						count++
						if len(missingDeps) > 0 {
							allWarns = append(allWarns, fmt.Sprintf("%s(缺依赖 %s)", m.plugins[i].Name, strings.Join(missingDeps, ", ")))
						}
					} else {
						failCount++
						lastErr = err
					}
				}
			}
			if count > 0 {
				m.pushUndo(UndoAction{Description: fmt.Sprintf("全量启用 %d 个插件", count), Items: items})
			}

			if failCount > 0 {
				m.statusMsg = fmt.Sprintf("已批量启用 %d 个插件，%d 个失败（%s）", count, failCount, lastErr.Error())
			} else if count > 0 {
				if len(allWarns) > 0 {
					m.statusMsg = fmt.Sprintf("已批量启用 %d 个插件。⚠️ 依赖告警: %s", count, strings.Join(allWarns, " | "))
				} else {
					m.statusMsg = fmt.Sprintf("已批量启用 %d 个插件", count)
				}
			} else {
				m.statusMsg = "所有插件已全部处于启用状态"
			}
		}

	// ---- Ctrl+D：批量禁用（支持标记集合或全量） ----
	case "ctrl+d":
		if m.markedCount() > 0 {
			var items []UndoItem
			var allWarns []string
			count := 0
			failCount := 0
			var lastErr error

			for i := range m.plugins {
				if m.isMarked(m.plugins[i].FilePath) && m.plugins[i].Enabled {
					oldPath := m.plugins[i].FilePath
					deps, _, err := manager.TogglePlugin(&m.plugins[i], m.plugins)
					if err == nil {
						items = append(items, UndoItem{OldPath: oldPath, NewPath: m.plugins[i].FilePath})
						count++
						if len(deps) > 0 {
							allWarns = append(allWarns, fmt.Sprintf("%s(被 %s 依赖)", m.plugins[i].Name, strings.Join(deps, ", ")))
						}
					} else {
						failCount++
						lastErr = err
					}
				}
			}
			m.clearMarks()
			if count > 0 {
				m.pushUndo(UndoAction{Description: fmt.Sprintf("禁用 %d 个标记插件", count), Items: items})
			}

			if failCount > 0 {
				m.statusMsg = fmt.Sprintf("已禁用 %d 个标记插件，%d 个失败（%s）", count, failCount, lastErr.Error())
			} else if count > 0 {
				if len(allWarns) > 0 {
					m.statusMsg = fmt.Sprintf("已成功禁用 %d 个标记插件。⚠️ 依赖告警: %s", count, strings.Join(allWarns, " | "))
				} else {
					m.statusMsg = fmt.Sprintf("已成功禁用 %d 个标记插件", count)
				}
			} else {
				m.statusMsg = "所选插件已处于禁用状态"
			}
		} else {
			var items []UndoItem
			var allWarns []string
			count := 0
			failCount := 0
			var lastErr error

			for i := range m.plugins {
				if m.plugins[i].Enabled {
					oldPath := m.plugins[i].FilePath
					deps, _, err := manager.TogglePlugin(&m.plugins[i], m.plugins)
					if err == nil {
						items = append(items, UndoItem{OldPath: oldPath, NewPath: m.plugins[i].FilePath})
						count++
						if len(deps) > 0 {
							allWarns = append(allWarns, fmt.Sprintf("%s(被 %s 依赖)", m.plugins[i].Name, strings.Join(deps, ", ")))
						}
					} else {
						failCount++
						lastErr = err
					}
				}
			}
			if count > 0 {
				m.pushUndo(UndoAction{Description: fmt.Sprintf("全量禁用 %d 个插件", count), Items: items})
			}

			if failCount > 0 {
				m.statusMsg = fmt.Sprintf("已批量禁用 %d 个插件，%d 个失败（%s）", count, failCount, lastErr.Error())
			} else if count > 0 {
				if len(allWarns) > 0 {
					m.statusMsg = fmt.Sprintf("已批量禁用 %d 个插件。⚠️ 依赖告警: %s", count, strings.Join(allWarns, " | "))
				} else {
					m.statusMsg = fmt.Sprintf("已批量禁用 %d 个插件", count)
				}
			} else {
				m.statusMsg = "所有插件已全部处于禁用状态"
			}
		}

	// ---- 撤销：u 或 Ctrl+Z ----
	case "u", "ctrl+z":
		action, ok := m.popUndo()
		if !ok {
			m.statusMsg = "没有可撤销的操作"
			return m, nil
		}
		// 逆序还原路径
		restoredCount := 0
		failCount := 0
		for i := len(action.Items) - 1; i >= 0; i-- {
			item := action.Items[i]
			if err := os.Rename(item.NewPath, item.OldPath); err == nil {
				restoredCount++
			} else {
				failCount++
			}
		}
		if failCount > 0 {
			m.statusMsg = fmt.Sprintf("撤销完成：%d 项成功还原，%d 项失败", restoredCount, failCount)
		} else {
			m.statusMsg = fmt.Sprintf("已成功撤销：%s", action.Description)
		}
		m = m.loadPlugins()
		return m, m.checkUpdatesCmd()

	// ---- 查看详情：i 或 Enter ----
	case "i", "enter":
		if len(m.filtered) > 0 {
			m.mode = ModeDetail
		}

	// ---- 快捷键帮助菜单：? ----
	case "?":
		m.mode = ModeHelp

	// ---- 退出标记或清除搜索过滤：Esc ----
	case "esc":
		if m.markedCount() > 0 {
			m.clearMarks()
			m.statusMsg = "已取消所有选择标记"
		} else if len(m.filtered) != len(m.plugins) || m.searchInput.Value() != "" {
			m.searchInput.SetValue("")
			m.resetFilter()
			m.statusMsg = "已重置并还原全量插件列表"
		}

	// ---- 切换主题：T ----
	case "t", "T":
		m.themeIndex = theme.NextThemeIndex(m.themeIndex)
		m.saveConfig()
		m.statusMsg = "主题已切换: " + theme.Themes[m.themeIndex].Name

	// ---- 切换排序方式：S ----
	case "s", "S":
		m.sortMode = SortMode((int(m.sortMode) + 1) % int(sortModeCount))
		m.saveConfig()
		m.sortPlugins()
		m.resetFilter()
		m.statusMsg = "排序方式: " + m.sortMode.Label()

	// ---- 进入搜索模式：/ 或 Ctrl+F ----
	case "/", "ctrl+f":
		m.mode = ModeSearch
		m.statusMsg = ""
		cmd := m.searchInput.Focus()
		return m, cmd

	// ---- 进入重命名模式：N ----
	case "n", "N":
		idx := m.selectedIndex()
		if idx >= 0 {
			m.mode = ModeRename
			m.renameInput.SetValue(m.plugins[idx].Name)
			m.statusMsg = ""
			cmd := m.renameInput.Focus()
			return m, cmd
		} else {
			m.statusMsg = "[错误] 没有可操作的插件"
		}

	// ---- 手动刷新：R ----
	case "r", "R":
		m.statusMsg = ""
		m = m.loadPlugins()
		return m, m.checkUpdatesCmd()
	}

	return m, nil
}

// updateSearch 处理搜索过滤模式
func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	// Esc：退出搜索并清除过滤
	case "esc":
		m.mode = ModeNormal
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		m.resetFilter()
		m.statusMsg = "已退出搜索"
		return m, nil

	// Enter：退出搜索但保留过滤结果
	case "enter":
		m.mode = ModeNormal
		m.searchInput.Blur()
		m.statusMsg = fmt.Sprintf("已筛选出 %d 个插件 (按 Esc 清除过滤)", len(m.filtered))
		return m, nil

	// 上下键：支持在搜索输入时同步上下导航预览
	case "up":
		if m.cursor > 0 {
			m.cursor--
			m.ensureCursorVisible()
		}
		return m, nil
	case "down":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.ensureCursorVisible()
		}
		return m, nil

	default:
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.applyFilter(m.searchInput.Value())
		return m, cmd
	}
}

// updateRename 处理重命名模式
func (m Model) updateRename(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = ModeNormal
		m.renameInput.Blur()
		m.statusMsg = "已取消重命名"
		return m, nil

	case "enter":
		idx := m.selectedIndex()
		if idx >= 0 {
			oldPath := m.plugins[idx].FilePath
			oldName := m.plugins[idx].Name
			newName := m.renameInput.Value()

			if err := manager.RenamePlugin(&m.plugins[idx], newName); err != nil {
				m.statusMsg = "[错误] " + err.Error()
			} else {
				m.pushUndo(UndoAction{
					Description: fmt.Sprintf("重命名 %s -> %s", oldName, m.plugins[idx].Name),
					Items:       []UndoItem{{OldPath: oldPath, NewPath: m.plugins[idx].FilePath}},
				})
				// 显式设置重命名成功的状态文本，loadPlugins 不会将其抹除
				m.statusMsg = fmt.Sprintf("已成功重命名为: %s", m.plugins[idx].Name)
				m = m.loadPlugins()
				m.mode = ModeNormal
				m.renameInput.Blur()
				return m, m.checkUpdatesCmd()
			}
		}
		m.mode = ModeNormal
		m.renameInput.Blur()
		return m, nil

	default:
		var cmd tea.Cmd
		m.renameInput, cmd = m.renameInput.Update(msg)
		return m, cmd
	}
}

// updateDetail 处理插件详情弹窗
func (m Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter", "i", "q":
		m.mode = ModeNormal
		return m, nil
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.ensureCursorVisible()
		}
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.ensureCursorVisible()
		}
	}
	return m, nil
}

// updateHelp 处理快捷键帮助弹窗
func (m Model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter", "?", "q":
		m.mode = ModeNormal
		return m, nil
	}
	return m, nil
}
