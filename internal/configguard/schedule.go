package configguard

import (
	"strings"

	"github.com/robfig/cron/v3"

	"github.com/baitaotao521/traffic_consumer/internal/model"
)

// ValidateSchedule 让 Web 配置只保留一个明确的调度来源。
func ValidateSchedule(cfg model.Config) error {
	cronExpr := strings.TrimSpace(cfg.CronExpr)
	hasInterval := cfg.Interval != nil && *cfg.Interval > 0
	if cronExpr == "" {
		return nil
	}
	if _, err := cron.ParseStandard(cronExpr); err != nil {
		return Error{"无效的Cron表达式"}
	}
	if hasInterval {
		return Error{"Cron 表达式和间隔调度不能同时设置。"}
	}
	return nil
}
