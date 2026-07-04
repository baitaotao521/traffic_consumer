package runtime

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	byName := plansByName(plans)
	if byName["one"].SchedulerRunning || !byName["two"].SchedulerRunning {
		t.Fatalf("other scheduled plan should remain active: %+v", plans)
	}
}

func TestStartConsumerAllowsMultipleConfigRuns(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	paths := runtimeTestPaths(t)
	manager := New(paths)
	t.Cleanup(func() { manager.StopAll() })

	if _, err := manager.StartConsumer(model.Config{
		ConfigName:  "one",
		URLs:        []string{server.URL},
		Threads:     1,
		URLStrategy: "round_robin",
	}); err != nil {
		t.Fatalf("start one: %v", err)
	}
	if _, err := manager.StartConsumer(model.Config{
		ConfigName:  "two",
		URLs:        []string{server.URL},
		Threads:     1,
		URLStrategy: "round_robin",
	}); err != nil {
		t.Fatalf("start two: %v", err)
	}

	waitUntil(t, 3*time.Second, func() bool {
		plans := plansByName(manager.PlanSummaries())
		return plans["one"].Running && plans["two"].Running
	})

	if !manager.StopRuntimeConfig("one") {
		t.Fatalf("expected one to stop")
	}
	waitUntil(t, 3*time.Second, func() bool {
		plans := plansByName(manager.PlanSummaries())
		return !plans["one"].Running && plans["two"].Running
	})
}

func TestStartConsumerRejectsDuplicateActiveConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	paths := runtimeTestPaths(t)
	manager := New(paths)
	t.Cleanup(func() { manager.StopAll() })

	cfg := model.Config{
		ConfigName:  "demo",
		URLs:        []string{server.URL},
		Threads:     1,
		URLStrategy: "round_robin",
	}
	if _, err := manager.StartConsumer(cfg); err != nil {
		t.Fatalf("start demo: %v", err)
	}
	if _, err := manager.StartConsumer(cfg); err == nil || !strings.Contains(err.Error(), "已在运行或等待调度") {
		t.Fatalf("expected duplicate start conflict, got %v", err)
	}
	if count := countPlanName(manager.PlanSummaries(), "demo"); count != 1 {
		t.Fatalf("duplicate start should not duplicate plan, got %d plans: %+v", count, manager.PlanSummaries())
	}
}

func TestStartConsumerRejectsConcurrentDuplicateConfig(t *testing.T) {
	paths := runtimeTestPaths(t)
	manager := New(paths)
	t.Cleanup(func() { manager.StopAll() })

	interval := int64(1)
	cfg := model.Config{
		ConfigName:  "demo",
		URLs:        []string{},
		Threads:     1,
		Interval:    &interval,
		URLStrategy: "round_robin",
	}
	const attempts = 12
	errs := make(chan error, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := manager.StartConsumer(cfg)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	successes := 0
	conflicts := 0
	for err := range errs {
		if err == nil {
			successes++
			continue
		}
		if strings.Contains(err.Error(), "已在运行或等待调度") {
			conflicts++
			continue
		}
		t.Fatalf("unexpected concurrent start error: %v", err)
	}
	if successes != 1 || conflicts != attempts-1 {
		t.Fatalf("expected one start and %d conflicts, got starts=%d conflicts=%d", attempts-1, successes, conflicts)
	}
	plans := plansByName(manager.PlanSummaries())
	if count := countPlanName(manager.PlanSummaries(), "demo"); count != 1 || !plans["demo"].SchedulerRunning {
		t.Fatalf("concurrent duplicate start should leave one scheduled row, count=%d plans=%+v", count, plans)
	}
}

func TestPlanSummariesClearCompletedRunMetricsAndKeepLastRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	paths := runtimeTestPaths(t)
	if err := storage.SaveConfigEntry(paths.ConfigFile, "demo", model.Config{
		URLs:        []string{server.URL},
		Threads:     1,
		URLStrategy: "round_robin",
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	manager := New(paths)
	t.Cleanup(func() { manager.StopAll() })

	count := int64(1)
	if _, err := manager.StartConsumer(model.Config{
		ConfigName:  "demo",
		URLs:        []string{server.URL},
		Threads:     1,
		Count:       &count,
		URLStrategy: "round_robin",
	}); err != nil {
		t.Fatalf("start demo: %v", err)
	}

	var plan model.PlanSummary
	var plans []model.PlanSummary
	waitUntilWithMessage(t, 3*time.Second, func() bool {
		plans = manager.PlanSummaries()
		plans := plansByName(manager.PlanSummaries())
		plan = plans["demo"]
		return plan.Name == "demo" && !plan.Running && plan.LastRun != nil
	}, func() string {
		runs, _ := manager.Stats.LoadRuns()
		return fmt.Sprintf("plans=%+v runs=%+v", plans, runs)
	})
	if plan.DownloadCount != 0 || plan.TotalBytesRaw != 0 || plan.TotalBytes != "0.00 B" {
		t.Fatalf("completed run metrics should be cleared from the stopped row: %+v", plan)
	}
	if plan.LastRun == nil || plan.LastRun.DownloadCount != 1 || plan.LastRun.Bytes != 2 {
		t.Fatalf("completed run result should be visible on plan row: %+v", plan.LastRun)
	}
}

func TestScheduledRunKeepsRuntimeItemAfterCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	paths := runtimeTestPaths(t)
	manager := New(paths)
	t.Cleanup(func() { manager.StopAll() })

	interval := int64(1)
	count := int64(1)
	if _, err := manager.StartConsumer(model.Config{
		ConfigName:  "demo",
		URLs:        []string{server.URL},
		Threads:     1,
		Count:       &count,
		Interval:    &interval,
		URLStrategy: "round_robin",
	}); err != nil {
		t.Fatalf("start scheduled demo: %v", err)
	}
	items := manager.items()
	if len(items) != 1 || items[0].Cron == nil {
		t.Fatalf("expected one scheduled runtime item, got %+v", items)
	}

	manager.runScheduledItem(items[0], context.Background())
	plans := plansByName(manager.PlanSummaries())
	if !plans["demo"].SchedulerRunning || plans["demo"].Running {
		t.Fatalf("scheduled runtime item should remain waiting after run completion: %+v", plans["demo"])
	}
	if plans["demo"].TotalBytesRaw != 0 || plans["demo"].DownloadCount != 0 || plans["demo"].LastRun == nil {
		t.Fatalf("scheduled run should clear row metrics and keep last result: %+v", plans["demo"])
	}
}

func TestStopRuntimeConfigRecordsRunAndClearsCurrentSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	paths := runtimeTestPaths(t)
	if err := storage.SaveConfigEntry(paths.ConfigFile, "demo", model.Config{
		URLs:        []string{server.URL},
		Threads:     1,
		URLStrategy: "round_robin",
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	manager := New(paths)
	t.Cleanup(func() { manager.StopAll() })

	if _, err := manager.StartConsumer(model.Config{
		ConfigName:  "demo",
		URLs:        []string{server.URL},
		Threads:     1,
		URLStrategy: "round_robin",
	}); err != nil {
		t.Fatalf("start demo: %v", err)
	}
	waitUntil(t, 3*time.Second, func() bool {
		plans := plansByName(manager.PlanSummaries())
		return plans["demo"].Running && plans["demo"].DownloadCount > 0
	})

	if !manager.StopRuntimeConfig("demo") {
		t.Fatalf("expected demo to stop")
	}
	plans := plansByName(manager.PlanSummaries())
	plan := plans["demo"]
	if plan.Running || plan.Speed != "0 B/s" {
		t.Fatalf("stopped plan should clear live snapshot: %+v", plan)
	}
	if plan.LastRun == nil || plan.LastRun.DownloadCount <= 0 {
		t.Fatalf("stopped plan should expose execution record: %+v", plan.LastRun)
	}
	if plan.DownloadCount != 0 || plan.TotalBytesRaw != 0 {
		t.Fatalf("stopped plan should clear current-run metrics from the row: %+v", plan)
	}
}

func TestStartTempTestRecordsResultForSavedConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		payload := make([]byte, 1024*1024)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	paths := runtimeTestPaths(t)
	cfg := model.Config{
		URLs:        []string{server.URL},
		Threads:     1,
		URLStrategy: "round_robin",
	}
	if err := storage.SaveConfigEntry(paths.ConfigFile, "demo", cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	manager := New(paths)
	t.Cleanup(func() { manager.StopAll() })

	trafficLimit := int64(1)
	cfg.ConfigName = "demo"
	cfg.TrafficLimit = &trafficLimit
	if _, err := manager.StartTempTest(cfg); err != nil {
		t.Fatalf("start temp test: %v", err)
	}
	waitUntil(t, 3*time.Second, func() bool {
		plans := plansByName(manager.PlanSummaries())
		return plans["demo"].LastRun != nil && plans["demo"].LastRun.Bytes >= 1024*1024
	})
}

func TestSaveAndDeleteConfigOnlyStopSameNamedRuntime(t *testing.T) {
	paths := runtimeTestPaths(t)
	manager := New(paths)
	t.Cleanup(func() { manager.StopAll() })

	interval := int64(1)
	for _, name := range []string{"one", "two"} {
		if _, err := manager.StartConsumer(model.Config{
			ConfigName:  name,
			URLs:        []string{},
			Threads:     1,
			Interval:    &interval,
			URLStrategy: "round_robin",
		}); err != nil {
			t.Fatalf("start %s: %v", name, err)
		}
	}

	if err := manager.SaveConfig("one", model.Config{URLs: []string{}, Threads: 1}); err != nil {
		t.Fatalf("save one: %v", err)
	}
	plans := plansByName(manager.PlanSummaries())
	if plans["one"].SchedulerRunning || !plans["two"].SchedulerRunning {
		t.Fatalf("saving one should stop only one: %+v", plans)
	}

	if deleted, err := manager.DeleteConfig("one"); err != nil || !deleted {
		t.Fatalf("delete one deleted=%v err=%v", deleted, err)
	}
	plans = plansByName(manager.PlanSummaries())
	if plans["two"].Name != "two" || !plans["two"].SchedulerRunning {
		t.Fatalf("deleting one should keep two scheduled: %+v", plans)
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

func waitUntilWithMessage(t *testing.T, timeout time.Duration, condition func() bool, message func() string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("condition was not met within %s: %s", timeout, message())
}

func plansByName(plans []model.PlanSummary) map[string]model.PlanSummary {
	result := make(map[string]model.PlanSummary, len(plans))
	for _, plan := range plans {
		result[plan.Name] = plan
	}
	return result
}

func countPlanName(plans []model.PlanSummary, name string) int {
	count := 0
	for _, plan := range plans {
		if plan.Name == name {
			count++
		}
	}
	return count
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
