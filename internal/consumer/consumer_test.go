package consumer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func floatPtr(value float64) *float64 {
	return &value
}
