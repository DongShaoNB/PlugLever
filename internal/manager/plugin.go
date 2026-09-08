// Package manager 负责 Minecraft 插件文件的扫描、元数据解析与状态管理。
package manager

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Plugin 表示一个 Minecraft 插件文件
type Plugin struct {
	Name          string    // 主文件名（不含 .jar / .jar.disabled 后缀）
	RawName       string    // 插件声明的内部名称（name / id）
	FileName      string    // 完整文件名（含后缀）
	Enabled       bool      // true=启用(.jar), false=禁用(.jar.disabled)
	Version       string    // 解析出的版本号
	Author        string    // 解析出的作者信息
	Description   string    // 插件功能描述
	Website          string    // 插件官网或发布页
	SpigotID         string    // 提取出的 Spigot 资源数字 ID
	UpdateVersion    string    // 异步检测到的最新可用版本号（若有更新）
	Dependencies     []string  // 强依赖列表（仅 depend）
	SoftDependencies []string  // 软依赖列表（仅 softdepend）
	FileSizeBytes    int64     // 文件大小（字节）
	FilePath         string    // 文件绝对路径
	ModTime          time.Time // 文件最后修改时间
}

// PluginMeta 存储从 Jar/Zip 中提取的关键元数据
type PluginMeta struct {
	RawName          string
	Version          string
	Author           string
	Description      string
	Website          string
	SpigotID         string
	Dependencies     []string
	SoftDependencies []string
}

// genericYml 覆盖 Spigot/Bukkit (plugin.yml)、BungeeCord (bungee.yml) 及 Paper 的常见字段
type genericYml struct {
	Name         interface{} `yaml:"name"`
	Version      interface{} `yaml:"version"`
	Author       interface{} `yaml:"author"`
	Authors      interface{} `yaml:"authors"`
	Description  interface{} `yaml:"description"`
	Website      interface{} `yaml:"website"`
	Depend       interface{} `yaml:"depend"`
	Depends      interface{} `yaml:"depends"`
	SoftDepend   interface{} `yaml:"softdepend"`
	SoftDepends  interface{} `yaml:"softdepends"`
	Dependencies interface{} `yaml:"dependencies"`
}

// velocityMeta 覆盖 Velocity 代理端 (velocity-plugin.json) 结构
type velocityMeta struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	URL          string   `json:"url"`
	Authors      []string `json:"authors"`
	Dependencies []struct {
		ID       string `json:"id"`
		Optional bool   `json:"optional"`
	} `json:"dependencies"`
}

const maxMetadataFileSize = 1 << 20 // 1MB 限制，防御 Zip 炸弹

// ParsePluginMetadata 从 jar/zip 包中读取并解析元数据文件。
// 支持 Bukkit/Spigot/Paper (plugin.yml, paper-plugin.yml)、BungeeCord (bungee.yml) 及 Velocity (velocity-plugin.json)。
// 若读取或解析失败则优雅降级返回默认值。
func ParsePluginMetadata(jarPath string) PluginMeta {
	meta := PluginMeta{
		Version: "Unknown",
		Author:  "Unknown",
	}

	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return meta
	}
	defer r.Close()

	// 按优先级寻找匹配的元数据文件
	var targetFile *zip.File
	var targetKind string // "yaml" 或 "velocity_json"

	for _, f := range r.File {
		cleanName := strings.TrimPrefix(strings.ToLower(f.Name), "/")
		switch cleanName {
		case "plugin.yml":
			targetFile = f
			targetKind = "yaml"
		case "paper-plugin.yml":
			if targetFile == nil || targetKind != "yaml" {
				targetFile = f
				targetKind = "yaml"
			}
		case "bungee.yml":
			if targetFile == nil {
				targetFile = f
				targetKind = "yaml"
			}
		case "velocity-plugin.json":
			if targetFile == nil {
				targetFile = f
				targetKind = "velocity_json"
			}
		}
		if cleanName == "plugin.yml" {
			break // plugin.yml 具有最高优先级，找到即可停止遍历
		}
	}

	if targetFile == nil {
		return meta
	}

	rc, err := targetFile.Open()
	if err != nil {
		return meta
	}
	defer rc.Close()

	// 限制读取最大 1MB 数据，防御超大文件或 Zip 炸弹
	data, err := io.ReadAll(io.LimitReader(rc, maxMetadataFileSize))
	if err != nil {
		return meta
	}

	if targetKind == "velocity_json" {
		var v velocityMeta
		if err := json.Unmarshal(data, &v); err == nil {
			if v.Name != "" {
				meta.RawName = v.Name
			} else {
				meta.RawName = v.ID
			}
			if v.Version != "" {
				meta.Version = v.Version
			}
			if len(v.Authors) > 0 {
				meta.Author = strings.Join(v.Authors, ", ")
			}
			meta.Description = v.Description
			meta.Website = v.URL
			for _, dep := range v.Dependencies {
				if dep.ID != "" {
					if !dep.Optional {
						meta.Dependencies = append(meta.Dependencies, dep.ID)
					} else {
						meta.SoftDependencies = append(meta.SoftDependencies, dep.ID)
					}
				}
			}
			meta.SpigotID = ExtractSpigotResourceID(meta.Website)
			return meta
		}
	}

	// 统一作为 YAML 解析（涵盖 plugin.yml, paper-plugin.yml, bungee.yml）
	var yml genericYml
	if err := yaml.Unmarshal(data, &yml); err != nil {
		return meta
	}

	if yml.Name != nil {
		meta.RawName = fmt.Sprintf("%v", yml.Name)
	}

	if yml.Version != nil {
		meta.Version = fmt.Sprintf("%v", yml.Version)
	}

	if yml.Description != nil {
		meta.Description = fmt.Sprintf("%v", yml.Description)
	}

	if yml.Website != nil {
		meta.Website = fmt.Sprintf("%v", yml.Website)
	}

	// 解析作者
	if yml.Author != nil && fmt.Sprintf("%v", yml.Author) != "" {
		meta.Author = fmt.Sprintf("%v", yml.Author)
	} else if yml.Authors != nil {
		meta.Author = parseStringList(yml.Authors, ", ")
	}

	// 提取强依赖 depend 与软依赖 softdepend
	var deps []string
	var softDeps []string
	if yml.Depend != nil {
		deps = append(deps, extractStringList(yml.Depend)...)
	}
	if yml.Depends != nil {
		deps = append(deps, extractStringList(yml.Depends)...)
	}
	if yml.SoftDepend != nil {
		softDeps = append(softDeps, extractStringList(yml.SoftDepend)...)
	}
	if yml.SoftDepends != nil {
		softDeps = append(softDeps, extractStringList(yml.SoftDepends)...)
	}
	if yml.Dependencies != nil {
		// 支持 Paper paper-plugin.yml 的 dependencies 节点
		req, opt := extractPaperDependencies(yml.Dependencies)
		deps = append(deps, req...)
		softDeps = append(softDeps, opt...)
	}

	meta.Dependencies = uniqueStrings(deps)
	meta.SoftDependencies = uniqueStrings(softDeps)
	meta.SpigotID = ExtractSpigotResourceID(meta.Website)
	return meta
}

// ParsePluginYml 兼容旧版调用，从 jar 中提取版本号与作者
func ParsePluginYml(jarPath string) (version, author string) {
	meta := ParsePluginMetadata(jarPath)
	return meta.Version, meta.Author
}

// parseStringList 将可能是 string、[]string 或 []interface{} 的值拼接为字符串
func parseStringList(v interface{}, sep string) string {
	list := extractStringList(v)
	if len(list) == 0 {
		return ""
	}
	return strings.Join(list, sep)
}

// extractStringList 从任意结构提取出字符串切片
func extractStringList(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case string:
		val = strings.TrimSpace(val)
		if val != "" {
			return []string{val}
		}
	case []interface{}:
		var result []string
		for _, item := range val {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				result = append(result, strings.TrimSpace(s))
			} else if item != nil {
				str := fmt.Sprintf("%v", item)
				if strings.TrimSpace(str) != "" {
					result = append(result, strings.TrimSpace(str))
				}
			}
		}
		return result
	case []string:
		var result []string
		for _, s := range val {
			if strings.TrimSpace(s) != "" {
				result = append(result, strings.TrimSpace(s))
			}
		}
		return result
	}
	return nil
}

// extractPaperDependencies 解析 paper-plugin.yml 中 dependencies 结构的强依赖与软依赖
func extractPaperDependencies(v interface{}) (required []string, optional []string) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return extractStringList(v), nil
	}

	for _, section := range []string{"server", "bootstrap"} {
		if secVal, exists := m[section]; exists {
			if secMap, ok := secVal.(map[string]interface{}); ok {
				for depName, depProps := range secMap {
					depName = strings.TrimSpace(depName)
					if depName == "" {
						continue
					}
					isReq := true
					if propsMap, ok := depProps.(map[string]interface{}); ok {
						if reqVal, exists := propsMap["required"]; exists {
							if b, ok := reqVal.(bool); ok {
								isReq = b
							}
						}
					}
					if isReq {
						required = append(required, depName)
					} else {
						optional = append(optional, depName)
					}
				}
			}
		}
	}
	return required, optional
}

// uniqueStrings 去重并保持原有顺序
func uniqueStrings(items []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range items {
		if !seen[item] && item != "" {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// BaseName 从文件名中提取主文件名（去除 .jar 或 .jar.disabled 后缀，忽略大小写）
func BaseName(fileName string) string {
	lower := strings.ToLower(fileName)
	if strings.HasSuffix(lower, ".jar.disabled") {
		return fileName[:len(fileName)-len(".jar.disabled")]
	}
	if strings.HasSuffix(lower, ".jar") {
		return fileName[:len(fileName)-len(".jar")]
	}
	return fileName
}

// IsEnabled 判断文件名是否表示启用状态（以 .jar 结尾且非 .jar.disabled）
func IsEnabled(fileName string) bool {
	lower := strings.ToLower(fileName)
	return strings.HasSuffix(lower, ".jar") && !strings.HasSuffix(lower, ".jar.disabled")
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
