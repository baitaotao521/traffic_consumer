package runtime

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/baitaotao521/traffic_consumer/internal/config"
	"github.com/baitaotao521/traffic_consumer/internal/consumer"
	"github.com/baitaotao521/traffic_consumer/internal/model"
	"github.com/baitaotao521/traffic_consumer/internal/stats"
	"github.com/baitaotao521/traffic_consumer/internal/storage"
)

// Manager 管理 Web 模式下所有运行态实例、调度器、日志和通知。
type Manager struct {
	// Paths 指向 Web 模式共享的配置和统计持久化文件。
	Paths config.Paths
	// Stats 负责读取和写入 stats.json，计划列表也从这里聚合历史。
	Stats *stats.Store

	// lifecycleMu 串行化启动、停止和临时测试注册流程，避免同名配置在半注册状态被重复启动。
	lifecycleMu sync.Mutex
	mu          sync.RWMutex
	itemsByName map[string]*Item
	temp        []*Item
	logSeq      int64
	logs        []model.LogEntry
	eventSeq    int64
	events      []model.Event
	location    *time.Location
}

// Item 是一个运行计划或临时测试实例。
type Item struct {
	// Name 是运行态实例对应的规范化配置名。
	Name string
	// Config 是该实例启动时使用的配置快照。
	Config model.Config
	// Consumer 是实际执行下载和统计保存的任务实例。
	Consumer *consumer.Consumer
	// Cron 是等待调度计划使用的后台调度器。
	Cron *cron.Cron
	// EntryID 标识 Cron 内部的当前计划任务。
	EntryID cron.EntryID
	// Temp 标记该实例是临时测试，计划列表不会展示它。
	Temp bool
	// AutoStart 标记该实例是否来自 Web 服务启动时的自动运行配置。
	AutoStart bool
	// Resume 保存临时测试完成后需要恢复的原调度计划。
	Resume *ResumeTarget
}

// ResumeTarget 保存临时测试结束后需要恢复的同名等待调度计划。
type ResumeTarget struct {
	// Name 是被临时测试暂停的计划名。
	Name string
	// Config 是恢复计划时使用的原配置快照。
	Config model.Config
	// AutoStart 表示恢复后应回到自启动实例列表还是主实例。
	AutoStart bool
}

// SaveConfigOptions 描述一次 Web 配置保存、重命名或覆盖操作。
type SaveConfigOptions struct {
	// SourceName 是编辑开始时的配置名，重命名时会在写入目标后删除。
	SourceName string
	// TargetName 是表单提交后的最终配置名。
	TargetName string
	// Config 是已经通过 Web 层业务校验的配置内容。
	Config model.Config
	// Force 允许覆盖不同名的已有目标配置。
	Force bool
	// RequireNew 表示本次操作来自“新建配置”，目标已存在时必须显式确认覆盖。
	RequireNew bool
}

// New 创建运行态管理器。
func New(paths config.Paths) *Manager {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.Local
	}
	return &Manager{
		Paths:       paths,
		Stats:       stats.NewStore(paths.StatsFile),
		itemsByName: map[string]*Item{},
		location:    loc,
	}
}

// AddLog 写入日志缓冲并同步打印到后端控制台，确保前后端看到的是同一条业务日志。
func (m *Manager) AddLog(configName string, source model.LogSource, level, message, color string) {
	entry := model.LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Source:    model.NormalizeLogSource(string(source)),
		Level:     model.NormalizeLogLevel(level),
		Message:   stripANSI(message),
		Color:     color,
		Config:    configName,
	}

	m.mu.Lock()
	m.logSeq++
	entry.Seq = m.logSeq
	m.logs = append(m.logs, entry)
	if len(m.logs) > 500 {
		m.logs = m.logs[len(m.logs)-500:]
	}
	m.mu.Unlock()

	fmt.Println(formatConsoleLog(entry))
}

// AddEvent 写入通知缓冲，替代旧 Socket.IO 的 invalid_url/history_update/error 推送。
func (m *Manager) AddEvent(event model.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventSeq++
	event.Seq = m.eventSeq
	if event.Type == "" {
		event.Type = firstNonEmpty(event.EventType, "notification")
	}
	m.events = append(m.events, event)
	if len(m.events) > 500 {
		m.events = m.events[len(m.events)-500:]
	}
}

// LogsSince 返回指定序号之后、符合来源和级别筛选条件的日志。
func (m *Manager) LogsSince(seq int64, sources []model.LogSource, levels []string) ([]model.LogEntry, int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sourceFilter := make(map[model.LogSource]bool, len(sources))
	for _, source := range sources {
		sourceFilter[model.NormalizeLogSource(string(source))] = true
	}
	levelFilter := make(map[string]bool, len(levels))
	for _, level := range levels {
		levelFilter[model.NormalizeLogLevel(level)] = true
	}
	result := make([]model.LogEntry, 0)
	for _, entry := range m.logs {
		if entry.Seq > seq && matchesLogFilter(entry, sourceFilter, levelFilter) {
			result = append(result, entry)
		}
	}
	return result, m.logSeq
}

func (m *Manager) addRuntimeLog(configName, level, message string) {
	m.AddLog(configName, model.LogSourceRuntime, level, message, colorForLevel(level))
}

func (m *Manager) addSchedulerLog(configName, level, message string) {
	m.AddLog(configName, model.LogSourceScheduler, level, message, colorForLevel(level))
}

// EventsSince 返回指定序号之后的通知。
func (m *Manager) EventsSince(seq int64) ([]model.Event, int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]model.Event, 0)
	for _, event := range m.events {
		if event.Seq > seq {
			result = append(result, event)
		}
	}
	return result, m.eventSeq
}

// LaunchAutoStartConfigs 启动所有标记为自启动的配置。
func (m *Manager) LaunchAutoStartConfigs() bool {
	names, err := storage.FindAutoStartConfigs(m.Paths.ConfigFile)
	if err != nil || len(names) == 0 {
		return false
	}
	startedNow := []string{}
	scheduled := []string{}
	for _, name := range names {
		cfg, ok, err := storage.LoadConfigEntry(m.Paths.ConfigFile, name)
		if err != nil || !ok {
			continue
		}
		item, err := m.createItem(name, cfg, false, nil)
		if err != nil {
			m.AddEvent(errorEvent(err.Error()))
			continue
		}
		item.AutoStart = true
		var done <-chan struct{}
		if item.isScheduled() {
			if err := m.startScheduler(item); err != nil {
				m.AddEvent(errorEvent(err.Error()))
				continue
			}
			m.addRuntimeLog(name, "info", "服务启动时已恢复等待调度的配置。")
			scheduled = append(scheduled, name)
		} else {
			done = item.Consumer.StartAsync()
			m.addRuntimeLog(name, "info", "服务启动时已恢复下载任务。")
			startedNow = append(startedNow, name)
		}
		m.mu.Lock()
		m.ensureItemsLocked()
		m.itemsByName[item.Name] = item
		m.mu.Unlock()
		if done != nil {
			go m.cleanupRuntimeWhenDone(item, done)
		}
	}
	if len(startedNow) == 0 && len(scheduled) == 0 {
		return false
	}
	parts := []string{}
	state := model.StateScheduled
	if len(startedNow) > 0 {
		state = model.StateRunning
		parts = append(parts, "立即执行: "+strings.Join(startedNow, ", "))
	}
	if len(scheduled) > 0 {
		parts = append(parts, "等待调度: "+strings.Join(scheduled, ", "))
	}
	m.AddEvent(model.Event{
		Type:    "status_update",
		Title:   "状态更新：",
		Variant: stateVariant(state),
		Message: "已自动启动配置（" + strings.Join(parts, "；") + "）",
	})
	return true
}

// StartConsumer 启动手动配置；若配置带调度参数则只进入等待调度。
func (m *Manager) StartConsumer(cfg model.Config) (model.StatusPayload, error) {
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()

	name := storage.NormalizeConfigName(firstNonEmpty(cfg.ConfigName, cfg.Name))
	item, err := m.createItem(name, cfg, false, nil)
	if err != nil {
		return model.StatusPayload{}, err
	}

	var stale *Item
	m.mu.Lock()
	m.ensureItemsLocked()
	if existing := m.itemsByName[name]; existing != nil {
		// 多配置运行允许不同配置并行，但同名配置只能有一个下载或调度实例。
		if existing.Consumer.IsActive() || existing.Cron != nil {
			m.mu.Unlock()
			return model.StatusPayload{}, fmt.Errorf("配置 \"%s\" 已在运行或等待调度。", name)
		}
		stale = existing
	}
	m.itemsByName[name] = item
	m.mu.Unlock()
	stopItem(stale)

	if item.isScheduled() {
		if err := m.startScheduler(item); err != nil {
			m.removeRuntimeItem(name, item)
			return model.StatusPayload{}, err
		}
		m.addRuntimeLog(name, "info", "配置已切换到等待调度状态。")
	} else {
		done := item.Consumer.StartAsync()
		go m.cleanupRuntimeWhenDone(item, done)
		m.addRuntimeLog(name, "info", "配置已启动下载任务。")
	}

	state := model.StateRunning
	message := fmt.Sprintf("流量消耗器已使用配置启动: %s", name)
	if item.isScheduled() {
		state = model.StateScheduled
		message = fmt.Sprintf("配置 \"%s\" 已启动，当前等待调度执行。", name)
	}
	m.AddEvent(model.Event{Type: "status_update", Title: "状态更新：", Variant: stateVariant(state), Message: message, Config: name})
	return model.StatusPayload{Running: true, State: state, Config: name, Message: message, ThreadStatus: map[int]string{}, URLUsageStats: []model.URLUsageStat{}}, nil
}

// StartTempTest 使用当前配置执行最多 100MB 的立即验证。
func (m *Manager) StartTempTest(cfg model.Config) (model.StatusPayload, error) {
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()

	name := storage.NormalizeConfigName(firstNonEmpty(cfg.ConfigName, cfg.Name))
	snapshot := m.snapshot()
	if snapshot.hasActiveDownload {
		return model.StatusPayload{}, errors.New("已有下载任务正在运行，请先停止后再发起临时测试。")
	}
	var resume *ResumeTarget
	if len(snapshot.scheduled) > 0 {
		if len(snapshot.scheduled) > 1 {
			return model.StatusPayload{}, errors.New("当前存在多个等待中的计划，请先停掉其他计划后再发起临时测试。")
		}
		scheduled := snapshot.scheduled[0]
		if scheduled.Name != name {
			return model.StatusPayload{}, fmt.Errorf("当前等待中的计划是 \"%s\"，请先切换到同名配置或停止该计划。", scheduled.Name)
		}
		resume = &ResumeTarget{Name: name, Config: scheduled.Config, AutoStart: scheduled.AutoStart}
		m.addRuntimeLog(name, "info", "临时测试开始前已暂停同名等待中的计划，测试结束后会自动恢复。")
		m.stopRuntimeConfig(name)
	}

	testCfg := cfg
	testCfg.CronExpr = ""
	testCfg.Interval = nil
	testCfg.AutoStart = false
	testCfg.Duration = nil
	testCfg.Count = nil
	testCfg.AutoRemoveFailedURL = false
	limit := int64(100)
	if testCfg.TrafficLimit != nil && *testCfg.TrafficLimit > 0 && *testCfg.TrafficLimit < limit {
		limit = *testCfg.TrafficLimit
	}
	testCfg.TrafficLimit = &limit

	item, err := m.createItem(name, testCfg, true, resume)
	if err != nil {
		return model.StatusPayload{}, err
	}
	done := item.Consumer.StartAsync()
	m.mu.Lock()
	m.temp = append(m.temp, item)
	m.mu.Unlock()
	go m.cleanupTempWhenDone(item, done)

	message := fmt.Sprintf("已开始临时测试验证配置 \"%s\"，本次最多下载 %d MB。", name, limit)
	m.addRuntimeLog(name, "info", message)
	m.AddEvent(model.Event{Type: "status_update", Title: "状态更新：", Variant: "success", Message: message, Config: name})
	return model.StatusPayload{Running: true, State: model.StateRunning, Config: name, Message: message, ThreadStatus: map[int]string{}, URLUsageStats: []model.URLUsageStat{}}, nil
}

// StopAll 停止所有下载与调度。
func (m *Manager) StopAll() bool {
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()
	return m.stopAll()
}

func (m *Manager) stopAll() bool {
	items := m.items()
	stoppedSchedules := []string{}
	stoppedRuns := []string{}
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.Cron != nil {
			stoppedSchedules = append(stoppedSchedules, item.Name)
		}
		if item.Consumer.IsActive() {
			stoppedRuns = append(stoppedRuns, item.Name)
		}
	}
	m.mu.Lock()
	m.itemsByName = map[string]*Item{}
	m.temp = nil
	m.mu.Unlock()
	stopped := false
	for _, item := range items {
		if stopItem(item) {
			stopped = true
		}
	}
	if stopped {
		if len(stoppedSchedules) > 0 {
			m.addSchedulerLog("", "info", fmt.Sprintf("已停止调度计划：%s。", strings.Join(stoppedSchedules, ", ")))
		}
		if len(stoppedRuns) > 0 {
			m.addRuntimeLog("", "info", fmt.Sprintf("已停止运行任务：%s。", strings.Join(stoppedRuns, ", ")))
		}
		m.AddEvent(model.Event{Type: "status_update", Title: "状态更新：", Variant: "secondary", Message: "流量消耗器已停止。"})
	}
	return stopped
}

// StopRuntimeConfig 仅停止指定名称的计划或任务。
func (m *Manager) StopRuntimeConfig(name string) bool {
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()
	return m.stopRuntimeConfig(name)
}

func (m *Manager) stopRuntimeConfig(name string) bool {
	name = storage.NormalizeConfigName(name)
	var targets []*Item
	m.mu.Lock()
	if item := m.itemsByName[name]; item != nil {
		targets = append(targets, item)
		delete(m.itemsByName, name)
	}
	m.temp, targets = removeNamed(m.temp, name, targets)
	m.mu.Unlock()

	stopped := false
	stoppedScheduler := false
	stoppedDownload := false
	for _, item := range targets {
		if item != nil && item.Cron != nil {
			stoppedScheduler = true
		}
		if item != nil && item.Consumer.IsActive() {
			stoppedDownload = true
		}
		if stopItem(item) {
			stopped = true
		}
	}
	if len(targets) > 0 || stopped {
		if stoppedScheduler {
			m.addSchedulerLog(name, "info", fmt.Sprintf("计划 \"%s\" 的调度已停止。", name))
		}
		if stoppedDownload {
			m.addRuntimeLog(name, "info", fmt.Sprintf("计划 \"%s\" 的运行任务已停止。", name))
		}
		m.AddEvent(model.Event{Type: "status_update", Title: "状态更新：", Variant: "secondary", Message: fmt.Sprintf("计划 \"%s\" 已停止。", name), Config: name})
	}
	return len(targets) > 0 || stopped
}

func (m *Manager) removeRuntimeItem(name string, target *Item) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if current := m.itemsByName[storage.NormalizeConfigName(name)]; current == target {
		delete(m.itemsByName, storage.NormalizeConfigName(name))
	}
}

// Status 返回当前状态面板数据。
func (m *Manager) Status() model.StatusPayload {
	snapshot := m.snapshot()
	state := model.StateStopped
	if snapshot.hasActiveDownload {
		state = model.StateRunning
	} else if snapshot.hasActiveScheduler {
		state = model.StateScheduled
	}

	plans := m.PlanSummaries()
	totalRaw := int64(0)
	downloadCount := int64(0)
	speedBytes := 0.0
	threadCount := 0
	activeConfigs := []string{}
	for _, plan := range plans {
		totalRaw += plan.TotalBytesRaw
		downloadCount += plan.DownloadCount
		speedBytes += plan.SpeedBytes
		if plan.Running || plan.SchedulerRunning {
			threadCount += plan.Threads
			activeConfigs = append(activeConfigs, plan.Name)
		}
	}

	threadStatus := map[int]string{}
	urlUsageStats := []model.URLUsageStat{}
	if snapshot.focus != nil && snapshot.focus.Consumer.IsActive() {
		item := snapshot.focus
		threadStatus = item.Consumer.URLMgr.ThreadSnapshot(item.Config.Threads)
		urlUsageStats = buildURLUsageStats(item.Consumer.URLMgr.URLs(), item.Consumer.URLMgr.UsageSnapshot())
	}
	configLabel := strings.Join(activeConfigs, ", ")
	if configLabel == "" {
		configLabel = "全部配置"
	}
	return model.StatusPayload{
		TotalBytes:    stats.FormatBytes(float64(totalRaw)),
		Speed:         formatSpeed(speedBytes),
		DownloadCount: downloadCount,
		Running:       state != model.StateStopped,
		State:         state,
		Config:        configLabel,
		ThreadCount:   threadCount,
		ThreadStatus:  threadStatus,
		URLUsageStats: urlUsageStats,
	}
}

// SchedulerStatus 返回调度和计划列表。
func (m *Manager) SchedulerStatus() model.SchedulerPayload {
	plans := m.PlanSummaries()
	nextRun := ""
	details := []string{}
	for _, item := range m.items() {
		if item.Cron == nil {
			continue
		}
		itemNext := item.nextRun()
		if itemNext != "" && (nextRun == "" || itemNext < nextRun) {
			nextRun = itemNext
		}
		if item.Config.CronExpr != "" {
			details = append(details, fmt.Sprintf("%s: Cron %s", item.Name, item.Config.CronExpr))
		} else if item.Config.Interval != nil {
			details = append(details, fmt.Sprintf("%s: Interval %d 分钟", item.Name, *item.Config.Interval))
		}
	}
	return model.SchedulerPayload{NextRunTime: nextRun, JobDetails: strings.Join(details, " | "), Plans: plans}
}

// PlanSummaries 返回保存配置和非临时运行态合并后的配置行。
func (m *Manager) PlanSummaries() []model.PlanSummary {
	historySummary, _ := m.Stats.SummaryByConfig()
	savedConfigs, _ := storage.LoadConfigs(m.Paths.ConfigFile)
	planByName := map[string]model.PlanSummary{}
	names := make([]string, 0, len(savedConfigs))
	for name, cfg := range savedConfigs {
		name = storage.NormalizeConfigName(name)
		names = append(names, name)
		normalized := cfg.Normalize(config.DefaultURLs, name, false)
		history := historySummary[name]
		planByName[name] = model.PlanSummary{
			Name:          name,
			TotalBytes:    stats.FormatBytes(0),
			TotalBytesRaw: 0,
			Speed:         formatSpeed(0),
			SpeedBytes:    0,
			DownloadCount: 0,
			Threads:       normalized.Threads,
			URLCount:      len(normalized.URLs),
			CronExpr:      normalized.CronExpr,
			Interval:      normalized.Interval,
			AutoStart:     normalized.AutoStart,
			URLStrategy:   normalized.URLStrategy,
			LastRun:       history.LastRun,
		}
	}
	for _, item := range m.items() {
		if item.Temp {
			continue
		}
		if _, exists := planByName[item.Name]; !exists {
			names = append(names, item.Name)
		}
		history := historySummary[item.Name]
		total, count, startedAt, active := item.Consumer.Snapshot()
		// 列表里的流量和下载数只展示本次正在运行的数据；停止后的结果进入 LastRun 和历史详情。
		currentTotal := int64(0)
		currentCount := int64(0)
		if active {
			currentTotal = total
			currentCount = count
		}
		totalRaw := currentTotal
		downloadCount := currentCount
		speedBytes := liveSpeedBytes(currentTotal, startedAt, active)
		planByName[item.Name] = model.PlanSummary{
			Name:             item.Name,
			Running:          active,
			SchedulerRunning: item.Cron != nil,
			NextRunTime:      item.nextRun(),
			TotalBytes:       stats.FormatBytes(float64(totalRaw)),
			TotalBytesRaw:    totalRaw,
			Speed:            formatSpeed(speedBytes),
			SpeedBytes:       speedBytes,
			DownloadCount:    downloadCount,
			Threads:          item.Config.Threads,
			URLCount:         len(item.Consumer.URLMgr.URLs()),
			CronExpr:         item.Config.CronExpr,
			Interval:         item.Config.Interval,
			AutoStart:        item.Config.AutoStart,
			URLStrategy:      item.Config.URLStrategy,
			LastRun:          history.LastRun,
		}
	}
	sort.Strings(names)
	plans := make([]model.PlanSummary, 0, len(names))
	for _, name := range names {
		plans = append(plans, planByName[name])
	}
	return plans
}

// PlanDetail 返回计划详情弹窗数据。
func (m *Manager) PlanDetail(name string) (model.PlanDetailPayload, error) {
	name = storage.NormalizeConfigName(name)
	cfg, ok, err := storage.LoadConfigEntry(m.Paths.ConfigFile, name)
	if err != nil {
		return model.PlanDetailPayload{}, err
	}
	if !ok {
		return model.PlanDetailPayload{}, errors.New("计划不存在或已被删除。")
	}
	cfg = cfg.Normalize(config.DefaultURLs, name, false)
	var summary model.PlanSummary
	for _, plan := range m.PlanSummaries() {
		if plan.Name == name {
			summary = plan
			break
		}
	}
	if summary.Name == "" {
		historySummary, _ := m.Stats.SummaryByConfig()
		summary = historySummary[name]
		summary.Name = name
		// 详情头部跟列表保持一致，只展示当前运行态；历史结果留在 history 明细中。
		summary.TotalBytes = stats.FormatBytes(0)
		summary.TotalBytesRaw = 0
		summary.Speed = formatSpeed(0)
		summary.SpeedBytes = 0
		summary.DownloadCount = 0
		summary.Threads = cfg.Threads
		summary.CronExpr = cfg.CronExpr
		summary.Interval = cfg.Interval
	}
	history, _ := m.Stats.HistoryForConfig(name, 50)
	return model.PlanDetailPayload{Name: name, Config: cfg, Summary: summary, History: history}, nil
}

// SaveConfig 保存配置；若同名运行态存在，先停止旧计划。
func (m *Manager) SaveConfig(name string, cfg model.Config) error {
	_, err := m.SaveConfigAs(SaveConfigOptions{
		SourceName: name,
		TargetName: name,
		Config:     cfg,
	})
	return err
}

// SaveConfigAs 保存、重命名或覆盖配置，并在写入前停止所有受影响的同名运行态。
func (m *Manager) SaveConfigAs(options SaveConfigOptions) (string, error) {
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()

	sourceName := storage.NormalizeConfigName(options.SourceName)
	targetName := storage.NormalizeConfigName(firstNonEmpty(options.TargetName, sourceName))
	configs, err := storage.LoadConfigs(m.Paths.ConfigFile)
	if err != nil {
		return "", err
	}
	_, targetExists := configs[targetName]
	if targetExists && (options.RequireNew || sourceName != targetName) && !options.Force {
		return "", storage.ConfigConflictError{Name: targetName}
	}

	stopNames := []string{sourceName}
	if targetName != sourceName {
		stopNames = append(stopNames, targetName)
	}
	for _, name := range uniqueNames(stopNames) {
		m.stopRuntimeConfig(name)
	}

	cfg := options.Config.Normalize(config.DefaultURLs, targetName, false)
	if _, err := storage.SaveConfigEntryAs(m.Paths.ConfigFile, storage.SaveConfigEntryOptions{
		SourceName: sourceName,
		TargetName: targetName,
		Config:     cfg,
		Force:      options.Force,
		RequireNew: options.RequireNew,
	}); err != nil {
		return "", err
	}
	return targetName, nil
}

// DeleteConfig 删除配置并同步停止同名运行态。
func (m *Manager) DeleteConfig(name string) (bool, error) {
	name = storage.NormalizeConfigName(name)
	m.StopRuntimeConfig(name)
	return storage.DeleteConfigEntry(m.Paths.ConfigFile, name)
}

func (m *Manager) createItem(name string, cfg model.Config, temp bool, resume *ResumeTarget) (*Item, error) {
	cfg = cfg.Normalize(config.DefaultURLs, name, cfg.URLs == nil)
	name = cfg.ConfigName
	logger := func(level, message, color string) {
		m.AddLog(name, model.LogSourceDownload, level, message, color)
	}
	eventer := func(event model.Event) {
		event.Config = name
		m.AddEvent(event)
	}
	c := consumer.New(cfg, m.Paths, m.Stats, logger, eventer)
	return &Item{Name: name, Config: cfg, Consumer: c, Temp: temp, Resume: resume, AutoStart: cfg.AutoStart}, nil
}

func (m *Manager) startScheduler(item *Item) error {
	c := cron.New(cron.WithLocation(m.location), cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)))
	spec := item.Config.CronExpr
	if spec == "" && item.Config.Interval != nil {
		spec = fmt.Sprintf("@every %dm", *item.Config.Interval)
	}
	if spec == "" {
		return nil
	}
	id, err := c.AddFunc(spec, func() {
		m.runScheduledItem(item, context.Background())
	})
	if err != nil {
		return err
	}
	item.Cron = c
	item.EntryID = id
	if item.Config.CronExpr != "" {
		m.addSchedulerLog(item.Name, "info", fmt.Sprintf("已登记 Cron 调度：%s。", item.Config.CronExpr))
	} else if item.Config.Interval != nil {
		m.addSchedulerLog(item.Name, "info", fmt.Sprintf("已登记间隔调度：每 %d 分钟执行一次。", *item.Config.Interval))
	}
	c.Start()
	m.addSchedulerLog(item.Name, "info", fmt.Sprintf("调度器已启动，下一次执行时间 %s。", m.describeNextRun(item)))
	return nil
}

func (m *Manager) cleanupTempWhenDone(item *Item, done <-chan struct{}) {
	<-done
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()

	var resume *ResumeTarget
	m.mu.Lock()
	next := make([]*Item, 0, len(m.temp))
	for _, candidate := range m.temp {
		if candidate == item {
			resume = candidate.Resume
			continue
		}
		next = append(next, candidate)
	}
	m.temp = next
	m.mu.Unlock()
	if resume == nil {
		return
	}
	m.addSchedulerLog(resume.Name, "info", "临时测试已结束，正在恢复原计划调度。")
	restored, err := m.createItem(resume.Name, resume.Config, false, nil)
	if err != nil {
		m.AddEvent(errorEvent(err.Error()))
		return
	}
	restored.AutoStart = resume.AutoStart
	if err := m.startScheduler(restored); err != nil {
		m.AddEvent(errorEvent(err.Error()))
		return
	}
	m.mu.Lock()
	m.ensureItemsLocked()
	m.itemsByName[restored.Name] = restored
	m.mu.Unlock()
}

func (m *Manager) cleanupRuntimeWhenDone(item *Item, done <-chan struct{}) {
	<-done
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()

	m.removeRuntimeItem(item.Name, item)
}

func (m *Manager) ensureItemsLocked() {
	if m.itemsByName == nil {
		m.itemsByName = map[string]*Item{}
	}
}

func (m *Manager) items() []*Item {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := []*Item{}
	seen := map[*Item]bool{}
	names := make([]string, 0, len(m.itemsByName))
	for name := range m.itemsByName {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		item := m.itemsByName[name]
		if item == nil || seen[item] {
			continue
		}
		seen[item] = true
		items = append(items, item)
	}
	for _, item := range m.temp {
		if item == nil || seen[item] {
			continue
		}
		seen[item] = true
		items = append(items, item)
	}
	return items
}

func (m *Manager) snapshot() runtimeSnapshot {
	items := m.items()
	snapshot := runtimeSnapshot{records: items}
	for _, item := range items {
		if item.Consumer.IsActive() {
			snapshot.active = append(snapshot.active, item)
			snapshot.hasActiveDownload = true
		}
		if item.Cron != nil {
			snapshot.scheduled = append(snapshot.scheduled, item)
			snapshot.hasActiveScheduler = true
		}
		snapshot.threadCount += item.Config.Threads
		snapshot.configNames = append(snapshot.configNames, item.Name)
	}
	if len(snapshot.active) > 0 {
		snapshot.focus = snapshot.active[0]
	} else if len(snapshot.scheduled) > 0 {
		snapshot.focus = snapshot.scheduled[0]
	} else if len(items) > 0 {
		snapshot.focus = items[0]
	}
	return snapshot
}

type runtimeSnapshot struct {
	records            []*Item
	active             []*Item
	scheduled          []*Item
	focus              *Item
	threadCount        int
	configNames        []string
	hasActiveDownload  bool
	hasActiveScheduler bool
}

func (i *Item) isScheduled() bool {
	return i.Config.CronExpr != "" || i.Config.Interval != nil && *i.Config.Interval > 0
}

func (m *Manager) runScheduledItem(item *Item, ctx context.Context) {
	m.addSchedulerLog(item.Name, "info", "计划任务已触发执行。")
	item.Consumer.RunOnce(ctx)
	m.addSchedulerLog(item.Name, "info", fmt.Sprintf("计划任务执行完成，下一次执行时间 %s。", m.describeNextRun(item)))
}

func (i *Item) nextRun() string {
	next := i.nextRunAt()
	if next.IsZero() {
		return ""
	}
	return next.Format(time.RFC3339)
}

func (i *Item) nextRunAt() time.Time {
	if i.Cron == nil {
		return time.Time{}
	}
	entry := i.Cron.Entry(i.EntryID)
	return entry.Next
}

func stopItem(item *Item) bool {
	if item == nil {
		return false
	}
	stopped := false
	if item.Consumer.IsActive() {
		stopped = true
	}
	done := item.Consumer.Done()
	item.Consumer.Stop()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
	}
	if item.Cron != nil {
		ctx := item.Cron.Stop()
		select {
		case <-ctx.Done():
		case <-time.After(3 * time.Second):
		}
		item.Cron = nil
		stopped = true
	}
	return stopped
}

func removeNamed(items []*Item, name string, targets []*Item) ([]*Item, []*Item) {
	remaining := make([]*Item, 0, len(items))
	for _, item := range items {
		if item != nil && item.Name == name {
			targets = append(targets, item)
			continue
		}
		remaining = append(remaining, item)
	}
	return remaining, targets
}

func liveSpeedBytes(totalBytes int64, startedAt time.Time, active bool) float64 {
	if !active || startedAt.IsZero() {
		return 0
	}
	elapsed := time.Since(startedAt).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return float64(totalBytes) / elapsed
}

func formatSpeed(bytesPerSecond float64) string {
	if bytesPerSecond <= 0 {
		return "0 B/s"
	}
	return stats.FormatBytes(bytesPerSecond) + "/s"
}

func buildURLUsageStats(urls []string, usage map[string]int64) []model.URLUsageStat {
	total := int64(0)
	for _, count := range usage {
		total += count
	}
	result := []model.URLUsageStat{}
	if len(urls) > 0 {
		for _, url := range urls {
			count := usage[url]
			result = append(result, model.URLUsageStat{URL: url, Count: count, Percentage: percent(count, total)})
		}
		return result
	}
	for url, count := range usage {
		result = append(result, model.URLUsageStat{URL: url, Count: count, Percentage: percent(count, total)})
	}
	return result
}

func percent(count, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(count) / float64(total) * 100
}

func matchesLogFilter(entry model.LogEntry, sources map[model.LogSource]bool, levels map[string]bool) bool {
	if len(sources) > 0 && !sources[entry.Source] {
		return false
	}
	if len(levels) > 0 && !levels[model.NormalizeLogLevel(entry.Level)] {
		return false
	}
	return true
}

func errorEvent(message string) model.Event {
	return model.Event{Type: "error", Title: "操作失败：", Variant: "danger", Message: message}
}

func stateVariant(state model.RuntimeState) string {
	switch state {
	case model.StateScheduled:
		return "info"
	case model.StateRunning:
		return "success"
	default:
		return "secondary"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func uniqueNames(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		name := storage.NormalizeConfigName(value)
		if seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}
	return result
}

func colorForLevel(level string) string {
	switch model.NormalizeLogLevel(level) {
	case "error":
		return "#dc3545"
	case "warning":
		return "#ffc107"
	case "success":
		return "#198754"
	default:
		return "#0dcaf0"
	}
}

func formatConsoleLog(entry model.LogEntry) string {
	parts := []string{
		"[" + entry.Timestamp + "]",
		"[" + string(entry.Source) + "]",
		"[" + entry.Level + "]",
	}
	if entry.Config != "" {
		parts = append(parts, "["+entry.Config+"]")
	}
	parts = append(parts, entry.Message)
	return strings.Join(parts, " ")
}

func (m *Manager) describeNextRun(item *Item) string {
	next := item.nextRunAt()
	if next.IsZero() {
		return "暂无后续触发时间"
	}
	return next.In(m.location).Format("2006-01-02 15:04:05")
}

func stripANSI(text string) string {
	replacer := strings.NewReplacer("\x1b[31m", "", "\x1b[32m", "", "\x1b[33m", "", "\x1b[34m", "", "\x1b[35m", "", "\x1b[36m", "", "\x1b[37m", "", "\x1b[0m", "")
	return replacer.Replace(text)
}
