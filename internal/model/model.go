package model

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// HeaderMap 保存下载请求的自定义请求头，兼容旧配置中的对象和字符串数组两种格式。
type HeaderMap map[string]string

// UnmarshalJSON 兼容 Web 表单、CLI 和旧配置写出的 Header 表示，避免迁移配置文件。
func (h *HeaderMap) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || len(data) == 0 {
		*h = nil
		return nil
	}

	var object map[string]string
	if err := json.Unmarshal(data, &object); err == nil {
		*h = cleanHeaders(object)
		return nil
	}

	var lines []string
	if err := json.Unmarshal(data, &lines); err == nil {
		parsed := map[string]string{}
		for _, line := range lines {
			name, value, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			parsed[strings.TrimSpace(name)] = strings.TrimSpace(value)
		}
		*h = cleanHeaders(parsed)
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		parsed := map[string]string{}
		for _, line := range strings.Split(text, "\n") {
			name, value, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			parsed[strings.TrimSpace(name)] = strings.TrimSpace(value)
		}
		*h = cleanHeaders(parsed)
		return nil
	}

	return fmt.Errorf("unsupported request_headers format")
}

// MarshalJSON 保持 Python 版 Web UI 保存配置时的对象格式。
func (h HeaderMap) MarshalJSON() ([]byte, error) {
	if len(h) == 0 {
		return []byte("null"), nil
	}
	return json.Marshal(map[string]string(h))
}

// Config 是持久化配置和 Web 启动 payload 的共同结构。
type Config struct {
	// URLs 是当前配置的下载链接列表，空数组表示用户明确保存了空链接配置。
	URLs []string `json:"urls"`
	// URL 兼容旧版单链接配置，读取后会归一化到 URLs。
	URL string `json:"url,omitempty"`
	// URLStrategy 决定多链接下载时按随机均衡还是轮询顺序分配链接。
	URLStrategy string `json:"url_strategy,omitempty"`
	// Threads 是本轮任务同时启动的下载 goroutine 数。
	Threads int `json:"threads,omitempty"`
	// LimitSpeed 是整轮任务共享的下载限速，单位为 MB/s，0 表示不限速。
	LimitSpeed float64 `json:"limit_speed,omitempty"`
	// Duration 限制单轮任务最长运行秒数，调度任务每次触发都会重新计算。
	Duration *int64 `json:"duration,omitempty"`
	// Count 限制单轮任务成功完成的响应次数。
	Count *int64 `json:"count,omitempty"`
	// CronExpr 保存五段 cron 表达式，存在时任务进入等待调度态。
	CronExpr string `json:"cron_expr,omitempty"`
	// TrafficLimit 限制单轮任务累计消耗流量，单位为 MB。
	TrafficLimit *int64 `json:"traffic_limit,omitempty"`
	// Interval 保存固定间隔调度分钟数，未设置 cron 时生效。
	Interval *int64 `json:"interval,omitempty"`
	// AutoRemoveFailedURL 控制坏链重试耗尽后是否同步移除运行态和配置文件中的 URL。
	AutoRemoveFailedURL bool `json:"auto_remove_failed_url,omitempty"`
	// AutoStart 标记 Web 服务启动时是否自动拉起该配置。
	AutoStart bool `json:"auto_start,omitempty"`
	// UserAgent 覆盖下载请求最终发送的 User-Agent。
	UserAgent string `json:"user_agent,omitempty"`
	// RequestHeaders 保存下载请求的额外 HTTP 头。
	RequestHeaders HeaderMap `json:"request_headers,omitempty"`
	// URLSwitchInterval 控制单条 URL 连续下载达到指定秒数后主动换链。
	URLSwitchInterval *float64 `json:"url_switch_interval,omitempty"`
	// ThreadStartDelay 控制多线程顺序启动的间隔秒数，减少所有连接同时打到同一路。
	ThreadStartDelay *float64 `json:"thread_start_delay,omitempty"`
	// ConfigName 是后端使用的规范化配置名。
	ConfigName string `json:"config_name,omitempty"`
	// Name 兼容前端 payload 中的配置名字段。
	Name string `json:"name,omitempty"`
}

// Normalize 补齐运行时默认值，并保留“显式空 URL 列表”这条业务语义。
func (c Config) Normalize(defaultURLs []string, name string, useDefaultURLs bool) Config {
	if c.ConfigName == "" {
		c.ConfigName = name
	}
	if c.Name == "" {
		c.Name = c.ConfigName
	}
	if c.ConfigName == "" {
		c.ConfigName = "default"
	}
	if c.Name == "" {
		c.Name = c.ConfigName
	}
	if c.URLStrategy == "" {
		c.URLStrategy = "random"
	}
	if c.Threads <= 0 {
		c.Threads = 4
	}
	if c.URLs == nil && c.URL != "" {
		c.URLs = []string{c.URL}
	}
	if c.URLs == nil && useDefaultURLs {
		c.URLs = append([]string{}, defaultURLs...)
	}
	c.URLs = CleanStringSlice(c.URLs)
	c.UserAgent = CleanHeaderValue(c.UserAgent)
	c.RequestHeaders = cleanHeaders(c.RequestHeaders)
	c.LimitSpeed = maxFloat(c.LimitSpeed, 0)
	if c.URLSwitchInterval != nil && *c.URLSwitchInterval <= 0 {
		c.URLSwitchInterval = nil
	}
	if c.ThreadStartDelay != nil && *c.ThreadStartDelay < 0 {
		c.ThreadStartDelay = nil
	}
	return c
}

// RuntimeState 是前端状态徽标使用的稳定状态枚举。
type RuntimeState string

const (
	// StateRunning 表示至少一个下载任务正在执行。
	StateRunning RuntimeState = "running"
	// StateScheduled 表示没有下载任务，但存在等待触发的计划。
	StateScheduled RuntimeState = "scheduled"
	// StateStopped 表示没有下载任务和计划。
	StateStopped RuntimeState = "stopped"
)

// StatusPayload 对应旧 Socket.IO 的 status_update 数据。
type StatusPayload struct {
	// TotalBytes 是当前运行态累计流量的人类可读展示值。
	TotalBytes string `json:"total_bytes,omitempty"`
	// Speed 是当前运行态的平均下载速度展示值。
	Speed string `json:"speed,omitempty"`
	// DownloadCount 是当前运行态成功完成的响应次数。
	DownloadCount int64 `json:"download_count,omitempty"`
	// Running 告诉前端是否存在运行或等待调度的任务。
	Running bool `json:"running"`
	// State 区分执行中、等待调度和停止三种按钮状态。
	State RuntimeState `json:"state"`
	// Config 是当前状态面板显示的配置名。
	Config string `json:"config,omitempty"`
	// ThreadCount 是线程表需要渲染的总线程数。
	ThreadCount int `json:"thread_count,omitempty"`
	// ThreadStatus 保存每条线程当前分配的 URL 或异常状态。
	ThreadStatus map[int]string `json:"thread_status"`
	// URLUsageStats 保存 URL 使用占比图表的数据源。
	URLUsageStats []URLUsageStat `json:"url_usage_stats"`
	// Message 是操作接口返回给前端弹窗的状态文案。
	Message string `json:"message,omitempty"`
}

// URLUsageStat 是前端 URL 占比列表和图表的数据项。
type URLUsageStat struct {
	// URL 是被统计的下载链接。
	URL string `json:"url"`
	// Count 是该链接成功完成响应的次数。
	Count int64 `json:"count"`
	// Percentage 是该链接在本轮成功下载次数中的占比。
	Percentage float64 `json:"percentage"`
}

// PlanSummary 是运行计划列表中的摘要数据。
type PlanSummary struct {
	// Name 是计划卡片和详情弹窗使用的配置名。
	Name string `json:"name"`
	// Running 表示该计划当前是否正在下载。
	Running bool `json:"running"`
	// SchedulerRunning 表示该计划是否仍在等待后续调度触发。
	SchedulerRunning bool `json:"scheduler_running"`
	// NextRunTime 是调度器计算出的下一次触发时间。
	NextRunTime string `json:"next_run_time,omitempty"`
	// TotalBytes 是历史累计加当前运行流量后的展示值。
	TotalBytes string `json:"total_bytes"`
	// TotalBytesRaw 是前端排序或继续聚合时使用的原始字节数。
	TotalBytesRaw int64 `json:"total_bytes_raw"`
	// DownloadCount 是历史累计加当前运行次数后的下载数。
	DownloadCount int64 `json:"download_count"`
	// Threads 是计划配置的线程数量。
	Threads int `json:"threads"`
	// URLCount 是计划当前可用链接数量。
	URLCount int `json:"url_count"`
	// CronExpr 是计划卡片展示的 cron 调度表达式。
	CronExpr string `json:"cron_expr,omitempty"`
	// Interval 是计划卡片展示的固定间隔分钟数。
	Interval *int64 `json:"interval,omitempty"`
	// AutoStart 表示计划是否来自 Web 服务启动时的自动运行配置。
	AutoStart bool `json:"auto_start"`
	// URLStrategy 是计划使用的 URL 分配策略。
	URLStrategy string `json:"url_strategy"`
	// StatsHistory 预留给旧前端读取计划内联历史，当前详情接口单独返回 History。
	StatsHistory []HistoryRecord `json:"stats_history,omitempty"`
}

// SchedulerPayload 对应旧 Socket.IO 的 scheduler_status_update 数据。
type SchedulerPayload struct {
	// NextRunTime 是所有计划中最早的下一次执行时间。
	NextRunTime string `json:"next_run_time"`
	// JobDetails 是调度 tab 顶部展示的紧凑计划说明。
	JobDetails string `json:"job_details"`
	// Plans 是运行计划列表的数据源。
	Plans []PlanSummary `json:"plans"`
	// Message 是停止或启动调度后的可选提示。
	Message string `json:"message,omitempty"`
}

// HistoryRecord 是内存历史和前端计划详情共用的数据项。
type HistoryRecord struct {
	// Timestamp 是单次运行结束或记录产生的时间。
	Timestamp string `json:"timestamp"`
	// Result 保存运行结果文案，前端直接展示。
	Result string `json:"result"`
	// BytesConsumed 是本次运行消耗流量的人类可读值。
	BytesConsumed string `json:"bytes_consumed,omitempty"`
	// DownloadCount 是本次运行成功完成的响应次数。
	DownloadCount int64 `json:"download_count"`
	// Bytes 是本次运行消耗流量的原始字节数。
	Bytes int64 `json:"bytes,omitempty"`
	// Speed 兼容旧历史记录中的速度字段。
	Speed float64 `json:"speed,omitempty"`
	// ElapsedSeconds 是本次运行持续秒数。
	ElapsedSeconds int64 `json:"elapsed_seconds,omitempty"`
}

// LogSource 区分日志来自调度、运行控制还是下载执行。
type LogSource string

const (
	// LogSourceScheduler 表示日志来自计划建立、触发或停止等调度链路。
	LogSourceScheduler LogSource = "scheduler"
	// LogSourceRuntime 表示日志来自运行态切换、临时测试和自动启动等控制链路。
	LogSourceRuntime LogSource = "runtime"
	// LogSourceDownload 表示日志来自实际下载执行和失败治理链路。
	LogSourceDownload LogSource = "download"
)

// StatsRun 是 stats.json 中单次运行的持久化结构。
type StatsRun struct {
	// ConfigName 是本次运行所属配置，用于计划详情过滤历史。
	ConfigName string `json:"config_name"`
	// URLs 保存运行结束时仍在运行态中的 URL 列表。
	URLs []string `json:"urls"`
	// URLStrategy 保存本次运行使用的 URL 分配策略。
	URLStrategy string `json:"url_strategy"`
	// URLUsage 保存本次运行每条 URL 的成功次数。
	URLUsage map[string]int64 `json:"url_usage"`
	// Threads 保存本次运行的线程数。
	Threads int `json:"threads"`
	// LimitSpeed 保存本次运行的限速配置，单位 MB/s。
	LimitSpeed float64 `json:"limit_speed"`
	// StartTime 是本次运行开始时间，沿用 Python 版字符串格式。
	StartTime string `json:"start_time"`
	// EndTime 是本次运行结束时间，沿用 Python 版字符串格式。
	EndTime string `json:"end_time"`
	// TotalBytes 是本次运行累计消耗的字节数。
	TotalBytes int64 `json:"total_bytes"`
	// DownloadCount 是本次运行成功完成的响应次数。
	DownloadCount int64 `json:"download_count"`
	// ElapsedSeconds 是本次运行持续秒数。
	ElapsedSeconds int64 `json:"elapsed_seconds"`
	// Result 是本次运行结果文案。
	Result string `json:"result"`
	// History 保存该 consumer 内存中的最近运行记录快照。
	History []HistoryRecord `json:"history"`
}

// LogEntry 是 HTTP 轮询日志流的数据项。
type LogEntry struct {
	// Seq 是日志轮询游标，前端按它增量读取。
	Seq int64 `json:"seq"`
	// Timestamp 是日志写入运行态缓冲的时间。
	Timestamp string `json:"timestamp"`
	// Source 区分日志来自调度、运行控制还是下载执行。
	Source LogSource `json:"source"`
	// Level 是日志级别，供前端保留扩展样式。
	Level string `json:"level"`
	// Message 是去掉 ANSI 控制符后的日志正文。
	Message string `json:"message"`
	// Color 是旧日志颜色映射到前端样式时使用的色值。
	Color string `json:"color,omitempty"`
	// Config 是产生日志的配置名。
	Config string `json:"config,omitempty"`
}

// Event 是旧 invalid_url/history_update/error/status_update 推送的 HTTP 轮询替代。
type Event struct {
	// Seq 是事件轮询游标，前端按它增量读取。
	Seq int64 `json:"seq"`
	// Type 是前端分派事件处理逻辑的主类型。
	Type string `json:"type"`
	// EventType 兼容旧 Socket.IO 推送事件名。
	EventType string `json:"event_type,omitempty"`
	// Level 是事件级别，保留给日志或扩展展示使用。
	Level string `json:"level,omitempty"`
	// Title 是通知条的标题文案。
	Title string `json:"title,omitempty"`
	// Variant 映射 Bootstrap alert 的颜色类型。
	Variant string `json:"variant,omitempty"`
	// Message 是通知条展示的业务信息。
	Message string `json:"message,omitempty"`
	// Error 保存失败 URL 或接口异常的底层错误详情。
	Error string `json:"error,omitempty"`
	// URL 是失效治理事件对应的下载链接。
	URL string `json:"url,omitempty"`
	// Config 是事件所属配置名。
	Config string `json:"config,omitempty"`
	// Retries 是 URL 被判定失效前已经尝试的次数。
	Retries int `json:"retries,omitempty"`
	// AllInvalid 表示所有 URL 都已经失效，任务应停止。
	AllInvalid bool `json:"all_invalid,omitempty"`
	// HistoryRecord 是运行完成后推给前端的最新历史记录。
	HistoryRecord *HistoryRecord `json:"history_record,omitempty"`
}

// PlanDetailPayload 是计划详情弹窗需要的完整数据。
type PlanDetailPayload struct {
	// Name 是详情弹窗标题显示的计划名。
	Name string `json:"name"`
	// Config 是详情弹窗读取的持久化配置。
	Config Config `json:"config"`
	// Summary 是详情顶部指标卡的数据源。
	Summary PlanSummary `json:"summary"`
	// History 是详情表格展示的最近运行记录。
	History []HistoryRecord `json:"history"`
}

// CleanStringSlice 去掉 URL 和 Header 输入中的空白项。
func CleanStringSlice(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		text := strings.TrimSpace(value)
		if text != "" {
			result = append(result, text)
		}
	}
	return result
}

// NormalizeLogSource 统一日志来源值，避免大小写和空白导致筛选失效。
func NormalizeLogSource(value string) LogSource {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(LogSourceScheduler):
		return LogSourceScheduler
	case string(LogSourceDownload):
		return LogSourceDownload
	default:
		return LogSourceRuntime
	}
}

// NormalizeLogLevel 统一日志级别值，便于前后端按级别筛选。
func NormalizeLogLevel(value string) string {
	text := strings.ToLower(strings.TrimSpace(value))
	if text == "" {
		return "info"
	}
	return text
}

// CleanHeaderValue 拒绝多行 Header 值，避免 HTTP 头注入。
func CleanHeaderValue(value string) string {
	text := strings.TrimSpace(value)
	if text == "" || strings.ContainsAny(text, "\r\n") {
		return ""
	}
	return text
}

// ParseHeaderLines 解析 CLI 的重复 --header 参数。
func ParseHeaderLines(lines []string) (HeaderMap, error) {
	headers := HeaderMap{}
	for _, line := range lines {
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("--header 必须使用 'Name: value' 格式: %s", line)
		}
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		if name == "" || value == "" {
			continue
		}
		headers[name] = value
	}
	return cleanHeaders(headers), nil
}

// ParseHeadersJSON 解析 CLI 的 --headers-json 参数。
func ParseHeadersJSON(text string) (HeaderMap, error) {
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	headers := HeaderMap{}
	if err := json.Unmarshal([]byte(text), &headers); err != nil {
		return nil, err
	}
	return cleanHeaders(headers), nil
}

// CoerceInt64Ptr 将动态 JSON 数字转为可选整数，供少量 API 兜底使用。
func CoerceInt64Ptr(value any) *int64 {
	switch v := value.(type) {
	case nil:
		return nil
	case float64:
		n := int64(v)
		return &n
	case int64:
		return &v
	case int:
		n := int64(v)
		return &n
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return nil
		}
		return &n
	default:
		return nil
	}
}

func cleanHeaders(headers map[string]string) HeaderMap {
	if len(headers) == 0 {
		return nil
	}
	cleaned := HeaderMap{}
	for name, value := range headers {
		headerName := strings.TrimSpace(name)
		headerValue := CleanHeaderValue(value)
		if headerName == "" || headerValue == "" || strings.ContainsAny(headerName, "\r\n:") {
			continue
		}
		cleaned[headerName] = headerValue
	}
	if len(cleaned) == 0 {
		return nil
	}
	return cleaned
}

func maxFloat(value, min float64) float64 {
	if value < min {
		return min
	}
	return value
}
