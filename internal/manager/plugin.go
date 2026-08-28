// Package manager 负责 Minecraft 插件文件的扫描、元数据解析与状态管理。
package manager

import (
	"archive/zip"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// Plugin 表示一个 Minecraft 插件文件
type Plugin struct {
	Name          string // 主文件名（不含 .jar / .jar.disabled 后缀）
	FileName      string // 完整文件名（含后缀）
	Enabled       bool   // true=启用(.jar), false=禁用(.jar.disabled)
	Version       string // 从 plugin.yml 解析的版本号
	Author        string // 从 plugin.yml 解析的作者
	FileSizeBytes int64  // 文件大小（字节）
	FilePath      string // 文件绝对路径
}

// pluginYml 用于反序列化 jar 包中 plugin.yml 的关键字段
type pluginYml struct {
	Version interface{} `yaml:"version"` // 版本号（可能是 string / int / float64）
	Author  string      `yaml:"author"`  // 单作者字段
	Authors interface{} `yaml:"authors"` // 多作者字段（可能是 []string 或 []interface{}）
}

// ParsePluginYml 从 jar/zip 包中读取并解析 plugin.yml，提取版本号与作者信息。
// 若读取或解析失败则优雅降级返回 "Unknown"。
func ParsePluginYml(jarPath string) (version, author string) {
	version = "Unknown"
	author = "Unknown"

	// 使用 archive/zip 打开 jar 包（jar 本质是 zip 格式）
	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return
	}
	defer r.Close()

	// 遍历 zip 内文件，查找根目录下的 plugin.yml
	for _, f := range r.File {
		if f.Name != "plugin.yml" {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return
		}

		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return
		}

		var yml pluginYml
		if err := yaml.Unmarshal(data, &yml); err != nil {
			return
		}

		// 解析版本号：YAML 可能将 "1.0" 解析为 float64，需统一转为字符串
		if yml.Version != nil {
			version = fmt.Sprintf("%v", yml.Version)
		}

		// 解析作者：优先使用 author 字段，其次拼接 authors 列表
		if yml.Author != "" {
			author = yml.Author
		} else if yml.Authors != nil {
			switch v := yml.Authors.(type) {
			case []interface{}:
				var names []string
				for _, a := range v {
					if s, ok := a.(string); ok {
						names = append(names, s)
					}
				}
				if len(names) > 0 {
					author = strings.Join(names, ", ")
				}
			}
		}

		return
	}

	return
}

// BaseName 从文件名中提取主文件名（去除 .jar 或 .jar.disabled 后缀）
func BaseName(fileName string) string {
	if strings.HasSuffix(fileName, ".jar.disabled") {
		return strings.TrimSuffix(fileName, ".jar.disabled")
	}
	if strings.HasSuffix(fileName, ".jar") {
		return strings.TrimSuffix(fileName, ".jar")
	}
	return fileName
}

// IsEnabled 判断文件名是否表示启用状态（以 .jar 结尾且非 .jar.disabled）
func IsEnabled(fileName string) bool {
	return strings.HasSuffix(fileName, ".jar") && !strings.HasSuffix(fileName, ".jar.disabled")
}

// FormatFileSize 将字节数格式化为人类可读的文件大小字符串
func FormatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
