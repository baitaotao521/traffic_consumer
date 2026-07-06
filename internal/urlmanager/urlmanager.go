package urlmanager

import (
	"math/rand"
	"sync"
	"time"

	"github.com/baitaotao521/traffic_consumer/internal/model"
)

// InvalidCallback 在 URL 重试耗尽时通知运行态，供 Web 端展示失败告警。
type InvalidCallback func(model.Event)

// Manager 负责 URL 选择、线程分配、失效标记和使用统计。
type Manager struct {
	mu           sync.Mutex
	urls         []string
	strategy     string
	maxRetries   int
	invalid      map[string]bool
	usage        map[string]int64
	threadStatus map[int]string
	// threadMetrics 保存线程当前响应耗时和响应体读取进度，供 Web 线程表实时刷新。
	threadMetrics map[int]model.ThreadMetric
	weights       []float64
	roundRobinPos int
	callback      InvalidCallback
}

// New 创建 URL 管理器，保留 Python 版 random/round_robin 两种策略。
func New(urls []string, strategy string, maxRetries int, callback InvalidCallback) *Manager {
	cleanURLs := model.CleanStringSlice(urls)
	usage := map[string]int64{}
	for _, url := range cleanURLs {
		usage[url] = 0
	}
	weights := make([]float64, len(cleanURLs))
	for i := range weights {
		weights[i] = 1
	}
	if strategy == "" {
		strategy = "random"
	}
	return &Manager{
		urls:          cleanURLs,
		strategy:      strategy,
		maxRetries:    maxRetries,
		invalid:       map[string]bool{},
		usage:         usage,
		threadStatus:  map[int]string{},
		threadMetrics: map[int]model.ThreadMetric{},
		weights:       weights,
		callback:      callback,
	}
}

// URLs 返回当前运行态 URL 副本。
func (m *Manager) URLs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.urls...)
}

// ResetRuntimeState 为下一轮调度执行清理线程分配、临时失效状态和 URL 使用统计。
func (m *Manager) ResetRuntimeState() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.threadStatus = map[int]string{}
	m.threadMetrics = map[int]model.ThreadMetric{}
	m.invalid = map[string]bool{}
	m.usage = map[string]int64{}
	for _, url := range m.urls {
		m.usage[url] = 0
	}
	for idx := range m.weights {
		m.weights[idx] = 1
	}
}

// ThreadSnapshot 返回所有线程的状态，缺失线程显示为等待中。
func (m *Manager) ThreadSnapshot(total int) map[int]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	snapshot := make(map[int]string, total)
	for i := 1; i <= total; i++ {
		value := m.threadStatus[i]
		if value == "" {
			value = "等待中..."
		}
		snapshot[i] = value
	}
	return snapshot
}

// ThreadMetricSnapshot 返回线程响应耗时和读取进度，缺失线程使用空快照。
func (m *Manager) ThreadMetricSnapshot(total int) map[int]model.ThreadMetric {
	m.mu.Lock()
	defer m.mu.Unlock()
	snapshot := make(map[int]model.ThreadMetric, total)
	for i := 1; i <= total; i++ {
		snapshot[i] = m.threadMetrics[i]
	}
	return snapshot
}

// UsageSnapshot 返回 URL 下载成功次数快照。
func (m *Manager) UsageSnapshot() map[string]int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	snapshot := make(map[string]int64, len(m.usage))
	for url, count := range m.usage {
		snapshot[url] = count
	}
	return snapshot
}

// IsInvalid 判断指定 URL 是否已在本轮被判定失效，供并发重试提前换链。
func (m *Manager) IsInvalid(url string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.invalid[url]
}

// RecordSuccess 在一次完整响应流结束后累计 URL 使用次数。
func (m *Manager) RecordSuccess(url string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.usage[url]++
}

// SetThreadStatus 记录线程当前状态。
func (m *Manager) SetThreadStatus(threadID int, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.threadStatus[threadID] = message
}

// RecordThreadResponse 记录线程本次请求拿到响应头所需时间和响应体长度。
func (m *Manager) RecordThreadResponse(threadID int, responseTime time.Duration, contentLength int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	metric := m.threadMetrics[threadID]
	metric.ResponseTimeMS = durationMillis(responseTime)
	if contentLength > 0 {
		metric.ContentLength = contentLength
	} else {
		metric.ContentLength = 0
	}
	metric.Progress = progressPercent(metric.BytesRead, metric.ContentLength)
	m.threadMetrics[threadID] = metric
}

// RecordThreadProgress 累计线程当前响应体已读取字节并刷新百分比。
func (m *Manager) RecordThreadProgress(threadID int, bytesRead int64) {
	if bytesRead <= 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	metric := m.threadMetrics[threadID]
	metric.BytesRead += bytesRead
	metric.Progress = progressPercent(metric.BytesRead, metric.ContentLength)
	m.threadMetrics[threadID] = metric
}

// CompleteThreadProgress 在一个完整响应读完后把当前响应进度固定为 100%。
func (m *Manager) CompleteThreadProgress(threadID int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	metric := m.threadMetrics[threadID]
	if metric.ContentLength > 0 && metric.BytesRead < metric.ContentLength {
		metric.BytesRead = metric.ContentLength
	}
	metric.Progress = 100
	m.threadMetrics[threadID] = metric
}

// GetURLForThread 根据当前策略为线程分配 URL。
func (m *Manager) GetURLForThread(threadID int) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	available := m.availableLocked()
	if len(available) == 0 {
		return ""
	}

	var selected string
	switch m.strategy {
	case "round_robin":
		selected = m.nextRoundRobinLocked()
		if selected == "" {
			selected = m.weightedRandomLocked(available)
		}
	case "random":
		selected = m.weightedRandomLocked(available)
	default:
		selected = available[0]
	}
	m.threadStatus[threadID] = selected
	// 新响应开始时重置读取进度，旧状态字符串继续负责当前任务展示。
	m.threadMetrics[threadID] = model.ThreadMetric{URL: selected}
	return selected
}

// MarkInvalid 在重试耗尽后标记 URL 失效，并返回是否所有 URL 都失效。
func (m *Manager) MarkInvalid(url string, errorDetail string) bool {
	var event model.Event
	var shouldNotify bool

	m.mu.Lock()
	if m.invalid[url] {
		allInvalid := len(m.availableLocked()) == 0
		m.mu.Unlock()
		return allInvalid
	}
	m.invalid[url] = true
	for threadID, assigned := range m.threadStatus {
		if assigned == url {
			m.threadStatus[threadID] = url + " (已失效)"
			metric := m.threadMetrics[threadID]
			metric.URL = url
			m.threadMetrics[threadID] = metric
		}
	}
	for idx, value := range m.urls {
		if value == url && idx < len(m.weights) {
			m.weights[idx] = 0
		}
	}
	allInvalid := len(m.availableLocked()) == 0
	if m.callback != nil {
		event = model.Event{
			Type:       "download_failure",
			EventType:  "invalid_url",
			Level:      "error",
			Title:      "下载失败：",
			Variant:    "danger",
			URL:        url,
			Message:    "链接已连续失败 " + intText(m.maxRetries) + " 次，已停止重试。",
			Retries:    m.maxRetries,
			Error:      errorDetail,
			AllInvalid: allInvalid,
		}
		shouldNotify = true
	}
	m.mu.Unlock()

	if shouldNotify {
		m.callback(event)
	}
	return allInvalid
}

// RemoveURL 从运行态彻底移除指定 URL。
func (m *Manager) RemoveURL(url string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if url == "" {
		return false
	}

	changed := false
	nextURLs := make([]string, 0, len(m.urls))
	nextWeights := make([]float64, 0, len(m.weights))
	for idx, value := range m.urls {
		if value == url {
			changed = true
			continue
		}
		nextURLs = append(nextURLs, value)
		if idx < len(m.weights) {
			nextWeights = append(nextWeights, m.weights[idx])
		}
	}
	if !changed {
		return false
	}

	m.urls = nextURLs
	m.weights = nextWeights
	delete(m.usage, url)
	delete(m.invalid, url)
	for threadID, assigned := range m.threadStatus {
		if assigned == url || len(assigned) >= len(url) && assigned[:len(url)] == url {
			m.threadStatus[threadID] = url + " (已移除)"
			metric := m.threadMetrics[threadID]
			metric.URL = url
			m.threadMetrics[threadID] = metric
		}
	}
	if len(m.urls) > 0 {
		m.roundRobinPos %= len(m.urls)
	} else {
		m.roundRobinPos = 0
	}
	return true
}

func durationMillis(duration time.Duration) int64 {
	if duration <= 0 {
		return 0
	}
	millis := duration.Milliseconds()
	if millis == 0 {
		return 1
	}
	return millis
}

func progressPercent(bytesRead int64, contentLength int64) float64 {
	if bytesRead <= 0 || contentLength <= 0 {
		return 0
	}
	progress := float64(bytesRead) / float64(contentLength) * 100
	if progress > 100 {
		return 100
	}
	return progress
}

func (m *Manager) availableLocked() []string {
	available := make([]string, 0, len(m.urls))
	for _, url := range m.urls {
		if !m.invalid[url] {
			available = append(available, url)
		}
	}
	return available
}

func (m *Manager) nextRoundRobinLocked() string {
	if len(m.urls) == 0 {
		return ""
	}
	for range m.urls {
		url := m.urls[m.roundRobinPos%len(m.urls)]
		m.roundRobinPos++
		if !m.invalid[url] {
			return url
		}
	}
	return ""
}

func (m *Manager) weightedRandomLocked(candidates []string) string {
	totalUsage := int64(0)
	for _, count := range m.usage {
		totalUsage += count
	}
	if totalUsage == 0 {
		return candidates[rand.Intn(len(candidates))]
	}

	expectedAvg := float64(totalUsage) / float64(maxInt(len(m.urls), 1))
	for idx, url := range m.urls {
		if m.invalid[url] {
			m.weights[idx] = 0
			continue
		}
		current := float64(m.usage[url])
		if current < expectedAvg {
			m.weights[idx] = expectedAvg - current + 1
		} else {
			m.weights[idx] = 1 / (current - expectedAvg + 1)
		}
	}

	totalWeight := 0.0
	weightsByURL := make(map[string]float64, len(m.urls))
	for idx, url := range m.urls {
		if _, exists := weightsByURL[url]; exists || idx >= len(m.weights) {
			continue
		}
		// 重复 URL 共享同一个失效和使用计数，候选权重也沿用第一条配置项，保持旧展示语义。
		weightsByURL[url] = m.weights[idx]
	}
	candidateWeights := make([]float64, len(candidates))
	for idx, url := range candidates {
		weight := 1.0
		if value, ok := weightsByURL[url]; ok {
			weight = value
		}
		candidateWeights[idx] = weight
		totalWeight += weight
	}
	if totalWeight <= 0 {
		return candidates[rand.Intn(len(candidates))]
	}
	r := rand.Float64() * totalWeight
	current := 0.0
	for idx, candidate := range candidates {
		current += candidateWeights[idx]
		if r <= current {
			return candidate
		}
	}
	return candidates[len(candidates)-1]
}

func intText(value int) string {
	return fmtInt(value)
}

func fmtInt(value int) string {
	if value == 0 {
		return "0"
	}
	digits := []byte{}
	n := value
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
