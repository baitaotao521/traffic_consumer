package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/baitaotao521/traffic_consumer/internal/config"
	"github.com/baitaotao521/traffic_consumer/internal/model"
	"github.com/baitaotao521/traffic_consumer/internal/stats"
	"github.com/baitaotao521/traffic_consumer/internal/storage"
)

func TestBuildConfigPreservesExplicitEmptyURLs(t *testing.T) {
	paths := testPaths(t)
	if err := storage.SaveConfigEntry(paths.ConfigFile, "empty", model.Config{URLs: []string{}, Threads: 1}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	cfg, err := buildConfig(Args{LoadConfig: true, Config: "empty", URLStrategy: "random", Threads: 8}, paths)
	if err != nil {
		t.Fatalf("build config: %v", err)
	}
	if cfg.URLs == nil {
		t.Fatalf("explicit empty URL list should stay non-nil")
	}
	if len(cfg.URLs) != 0 {
		t.Fatalf("explicit empty URL list should not fall back to defaults: %+v", cfg.URLs)
	}
}

func TestBuildConfigUsesDefaultsWhenURLsAreMissing(t *testing.T) {
	paths := testPaths(t)
	cfg, err := buildConfig(Args{Config: "default", URLStrategy: "random", Threads: 8}, paths)
	if err != nil {
		t.Fatalf("build config: %v", err)
	}
	if len(cfg.URLs) != len(config.DefaultURLs) {
		t.Fatalf("missing URLs should use defaults, got %+v", cfg.URLs)
	}
}

func TestScheduleSpecUsesCronOrInterval(t *testing.T) {
	spec, description, err := scheduleSpec(model.Config{CronExpr: "*/5 * * * *"})
	if err != nil {
		t.Fatalf("cron spec: %v", err)
	}
	if spec != "*/5 * * * *" || description == "" {
		t.Fatalf("unexpected cron spec=%q description=%q", spec, description)
	}

	interval := int64(15)
	spec, description, err = scheduleSpec(model.Config{Interval: &interval})
	if err != nil {
		t.Fatalf("interval spec: %v", err)
	}
	if spec != "@every 15m" || description == "" {
		t.Fatalf("unexpected interval spec=%q description=%q", spec, description)
	}
}

func TestRunScheduledStopsWhenContextIsCanceled(t *testing.T) {
	paths := testPaths(t)
	interval := int64(1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runScheduled(ctx, model.Config{
		ConfigName:  "demo",
		URLs:        []string{"http://127.0.0.1/unused"},
		Threads:     1,
		Interval:    &interval,
		URLStrategy: "round_robin",
	}, paths, stats.NewStore(paths.StatsFile), func(level, message, color string) {})
	if err != nil {
		t.Fatalf("run scheduled: %v", err)
	}
}

func TestShowStatsIncludesPythonCLIFields(t *testing.T) {
	paths := testPaths(t)
	store := stats.NewStore(paths.StatsFile)
	if err := store.SaveRun(model.StatsRun{
		ConfigName:     "demo",
		StartTime:      "2026-07-04 00:00:00",
		EndTime:        "2026-07-04 00:00:01",
		Result:         "成功",
		TotalBytes:     1024,
		DownloadCount:  3,
		ElapsedSeconds: 1,
	}); err != nil {
		t.Fatalf("save stats: %v", err)
	}

	output := captureStdout(t, func() {
		if err := showStats(store, 5); err != nil {
			t.Fatalf("show stats: %v", err)
		}
	})
	for _, expected := range []string{"配置名称: demo", "开始时间: 2026-07-04 00:00:00", "结束时间: 2026-07-04 00:00:01", "总消耗流量: 1.00 KB", "下载次数: 3"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("stats output missing %q:\n%s", expected, output)
		}
	}
}

func TestListConfigsIncludesPythonCLIFields(t *testing.T) {
	paths := testPaths(t)
	duration := int64(30)
	count := int64(2)
	interval := int64(15)
	switchInterval := 3.5
	startDelay := 0.5
	if err := storage.SaveConfigEntry(paths.ConfigFile, "demo", model.Config{
		URLs:                []string{"https://example.com/a"},
		URLStrategy:         "round_robin",
		Threads:             2,
		LimitSpeed:          1.5,
		Duration:            &duration,
		Count:               &count,
		CronExpr:            "*/5 * * * *",
		Interval:            &interval,
		AutoStart:           true,
		UserAgent:           "TrafficConsumer-Test",
		RequestHeaders:      model.HeaderMap{"Referer": "https://example.com"},
		URLSwitchInterval:   &switchInterval,
		ThreadStartDelay:    &startDelay,
		AutoRemoveFailedURL: true,
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output := captureStdout(t, func() {
		if err := listConfigs(paths.ConfigFile); err != nil {
			t.Fatalf("list configs: %v", err)
		}
	})
	for _, expected := range []string{"持续时间: 30s", "下载次数: 2", "Cron表达式: */5 * * * *", "间隔执行: 每 15 分钟", "Web启动自启: 开启", "User-Agent: TrafficConsumer-Test", "自定义请求头:", "URL切换时限: 3.5 秒", "线程启动间隔: 0.5 秒", "失败链接自动移除: 开启"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("config output missing %q:\n%s", expected, output)
		}
	}
}

func testPaths(t *testing.T) config.Paths {
	t.Helper()
	dir := t.TempDir()
	return config.Paths{
		ConfigDir:  dir,
		ConfigFile: filepath.Join(dir, "config.json"),
		StatsFile:  filepath.Join(dir, "stats.json"),
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	os.Stdout = writer
	fn()
	_ = writer.Close()
	os.Stdout = old
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return string(data)
}
