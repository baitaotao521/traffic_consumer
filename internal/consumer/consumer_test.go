package consumer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/baitaotao521/traffic_consumer/internal/config"
	"github.com/baitaotao521/traffic_consumer/internal/model"
	"github.com/baitaotao521/traffic_consumer/internal/stats"
	"github.com/baitaotao521/traffic_consumer/internal/storage"
)

func TestConsumerDownloadsOnceAndWritesStats(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))
	defer testServer.Close()

	dir := t.TempDir()
	paths := config.Paths{
		ConfigDir:  dir,
		ConfigFile: filepath.Join(dir, "config.json"),
		StatsFile:  filepath.Join(dir, "stats.json"),
	}
	count := int64(1)
	cfg := model.Config{
		ConfigName:  "demo",
		URLs:        []string{testServer.URL},
		Threads:     1,
		Count:       &count,
		URLStrategy: "round_robin",
	}
	c := New(cfg, paths, stats.NewStore(paths.StatsFile), nil, nil)
	c.RunOnce(context.Background())

	data, err := os.ReadFile(paths.StatsFile)
	if err != nil {
		t.Fatalf("stats file not written: %v", err)
	}
	var runs map[string]model.StatsRun
	if err := json.Unmarshal(data, &runs); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected one stats run, got %d", len(runs))
	}
	for _, run := range runs {
		if run.DownloadCount != 1 || run.TotalBytes == 0 || run.ConfigName != "demo" {
			t.Fatalf("unexpected run: %+v", run)
		}
	}
}

func TestRunOnceCanBeStoppedWhileRequestBlocked(t *testing.T) {
	started := make(chan struct{})
	released := make(chan struct{})
	testServer := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(released)
	}))
	defer testServer.Close()

	dir := t.TempDir()
	paths := config.Paths{
		ConfigDir:  dir,
		ConfigFile: filepath.Join(dir, "config.json"),
		StatsFile:  filepath.Join(dir, "stats.json"),
	}
	cfg := model.Config{
		ConfigName:  "blocked",
		URLs:        []string{testServer.URL},
		Threads:     1,
		URLStrategy: "round_robin",
	}
	c := New(cfg, paths, stats.NewStore(paths.StatsFile), func(string, string, string) {}, nil)
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.RunOnce(context.Background())
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatalf("server did not receive blocked request")
	}
	c.Stop()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("RunOnce should stop after Consumer.Stop cancels the active request")
	}
	select {
	case <-released:
	case <-time.After(time.Second):
		t.Fatalf("blocked request context should be canceled")
	}
	history := c.History()
	if len(history) != 1 || history[0].Result != "已停止" {
		t.Fatalf("user-stopped run should record its finish reason: %+v", history)
	}
}

func TestRunOnceCanceledBeforeStartDoesNotWriteSuccessHistory(t *testing.T) {
	dir := t.TempDir()
	paths := config.Paths{
		ConfigDir:  dir,
		ConfigFile: filepath.Join(dir, "config.json"),
		StatsFile:  filepath.Join(dir, "stats.json"),
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := New(model.Config{
		ConfigName:  "pre-canceled",
		URLs:        []string{"https://example.com/file"},
		Threads:     1,
		URLStrategy: "round_robin",
	}, paths, stats.NewStore(paths.StatsFile), func(string, string, string) {}, nil)

	c.RunOnce(ctx)

	if history := c.History(); len(history) != 0 {
		t.Fatalf("pre-canceled run should not create a fake success history: %+v", history)
	}
}

func TestRunOnceParentContextCancellationRecordsStoppedResult(t *testing.T) {
	started := make(chan struct{})
	released := make(chan struct{})
	testServer := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(released)
	}))
	defer testServer.Close()

	dir := t.TempDir()
	paths := config.Paths{
		ConfigDir:  dir,
		ConfigFile: filepath.Join(dir, "config.json"),
		StatsFile:  filepath.Join(dir, "stats.json"),
	}
	ctx, cancel := context.WithCancel(context.Background())
	c := New(model.Config{
		ConfigName:  "parent-canceled",
		URLs:        []string{testServer.URL},
		Threads:     1,
		URLStrategy: "round_robin",
	}, paths, stats.NewStore(paths.StatsFile), func(string, string, string) {}, nil)
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.RunOnce(ctx)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatalf("server did not receive blocked request")
	}
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("RunOnce should stop after parent context cancellation")
	}
	select {
	case <-released:
	case <-time.After(time.Second):
		t.Fatalf("blocked request context should be canceled by parent context")
	}
	history := c.History()
	if len(history) != 1 || history[0].Result != "已停止" {
		t.Fatalf("parent-canceled run should record stopped result: %+v", history)
	}
}

func TestRateLimiterConsumesInitialTokens(t *testing.T) {
	limiter := NewRateLimiter(1024)
	limiter.Acquire(512)

	limiter.mu.Lock()
	tokens := limiter.tokens
	limiter.mu.Unlock()
	if tokens <= 0 || tokens >= 1024 {
		t.Fatalf("expected tokens to be partially consumed, got %.2f", tokens)
	}
}

func TestRateLimiterAcquireContextCanBeCanceled(t *testing.T) {
	limiter := NewRateLimiter(1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if limiter.AcquireContext(ctx, 1) {
		t.Fatalf("canceled context should interrupt limiter acquisition")
	}
}

func TestConsumerAppliesRequestHeadersAndUserAgent(t *testing.T) {
	requests := make(chan http.Header, 1)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Header.Clone()
		_, _ = w.Write([]byte("ok"))
	}))
	defer testServer.Close()

	dir := t.TempDir()
	paths := config.Paths{
		ConfigDir:  dir,
		ConfigFile: filepath.Join(dir, "config.json"),
		StatsFile:  filepath.Join(dir, "stats.json"),
	}
	count := int64(1)
	cfg := model.Config{
		ConfigName:     "headers",
		URLs:           []string{testServer.URL},
		Threads:        1,
		Count:          &count,
		UserAgent:      "TrafficConsumer-Test",
		RequestHeaders: model.HeaderMap{"Referer": "https://example.com", "User-Agent": "ignored"},
	}
	New(cfg, paths, stats.NewStore(paths.StatsFile), nil, nil).RunOnce(context.Background())

	select {
	case headers := <-requests:
		if headers.Get("Referer") != "https://example.com" {
			t.Fatalf("referer was not sent: %+v", headers)
		}
		if headers.Get("User-Agent") != "TrafficConsumer-Test" {
			t.Fatalf("user agent field should override request_headers: %+v", headers.Values("User-Agent"))
		}
	case <-time.After(time.Second):
		t.Fatalf("server did not receive request")
	}
}

func TestDownloadWithRetriesSkipsAlreadyInvalidURL(t *testing.T) {
	var requests int32
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&requests, 1)
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer testServer.Close()

	dir := t.TempDir()
	paths := config.Paths{
		ConfigDir:  dir,
		ConfigFile: filepath.Join(dir, "config.json"),
		StatsFile:  filepath.Join(dir, "stats.json"),
	}
	c := New(model.Config{
		ConfigName:  "invalid",
		URLs:        []string{testServer.URL},
		Threads:     1,
		URLStrategy: "round_robin",
	}, paths, stats.NewStore(paths.StatsFile), func(string, string, string) {}, nil)
	c.mu.Lock()
	c.active = true
	c.mu.Unlock()
	c.URLMgr.MarkInvalid(testServer.URL, "boom")

	if c.downloadWithRetries(context.Background(), testServer.Client(), testServer.URL, 1) {
		t.Fatalf("invalid URL should not complete a download")
	}
	if got := atomic.LoadInt32(&requests); got != 0 {
		t.Fatalf("already invalid URL should not be retried, requests=%d", got)
	}
}

func TestAutoRemoveFailedURLUpdatesRuntimeAndConfig(t *testing.T) {
	dir := t.TempDir()
	paths := config.Paths{
		ConfigDir:  dir,
		ConfigFile: filepath.Join(dir, "config.json"),
		StatsFile:  filepath.Join(dir, "stats.json"),
	}
	cfg := model.Config{
		ConfigName:          "demo",
		URLs:                []string{"https://example.com/a", "https://example.com/b"},
		Threads:             1,
		AutoRemoveFailedURL: true,
		URLStrategy:         "round_robin",
		ThreadStartDelay:    floatPtr(0),
		URLSwitchInterval:   nil,
		RequestHeaders:      nil,
		UserAgent:           "",
		Duration:            nil,
		Count:               nil,
		TrafficLimit:        nil,
		CronExpr:            "",
		Interval:            nil,
		AutoStart:           false,
	}
	if err := storage.SaveConfigEntry(paths.ConfigFile, "demo", cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	c := New(cfg, paths, stats.NewStore(paths.StatsFile), nil, nil)

	c.handleAutoRemoveFailedURL("https://example.com/a")

	if urls := c.URLMgr.URLs(); len(urls) != 1 || urls[0] != "https://example.com/b" {
		t.Fatalf("runtime URL was not removed: %+v", urls)
	}
	loaded, ok, err := storage.LoadConfigEntry(paths.ConfigFile, "demo")
	if err != nil || !ok {
		t.Fatalf("load config ok=%v err=%v", ok, err)
	}
	if len(loaded.URLs) != 1 || loaded.URLs[0] != "https://example.com/b" {
		t.Fatalf("config URL was not removed: %+v", loaded.URLs)
	}
}

func TestShouldSwitchURLAfterConfiguredInterval(t *testing.T) {
	interval := 0.1
	c := New(model.Config{
		ConfigName:          "switch",
		URLs:                []string{"https://example.com/a"},
		Threads:             1,
		URLSwitchInterval:   &interval,
		ThreadStartDelay:    floatPtr(0),
		URLStrategy:         "round_robin",
		AutoRemoveFailedURL: false,
	}, config.Paths{}, stats.NewStore(filepath.Join(t.TempDir(), "stats.json")), nil, nil)

	if c.shouldSwitchURL(time.Now()) {
		t.Fatalf("fresh URL should not switch immediately")
	}
	if !c.shouldSwitchURL(time.Now().Add(-time.Second)) {
		t.Fatalf("expired URL interval should request a switch")
	}
}

func TestConsumerRecordsBusinessFinishReasons(t *testing.T) {
	testCases := []struct {
		name    string
		cfg     model.Config
		trigger func(*Consumer)
		want    string
	}{
		{
			name: "duration limit",
			trigger: func(c *Consumer) {
				c.stopWithReason("达到时长限制")
			},
			want: "达到时长限制",
		},
		{
			name: "traffic limit",
			cfg:  model.Config{TrafficLimit: int64Ptr(1)},
			trigger: func(c *Consumer) {
				c.totalBytes = 1024 * 1024
				c.checkTrafficLimit()
			},
			want: "达到流量限制",
		},
		{
			name: "count limit",
			cfg:  model.Config{Count: int64Ptr(1)},
			trigger: func(c *Consumer) {
				c.downloadCount = 1
				c.stopDueToCount()
			},
			want: "达到下载次数限制",
		},
		{
			name: "no available URL",
			trigger: func(c *Consumer) {
				c.stopWithReason("无可用链接")
			},
			want: "无可用链接",
		},
		{
			name: "all URLs invalid",
			trigger: func(c *Consumer) {
				c.stopWithReason("全部链接失效")
			},
			want: "全部链接失效",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			paths := config.Paths{
				ConfigDir:  dir,
				ConfigFile: filepath.Join(dir, "config.json"),
				StatsFile:  filepath.Join(dir, "stats.json"),
			}
			cfg := tc.cfg
			cfg.ConfigName = "finish-reason"
			cfg.URLs = []string{"https://example.com/file"}
			cfg.Threads = 1
			cfg.URLStrategy = "round_robin"
			c := New(cfg, paths, stats.NewStore(paths.StatsFile), func(string, string, string) {}, nil)

			c.prepareRun()
			tc.trigger(c)
			c.finishRun()

			history := c.History()
			if len(history) != 1 || history[0].Result != tc.want {
				t.Fatalf("run should record finish reason %q: %+v", tc.want, history)
			}
			runs, err := stats.NewStore(paths.StatsFile).LoadRuns()
			if err != nil {
				t.Fatalf("load stats: %v", err)
			}
			for _, run := range runs {
				if run.Result != tc.want {
					t.Fatalf("stats run should persist finish reason %q: %+v", tc.want, run)
				}
				return
			}
			t.Fatalf("expected a persisted stats run")
		})
	}
}

func floatPtr(value float64) *float64 {
	return &value
}

func int64Ptr(value int64) *int64 {
	return &value
}
