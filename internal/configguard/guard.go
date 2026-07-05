package configguard

import (
	"errors"
	"fmt"
	"strings"

	"github.com/baitaotao521/traffic_consumer/internal/model"
)

// Mode 决定同一份 Web 配置 payload 在保存、启动和临时测试时的业务校验差异。
type Mode string

const (
	// ModeSave 允许保存显式空 URL 列表，保留配置占位语义。
	ModeSave Mode = "save"
	// ModeStart 要求配置已经具备可以启动下载的 URL 列表。
	ModeStart Mode = "start"
	// ModeTempTest 要求当前表单可以立即执行一次临时验证。
	ModeTempTest Mode = "temp-test"
)

const (
	maxConfigNameRunes     = 80
	maxConfigURLs          = 500
	maxConfigURLLength     = 4096
	maxConfigThreads       = 512
	maxLimitSpeedMB        = 10240
	maxTrafficLimitMB      = 1073741824
	maxDurationSeconds     = 31536000
	maxDownloadCount       = 1000000000
	maxScheduleIntervalMin = 525600
	maxSecondsValue        = 86400
	minSwitchSeconds       = 0.001
)

// Error 表示 Web 配置 payload 违反业务校验，HTTP 层会转成 400。
type Error struct {
	// Message 是前端直接展示的用户可见错误。
	Message string
}

func (e Error) Error() string {
	return e.Message
}

// IsValidationError 判断错误是否来自 Web 配置业务校验。
func IsValidationError(err error) bool {
	var validationError Error
	return errors.As(err, &validationError)
}

// Validate 归一化并校验 Web 配置 payload，返回最终目标配置名。
func Validate(sourceName string, cfg model.Config, mode Mode) (model.Config, string, error) {
	targetName := firstNonEmpty(cfg.ConfigName, cfg.Name, sourceName)
	if err := ValidateName(targetName); err != nil {
		return model.Config{}, "", err
	}

	urls, err := ValidateURLs(cfg, mode)
	if err != nil {
		return model.Config{}, "", err
	}
	if err := ValidateStrategy(cfg.URLStrategy); err != nil {
		return model.Config{}, "", err
	}
	if err := ValidateHeaders(cfg); err != nil {
		return model.Config{}, "", err
	}
	if err := ValidateSchedule(cfg); err != nil {
		return model.Config{}, "", err
	}
	if err := ValidateNumbers(cfg); err != nil {
		return model.Config{}, "", err
	}

	cfg.ConfigName = targetName
	cfg.Name = targetName
	cfg.URLs = urls
	cfg.URL = ""
	if cfg.URLStrategy == "" {
		cfg.URLStrategy = "random"
	}
	cfg.CronExpr = strings.TrimSpace(cfg.CronExpr)
	cfg.UserAgent = strings.TrimSpace(cfg.UserAgent)
	return cfg, targetName, nil
}

func guardError(format string, args ...any) Error {
	return Error{Message: fmt.Sprintf(format, args...)}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
