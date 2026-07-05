package configguard

import (
	"strings"
	"unicode"

	"github.com/baitaotao521/traffic_consumer/internal/model"
)

// ValidateHeaders 保证请求头不会被静默覆盖或产生 HTTP 头注入风险。
func ValidateHeaders(cfg model.Config) error {
	seen := map[string]bool{}
	userAgent := strings.TrimSpace(cfg.UserAgent)
	if userAgent != "" && strings.ContainsAny(userAgent, "\r\n") {
		return Error{"User-Agent 不能包含换行。"}
	}
	for name, value := range cfg.RequestHeaders {
		headerName := strings.TrimSpace(name)
		headerValue := strings.TrimSpace(value)
		if headerName == "" || headerValue == "" {
			return Error{"自定义请求头必须使用非空名称和值。"}
		}
		if invalidHeaderName(headerName) {
			return Error{"自定义请求头名称不能包含冒号、空白或控制字符。"}
		}
		if strings.ContainsAny(headerValue, "\r\n") {
			return Error{"自定义请求头值不能包含换行。"}
		}
		key := strings.ToLower(headerName)
		if seen[key] {
			return Error{"自定义请求头名称不能重复。"}
		}
		seen[key] = true
		if userAgent != "" && key == "user-agent" {
			return Error{"User-Agent 请只在专用字段中配置。"}
		}
	}
	return nil
}

func invalidHeaderName(name string) bool {
	for _, r := range name {
		if r == ':' || unicode.IsSpace(r) || unicode.IsControl(r) {
			return true
		}
	}
	return false
}
