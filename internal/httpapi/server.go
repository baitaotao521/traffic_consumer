package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/robfig/cron/v3"

	"github.com/baitaotao521/traffic_consumer/internal/model"
	appruntime "github.com/baitaotao521/traffic_consumer/internal/runtime"
	"github.com/baitaotao521/traffic_consumer/internal/storage"
)

// Server 封装 Web 控制台页面和 HTTP JSON API。
type Server struct {
	// Manager 承载 Web 控制台的运行态、调度器、日志和事件缓冲。
	Manager *appruntime.Manager
	// BaseDir 是模板和静态资源所在的仓库根目录。
	BaseDir string
}

// New 创建 HTTP 服务。
func New(manager *appruntime.Manager, baseDir string) *Server {
	return &Server{Manager: manager, BaseDir: baseDir}
}

// Router 注册页面、静态资源和 API 路由。
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)

	r.Get("/", s.index)
	r.Get("/favicon.ico", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(s.BaseDir, "static")))))

	r.Route("/api", func(r chi.Router) {
		r.Post("/preview_cron", s.previewCron)
		r.Get("/configs", s.getConfigs)
		r.Get("/configs/{name}", s.getConfig)
		r.Put("/configs/{name}", s.saveConfig)
		r.Delete("/configs/{name}", s.deleteConfig)
		r.Post("/runtime/start", s.startConsumer)
		r.Post("/runtime/temp-test", s.startTempTest)
		r.Post("/runtime/stop", s.stopConsumer)
		r.Post("/runtime/scheduler/stop", s.stopScheduler)
		r.Post("/runtime/plans/{name}/stop", s.stopPlan)
		r.Get("/runtime/status", s.status)
		r.Get("/runtime/scheduler", s.scheduler)
		r.Get("/runtime/plans", s.plans)
		r.Get("/runtime/plans/{name}", s.planDetail)
		r.Get("/runtime/logs", s.logs)
		r.Get("/runtime/events", s.events)
	})
	return r
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(s.BaseDir, "templates", "index.html")
	data, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	html := strings.ReplaceAll(string(data), "{{ url_for('static', filename='css/style.css') }}", "/static/css/style.css")
	html = strings.ReplaceAll(html, "{{ url_for('static', filename='js/main.js') }}", "/static/js/main.js")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

func (s *Server) previewCron(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		CronExpr string `json:"cron_expr"`
	}
	if err := decodeJSON(r, &payload); err != nil || strings.TrimSpace(payload.CronExpr) == "" {
		writeError(w, http.StatusBadRequest, "无效的Cron表达式")
		return
	}
	schedule, err := cron.ParseStandard(payload.CronExpr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的Cron表达式")
		return
	}
	now := time.Now()
	result := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		now = schedule.Next(now)
		result = append(result, now.Format(time.RFC3339))
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) getConfigs(w http.ResponseWriter, _ *http.Request) {
	names, err := storage.ListConfigNames(s.Manager.Paths.ConfigFile)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"configs": names})
}

func (s *Server) getConfig(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	cfg, ok, err := storage.LoadConfigEntry(s.Manager.Paths.ConfigFile, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "配置不存在。")
		return
	}
	cfg = cfg.Normalize(nil, storage.NormalizeConfigName(name), false)
	writeJSON(w, http.StatusOK, map[string]any{"name": storage.NormalizeConfigName(name), "config": cfg})
}

func (s *Server) saveConfig(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var cfg model.Config
	if err := decodeJSON(r, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "配置格式无效。")
		return
	}
	if err := s.Manager.SaveConfig(name, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "配置 \"" + storage.NormalizeConfigName(name) + "\" 已保存。"})
}

func (s *Server) deleteConfig(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	deleted, err := s.Manager.DeleteConfig(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, "配置 \""+storage.NormalizeConfigName(name)+"\" 不存在。")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "配置 \"" + storage.NormalizeConfigName(name) + "\" 已删除。"})
}

func (s *Server) startConsumer(w http.ResponseWriter, r *http.Request) {
	var cfg model.Config
	if err := decodeJSON(r, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "启动参数无效。")
		return
	}
	status, err := s.Manager.StartConsumer(cfg)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) startTempTest(w http.ResponseWriter, r *http.Request) {
	var cfg model.Config
	if err := decodeJSON(r, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "临时测试参数无效。")
		return
	}
	status, err := s.Manager.StartTempTest(cfg)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) stopConsumer(w http.ResponseWriter, _ *http.Request) {
	if !s.Manager.StopAll() {
		writeError(w, http.StatusConflict, "流量消耗器未在运行。")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "流量消耗器已停止。"})
}

func (s *Server) stopScheduler(w http.ResponseWriter, _ *http.Request) {
	if !s.Manager.StopAll() {
		writeError(w, http.StatusConflict, "调度器未在运行。")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "调度器已停止。"})
}

func (s *Server) stopPlan(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if !s.Manager.StopRuntimeConfig(name) {
		writeError(w, http.StatusNotFound, "计划 \""+storage.NormalizeConfigName(name)+"\" 未在运行。")
		return
	}
	writeJSON(w, http.StatusOK, s.Manager.SchedulerStatus())
}

func (s *Server) status(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.Manager.Status())
}

func (s *Server) scheduler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.Manager.SchedulerStatus())
}

func (s *Server) plans(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"plans": s.Manager.PlanSummaries()})
}

func (s *Server) planDetail(w http.ResponseWriter, r *http.Request) {
	payload, err := s.Manager.PlanDetail(chi.URLParam(r, "name"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "不存在") {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) logs(w http.ResponseWriter, r *http.Request) {
	entries, latest := s.Manager.LogsSince(querySeq(r))
	writeJSON(w, http.StatusOK, map[string]any{"logs": entries, "latest": latest})
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	events, latest := s.Manager.EventsSince(querySeq(r))
	writeJSON(w, http.StatusOK, map[string]any{"events": events, "latest": latest})
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message, "message": message})
}

func querySeq(r *http.Request) int64 {
	value := r.URL.Query().Get("since")
	if value == "" {
		return 0
	}
	seq, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return seq
}
