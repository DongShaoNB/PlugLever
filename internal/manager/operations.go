package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TogglePlugin 切换单个插件的启用/禁用状态。
// 启用(.jar) → 禁用(.jar.disabled)，或反之。
// 在执行重命名前进行目标防冲突校验，杜绝 Linux 下静默覆盖与 Windows 异常。
// 若当前操作为禁用，且 allPlugins 中存在其他已启用的插件依赖当前插件，将受影响的插件名称列表作为 dependents 返回供告警。
// 若当前操作为启用，检查当前插件声明的强依赖是否在 allPlugins 中存在且已启用，将缺失或未启用的依赖列表作为 missingDeps 返回供告警。
func TogglePlugin(p *Plugin, allPlugins []Plugin) (dependents []string, missingDeps []string, err error) {
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

	// 重名冲突检查：确认目标文件不存在，防止静默覆盖
	if _, err := os.Stat(newPath); err == nil {
		return nil, nil, fmt.Errorf("目标文件 '%s' 已存在，操作已取消", newFileName)
	}

	// 若当前操作为【禁用】，检查是否有其他已启用的插件强依赖此插件
	if p.Enabled && len(allPlugins) > 0 {
		targetIdentifiers := []string{
			strings.ToLower(p.Name),
		}
		if p.RawName != "" && !strings.EqualFold(p.RawName, p.Name) {
			targetIdentifiers = append(targetIdentifiers, strings.ToLower(p.RawName))
		}

		for _, other := range allPlugins {
			if !other.Enabled || other.FilePath == p.FilePath {
				continue // 忽略自身和已禁用的插件
			}
			for _, dep := range other.Dependencies {
				depLower := strings.ToLower(dep)
				for _, target := range targetIdentifiers {
					if depLower == target {
						displayName := other.RawName
						if displayName == "" {
							displayName = other.Name
						}
						dependents = append(dependents, displayName)
						break
					}
				}
			}
		}
	}

	// 若当前操作为【启用】，检查当前插件自身声明的强依赖是否缺失或未启用
	if !p.Enabled && len(p.Dependencies) > 0 && len(allPlugins) > 0 {
		for _, dep := range p.Dependencies {
			depLower := strings.ToLower(dep)
			found := false
			foundEnabled := false

			for _, other := range allPlugins {
				if other.FilePath == p.FilePath {
					continue
				}
				if strings.ToLower(other.Name) == depLower || (other.RawName != "" && strings.ToLower(other.RawName) == depLower) {
					found = true
					if other.Enabled {
						foundEnabled = true
					}
					break
				}
			}

			if !found {
				missingDeps = append(missingDeps, fmt.Sprintf("%s(未安装)", dep))
			} else if !foundEnabled {
				missingDeps = append(missingDeps, fmt.Sprintf("%s(已禁用)", dep))
			}
		}
	}

	// 执行文件重命名（即状态切换的实际操作）
	if err := os.Rename(p.FilePath, newPath); err != nil {
		return nil, nil, wrapFileError(err)
	}

	// 更新内存中的插件状态
	p.FileName = newFileName
	p.FilePath = newPath
	p.Enabled = !p.Enabled
	p.Name = BaseName(newFileName)

	return dependents, missingDeps, nil
}

// EnableAll 批量启用所有处于禁用状态的插件。
// 返回成功启用的数量。若中途遇到错误则停止并返回已完成的数量与错误。
func EnableAll(plugins []Plugin) (int, error) {
	count := 0
	for i := range plugins {
		if !plugins[i].Enabled {
			if _, _, err := TogglePlugin(&plugins[i], plugins); err != nil {
				return count, fmt.Errorf("启用 '%s' 失败: %w", plugins[i].Name, err)
			}
			count++
		}
	}
	return count, nil
}

// DisableAll 批量禁用所有处于启用状态的插件。
// 返回成功禁用的数量。若中途遇到错误则停止并返回已完成的数量与错误。
func DisableAll(plugins []Plugin) (int, error) {
	count := 0
	for i := range plugins {
		if plugins[i].Enabled {
			if _, _, err := TogglePlugin(&plugins[i], plugins); err != nil {
				return count, fmt.Errorf("禁用 '%s' 失败: %w", plugins[i].Name, err)
			}
			count++
		}
	}
	return count, nil
}

// RenamePlugin 重命名插件文件（仅修改主文件名，强制保留原始后缀）。
// 执行路径合法性清洗、去除多余 .jar 后缀并进行防重名覆盖检查。
func RenamePlugin(p *Plugin, newBaseName string) error {
	newBaseName = strings.TrimSpace(newBaseName)
	if newBaseName == "" {
		return fmt.Errorf("文件名不能为空")
	}

	// 自动去除用户误输入的后缀，避免生成 .jar.jar
	for {
		lower := strings.ToLower(newBaseName)
		if strings.HasSuffix(lower, ".jar.disabled") {
			newBaseName = strings.TrimSpace(newBaseName[:len(newBaseName)-len(".jar.disabled")])
			continue
		}
		if strings.HasSuffix(lower, ".jar") {
			newBaseName = strings.TrimSpace(newBaseName[:len(newBaseName)-len(".jar")])
			continue
		}
		break
	}

	if newBaseName == "" {
		return fmt.Errorf("文件名不能为空")
	}

	// 安全校验：禁止包含路径分隔符和非法字符，防止路径穿越
	if strings.ContainsAny(newBaseName, `/\:*?"<>|`) || filepath.Base(newBaseName) != newBaseName {
		return fmt.Errorf("文件名包含非法字符或路径分隔符")
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

	// 处理仅大小写改变的情况（针对 Windows / macOS 大小写不敏感文件系统）
	if strings.EqualFold(p.FileName, newFileName) {
		// 采用两步安全中转重命名，规避文件系统目标冲突
		tmpPath := filepath.Join(dir, fmt.Sprintf(".rename_tmp_%d_%s", time.Now().UnixNano(), newFileName))
		if err := os.Rename(p.FilePath, tmpPath); err != nil {
			return wrapFileError(err)
		}
		if err := os.Rename(tmpPath, newPath); err != nil {
			_ = os.Rename(tmpPath, p.FilePath) // 尝试还原
			return wrapFileError(err)
		}

		p.Name = newBaseName
		p.FileName = newFileName
		p.FilePath = newPath
		return nil
	}

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
// 通过 isFileLockedOrPermission 结合系统错误码与中文错误文本判断。
func wrapFileError(err error) error {
	if isFileLockedOrPermission(err) {
		return fmt.Errorf("文件被占用（服务器正在运行？）或权限不足，操作失败")
	}

	msg := err.Error()
	// 匹配 Windows 和 Linux 下常见的文件占用与权限错误描述（中英双语兜底）
	if strings.Contains(msg, "Access is denied") ||
		strings.Contains(msg, "access is denied") ||
		strings.Contains(msg, "text file busy") ||
		strings.Contains(msg, "being used by another process") ||
		strings.Contains(msg, "permission denied") ||
		strings.Contains(msg, "拒绝访问") ||
		strings.Contains(msg, "另一个程序正在使用") ||
		strings.Contains(msg, "无法访问") {
		return fmt.Errorf("文件被占用（服务器正在运行？），操作失败")
	}
	return fmt.Errorf("文件操作失败: %w", err)
}
