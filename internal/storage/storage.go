package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/baitaotao521/traffic_consumer/internal/model"
)

// ReadJSON 安全读取 JSON；文件不存在或内容损坏时返回空对象，延续 Python 版容错行为。
func ReadJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, target); err != nil {
		return nil
	}
	return nil
}

// WriteJSON 写回 JSON 文件，并自动创建父目录。
func WriteJSON(path string, data any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return os.WriteFile(path, payload, 0o644)
}

// NormalizeConfigName 统一配置名，避免空字符串写入奇怪 key。
func NormalizeConfigName(name string) string {
	text := strings.TrimSpace(name)
	if text == "" {
		return "default"
	}
	return text
}

// LoadConfigs 读取全部保存配置。
func LoadConfigs(path string) (map[string]model.Config, error) {
	configs := map[string]model.Config{}
	if err := ReadJSON(path, &configs); err != nil {
		return map[string]model.Config{}, err
	}
	return configs, nil
}

// SaveConfigEntry 将配置写入指定名称，保留现有 config.json 外层结构。
func SaveConfigEntry(path, name string, cfg model.Config) error {
	configs, err := LoadConfigs(path)
	if err != nil {
		return err
	}
	name = NormalizeConfigName(name)
	cfg.ConfigName = ""
	cfg.Name = ""
	configs[name] = cfg
	return WriteJSON(path, configs)
}

// LoadConfigEntry 读取指定配置；第二个返回值表示配置是否存在。
func LoadConfigEntry(path, name string) (model.Config, bool, error) {
	configs, err := LoadConfigs(path)
	if err != nil {
		return model.Config{}, false, err
	}
	cfg, ok := configs[NormalizeConfigName(name)]
	return cfg, ok, nil
}

// ListConfigNames 返回按名称排序的配置列表，方便前端稳定渲染。
func ListConfigNames(path string) ([]string, error) {
	configs, err := LoadConfigs(path)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(configs))
	for name := range configs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// DeleteConfigEntry 删除指定配置。
func DeleteConfigEntry(path, name string) (bool, error) {
	configs, err := LoadConfigs(path)
	if err != nil {
		return false, err
	}
	name = NormalizeConfigName(name)
	if _, ok := configs[name]; !ok {
		return false, nil
	}
	delete(configs, name)
	return true, WriteJSON(path, configs)
}

// RemoveURLFromConfig 在失败 URL 自动移除时同步更新持久化配置。
func RemoveURLFromConfig(path, name, url string) (bool, error) {
	if strings.TrimSpace(url) == "" {
		return false, nil
	}
	configs, err := LoadConfigs(path)
	if err != nil {
		return false, err
	}
	name = NormalizeConfigName(name)
	cfg, ok := configs[name]
	if !ok {
		return false, nil
	}
	changed := false
	filtered := make([]string, 0, len(cfg.URLs))
	for _, item := range cfg.URLs {
		if item == url {
			changed = true
			continue
		}
		filtered = append(filtered, item)
	}
	if changed {
		cfg.URLs = filtered
	}
	if cfg.URL == url {
		cfg.URL = ""
		changed = true
	}
	if !changed {
		return false, nil
	}
	configs[name] = cfg
	return true, WriteJSON(path, configs)
}

// FindAutoStartConfigs 找出所有启用 Web 启动自启的配置。
func FindAutoStartConfigs(path string) ([]string, error) {
	configs, err := LoadConfigs(path)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0)
	for name, cfg := range configs {
		if cfg.AutoStart {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

// MustWriteJSON 用于测试辅助，业务代码不要吞掉写入错误。
func MustWriteJSON(path string, data any) {
	if err := WriteJSON(path, data); err != nil {
		panic(fmt.Sprintf("write json: %v", err))
	}
}
