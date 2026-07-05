package consumer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/baitaotao521/traffic_consumer/internal/config"
	"github.com/baitaotao521/traffic_consumer/internal/model"
	"github.com/baitaotao521/traffic_consumer/internal/stats"
	"github.com/baitaotao521/traffic_consumer/internal/storage"
	"github.com/baitaotao521/traffic_consumer/internal/urlmanager"
)

// Logger 将下载器内部日志转交给 CLI 或 Web 运行态。
type Logger func(level, message, color string)

// Eventer 将失败 URL 等业务事件转交给 Web 运行态。
type Eventer func(model.Event)

// Consumer 执行一次或多次流量消耗任务。
type Consumer struct {
	// Config 是本次 consumer 启动时使用的配置快照。
	Config model.Config
	// ConfigDir 保存配置和统计文件路径，失败 URL 自动移除会写回这里的 config.json。
	ConfigDir config.Paths
	// Stats 负责在每轮任务结束后写入 stats.json。
	Stats *stats.Store
	// URLMgr 负责多 URL 分配、失效标记、线程状态和使用次数统计。
	URLMgr *urlmanager.Manager
	// Logger 把下载过程中的业务日志交给 CLI 或 Web 运行态。
	Logger Logger
	// Eventer 把失效 URL 和历史更新事件交给 Web 运行态。
	Eventer Eventer

	mu              sync.Mutex
	active          bool
	status          string
	startTime       time.Time
	totalBytes      int64
	downloadCount   int64
	cancel          context.CancelFunc
	done            chan struct{}
	rateLimiter     *RateLimiter
	trafficNotified bool
	countNotified   bool
	retryLogKeys    map[string]bool
	history         []model.HistoryRecord
}

const (
	connectTimeout = 10 * time.Second
	readTimeout    = 30 * time.Second
	maxRetries     = 5
	retryBackoff   = 1500 * time.Millisecond
)

// New 创建下载器，并把 URL 失效事件连接到运行态。
func New(cfg model.Config, paths config.Paths, store *stats.Store, logger Logger, eventer Eventer) *Consumer {
	cfg = cfg.Normalize(config.DefaultURLs, cfg.ConfigName, cfg.URLs == nil)
	if logger == nil {
		logger = func(level, message, color string) { fmt.Println(message) }
	}
	c := &Consumer{
		Config:       cfg,
		ConfigDir:    paths,
		Stats:        store,
		Logger:       logger,
		Eventer:      eventer,
		done:         closedDone(),
		retryLogKeys: map[string]bool{},
	}
	c.URLMgr = urlmanager.New(cfg.URLs, cfg.URLStrategy, maxRetries, func(event model.Event) {
		event.Config = cfg.ConfigName
		if c.Eventer != nil {
			c.Eventer(event)
		}
	})
	if cfg.LimitSpeed > 0 {
		c.rateLimiter = NewRateLimiter(int64(cfg.LimitSpeed * 1024 * 1024))
	}
	return c
}

// StartAsync 在后台执行一次完整下载任务。
func (c *Consumer) StartAsync() <-chan struct{} {
	c.mu.Lock()
	if c.active {
		done := c.done
		c.mu.Unlock()
		return done
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.done = make(chan struct{})
	c.active = true
	done := c.done
	c.mu.Unlock()

	go func() {
		defer close(done)
		c.RunOnce(ctx)
	}()
	return done
}

// RunOnce 同步执行一轮下载，供 CLI 和调度器调用。
func (c *Consumer) RunOnce(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	c.prepareRun()
	defer c.finishRun()

	var wg sync.WaitGroup
	for i := 1; i <= c.Config.Threads; i++ {
		if !c.IsActive() || ctx.Err() != nil {
			break
		}
		threadID := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.downloadLoop(ctx, threadID)
		}()
		// 顺序发起线程可减少所有连接被上游负载均衡打到同一路的概率。
		if c.Config.ThreadStartDelay != nil && *c.Config.ThreadStartDelay > 0 && i < c.Config.Threads {
			if !sleepWithContext(ctx, time.Duration(*c.Config.ThreadStartDelay*float64(time.Second))) {
				break
			}
		}
	}

	if c.Config.Duration != nil && *c.Config.Duration > 0 {
		sleepWithContext(ctx, time.Duration(*c.Config.Duration)*time.Second)
		c.Stop()
	} else {
		for c.IsActive() && ctx.Err() == nil {
			time.Sleep(100 * time.Millisecond)
		}
	}
	wg.Wait()
}

// Stop 请求停止下载任务。
func (c *Consumer) Stop() {
	c.mu.Lock()
	cancel := c.cancel
	c.active = false
	c.status = "已停止"
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Done 返回当前后台任务结束信号。
func (c *Consumer) Done() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.done
}

// IsActive 返回当前是否正在下载。
func (c *Consumer) IsActive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.active
}

// Snapshot 返回前端状态面板需要的运行数据。
func (c *Consumer) Snapshot() (totalBytes int64, downloadCount int64, startTime time.Time, active bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.totalBytes, c.downloadCount, c.startTime, c.active
}

// History 返回内存历史副本。
func (c *Consumer) History() []model.HistoryRecord {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]model.HistoryRecord{}, c.history...)
}

func (c *Consumer) prepareRun() {
	c.mu.Lock()
	c.active = true
	c.status = "正在执行"
	c.startTime = time.Now()
	c.totalBytes = 0
	c.downloadCount = 0
	c.trafficNotified = false
	c.countNotified = false
	c.retryLogKeys = map[string]bool{}
	c.mu.Unlock()
	c.URLMgr.ResetRuntimeState()
	c.log("info", "流量消耗器启动", "#0dcaf0")
}

func (c *Consumer) finishRun() {
	c.mu.Lock()
	c.active = false
	startTime := c.startTime
	totalBytes := c.totalBytes
	downloadCount := c.downloadCount
	c.status = "已停止"
	c.mu.Unlock()

	if !startTime.IsZero() {
		historyRecord := model.HistoryRecord{
			Timestamp:      time.Now().Format(time.RFC3339),
			Result:         "成功",
			BytesConsumed:  stats.FormatBytes(float64(totalBytes)),
			DownloadCount:  downloadCount,
			Bytes:          totalBytes,
			ElapsedSeconds: int64(time.Since(startTime).Seconds()),
		}
		c.mu.Lock()
		c.history = append([]model.HistoryRecord{historyRecord}, c.history...)
		if len(c.history) > 50 {
			c.history = c.history[:50]
		}
		historySnapshot := append([]model.HistoryRecord{}, c.history...)
		c.mu.Unlock()
		run := model.StatsRun{
			ConfigName:     c.Config.ConfigName,
			URLs:           c.URLMgr.URLs(),
			URLStrategy:    c.Config.URLStrategy,
			URLUsage:       c.URLMgr.UsageSnapshot(),
			Threads:        c.Config.Threads,
			LimitSpeed:     c.Config.LimitSpeed,
			StartTime:      startTime.Format("2006-01-02 15:04:05"),
			EndTime:        time.Now().Format("2006-01-02 15:04:05"),
			TotalBytes:     totalBytes,
			DownloadCount:  downloadCount,
			ElapsedSeconds: int64(time.Since(startTime).Seconds()),
			Result:         "成功",
			History:        historySnapshot,
		}
		if err := c.Stats.SaveRun(run); err != nil {
			c.log("error", "保存统计数据失败: "+err.Error(), "#dc3545")
		}
		if c.Eventer != nil {
			c.Eventer(model.Event{
				Type:          "history_update",
				EventType:     "history_update",
				Level:         "info",
				Title:         "历史更新：",
				Variant:       "info",
				Config:        c.Config.ConfigName,
				HistoryRecord: &historyRecord,
			})
		}
	}
	// 本次运行的结果已写入历史，运行态计数归零，避免停止后的配置行继续显示旧快照。
	c.mu.Lock()
	c.totalBytes = 0
	c.downloadCount = 0
	c.startTime = time.Time{}
	c.mu.Unlock()
	c.log("info", "任务已停止。", "#0dcaf0")
}

func (c *Consumer) downloadLoop(ctx context.Context, threadID int) {
	client := c.httpClient()
	for c.IsActive() && ctx.Err() == nil {
		if c.Config.Count != nil {
			c.mu.Lock()
			reached := c.downloadCount >= *c.Config.Count
			c.mu.Unlock()
			if reached {
				c.stopDueToCount()
				return
			}
		}

		currentURL := c.URLMgr.GetURLForThread(threadID)
		if currentURL == "" {
			c.log("error", "未找到可用的下载链接，任务将停止。", "#dc3545")
			c.URLMgr.SetThreadStatus(threadID, "无可用链接")
			c.Stop()
			return
		}

		completed := c.downloadWithRetries(ctx, client, currentURL, threadID)
		if !c.IsActive() || ctx.Err() != nil {
			return
		}
		if !completed {
			continue
		}

		reachedCount := false
		c.mu.Lock()
		c.URLMgr.RecordSuccess(currentURL)
		c.downloadCount++
		if c.Config.Count != nil && c.downloadCount >= *c.Config.Count {
			reachedCount = true
		}
		c.mu.Unlock()
		if reachedCount {
			c.stopDueToCount()
			return
		}
	}
}

func (c *Consumer) downloadWithRetries(ctx context.Context, client *http.Client, url string, threadID int) bool {
	backoff := retryBackoff
	for attempt := 1; attempt <= maxRetries && c.IsActive() && ctx.Err() == nil; attempt++ {
		completed, err := c.streamDownload(ctx, client, url, threadID)
		if err == nil {
			return completed
		}
		if !c.IsActive() || ctx.Err() != nil {
			return false
		}
		errorDetail := formatErrorDetail(err)
		c.logRetryFailure(threadID, url, attempt, errorDetail)
		if attempt >= maxRetries {
			allInvalid := c.URLMgr.MarkInvalid(url, errorDetail)
			if c.Config.AutoRemoveFailedURL {
				c.handleAutoRemoveFailedURL(url)
			}
			if allInvalid {
				c.Stop()
			}
			return false
		}
		if !sleepWithContext(ctx, backoff) {
			return false
		}
		backoff *= 2
		if backoff > 8*time.Second {
			backoff = 8 * time.Second
		}
	}
	return false
}

func (c *Consumer) streamDownload(ctx context.Context, client *http.Client, url string, threadID int) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Expires", "0")
	for name, value := range c.Config.RequestHeaders {
		req.Header.Set(name, value)
	}
	// 专用 UA 字段覆盖自定义 Header 中的 User-Agent，避免界面两处配置互相打架。
	if c.Config.UserAgent != "" {
		req.Header.Set("User-Agent", c.Config.UserAgent)
	}

	requestStartedAt := time.Now()
	resp, err := client.Do(req)
	responseTime := time.Since(requestStartedAt)
	if err != nil {
		c.URLMgr.RecordThreadResponse(threadID, responseTime, 0)
		return false, err
	}
	defer resp.Body.Close()
	c.URLMgr.RecordThreadResponse(threadID, responseTime, resp.ContentLength)
	if resp.StatusCode >= 400 {
		return false, fmt.Errorf("HTTP %d %s", resp.StatusCode, resp.Status)
	}

	startedAt := time.Now()
	buf := make([]byte, config.DefaultChunkSize)
	for c.IsActive() && ctx.Err() == nil {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if c.rateLimiter != nil {
				c.rateLimiter.Acquire(n)
			}
			c.URLMgr.RecordThreadProgress(threadID, int64(n))
			c.addBytes(int64(n))
			if c.checkTrafficLimit() {
				return false, nil
			}
			if c.shouldSwitchURL(startedAt) {
				c.log("info", fmt.Sprintf("链接已连续下载超过 %.3g 秒，切换到下一条 URL。", *c.Config.URLSwitchInterval), "#0dcaf0")
				return false, nil
			}
		}
		if readErr == nil {
			continue
		}
		if errors.Is(readErr, io.EOF) {
			c.URLMgr.CompleteThreadProgress(threadID)
			return true, nil
		}
		return false, readErr
	}
	return false, nil
}

func (c *Consumer) addBytes(n int64) {
	c.mu.Lock()
	c.totalBytes += n
	c.mu.Unlock()
}

func (c *Consumer) checkTrafficLimit() bool {
	if c.Config.TrafficLimit == nil {
		return false
	}
	limitBytes := *c.Config.TrafficLimit * 1024 * 1024
	c.mu.Lock()
	if c.trafficNotified || c.totalBytes < limitBytes {
		c.mu.Unlock()
		return false
	}
	c.trafficNotified = true
	c.active = false
	c.status = "等待下次执行"
	c.mu.Unlock()
	c.log("warning", fmt.Sprintf("已达到流量限制 %d MB", *c.Config.TrafficLimit), "#ffc107")
	return true
}

func (c *Consumer) stopDueToCount() {
	c.mu.Lock()
	if c.Config.Count == nil || c.countNotified {
		c.mu.Unlock()
		return
	}
	count := *c.Config.Count
	c.countNotified = true
	c.active = false
	c.status = "等待下次执行"
	c.mu.Unlock()
	c.log("warning", fmt.Sprintf("已达到下载次数限制 %d", count), "#ffc107")
}

func (c *Consumer) shouldSwitchURL(startedAt time.Time) bool {
	return c.Config.URLSwitchInterval != nil && time.Since(startedAt) >= time.Duration(*c.Config.URLSwitchInterval*float64(time.Second))
}

func (c *Consumer) handleAutoRemoveFailedURL(url string) {
	removedRuntime := c.URLMgr.RemoveURL(url)
	removedConfig, err := storage.RemoveURLFromConfig(c.ConfigDir.ConfigFile, c.Config.ConfigName, url)
	if err != nil {
		c.log("warning", "自动移除失败链接时写配置失败: "+err.Error(), "#ffc107")
	}
	if removedRuntime || removedConfig {
		c.log("warning", fmt.Sprintf("链接 %s 失败后已自动移除，剩余 %d 条可用链接", url, len(c.URLMgr.URLs())), "#ffc107")
	}
}

func (c *Consumer) httpClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: connectTimeout,
			}).DialContext,
			ResponseHeaderTimeout: readTimeout,
		},
	}
}

func (c *Consumer) logRetryFailure(threadID int, url string, attempt int, errorDetail string) {
	key := fmt.Sprintf("%s|%d|%s", url, attempt, errorDetail)
	c.mu.Lock()
	if c.retryLogKeys[key] {
		c.mu.Unlock()
		return
	}
	c.retryLogKeys[key] = true
	c.mu.Unlock()
	c.log("error", fmt.Sprintf("线程 %d 下载出错 (第%d次尝试/%d): %s", threadID, attempt, maxRetries, errorDetail), "#dc3545")
}

func (c *Consumer) log(level, message, color string) {
	if c.Logger != nil {
		c.Logger(level, message, color)
	}
}

func formatErrorDetail(err error) string {
	if err == nil {
		return "请求失败，但底层异常未返回详细信息。"
	}
	text := strings.TrimSpace(err.Error())
	if text == "" {
		return fmt.Sprintf("%T", err)
	}
	return text
}

func sleepWithContext(ctx context.Context, duration time.Duration) bool {
	if duration <= 0 {
		return true
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func closedDone() chan struct{} {
	done := make(chan struct{})
	close(done)
	return done
}
