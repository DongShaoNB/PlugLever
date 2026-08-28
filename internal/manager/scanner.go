package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ScanPlugins 扫描指定目录下的所有 Minecraft 插件文件。
// 仅识别 .jar 和 .jar.disabled 后缀的文件，自动忽略子目录和其他无关文件。
// 返回按名称字母序（不区分大小写）排列的插件列表。
func ScanPlugins(dir string) ([]Plugin, error) {
	// 验证目录是否存在且可访问
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("无法访问目录 '%s': %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("'%s' 不是一个有效的目录", dir)
	}

	// 读取目录内容
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("无法读取目录 '%s': %w", dir, err)
	}

	var plugins []Plugin

	for _, entry := range entries {
		// 跳过子目录
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// 仅匹配 .jar 和 .jar.disabled 后缀
		isDisabled := strings.HasSuffix(name, ".jar.disabled")
		isJar := strings.HasSuffix(name, ".jar") && !isDisabled

		if !isJar && !isDisabled {
			continue
		}

		// 获取文件信息（大小等）
		fileInfo, err := entry.Info()
		if err != nil {
			continue // 跳过无法读取信息的文件
		}

		// 构建绝对路径
		absPath, err := filepath.Abs(filepath.Join(dir, name))
		if err != nil {
			absPath = filepath.Join(dir, name)
		}

		// 从 jar 包中解析 plugin.yml 元数据
		version, author := ParsePluginYml(absPath)

		p := Plugin{
			Name:          BaseName(name),
			FileName:      name,
			Enabled:       isJar, // .jar = 启用, .jar.disabled = 禁用
			Version:       version,
			Author:        author,
			FileSizeBytes: fileInfo.Size(),
			FilePath:      absPath,
		}

		plugins = append(plugins, p)
	}

	// 按插件名称字母序排列（不区分大小写）
	sort.Slice(plugins, func(i, j int) bool {
		return strings.ToLower(plugins[i].Name) < strings.ToLower(plugins[j].Name)
	})

	return plugins, nil
}
