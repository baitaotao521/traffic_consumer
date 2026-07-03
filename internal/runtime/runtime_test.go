package runtime

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/baitaotao521/traffic_consumer/internal/config"
	"github.com/baitaotao521/traffic_consumer/internal/model"
	"github.com/baitaotao521/traffic_consumer/internal/storage"
)

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
