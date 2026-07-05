package configguard

import "strings"

// ValidateStrategy 限定 URL 选择策略，避免错误字符串被运行态悄悄解释成另一种行为。
func ValidateStrategy(strategy string) error {
	text := strings.TrimSpace(strategy)
	if text == "" || text == "random" || text == "round_robin" {
		return nil
	}
	return Error{"URL 策略只能是随机均衡或轮询顺序。"}
}
