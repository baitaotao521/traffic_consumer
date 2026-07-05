package configguard

import (
	"math"

	"github.com/baitaotao521/traffic_consumer/internal/model"
)

// ValidateNumbers 限制 Web 表单数值，避免 0 值误触发立即停止或超大值拖垮本机。
func ValidateNumbers(cfg model.Config) error {
	if cfg.Threads < 1 || cfg.Threads > maxConfigThreads {
		return guardError("线程数量必须在 1 到 %d 之间。", maxConfigThreads)
	}
	if math.IsNaN(cfg.LimitSpeed) || math.IsInf(cfg.LimitSpeed, 0) || cfg.LimitSpeed < 0 || cfg.LimitSpeed > maxLimitSpeedMB {
		return guardError("限速必须在 0 到 %d MB/s 之间。", maxLimitSpeedMB)
	}
	if err := validateOptionalInt64("总流量", cfg.TrafficLimit, 1, maxTrafficLimitMB); err != nil {
		return err
	}
	if err := validateOptionalInt64("时长", cfg.Duration, 1, maxDurationSeconds); err != nil {
		return err
	}
	if err := validateOptionalInt64("下载次数", cfg.Count, 1, maxDownloadCount); err != nil {
		return err
	}
	if err := validateOptionalInt64("间隔", cfg.Interval, 1, maxScheduleIntervalMin); err != nil {
		return err
	}
	if err := validateOptionalFloat64("URL 切换时限", cfg.URLSwitchInterval, minSwitchSeconds, maxSecondsValue); err != nil {
		return err
	}
	if err := validateOptionalFloat64("线程启动间隔", cfg.ThreadStartDelay, 0, maxSecondsValue); err != nil {
		return err
	}
	return nil
}

func validateOptionalInt64(label string, value *int64, min, max int64) error {
	if value == nil {
		return nil
	}
	if *value < min || *value > max {
		return guardError("%s必须在 %d 到 %d 之间。", label, min, max)
	}
	return nil
}

func validateOptionalFloat64(label string, value *float64, min, max float64) error {
	if value == nil {
		return nil
	}
	if math.IsNaN(*value) || math.IsInf(*value, 0) || *value < min || *value > max {
		return guardError("%s必须在 %.3g 到 %.3g 之间。", label, min, max)
	}
	return nil
}
