package configguard

import (
	"unicode"
	"unicode/utf8"
)

// ValidateName 保证配置名可以安全用于保存、路由路径和列表展示。
func ValidateName(name string) error {
	if name == "" {
		return Error{"请填写配置名称。"}
	}
	if utf8.RuneCountInString(name) > maxConfigNameRunes {
		return guardError("配置名称不能超过 %d 个字符。", maxConfigNameRunes)
	}
	for _, r := range name {
		if r == '/' || r == '\\' || unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			return Error{"配置名称不能包含路径分隔符或控制字符。"}
		}
	}
	return nil
}
