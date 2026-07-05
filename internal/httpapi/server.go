package httpapi

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/robfig/cron/v3"

	"github.com/baitaotao521/traffic_consumer/internal/configguard"
	"github.com/baitaotao521/traffic_consumer/internal/model"
	appruntime "github.com/baitaotao521/traffic_consumer/internal/runtime"
	"github.com/baitaotao521/traffic_consumer/internal/storage"
)

// Server 封装 Web 控制台页面和 HTTP JSON API。
type Server struct {
	// Manager 承载 Web 控制台的运行态、调度器、日志和事件缓冲。
	Manager *appruntime.Manager
	// BaseDir 是仓库或发布包根目录，前端产物固定读取 BaseDir/frontend/dist。
	BaseDir string
}

// ConfigSummary 是 dashboard 配置选择器中的轻量数据项。
type ConfigSummary struct {
	// Name 是已经规范化的配置名称。
	Name string `json:"name" doc:"配置名称"`
}

// ConfigResource 是配置编辑面板读取和保存的配置详情。
type ConfigResource struct {
	// Name 是已经规范化的配置名称。
	Name string `json:"name" doc:"配置名称"`
	// Config 是保存或启动运行态时使用的配置内容。
	Config model.Config `json:"config" doc:"配置内容"`
}

// DashboardSnapshot 是前端主界面的单一服务端快照。
type DashboardSnapshot struct {
	// Configs 保存配置选择器需要的全部配置摘要。
	Configs []ConfigSummary `json:"configs" doc:"配置摘要列表"`
	// Status 保存 KPI、按钮状态和线程表使用的运行态快照。
	Status model.StatusPayload `json:"status" doc:"运行态快照"`
	// Scheduler 保存调度摘要和计划列表的调度态快照。
	Scheduler model.SchedulerPayload `json:"scheduler" doc:"调度态快照"`
	// Plans 保存当前所有非临时运行计划。
	Plans []model.PlanSummary `json:"plans" doc:"运行计划列表"`
}

// DashboardResponse 是主界面聚合快照响应体。
type DashboardResponse struct {
	// Dashboard 是前端主界面的服务端快照。
	Dashboard DashboardSnapshot `json:"dashboard" doc:"Dashboard 快照"`
}

// ConfigResourceResponse 是配置详情响应体。
type ConfigResourceResponse struct {
	// Item 保存当前配置详情。
	Item ConfigResource `json:"item" doc:"配置详情"`
}

// ConfigCommandResponse 是保存或删除配置后的响应体。
type ConfigCommandResponse struct {
	// Item 保存写入后的配置详情；删除配置时为空。
	Item *ConfigResource `json:"item,omitempty" doc:"配置详情"`
	// Dashboard 是命令执行后的主界面快照。
	Dashboard DashboardSnapshot `json:"dashboard" doc:"Dashboard 快照"`
	// Message 是用户可见的命令结果。
	Message string `json:"message" doc:"操作结果"`
}

// RuntimeCommandResponse 是启动、停止等运行态命令后的响应体。
type RuntimeCommandResponse struct {
	// Dashboard 是命令执行后的主界面快照。
	Dashboard DashboardSnapshot `json:"dashboard" doc:"Dashboard 快照"`
	// Message 是用户可见的命令结果。
	Message string `json:"message" doc:"操作结果"`
}

// PlanDetailResponse 是运行计划详情响应体。
type PlanDetailResponse struct {
	// Item 保存单个运行计划的详情。
	Item model.PlanDetailPayload `json:"item" doc:"运行计划详情"`
}

// ActivityResponse 是日志和运行态事件的统一增量响应。
type ActivityResponse struct {
	// Logs 保存日志游标之后新增的日志。
	Logs []model.LogEntry `json:"logs" doc:"日志增量"`
	// Events 保存事件游标之后新增的运行态事件。
	Events []model.Event `json:"events" doc:"事件增量"`
	// LogCursor 是本次响应后的最新日志游标。
	LogCursor int64 `json:"log_cursor" doc:"最新日志游标"`
	// EventCursor 是本次响应后的最新事件游标。
	EventCursor int64 `json:"event_cursor" doc:"最新事件游标"`
}

// CronPreviewRequest 是 cron 预览请求体。
type CronPreviewRequest struct {
	// Expression 是五段 cron 表达式。
	Expression string `json:"expression" doc:"Cron 表达式"`
}

// CronPreviewResponse 是 cron 预览响应体。
type CronPreviewResponse struct {
	// Items 保存接下来五次触发时间，格式为 RFC3339。
	Items []string `json:"items" doc:"接下来五次触发时间"`
}

type dashboardOutput struct {
	// Body 是 Huma 写入响应的 dashboard 快照。
	Body DashboardResponse
}

type configNameInput struct {
	// Name 是路径中的配置或计划名称。
	Name string `path:"name" doc:"配置或计划名称"`
}

type getConfigOutput struct {
	// Body 是 Huma 写入响应的配置详情。
	Body ConfigResourceResponse
}

type saveConfigInput struct {
	// Name 是路径中的配置名称。
	Name string `path:"name" doc:"配置名称"`
	// Force 是旧前端覆盖确认遗留的兼容参数；重名目标仍会被拒绝。
	Force bool `query:"force" doc:"兼容旧覆盖确认参数"`
	// Mode 区分新建和编辑，create 模式遇到已有目标会直接返回冲突。
	Mode string `query:"mode" doc:"保存模式：create 或 edit"`
	// Body 是要保存的配置内容。
	Body model.Config
}

type configCommandOutput struct {
	// Body 是 Huma 写入响应的配置命令结果。
	Body ConfigCommandResponse
}

type configRuntimeInput struct {
	// Body 是启动运行态使用的配置内容。
	Body model.Config
}

type runtimeCommandOutput struct {
	// Body 是 Huma 写入响应的运行态命令结果。
	Body RuntimeCommandResponse
}

type planDetailOutput struct {
	// Body 是 Huma 写入响应的运行计划详情。
	Body PlanDetailResponse
}

type activityInput struct {
	// AfterLog 是客户端已经消费过的最后一个日志游标。
	AfterLog int64 `query:"after_log" minimum:"0" doc:"已消费的最后一个日志游标"`
	// AfterEvent 是客户端已经消费过的最后一个事件游标。
	AfterEvent int64 `query:"after_event" minimum:"0" doc:"已消费的最后一个事件游标"`
	// IncludeLogs 控制本次 activity 轮询是否推进日志游标。
	IncludeLogs string `query:"include_logs" doc:"是否返回日志并推进日志游标"`
	// LogSources 是逗号分隔的日志来源筛选，例如 scheduler,runtime。
	LogSources string `query:"log_sources" doc:"逗号分隔的日志来源筛选"`
	// LogLevels 是逗号分隔的日志级别筛选，例如 info,error。
	LogLevels string `query:"log_levels" doc:"逗号分隔的日志级别筛选"`
}

type activityOutput struct {
	// Body 是 Huma 写入响应的统一活动增量。
	Body ActivityResponse
}

type cronPreviewInput struct {
	// Body 是 cron 预览请求体。
	Body CronPreviewRequest
}

type cronPreviewOutput struct {
	// Body 是 Huma 写入响应的 cron 预览结果。
	Body CronPreviewResponse
}

// New 创建 HTTP 服务。
func New(manager *appruntime.Manager, baseDir string) *Server {
	return &Server{Manager: manager, BaseDir: baseDir}
}

// Router 注册页面、静态资源和重新设计后的 OpenAPI JSON API。
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)

	r.Get("/", s.index)
	r.Get("/favicon.ico", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	// Vite 前端产物统一放在 frontend/dist，避免模板和静态资源分散到多个运行目录。
	frontendFiles := http.FileServer(http.Dir(s.frontendDir()))
	r.Handle("/assets/*", frontendFiles)
	r.Get("/favicon.svg", s.frontendFile("favicon.svg"))
	r.Get("/icons.svg", s.frontendFile("icons.svg"))
	r.Route("/api", func(r chi.Router) {
		s.registerAPI(r)
	})
	return r
}

func (s *Server) index(w http.ResponseWriter, _ *http.Request) {
	path := filepath.Join(s.frontendDir(), "index.html")
	data, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "frontend bundle not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (s *Server) frontendDir() string {
	return filepath.Join(s.BaseDir, "frontend", "dist")
}

func (s *Server) frontendFile(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(s.frontendDir(), name))
	}
}

func (s *Server) registerAPI(r chi.Router) huma.API {
	config := huma.DefaultConfig("Traffic Consumer API", "2.0.0")
	config.Servers = []*huma.Server{{URL: "/api"}}
	// Web API 响应只承载 dashboard 业务 payload；类型合同统一由 /api/openapi.json 提供给前端生成 SDK。
	config.SchemasPath = ""
	config.CreateHooks = nil
	api := humachi.New(r, config)

	huma.Register(api, huma.Operation{
		OperationID: "getDashboard",
		Method:      http.MethodGet,
		Path:        "/dashboard",
		Summary:     "读取 Dashboard 快照",
		Tags:        []string{"Dashboard"},
		Errors:      []int{http.StatusInternalServerError},
	}, func(ctx context.Context, input *struct{}) (*dashboardOutput, error) {
		snapshot, err := s.dashboardSnapshot()
		if err != nil {
			return nil, serverError(err)
		}
		return &dashboardOutput{Body: DashboardResponse{Dashboard: snapshot}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getConfiguration",
		Method:      http.MethodGet,
		Path:        "/configurations/{name}",
		Summary:     "读取配置详情",
		Tags:        []string{"Configurations"},
		Errors:      []int{http.StatusNotFound, http.StatusInternalServerError},
	}, func(ctx context.Context, input *configNameInput) (*getConfigOutput, error) {
		resource, err := s.configResource(input.Name)
		if err != nil {
			return nil, err
		}
		return &getConfigOutput{Body: ConfigResourceResponse{Item: resource}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "saveConfiguration",
		Method:      http.MethodPut,
		Path:        "/configurations/{name}",
		Summary:     "保存配置",
		Tags:        []string{"Configurations"},
		Errors:      []int{http.StatusBadRequest, http.StatusConflict, http.StatusInternalServerError},
	}, func(ctx context.Context, input *saveConfigInput) (*configCommandOutput, error) {
		sourceName := storage.NormalizeConfigName(input.Name)
		cfg, targetName, err := configguard.Validate(sourceName, input.Body, configguard.ModeSave)
		if err != nil {
			return nil, commandError(err)
		}
		savedName, err := s.Manager.SaveConfigAs(appruntime.SaveConfigOptions{
			SourceName: sourceName,
			TargetName: targetName,
			Config:     cfg,
			RequireNew: strings.EqualFold(strings.TrimSpace(input.Mode), "create"),
		})
		if err != nil {
			return nil, commandError(err)
		}
		resource, err := s.configResource(savedName)
		if err != nil {
			return nil, err
		}
		dashboard, err := s.dashboardSnapshot()
		if err != nil {
			return nil, serverError(err)
		}
		return &configCommandOutput{Body: ConfigCommandResponse{
			Item:      &resource,
			Dashboard: dashboard,
			Message:   "配置 \"" + savedName + "\" 已保存。",
		}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "deleteConfiguration",
		Method:      http.MethodDelete,
		Path:        "/configurations/{name}",
		Summary:     "删除配置",
		Tags:        []string{"Configurations"},
		Errors:      []int{http.StatusNotFound, http.StatusInternalServerError},
	}, func(ctx context.Context, input *configNameInput) (*configCommandOutput, error) {
		name := storage.NormalizeConfigName(input.Name)
		deleted, err := s.Manager.DeleteConfig(name)
		if err != nil {
			return nil, serverError(err)
		}
		if !deleted {
			return nil, huma.Error404NotFound("配置 \"" + name + "\" 不存在。")
		}
		dashboard, err := s.dashboardSnapshot()
		if err != nil {
			return nil, serverError(err)
		}
		return &configCommandOutput{Body: ConfigCommandResponse{
			Dashboard: dashboard,
			Message:   "配置 \"" + name + "\" 已删除。",
		}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "startRuntimeCommand",
		Method:      http.MethodPost,
		Path:        "/commands/start-runtime",
		Summary:     "启动运行态",
		Tags:        []string{"Commands"},
		Errors:      []int{http.StatusConflict, http.StatusInternalServerError},
	}, func(ctx context.Context, input *configRuntimeInput) (*runtimeCommandOutput, error) {
		cfg, _, err := configguard.Validate("", input.Body, configguard.ModeStart)
		if err != nil {
			return nil, commandError(err)
		}
		status, err := s.Manager.StartConsumer(cfg)
		if err != nil {
			return nil, huma.Error409Conflict(err.Error())
		}
		return s.runtimeCommand(status.Message)
	})

	huma.Register(api, huma.Operation{
		OperationID: "startTempTestCommand",
		Method:      http.MethodPost,
		Path:        "/commands/start-temp-test",
		Summary:     "启动临时测试",
		Tags:        []string{"Commands"},
		Errors:      []int{http.StatusConflict, http.StatusInternalServerError},
	}, func(ctx context.Context, input *configRuntimeInput) (*runtimeCommandOutput, error) {
		cfg, _, err := configguard.Validate("", input.Body, configguard.ModeTempTest)
		if err != nil {
			return nil, commandError(err)
		}
		status, err := s.Manager.StartTempTest(cfg)
		if err != nil {
			return nil, huma.Error409Conflict(err.Error())
		}
		return s.runtimeCommand(status.Message)
	})

	huma.Register(api, huma.Operation{
		OperationID: "stopRuntimeCommand",
		Method:      http.MethodPost,
		Path:        "/commands/stop-runtime",
		Summary:     "停止运行态",
		Tags:        []string{"Commands"},
		Errors:      []int{http.StatusConflict, http.StatusInternalServerError},
	}, func(ctx context.Context, input *struct{}) (*runtimeCommandOutput, error) {
		if !s.Manager.StopAll() {
			return nil, huma.Error409Conflict("流量消耗器未在运行。")
		}
		return s.runtimeCommand("流量消耗器已停止。")
	})

	huma.Register(api, huma.Operation{
		OperationID: "stopSchedulerCommand",
		Method:      http.MethodPost,
		Path:        "/commands/stop-scheduler",
		Summary:     "停止调度器",
		Tags:        []string{"Commands"},
		Errors:      []int{http.StatusConflict, http.StatusInternalServerError},
	}, func(ctx context.Context, input *struct{}) (*runtimeCommandOutput, error) {
		if !s.Manager.StopAll() {
			return nil, huma.Error409Conflict("调度器未在运行。")
		}
		return s.runtimeCommand("调度器已停止。")
	})

	huma.Register(api, huma.Operation{
		OperationID: "stopPlanCommand",
		Method:      http.MethodPost,
		Path:        "/commands/stop-plan/{name}",
		Summary:     "停止运行计划",
		Tags:        []string{"Commands"},
		Errors:      []int{http.StatusNotFound, http.StatusInternalServerError},
	}, func(ctx context.Context, input *configNameInput) (*runtimeCommandOutput, error) {
		name := storage.NormalizeConfigName(input.Name)
		if !s.Manager.StopRuntimeConfig(name) {
			return nil, huma.Error404NotFound("计划 \"" + name + "\" 未在运行。")
		}
		return s.runtimeCommand("计划 \"" + name + "\" 已停止。")
	})

	huma.Register(api, huma.Operation{
		OperationID: "getPlanDetail",
		Method:      http.MethodGet,
		Path:        "/plans/{name}",
		Summary:     "读取运行计划详情",
		Tags:        []string{"Plans"},
		Errors:      []int{http.StatusNotFound, http.StatusInternalServerError},
	}, func(ctx context.Context, input *configNameInput) (*planDetailOutput, error) {
		payload, err := s.Manager.PlanDetail(input.Name)
		if err != nil {
			return nil, planDetailError(err)
		}
		return &planDetailOutput{Body: PlanDetailResponse{Item: payload}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getActivity",
		Method:      http.MethodGet,
		Path:        "/activity",
		Summary:     "读取日志和事件增量",
		Tags:        []string{"Activity"},
	}, func(ctx context.Context, input *activityInput) (*activityOutput, error) {
		logs := []model.LogEntry{}
		logCursor := input.AfterLog
		includeLogs := shouldIncludeLogs(input.IncludeLogs)
		if includeLogs {
			logs, logCursor = s.Manager.LogsSince(
				input.AfterLog,
				parseLogSources(input.LogSources),
				parseCSV(input.LogLevels),
			)
		}
		events, eventCursor := s.Manager.EventsSince(input.AfterEvent)
		return &activityOutput{Body: ActivityResponse{
			Logs:        logs,
			Events:      events,
			LogCursor:   logCursor,
			EventCursor: eventCursor,
		}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "previewCron",
		Method:      http.MethodPost,
		Path:        "/cron/preview",
		Summary:     "预览 Cron",
		Tags:        []string{"Utilities"},
		Errors:      []int{http.StatusBadRequest},
	}, func(ctx context.Context, input *cronPreviewInput) (*cronPreviewOutput, error) {
		expression := strings.TrimSpace(input.Body.Expression)
		if expression == "" {
			return nil, huma.Error400BadRequest("无效的Cron表达式")
		}
		schedule, err := cron.ParseStandard(expression)
		if err != nil {
			return nil, huma.Error400BadRequest("无效的Cron表达式")
		}
		now := time.Now()
		result := make([]string, 0, 5)
		for i := 0; i < 5; i++ {
			now = schedule.Next(now)
			result = append(result, now.Format(time.RFC3339))
		}
		return &cronPreviewOutput{Body: CronPreviewResponse{Items: result}}, nil
	})

	return api
}

func (s *Server) dashboardSnapshot() (DashboardSnapshot, error) {
	names, err := storage.ListConfigNames(s.Manager.Paths.ConfigFile)
	if err != nil {
		return DashboardSnapshot{}, err
	}
	configs := make([]ConfigSummary, 0, len(names))
	for _, name := range names {
		configs = append(configs, ConfigSummary{Name: name})
	}
	plans := s.Manager.PlanSummaries()
	scheduler := s.Manager.SchedulerStatus()
	// Dashboard 聚合响应是前端唯一的主界面读模型，计划列表保持与调度快照同步。
	scheduler.Plans = plans
	return DashboardSnapshot{
		Configs:   configs,
		Status:    s.Manager.Status(),
		Scheduler: scheduler,
		Plans:     plans,
	}, nil
}

func (s *Server) runtimeCommand(message string) (*runtimeCommandOutput, error) {
	dashboard, err := s.dashboardSnapshot()
	if err != nil {
		return nil, serverError(err)
	}
	return &runtimeCommandOutput{Body: RuntimeCommandResponse{Dashboard: dashboard, Message: message}}, nil
}

func (s *Server) configResource(name string) (ConfigResource, error) {
	name = storage.NormalizeConfigName(name)
	cfg, ok, err := storage.LoadConfigEntry(s.Manager.Paths.ConfigFile, name)
	if err != nil {
		return ConfigResource{}, serverError(err)
	}
	if !ok {
		return ConfigResource{}, huma.Error404NotFound("配置不存在。")
	}
	cfg = cfg.Normalize(nil, name, false)
	return ConfigResource{Name: name, Config: cfg}, nil
}

func planDetailError(err error) error {
	if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "不存在") {
		return huma.Error404NotFound(err.Error())
	}
	return serverError(err)
}

func serverError(err error) error {
	return huma.Error500InternalServerError(err.Error())
}

func commandError(err error) error {
	if configguard.IsValidationError(err) {
		return huma.Error400BadRequest(err.Error())
	}
	if errors.Is(err, storage.ErrConfigConflict) {
		return huma.Error409Conflict(err.Error())
	}
	return serverError(err)
}

func parseLogSources(raw string) []model.LogSource {
	values := parseCSV(raw)
	result := make([]model.LogSource, 0, len(values))
	for _, value := range values {
		result = append(result, model.NormalizeLogSource(value))
	}
	return result
}

func parseCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		text := strings.TrimSpace(part)
		if text == "" {
			continue
		}
		result = append(result, text)
	}
	return result
}

func shouldIncludeLogs(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}
