package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config 存储用户在 PlugLever 中的个性化偏好配置
type Config struct {
	ThemeIndex int `json:"theme_index"` // 当前选择的主题索引
	SortMode   int `json:"sort_mode"`   // 当前选择的排序模式
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		ThemeIndex: 0,
		SortMode:   0,
	}
}

// configFilePath 返回配置文件的绝对存储路径
func configFilePath() (string, error) {
	baseDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, "pluglever", "config.json"), nil
}

// Load 读取并解析配置文件。若文件不存在或读取失败，则返回默认配置。
func Load() Config {
	path, err := configFilePath()
	if err != nil {
		return DefaultConfig()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig()
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig()
	}

	return cfg
}

// Save 将当前配置持久化保存到磁盘文件中
func Save(cfg Config) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
