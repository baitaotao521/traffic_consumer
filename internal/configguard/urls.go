package configguard

import (
	"net/url"
	"strings"

	"github.com/baitaotao521/traffic_consumer/internal/model"
)

// ValidateURLs 清理 URL 列表，并按保存/运行场景区分是否允许空列表。
func ValidateURLs(cfg model.Config, mode Mode) ([]string, error) {
	rawURLs := cfg.URLs
	if rawURLs == nil && strings.TrimSpace(cfg.URL) != "" {
		rawURLs = []string{cfg.URL}
	}

	cleaned := make([]string, 0, len(rawURLs))
	for _, item := range rawURLs {
		text := strings.TrimSpace(item)
		if text == "" {
			continue
		}
		if len(text) > maxConfigURLLength {
			return nil, guardError("URL 不能超过 %d 个字符。", maxConfigURLLength)
		}
		parsed, err := url.ParseRequestURI(text)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return nil, Error{"URL 必须是有效的 http 或 https 地址。"}
		}
		// 重复 URL 保留为用户明确配置的多条下载目标，校验只剔除空白项并拒绝非法地址。
		cleaned = append(cleaned, text)
		if len(cleaned) > maxConfigURLs {
			return nil, guardError("URL 数量不能超过 %d 条。", maxConfigURLs)
		}
	}
	if mode != ModeSave && len(cleaned) == 0 {
		return nil, Error{"启动或测试配置前至少需要 1 条 URL。"}
	}
	return cleaned, nil
}
