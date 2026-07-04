package runtime

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/baitaotao521/traffic_consumer/internal/config"
	"github.com/baitaotao521/traffic_consumer/internal/model"
	"github.com/baitaotao521/traffic_consumer/internal/storage"
)

func TestAddLogWritesConsoleAndSupportsFilters(t *testing.T) {
	manager := New(runtimeTestPaths(t))

	output := captureStdout(t, func() {
		manager.AddLog("demo", model.LogSourceScheduler, "info", "计划已登记。", "")
		manager.AddLog("demo", model.LogSourceDownload, "error", "下载失败。", "")
	})

	if !strings.Contains(output, "[scheduler]") || !strings.Contains(output, "计划已登记。") {
		t.Fatalf("console output should contain scheduler log, got %q", output)
	}
	if !strings.Contains(output, "[download]") || !strings.Contains(output, "下载失败。") {
		t.Fatalf("console output should contain download log, got %q", output)
	}

	logs, cursor := manager.LogsSince(0, []model.LogSource{model.LogSourceScheduler}, []string{"info"})
	if cursor != 2 {
		t.Fatalf("expected latest cursor 2, got %d", cursor)
	}
	if len(logs) != 1 || logs[0].Source != model.LogSourceScheduler || logs[0].Level != "info" {
		t.Fatalf("unexpected filtered logs: %+v", logs)
	}
}

func TestRunScheduledItemEmitsLifecycleLogs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	paths := runtimeTestPaths(t)
	manager := New(paths)
	t.Cleanup(func() { manager.StopAll() })

	interval := int64(1)
	count := int64(1)
	item, err := manager.createItem("demo", model.Config{
		ConfigName:  "demo",
		URLs:        []string{server.URL},
		Threads:     1,
		Count:       &count,
		Interval:    &interval,
		URLStrategy: "round_robin",
	}, false, nil)
	if err != nil {
		t.Fatalf("create item: %v", err)
	}
	if err := manager.startScheduler(item); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}

	manager.runScheduledItem(item, context.Background())

	logs, _ := manager.LogsSince(0, []model.LogSource{model.LogSourceScheduler}, []string{"info"})
	joined := make([]string, 0, len(logs))
	for _, entry := range logs {
		joined = append(joined, entry.Message)
	}
	text := strings.Join(joined, "\n")
	if !strings.Contains(text, "已登记间隔调度") {
		t.Fatalf("scheduler setup log missing: %s", text)
	}
	if !strings.Contains(text, "计划任务已触发执行") {
		t.Fatalf("scheduler start log missing: %s", text)
	}
	if !strings.Contains(text, "计划任务执行完成") {
		t.Fatalf("scheduler completion log missing: %s", text)
	}
}

func TestStopRuntimeConfigKeepsOtherAutoStartPlans(t *testing.T) {
	paths := runtimeTestPaths(t)
	interval := int64(1)
	for _, name := range []string{"one", "two"} {
		if err := storage.SaveConfigEntry(paths.ConfigFile, name, model.Config{
			URLs:      []string{},
			Threads:   1,
			Interval:  &interval,
			AutoStart: true,
		}); err != nil {
			t.Fatalf("save %s: %v", name, err)
		}
	}

	manager := New(paths)
	t.Cleanup(func() { manager.StopAll() })
	if !manager.LaunchAutoStartConfigs() {
		t.Fatalf("expected autostart plans")
	}
	if !manager.StopRuntimeConfig("one") {
		t.Fatalf("expected one plan to stop")
	}

	plans := manager.PlanSummaries()
	if len(plans) != 1 || plans[0].Name != "two" || !plans[0].SchedulerRunning {
		t.Fatalf("other scheduled plan should remain active: %+v", plans)
	}
}

func TestTempTestRestoresPausedScheduledPlan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		payload := make([]byte, 1024*1024)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	paths := runtimeTestPaths(t)
	manager := New(paths)
	t.Cleanup(func() { manager.StopAll() })

	interval := int64(1)
	trafficLimit := int64(1)
	cfg := model.Config{
		ConfigName:   "demo",
		URLs:         []string{server.URL},
		Threads:      1,
		Interval:     &interval,
		TrafficLimit: &trafficLimit,
		URLStrategy:  "round_robin",
	}
	if _, err := manager.StartConsumer(cfg); err != nil {
		t.Fatalf("start scheduled consumer: %v", err)
	}
	if _, err := manager.StartTempTest(cfg); err != nil {
		t.Fatalf("start temp test: %v", err)
	}

	waitUntil(t, 5*time.Second, func() bool {
		for _, plan := range manager.PlanSummaries() {
			if plan.Name == "demo" && plan.SchedulerRunning && !plan.Running {
				return true
			}
		}
		return false
	})
}

func runtimeTestPaths(t *testing.T) config.Paths {
	t.Helper()
	dir := t.TempDir()
	return config.Paths{
		ConfigDir:  dir,
		ConfigFile: filepath.Join(dir, "config.json"),
		StatsFile:  filepath.Join(dir, "stats.json"),
	}
}

func waitUntil(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("condition was not met within %s", timeout)
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = original
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		t.Fatalf("read console output: %v", err)
	}
	return buf.String()
}
