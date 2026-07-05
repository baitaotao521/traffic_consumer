package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/baitaotao521/traffic_consumer/internal/model"
)

var jsonLocks sync.Map

// ErrConfigConflict 表示保存或重命名配置时目标名称已经存在且未确认覆盖。
var ErrConfigConflict = errors.New("config target already exists")

// ConfigConflictError 保留可匹配的冲突类型，同时提供用户可见的中文错误。
type ConfigConflictError struct {
	// Name 是已经存在、需要确认覆盖的配置名。
	Name string
}

func (e ConfigConflictError) Error() string {
	return fmt.Sprintf("配置 \"%s\" 已存在。", e.Name)
}

func (e ConfigConflictError) Is(target error) bool {
	return target == ErrConfigConflict
}

// SaveConfigEntryResult 描述一次配置保存是否覆盖或重命名了已有条目。
type SaveConfigEntryResult struct {
	// SourceExisted 表示路径中的原配置名称在保存前已经存在。
	SourceExisted bool
	// TargetExisted 表示目标配置名称在保存前已经存在。
	TargetExisted bool
	// Renamed 表示本次保存把源配置改名为目标配置。
	Renamed bool
	// Overwritten 表示本次保存覆盖了需要用户确认的已有目标配置。
	Overwritten bool
}

// SaveConfigEntryOptions 描述一次 Web 保存需要在同一把 config.json 锁内判断的命名语义。
type SaveConfigEntryOptions struct {
	// SourceName 是编辑入口打开时对应的原配置名。
	SourceName string
	// TargetName 是表单保存后的最终配置名。
	TargetName string
	// Config 是要写入目标名称下的配置体。
	Config model.Config
	// Force 表示用户已确认覆盖已有目标配置。
	Force bool
	// RequireNew 表示新建入口保存，目标已存在时必须显式确认覆盖。
	RequireNew bool
}

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

// SaveConfigEntryAs 在同一个 config.json 锁内完成创建、重命名和确认覆盖。
func SaveConfigEntryAs(path string, options SaveConfigEntryOptions) (SaveConfigEntryResult, error) {
	configs := map[string]model.Config{}
	sourceName := NormalizeConfigName(options.SourceName)
	targetName := NormalizeConfigName(options.TargetName)
	result := SaveConfigEntryResult{}
	err := MutateJSON(path, &configs, func() (bool, error) {
		_, result.SourceExisted = configs[sourceName]
		_, result.TargetExisted = configs[targetName]
		result.Renamed = sourceName != targetName
		result.Overwritten = result.TargetExisted && (result.Renamed || options.RequireNew)
		if result.Overwritten && !options.Force {
			return false, ConfigConflictError{Name: targetName}
		}

		// 命名配置的真实身份由外层 JSON key 表示，配置体内的临时名称字段不能写回文件。
		cfg := options.Config
		cfg.ConfigName = ""
		cfg.Name = ""
		configs[targetName] = cfg
		if result.Renamed {
			delete(configs, sourceName)
		}
		return true, nil
	})
	return result, err
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
