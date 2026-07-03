package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/baitaotao521/traffic_consumer/internal/config"
	"github.com/baitaotao521/traffic_consumer/internal/model"
	appruntime "github.com/baitaotao521/traffic_consumer/internal/runtime"
)

func TestPreviewCronReturnsFiveRuns(t *testing.T) {
	server := newTestServer(t)
	body := bytes.NewBufferString(`{"cron_expr":"*/5 * * * *"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/preview_cron", body)
	rec := httptest.NewRecorder()

	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result []string
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(result) != 5 {
		t.Fatalf("expected 5 runs, got %d", len(result))
	}
}

func TestConfigHTTPCRUD(t *testing.T) {
	server := newTestServer(t)
	payload := model.Config{URLs: []string{"https://example.com/a"}, Threads: 2, URLStrategy: "random"}
	data, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/configs/demo", bytes.NewReader(data))
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("save status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/configs/demo", nil)
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Name   string       `json:"name"`
		Config model.Config `json:"config"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if response.Name != "demo" || response.Config.Threads != 2 {
		t.Fatalf("unexpected response: %+v", response)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/configs/demo", nil)
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
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
