package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TogglePlugin 切换单个插件的启用/禁用状态。
// 启用(.jar) → 禁用(.jar.disabled)，或反之。
// 同时更新 Plugin 结构体中的相关字段。
func TogglePlugin(p *Plugin) error {
	var newFileName string
	if p.Enabled {
		// 启用 → 禁用：在 .jar 后追加 .disabled
		newFileName = p.FileName + ".disabled"
	} else {
		// 禁用 → 启用：移除末尾的 .disabled
		newFileName = strings.TrimSuffix(p.FileName, ".disabled")
	}

	dir := filepath.Dir(p.FilePath)
	newPath := filepath.Join(dir, newFileName)

	// 执行文件重命名（即状态切换的实际操作）
	if err := os.Rename(p.FilePath, newPath); err != nil {
		return wrapFileError(err)
	}

	// 更新内存中的插件状态
	p.FileName = newFileName
	p.FilePath = newPath
	p.Enabled = !p.Enabled
	p.Name = BaseName(newFileName)

	return nil
}

// EnableAll 批量启用所有处于禁用状态的插件。
// 返回成功启用的数量。若中途遇到错误则停止并返回已完成的数量。
func EnableAll(plugins []Plugin) (int, error) {
	count := 0
	for i := range plugins {
		if !plugins[i].Enabled {
			if err := TogglePlugin(&plugins[i]); err != nil {
				return count, fmt.Errorf("启用 '%s' 失败: %w", plugins[i].Name, err)
			}
			count++
		}
	}
	return count, nil
}

// DisableAll 批量禁用所有处于启用状态的插件。
// 返回成功禁用的数量。若中途遇到错误则停止并返回已完成的数量。
func DisableAll(plugins []Plugin) (int, error) {
	count := 0
	for i := range plugins {
		if plugins[i].Enabled {
			if err := TogglePlugin(&plugins[i]); err != nil {
				return count, fmt.Errorf("禁用 '%s' 失败: %w", plugins[i].Name, err)
			}
			count++
		}
	}
	return count, nil
}

// RenamePlugin 重命名插件文件（仅修改主文件名，强制保留原始后缀）。
// 执行重名覆盖检查，若目标文件已存在则拒绝操作。
func RenamePlugin(p *Plugin, newBaseName string) error {
	newBaseName = strings.TrimSpace(newBaseName)
	if newBaseName == "" {
		return fmt.Errorf("文件名不能为空")
	}

	// 根据当前状态保留对应后缀
	var suffix string
	if p.Enabled {
		suffix = ".jar"
	} else {
		suffix = ".jar.disabled"
	}

	newFileName := newBaseName + suffix

	// 如果名称未改变则跳过
	if newFileName == p.FileName {
		return nil
	}

	dir := filepath.Dir(p.FilePath)
	newPath := filepath.Join(dir, newFileName)

	// 重名覆盖检查：确认目标文件不存在
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("文件 '%s' 已存在，无法重命名", newFileName)
	}

	// 执行重命名
	if err := os.Rename(p.FilePath, newPath); err != nil {
		return wrapFileError(err)
	}

	// 更新内存中的插件信息
	p.Name = newBaseName
	p.FileName = newFileName
	p.FilePath = newPath

	return nil
}

// wrapFileError 捕获底层 I/O 错误并转换为用户友好的中文提示。
// 主要处理 Windows 下文件被占用（服务器运行中）的场景。
func wrapFileError(err error) error {
	msg := err.Error()
	// 匹配 Windows 和 Linux 下常见的文件占用错误信息
	if strings.Contains(msg, "Access is denied") ||
		strings.Contains(msg, "access is denied") ||
		strings.Contains(msg, "text file busy") ||
		strings.Contains(msg, "being used by another process") ||
		strings.Contains(msg, "permission denied") {
		return fmt.Errorf("文件被占用（服务器正在运行？），操作失败")
	}
	return fmt.Errorf("文件操作失败: %w", err)
}
