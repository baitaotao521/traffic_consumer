package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/spf13/pflag"

	"github.com/baitaotao521/traffic_consumer/internal/config"
	"github.com/baitaotao521/traffic_consumer/internal/consumer"
	"github.com/baitaotao521/traffic_consumer/internal/model"
	"github.com/baitaotao521/traffic_consumer/internal/stats"
	"github.com/baitaotao521/traffic_consumer/internal/storage"
)

// Args 保存 CLI 参数，字段名对应 README 中的现有参数。
type Args struct {
	// URLs 是命令行直接传入的下载链接列表。
	URLs []string
	// URLStrategy 决定多链接下载时使用随机均衡还是轮询顺序。
	URLStrategy string
	// Threads 是本次 CLI 任务启动的并发下载线程数。
	Threads int
	// LimitSpeed 是本次 CLI 任务共享限速，单位 MB/s。
	LimitSpeed float64
	// Duration 限制本次 CLI 任务最长运行秒数。
	Duration int64
	// Count 限制本次 CLI 任务成功完成的响应次数。
	Count int64
	// CronExpr 让 CLI 进入 cron 调度等待模式。
	CronExpr string
	// TrafficLimit 限制本次 CLI 任务单轮累计消耗流量，单位 MB。
	TrafficLimit int64
	// Interval 让 CLI 按固定分钟间隔重复执行任务。
	Interval int64
	// AutoRemoveFailedURL 控制坏链重试耗尽后是否写回配置删除 URL。
	AutoRemoveFailedURL bool
	// AutoStart 保存配置时标记 Web 服务启动后自动运行。
	AutoStart bool
	// UserAgent 覆盖下载请求最终发送的 User-Agent。
	UserAgent string
	// Headers 保存重复 --header 传入的额外请求头。
	Headers []string
	// HeadersJSON 保存 JSON 对象形式传入的额外请求头。
	HeadersJSON string
	// URLSwitchInterval 控制单条 URL 连续下载达到指定秒数后主动换链。
	URLSwitchInterval float64
	// ThreadStartDelay 控制多线程顺序启动的间隔秒数。
	ThreadStartDelay float64
	// Config 是配置保存、加载、删除和统计归属使用的名称。
	Config string
	// SaveConfig 表示只保存当前参数为配置，不立即运行。
	SaveConfig bool
	// LoadConfig 表示运行前先读取指定持久化配置。
	LoadConfig bool
	// ListConfigs 表示只列出当前保存的配置。
	ListConfigs bool
	// DeleteConfig 表示只删除指定持久化配置。
	DeleteConfig bool
	// ShowStats 表示只展示历史统计。
	ShowStats bool
	// StatsLimit 限制 CLI 历史统计展示条数。
	StatsLimit int
	// NoGUI 表示使用 CLI 路径而不是默认 Web 控制台。
	NoGUI bool
	// Help 表示 pflag 已经处理过帮助输出。
	Help bool
}

// Parse 解析 GNU 风格长参数，并兼容 `--urls a b` 这种旧 README 写法。
func Parse(argv []string) (Args, *pflag.FlagSet, error) {
	args := Args{URLStrategy: "random", Threads: 8, Config: "default", StatsLimit: 5}
	fs := pflag.NewFlagSet("traffic_consumer", pflag.ContinueOnError)
	fs.StringArrayVarP(&args.URLs, "urls", "u", nil, "要下载的URL列表，可以重复指定")
	fs.StringVar(&args.URLStrategy, "url-strategy", "random", "URL选择策略: random 或 round_robin")
	fs.IntVarP(&args.Threads, "threads", "t", 8, "下载线程数")
	fs.Float64VarP(&args.LimitSpeed, "limit", "l", 0, "下载速度限制，单位MB/s，0表示不限速")
	fs.Int64VarP(&args.Duration, "duration", "d", 0, "持续时间，单位秒")
	fs.Int64VarP(&args.Count, "count", "c", 0, "下载次数")
	fs.StringVar(&args.CronExpr, "cron", "", "Cron表达式")
	fs.Int64Var(&args.TrafficLimit, "traffic-limit", 0, "流量限制，单位MB")
	fs.Int64Var(&args.Interval, "interval", 0, "间隔执行时间，单位分钟")
	fs.BoolVar(&args.AutoRemoveFailedURL, "auto-remove-failed-url", false, "失败 URL 自动移除")
	fs.BoolVar(&args.AutoStart, "auto-start", false, "保存配置时标记为 Web 启动自启")
	fs.StringVar(&args.UserAgent, "user-agent", "", "自定义下载请求的 User-Agent")
	fs.StringArrayVar(&args.Headers, "header", nil, "自定义请求头，格式为 'Header-Name: value'")
	fs.StringVar(&args.HeadersJSON, "headers-json", "", "以 JSON 对象形式传入自定义请求头")
	fs.Float64Var(&args.URLSwitchInterval, "url-switch-interval", 0, "单条 URL 连续下载后强制切换")
	fs.Float64Var(&args.ThreadStartDelay, "thread-start-delay", 0, "多线程启动间隔，单位秒")
	fs.StringVar(&args.Config, "config", "default", "配置名称")
	fs.BoolVar(&args.SaveConfig, "save-config", false, "保存当前配置")
	fs.BoolVar(&args.LoadConfig, "load-config", false, "加载指定配置")
	fs.BoolVar(&args.ListConfigs, "list-configs", false, "列出所有保存的配置")
	fs.BoolVar(&args.DeleteConfig, "delete-config", false, "删除指定配置")
	fs.BoolVar(&args.ShowStats, "show-stats", false, "显示历史统计数据")
	fs.IntVar(&args.StatsLimit, "stats-limit", 5, "显示的历史统计数据条数")
	fs.BoolVar(&args.NoGUI, "no-gui", false, "不启动Web UI，仅使用命令行")
	if err := fs.Parse(argv); err != nil {
		if err == pflag.ErrHelp {
			args.Help = true
			return args, fs, nil
		}
		return args, fs, err
	}
	if fs.Changed("urls") {
		for _, leftover := range fs.Args() {
			if strings.HasPrefix(leftover, "-") {
				continue
			}
			args.URLs = append(args.URLs, leftover)
		}
	}
	return args, fs, nil
}

// IsCLIMode 判断是否进入 CLI 路径。
func (a Args) IsCLIMode() bool {
	return a.NoGUI || a.SaveConfig || a.ListConfigs || a.DeleteConfig || a.ShowStats
}

// Run 执行 CLI 命令。
func Run(args Args, paths config.Paths) error {
	store := stats.NewStore(paths.StatsFile)
	if args.ListConfigs {
		return listConfigs(paths.ConfigFile)
	}
	if args.DeleteConfig {
		deleted, err := storage.DeleteConfigEntry(paths.ConfigFile, args.Config)
		if err != nil {
			return err
		}
		if deleted {
			fmt.Printf("配置 '%s' 已删除\n", storage.NormalizeConfigName(args.Config))
		} else {
			fmt.Printf("配置 '%s' 不存在\n", storage.NormalizeConfigName(args.Config))
		}
		return nil
	}
	if args.ShowStats {
		return showStats(store, args.StatsLimit)
	}

	cfg, err := buildConfig(args, paths)
	if err != nil {
		return err
	}
	if args.SaveConfig {
		if err := storage.SaveConfigEntry(paths.ConfigFile, args.Config, cfg); err != nil {
			return err
		}
		fmt.Printf("配置 '%s' 已保存\n", storage.NormalizeConfigName(args.Config))
		return nil
	}

	logger := func(level, message, color string) {
		fmt.Println(message)
	}
	c := consumer.New(cfg, paths, store, logger, nil)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if isScheduled(cfg) {
		return runScheduled(ctx, cfg, paths, store, logger)
	}
	c.RunOnce(ctx)
	return nil
}

func buildConfig(args Args, paths config.Paths) (model.Config, error) {
	var cfg model.Config
	loadedConfig := false
	if args.LoadConfig {
		loaded, ok, err := storage.LoadConfigEntry(paths.ConfigFile, args.Config)
		if err != nil {
			return model.Config{}, err
		}
		if ok {
			cfg = loaded
			loadedConfig = true
		}
	}

	// 显式空 URL 列表表示配置刻意禁用下载链接，不能退回默认测试 URL。
	configHasExplicitURLs := loadedConfig && (cfg.URLs != nil || cfg.URL != "")
	if !configHasExplicitURLs && len(args.URLs) > 0 {
		cfg.URLs = args.URLs
	}
	if cfg.URLStrategy == "" {
		cfg.URLStrategy = args.URLStrategy
	}
	if cfg.Threads == 0 {
		cfg.Threads = args.Threads
	}
	if cfg.LimitSpeed == 0 {
		cfg.LimitSpeed = args.LimitSpeed
	}
	if cfg.Duration == nil && args.Duration > 0 {
		cfg.Duration = &args.Duration
	}
	if cfg.Count == nil && args.Count > 0 {
		cfg.Count = &args.Count
	}
	if cfg.CronExpr == "" {
		cfg.CronExpr = args.CronExpr
	}
	if cfg.TrafficLimit == nil && args.TrafficLimit > 0 {
		cfg.TrafficLimit = &args.TrafficLimit
	}
	if cfg.Interval == nil && args.Interval > 0 {
		cfg.Interval = &args.Interval
	}
	if !cfg.AutoRemoveFailedURL {
		cfg.AutoRemoveFailedURL = args.AutoRemoveFailedURL
	}
	if !cfg.AutoStart {
		cfg.AutoStart = args.AutoStart
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = args.UserAgent
	}
	if len(cfg.RequestHeaders) == 0 {
		headers, err := mergeHeaders(args.Headers, args.HeadersJSON)
		if err != nil {
			return model.Config{}, err
		}
		cfg.RequestHeaders = headers
	}
	if cfg.URLSwitchInterval == nil && args.URLSwitchInterval > 0 {
		cfg.URLSwitchInterval = &args.URLSwitchInterval
	}
	if cfg.ThreadStartDelay == nil && args.ThreadStartDelay >= 0 {
		cfg.ThreadStartDelay = &args.ThreadStartDelay
	}
	cfg.ConfigName = storage.NormalizeConfigName(args.Config)
	useDefaultURLs := !configHasExplicitURLs && len(args.URLs) == 0
	return cfg.Normalize(config.DefaultURLs, args.Config, useDefaultURLs), nil
}

func isScheduled(cfg model.Config) bool {
	return strings.TrimSpace(cfg.CronExpr) != "" || cfg.Interval != nil && *cfg.Interval > 0
}

func runScheduled(ctx context.Context, cfg model.Config, paths config.Paths, store *stats.Store, logger consumer.Logger) error {
	spec, description, err := scheduleSpec(cfg)
	if err != nil {
		return err
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.Local
	}
	scheduler := cron.New(cron.WithLocation(loc), cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)))
	c := consumer.New(cfg, paths, store, logger, nil)
	id, err := scheduler.AddFunc(spec, func() {
		c.RunOnce(ctx)
	})
	if err != nil {
		return fmt.Errorf("无效的调度配置: %w", err)
	}
	scheduler.Start()
	defer func() {
		stopCtx := scheduler.Stop()
		select {
		case <-stopCtx.Done():
		case <-time.After(3 * time.Second):
		}
	}()

	fmt.Println(description)
	fmt.Println("调度器已启动。按Ctrl+C停止。")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	printNextRun := func() {
		entry := scheduler.Entry(id)
		if entry.Next.IsZero() {
			return
		}
		remaining := time.Until(entry.Next).Round(time.Second)
		if remaining < 0 {
			remaining = 0
		}
		fmt.Printf("\r状态: 等待执行 | 下一次执行时间: %s | 距离下次执行还有: %s", entry.Next.Format("2006-01-02 15:04:05"), remaining)
	}
	printNextRun()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("\n接收到停止信号，正在停止调度器...")
			c.Stop()
			return nil
		case <-ticker.C:
			printNextRun()
		}
	}
}

func scheduleSpec(cfg model.Config) (spec string, description string, err error) {
	if strings.TrimSpace(cfg.CronExpr) != "" {
		return strings.TrimSpace(cfg.CronExpr), "已设置Cron调度: " + strings.TrimSpace(cfg.CronExpr), nil
	}
	if cfg.Interval != nil && *cfg.Interval > 0 {
		return fmt.Sprintf("@every %dm", *cfg.Interval), fmt.Sprintf("已设置间隔调度: 每%d分钟执行一次", *cfg.Interval), nil
	}
	return "", "", fmt.Errorf("未设置有效调度参数")
}

func mergeHeaders(lines []string, headersJSON string) (model.HeaderMap, error) {
	headers := model.HeaderMap{}
	fromJSON, err := model.ParseHeadersJSON(headersJSON)
	if err != nil {
		return nil, err
	}
	for name, value := range fromJSON {
		headers[name] = value
	}
	fromLines, err := model.ParseHeaderLines(lines)
	if err != nil {
		return nil, err
	}
	for name, value := range fromLines {
		headers[name] = value
	}
	if len(headers) == 0 {
		return nil, nil
	}
	return headers, nil
}

func listConfigs(path string) error {
	configs, err := storage.LoadConfigs(path)
	if err != nil {
		return err
	}
	if len(configs) == 0 {
		fmt.Println("没有保存的配置")
		return nil
	}
	fmt.Println("=== 保存的配置 ===")
	names, _ := storage.ListConfigNames(path)
	for _, name := range names {
		cfg := configs[name].Normalize(config.DefaultURLs, name, false)
		fmt.Printf("\n配置名称: %s\n", name)
		if len(cfg.URLs) > 0 {
			fmt.Printf("  URLs: %v\n", cfg.URLs)
		} else if cfg.URL != "" {
			fmt.Printf("  URL: %s\n", cfg.URL)
		} else {
			fmt.Println("  URLs: []")
		}
		fmt.Printf("  URL策略: %s\n", firstNonEmpty(cfg.URLStrategy, "random"))
		fmt.Printf("  线程数: %d\n", cfg.Threads)
		fmt.Printf("  限速: %.2f MB/s (0表示不限速)\n", cfg.LimitSpeed)
		if cfg.Duration != nil {
			fmt.Printf("  持续时间: %s\n", time.Duration(*cfg.Duration)*time.Second)
		} else {
			fmt.Println("  持续时间: 无限制")
		}
		if cfg.Count != nil {
			fmt.Printf("  下载次数: %d\n", *cfg.Count)
		} else {
			fmt.Println("  下载次数: 无限制")
		}
		if cfg.CronExpr != "" {
			fmt.Printf("  Cron表达式: %s\n", cfg.CronExpr)
		}
		if cfg.Interval != nil {
			fmt.Printf("  间隔执行: 每 %d 分钟\n", *cfg.Interval)
		}
		if cfg.AutoStart {
			fmt.Println("  Web启动自启: 开启")
		}
		if cfg.UserAgent != "" {
			fmt.Printf("  User-Agent: %s\n", cfg.UserAgent)
		}
		if len(cfg.RequestHeaders) > 0 {
			fmt.Printf("  自定义请求头: %v\n", map[string]string(cfg.RequestHeaders))
		}
		if cfg.URLSwitchInterval != nil {
			fmt.Printf("  URL切换时限: %.3g 秒\n", *cfg.URLSwitchInterval)
		}
		if cfg.ThreadStartDelay != nil {
			fmt.Printf("  线程启动间隔: %.3g 秒\n", *cfg.ThreadStartDelay)
		}
		fmt.Printf("  失败链接自动移除: %s\n", onOff(cfg.AutoRemoveFailedURL))
	}
	return nil
}

func showStats(store *stats.Store, limit int) error {
	runs, err := store.LoadRuns()
	if err != nil {
		return err
	}
	if len(runs) == 0 {
		fmt.Println("没有历史统计数据")
		return nil
	}
	type runEntry struct {
		id  string
		run model.StatsRun
	}
	entries := make([]runEntry, 0, len(runs))
	for id, run := range runs {
		entries = append(entries, runEntry{id: id, run: run})
	}
	sort.Slice(entries, func(i, j int) bool {
		left := firstNonEmpty(entries[i].run.EndTime, entries[i].run.StartTime, entries[i].id)
		right := firstNonEmpty(entries[j].run.EndTime, entries[j].run.StartTime, entries[j].id)
		return left > right
	})
	if limit <= 0 || limit > len(entries) {
		limit = len(entries)
	}
	fmt.Printf("=== 流量消耗历史记录 (最近 %d 条) ===\n", limit)
	for i := 0; i < limit; i++ {
		entry := entries[i]
		run := entry.run
		fmt.Printf("\n运行ID: %s\n", entry.id)
		fmt.Printf("  配置名称: %s\n", firstNonEmpty(run.ConfigName, "默认"))
		fmt.Printf("  开始时间: %s\n", firstNonEmpty(run.StartTime, "N/A"))
		fmt.Printf("  结束时间: %s\n", firstNonEmpty(run.EndTime, "N/A"))
		fmt.Printf("  结果: %s\n", firstNonEmpty(run.Result, "成功"))
		fmt.Printf("  总消耗流量: %s\n", stats.FormatBytes(float64(run.TotalBytes)))
		fmt.Printf("  下载次数: %d\n", run.DownloadCount)
		fmt.Printf("  运行时间: %s\n", time.Duration(run.ElapsedSeconds)*time.Second)
		if i < limit-1 {
			fmt.Println("------------------------")
		}
	}
	return nil
}

func onOff(value bool) string {
	if value {
		return "开启"
	}
	return "关闭"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// DumpConfig 方便测试和排错时查看 CLI 合成后的配置。
func DumpConfig(cfg model.Config) {
	_ = json.NewEncoder(os.Stdout).Encode(cfg)
}
