package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/baitaotao521/traffic_consumer/internal/model"
)

var jsonLocks sync.Map

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
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')

	// 配置和统计文件会被多个运行态同时写入，先写同目录临时文件再替换，
	// 避免读端看到半截 JSON。
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()
	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// MutateJSON 在同一进程内串行化同一路径的读改写，避免并发运行配置互相覆盖。
func MutateJSON(path string, target any, mutate func() (bool, error)) error {
	lock := jsonLock(path)
	lock.Lock()
	defer lock.Unlock()
	if err := ReadJSON(path, target); err != nil {
		return err
	}
	changed, err := mutate()
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	return WriteJSON(path, target)
}

func jsonLock(path string) *sync.Mutex {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = filepath.Clean(path)
	}
	lock, _ := jsonLocks.LoadOrStore(abs, &sync.Mutex{})
	return lock.(*sync.Mutex)
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
	configs := map[string]model.Config{}
	return MutateJSON(path, &configs, func() (bool, error) {
		name = NormalizeConfigName(name)
		cfg.ConfigName = ""
		cfg.Name = ""
		configs[name] = cfg
		return true, nil
	})
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
	configs := map[string]model.Config{}
	deleted := false
	err := MutateJSON(path, &configs, func() (bool, error) {
		name = NormalizeConfigName(name)
		if _, ok := configs[name]; !ok {
			return false, nil
		}
		delete(configs, name)
		deleted = true
		return true, nil
	})
	return deleted, err
}

// RemoveURLFromConfig 在失败 URL 自动移除时同步更新持久化配置。
func RemoveURLFromConfig(path, name, url string) (bool, error) {
	if strings.TrimSpace(url) == "" {
		return false, nil
	}
	configs := map[string]model.Config{}
	changed := false
	err := MutateJSON(path, &configs, func() (bool, error) {
		name = NormalizeConfigName(name)
		cfg, ok := configs[name]
		if !ok {
			return false, nil
		}
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
		return true, nil
	})
	return changed, err
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
