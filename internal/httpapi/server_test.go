package httpapi

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/baitaotao521/traffic_consumer/internal/config"
	"github.com/baitaotao521/traffic_consumer/internal/model"
	appruntime "github.com/baitaotao521/traffic_consumer/internal/runtime"
	"github.com/baitaotao521/traffic_consumer/internal/storage"
)

func TestDashboardReturnsAggregateSnapshot(t *testing.T) {
	server := newTestServer(t)
	payload := model.Config{URLs: []string{"https://example.com/a"}, Threads: 2, URLStrategy: "random"}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/api/configurations/demo", bytes.NewReader(data))
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("save status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response DashboardResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode dashboard: %v", err)
	}
	if len(response.Dashboard.Configs) != 1 || response.Dashboard.Configs[0].Name != "demo" {
		t.Fatalf("unexpected configs: %+v", response.Dashboard.Configs)
	}
	if response.Dashboard.Status.State != model.StateStopped {
		t.Fatalf("unexpected status: %+v", response.Dashboard.Status)
	}
	if response.Dashboard.Status.ThreadMetrics == nil {
		t.Fatalf("dashboard status should include thread metrics: %+v", response.Dashboard.Status)
	}
	if response.Dashboard.Plans == nil {
		t.Fatalf("dashboard should normalize plan slices: %+v", response.Dashboard)
	}
	plan, ok := findPlan(response.Dashboard.Plans, "demo")
	if !ok || plan.Running || plan.Threads != 2 || plan.Speed != "0 B/s" {
		t.Fatalf("saved stopped config should appear as a stopped plan row: %+v", response.Dashboard.Plans)
	}
	var raw struct {
		Dashboard struct {
			Scheduler map[string]any `json:"scheduler"`
		} `json:"dashboard"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode raw dashboard: %v", err)
	}
	if _, ok := raw.Dashboard.Scheduler["plans"]; ok {
		t.Fatalf("scheduler should not duplicate dashboard plans: %s", rec.Body.String())
	}
}

func TestDashboardSupportsGzipCompression(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("dashboard should be gzip compressed when accepted, got %q", got)
	}
	reader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("open gzip body: %v", err)
	}
	defer reader.Close()
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read gzip body: %v", err)
	}
	var response DashboardResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode compressed dashboard: %v", err)
	}
	if response.Dashboard.Status.State != model.StateStopped {
		t.Fatalf("unexpected compressed dashboard status: %+v", response.Dashboard.Status)
	}
}

func TestStartRuntimeReturnsDashboardWithRunningConfigMetrics(t *testing.T) {
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer downloadServer.Close()

	server := newTestServer(t)
	t.Cleanup(func() { server.Manager.StopAll() })
	payload := model.Config{ConfigName: "demo", URLs: []string{downloadServer.URL}, Threads: 1, URLStrategy: "round_robin"}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/commands/start-runtime", bytes.NewReader(data))
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", rec.Code, rec.Body.String())
	}
	var started RuntimeCommandResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	plan, ok := findPlan(started.Dashboard.Plans, "demo")
	if !ok || !plan.Running || plan.Threads != 1 || plan.Speed == "" {
		t.Fatalf("running config metrics missing: %+v", started.Dashboard.Plans)
	}
	if started.Dashboard.Status.State != model.StateRunning || started.Dashboard.Status.ThreadCount != 1 {
		t.Fatalf("aggregate status should reflect running config: %+v", started.Dashboard.Status)
	}
}

func TestConfigurationCommandReturnsUpdatedDashboard(t *testing.T) {
	server := newTestServer(t)
	payload := model.Config{URLs: []string{"https://example.com/a"}, Threads: 2, URLStrategy: "random"}
	data, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/configurations/demo", bytes.NewReader(data))
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("save status=%d body=%s", rec.Code, rec.Body.String())
	}
	var saved ConfigCommandResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &saved); err != nil {
		t.Fatalf("decode save response: %v", err)
	}
	if saved.Item == nil || saved.Item.Name != "demo" || len(saved.Dashboard.Configs) != 1 {
		t.Fatalf("unexpected save response: %+v", saved)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/configurations/demo", nil)
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rec.Code, rec.Body.String())
	}
	var detail ConfigResourceResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if detail.Item.Name != "demo" || detail.Item.Config.Threads != 2 {
		t.Fatalf("unexpected response: %+v", detail)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/configurations/demo", nil)
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
	}
	var deleted ConfigCommandResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &deleted); err != nil {
		t.Fatalf("decode delete response: %v", err)
	}
	if len(deleted.Dashboard.Configs) != 0 {
		t.Fatalf("delete should return refreshed dashboard: %+v", deleted.Dashboard.Configs)
	}
}

func TestSaveConfigurationValidatesAndRejectsDuplicateCreate(t *testing.T) {
	server := newTestServer(t)
	initial := model.Config{URLs: []string{"https://example.com/a"}, Threads: 2, URLStrategy: "random"}
	if err := storage.SaveConfigEntry(server.Manager.Paths.ConfigFile, "demo", initial); err != nil {
		t.Fatalf("seed demo: %v", err)
	}

	invalid := model.Config{URLs: []string{"https://example.com/a"}, Threads: 0, URLStrategy: "random"}
	data, _ := json.Marshal(invalid)
	req := httptest.NewRequest(http.MethodPut, "/api/configurations/bad?mode=create", bytes.NewReader(data))
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid save should return 400, status=%d body=%s", rec.Code, rec.Body.String())
	}

	replacement := model.Config{Name: "demo", URLs: []string{"https://example.com/b"}, Threads: 3, URLStrategy: "random"}
	data, _ = json.Marshal(replacement)
	req = httptest.NewRequest(http.MethodPut, "/api/configurations/demo?mode=create", bytes.NewReader(data))
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("create existing should return conflict, status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/api/configurations/demo?mode=create&force=true", bytes.NewReader(data))
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("forced create duplicate should still conflict, status=%d body=%s", rec.Code, rec.Body.String())
	}
	cfg, ok, err := storage.LoadConfigEntry(server.Manager.Paths.ConfigFile, "demo")
	if err != nil || !ok || cfg.Threads != 2 {
		t.Fatalf("duplicate create should keep original target ok=%v err=%v cfg=%+v", ok, err, cfg)
	}

	sameNameEdit := model.Config{Name: "demo", ConfigName: "demo", URLs: []string{"https://example.com/c"}, Threads: 4, URLStrategy: "round_robin"}
	data, _ = json.Marshal(sameNameEdit)
	req = httptest.NewRequest(http.MethodPut, "/api/configurations/demo?mode=edit", bytes.NewReader(data))
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("same-name edit should not require force, status=%d body=%s", rec.Code, rec.Body.String())
	}
	cfg, ok, err = storage.LoadConfigEntry(server.Manager.Paths.ConfigFile, "demo")
	if err != nil || !ok || cfg.Threads != 4 || cfg.URLStrategy != "round_robin" {
		t.Fatalf("same-name edit should update target ok=%v err=%v cfg=%+v", ok, err, cfg)
	}
}

func TestSaveConfigurationRenamesRejectExistingTarget(t *testing.T) {
	server := newTestServer(t)
	if err := storage.SaveConfigEntry(server.Manager.Paths.ConfigFile, "one", model.Config{URLs: []string{"https://example.com/one"}, Threads: 1, URLStrategy: "random"}); err != nil {
		t.Fatalf("seed one: %v", err)
	}
	if err := storage.SaveConfigEntry(server.Manager.Paths.ConfigFile, "two", model.Config{URLs: []string{"https://example.com/two"}, Threads: 2, URLStrategy: "random"}); err != nil {
		t.Fatalf("seed two: %v", err)
	}

	rename := model.Config{Name: "two", ConfigName: "two", URLs: []string{"https://example.com/new"}, Threads: 4, URLStrategy: "round_robin"}
	data, _ := json.Marshal(rename)
	req := httptest.NewRequest(http.MethodPut, "/api/configurations/one?mode=edit", bytes.NewReader(data))
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate rename should conflict, status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/api/configurations/one?mode=edit&force=true", bytes.NewReader(data))
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("forced rename duplicate should still conflict, status=%d body=%s", rec.Code, rec.Body.String())
	}
	if cfg, ok, _ := storage.LoadConfigEntry(server.Manager.Paths.ConfigFile, "one"); !ok || cfg.Threads != 1 {
		t.Fatalf("source config should remain after rejected rename ok=%v cfg=%+v", ok, cfg)
	}
	cfg, ok, err := storage.LoadConfigEntry(server.Manager.Paths.ConfigFile, "two")
	if err != nil || !ok || cfg.Threads != 2 || cfg.URLStrategy != "random" {
		t.Fatalf("target config should remain after rejected rename ok=%v err=%v cfg=%+v", ok, err, cfg)
	}
}

func TestSaveConfigurationRenamesToUnusedTarget(t *testing.T) {
	server := newTestServer(t)
	if err := storage.SaveConfigEntry(server.Manager.Paths.ConfigFile, "one", model.Config{URLs: []string{"https://example.com/one"}, Threads: 1, URLStrategy: "random"}); err != nil {
		t.Fatalf("seed one: %v", err)
	}

	rename := model.Config{Name: "three", ConfigName: "three", URLs: []string{"https://example.com/three"}, Threads: 3, URLStrategy: "round_robin"}
	data, _ := json.Marshal(rename)
	req := httptest.NewRequest(http.MethodPut, "/api/configurations/one?mode=edit", bytes.NewReader(data))
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("rename to unused target status=%d body=%s", rec.Code, rec.Body.String())
	}
	if _, ok, _ := storage.LoadConfigEntry(server.Manager.Paths.ConfigFile, "one"); ok {
		t.Fatalf("source config should be removed after rename")
	}
	cfg, ok, err := storage.LoadConfigEntry(server.Manager.Paths.ConfigFile, "three")
	if err != nil || !ok || cfg.Threads != 3 || cfg.URLStrategy != "round_robin" {
		t.Fatalf("target config should be saved ok=%v err=%v cfg=%+v", ok, err, cfg)
	}
}

func TestRuntimeCommandsValidateWebConfigBeforeStart(t *testing.T) {
	server := newTestServer(t)
	payload := model.Config{ConfigName: "demo", URLs: []string{}, Threads: 1, URLStrategy: "random"}
	data, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/commands/start-runtime", bytes.NewReader(data))
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("start empty URLs should return 400, status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/commands/start-temp-test", bytes.NewReader(data))
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("temp test empty URLs should return 400, status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPreviewCronReturnsFiveRuns(t *testing.T) {
	server := newTestServer(t)
	body := bytes.NewBufferString(`{"expression":"*/5 * * * *"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/cron/preview", body)
	rec := httptest.NewRecorder()

	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response CronPreviewResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Items) != 5 {
		t.Fatalf("expected 5 runs, got %d", len(response.Items))
	}
}

func TestActivityUsesSeparateLogAndEventCursors(t *testing.T) {
	server := newTestServer(t)
	server.Manager.AddLog("demo", model.LogSourceScheduler, "info", "hello", "")
	server.Manager.AddLog("demo", model.LogSourceDownload, "error", "boom", "")
	server.Manager.AddEvent(model.Event{Type: "status_update", Message: "done"})

	req := httptest.NewRequest(http.MethodGet, "/api/activity?after_log=0&after_event=0&log_sources=scheduler&log_levels=info", nil)
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("activity status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response ActivityResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode activity: %v", err)
	}
	if len(response.Logs) != 1 || response.Logs[0].Source != model.LogSourceScheduler || len(response.Events) != 1 || response.LogCursor != 2 || response.EventCursor != 1 {
		t.Fatalf("unexpected activity response: %+v", response)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/activity?after_log=2&after_event=1", nil)
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("activity status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode activity: %v", err)
	}
	if len(response.Logs) != 0 || len(response.Events) != 0 || response.LogCursor != 2 || response.EventCursor != 1 {
		t.Fatalf("cursor filtering failed: %+v", response)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/activity?after_log=0&after_event=0&include_logs=false", nil)
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("activity status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode activity: %v", err)
	}
	if len(response.Logs) != 0 || response.LogCursor != 0 || len(response.Events) != 1 || response.EventCursor != 1 {
		t.Fatalf("include_logs=false should preserve log cursor: %+v", response)
	}
}

func TestOpenAPISchemaAndOldRoutes(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil)
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("openapi status=%d body=%s", rec.Code, rec.Body.String())
	}
	var spec struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]any `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
		Paths map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &spec); err != nil {
		t.Fatalf("decode openapi: %v", err)
	}
	if _, ok := spec.Paths["/dashboard"]; !ok {
		t.Fatalf("new dashboard path missing from OpenAPI: %+v", spec.Paths)
	}
	if _, ok := spec.Components.Schemas["DashboardResponse"].Properties["$schema"]; ok {
		t.Fatalf("OpenAPI response schema should not expose Huma metadata field")
	}
	if _, ok := spec.Components.Schemas["SchedulerPayload"].Properties["plans"]; ok {
		t.Fatalf("scheduler schema should not duplicate dashboard plans")
	}
	if _, ok := spec.Paths["/runtime/status"]; ok {
		t.Fatalf("old runtime path should not be present in OpenAPI")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard status=%d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Header().Get("Link"), "describedby") {
		t.Fatalf("dashboard response should not advertise schema metadata: %s", rec.Header().Get("Link"))
	}
	var dashboard map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &dashboard); err != nil {
		t.Fatalf("decode dashboard body: %v", err)
	}
	if _, ok := dashboard["$schema"]; ok {
		t.Fatalf("dashboard response body should not include Huma metadata: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/runtime/status", nil)
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("old runtime route should be gone, status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/schemas/DashboardResponse.json", nil)
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("schema route should be disabled, status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestIndexServesVueBundle(t *testing.T) {
	baseDir := t.TempDir()
	distDir := filepath.Join(baseDir, "frontend", "dist")
	if err := os.MkdirAll(filepath.Join(distDir, "assets"), 0o755); err != nil {
		t.Fatalf("create frontend dist: %v", err)
	}
	index := `<!DOCTYPE html><script type="module" src="/assets/app.js"></script>`
	if err := os.WriteFile(filepath.Join(distDir, "index.html"), []byte(index), 0o644); err != nil {
		t.Fatalf("write frontend index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "assets", "app.js"), []byte("console.log('ok')"), 0o644); err != nil {
		t.Fatalf("write frontend asset: %v", err)
	}

	server := newTestServerWithBaseDir(t, baseDir)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("index status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "/assets/app.js") {
		t.Fatalf("index should load Vue dashboard bundle: %s", body)
	}
	if strings.Contains(body, "/static/") {
		t.Fatalf("index should not load retired static directory: %s", body)
	}

	req = httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "console.log('ok')" {
		t.Fatalf("asset route status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	return newTestServerWithBaseDir(t, t.TempDir())
}

func newTestServerWithBaseDir(t *testing.T, baseDir string) *Server {
	t.Helper()
	dir := t.TempDir()
	paths := config.Paths{
		ConfigDir:  dir,
		ConfigFile: filepath.Join(dir, "config.json"),
		StatsFile:  filepath.Join(dir, "stats.json"),
	}
	return New(appruntime.New(paths), baseDir)
}

func findPlan(plans []model.PlanSummary, name string) (model.PlanSummary, bool) {
	for _, plan := range plans {
		if plan.Name == name {
			return plan, true
		}
	}
	return model.PlanSummary{}, false
}
