package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/baitaotao521/traffic_consumer/internal/config"
	"github.com/baitaotao521/traffic_consumer/internal/model"
	appruntime "github.com/baitaotao521/traffic_consumer/internal/runtime"
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
	if response.Dashboard.Plans == nil || response.Dashboard.Scheduler.Plans == nil {
		t.Fatalf("dashboard should normalize plan slices: %+v", response.Dashboard)
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
		Paths map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &spec); err != nil {
		t.Fatalf("decode openapi: %v", err)
	}
	if _, ok := spec.Paths["/dashboard"]; !ok {
		t.Fatalf("new dashboard path missing from OpenAPI: %+v", spec.Paths)
	}
	if _, ok := spec.Paths["/runtime/status"]; ok {
		t.Fatalf("old runtime path should not be present in OpenAPI")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/runtime/status", nil)
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("old runtime route should be gone, status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestIndexServesVueBundle(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("index status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "/static/app/assets/app.js") {
		t.Fatalf("index should load Vue dashboard bundle: %s", body)
	}
	if strings.Contains(body, "/static/js/main.js") {
		t.Fatalf("index should not load retired dashboard script: %s", body)
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	paths := config.Paths{
		ConfigDir:  dir,
		ConfigFile: filepath.Join(dir, "config.json"),
		StatsFile:  filepath.Join(dir, "stats.json"),
	}
	return New(appruntime.New(paths), "../..")
}
