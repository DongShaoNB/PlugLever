package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// scanItem 内部传递待解析的文件信息
type scanItem struct {
	name     string
	isJar    bool
	size     int64
	absPath  string
	modTime  time.Time
}

// ScanPlugins 扫描指定目录下的所有 Minecraft 插件文件。
// 仅识别 .jar 和 .jar.disabled 后缀的文件，自动忽略子目录和其他无关文件。
// 采用 Worker Pool 并发读取解析 Jar 包内部元数据，返回按名称字母序（不区分大小写）排列的插件列表。
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

	var items []scanItem
	for _, entry := range entries {
		// 跳过子目录
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// 忽略隐藏文件与 macOS 资源分叉文件（如 ._EssentialsX.jar）
		if strings.HasPrefix(name, ".") {
			continue
		}

		lower := strings.ToLower(name)

		// 仅匹配 .jar 和 .jar.disabled 后缀（忽略大小写）
		isDisabled := strings.HasSuffix(lower, ".jar.disabled")
		isJar := strings.HasSuffix(lower, ".jar") && !isDisabled

		if !isJar && !isDisabled {
			continue
		}

		// 获取文件信息（大小与修改时间）
		fileInfo, err := entry.Info()
		if err != nil || fileInfo.IsDir() {
			continue // 跳过无法读取信息的文件或指向目录的软链接
		}

		// 构建绝对路径
		absPath, err := filepath.Abs(filepath.Join(dir, name))
		if err != nil {
			absPath = filepath.Join(dir, name)
		}

		items = append(items, scanItem{
			name:    name,
			isJar:   isJar,
			size:    fileInfo.Size(),
			absPath: absPath,
			modTime: fileInfo.ModTime(),
		})
	}

	if len(items) == 0 {
		return nil, nil
	}

	// 使用 Worker Pool 并发解析 Jar 包元数据（适度控制并发数防 I/O 抖动）
	workerCount := runtime.NumCPU() * 2
	if workerCount < 4 {
		workerCount = 4
	}
	if workerCount > 16 {
		workerCount = 16
	}
	if workerCount > len(items) {
		workerCount = len(items)
	}

	itemChan := make(chan scanItem, len(items))
	for _, it := range items {
		itemChan <- it
	}
	close(itemChan)

	var wg sync.WaitGroup
	var mu sync.Mutex
	plugins := make([]Plugin, 0, len(items))

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for it := range itemChan {
				meta := ParsePluginMetadata(it.absPath)
				baseName := BaseName(it.name)

				rawName := meta.RawName
				if rawName == "" {
					rawName = baseName
				}

				p := Plugin{
					Name:             baseName,
					RawName:          rawName,
					FileName:         it.name,
					Enabled:          it.isJar, // .jar = 启用, .jar.disabled = 禁用
					Version:          meta.Version,
					Author:           meta.Author,
					Description:      meta.Description,
					Website:          meta.Website,
					SpigotID:         meta.SpigotID,
					Dependencies:     meta.Dependencies,
					SoftDependencies: meta.SoftDependencies,
					FileSizeBytes:    it.size,
					FilePath:         it.absPath,
					ModTime:          it.modTime,
				}

				mu.Lock()
				plugins = append(plugins, p)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// 按插件名称字母序排列（不区分大小写）
	sort.Slice(plugins, func(i, j int) bool {
		return strings.ToLower(plugins[i].Name) < strings.ToLower(plugins[j].Name)
	})

	return plugins, nil
}
