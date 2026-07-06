package stats

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/baitaotao521/traffic_consumer/internal/model"
	"github.com/baitaotao521/traffic_consumer/internal/storage"
)

// Store 封装 stats.json 的读写和聚合。
type Store struct {
	// Path 是 stats.json 的完整路径。
	Path string
	// mu 保护 stats.json 解析缓存，避免高频 dashboard 轮询重复解析同一个文件。
	mu sync.Mutex
	// cache 保存最近一次读取 stats.json 后得到的排序键和聚合结果。
	cache statsCache
}

type statsCache struct {
	// valid 表示当前缓存已经从 stats.json 成功加载过，可参与文件状态比对。
	valid bool
	// state 记录加载缓存时的文件状态，用于识别外部写入或删除。
	state statsFileState
	// runs 保存按运行 ID 索引的原始统计记录。
	runs map[string]model.StatsRun
	// keys 保存倒序排列的运行 ID，历史展示和汇总共用这一份排序结果。
	keys []string
	// summary 保存按配置名聚合后的历史总量和最近一次运行。
	summary map[string]model.PlanSummary
}

type statsFileState struct {
	// exists 区分“文件不存在”和“存在但为空文件”，两者都读成空对象但缓存失效条件不同。
	exists bool
	// size 是 stats.json 当前字节数，用于捕捉外部写入。
	size int64
	// modTime 是 stats.json 当前修改时间，用于捕捉外部写入。
	modTime time.Time
}

// NewStore 创建统计存储。
func NewStore(path string) *Store {
	return &Store{Path: path}
}

// FormatBytes 保持 Python 版两位小数的人类可读格式。
func FormatBytes(bytesValue float64) string {
	if bytesValue < 1024 {
		return fmt.Sprintf("%.2f B", bytesValue)
	}
	if bytesValue < 1024*1024 {
		return fmt.Sprintf("%.2f KB", bytesValue/1024)
	}
	if bytesValue < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", bytesValue/(1024*1024))
	}
	return fmt.Sprintf("%.2f GB", bytesValue/(1024*1024*1024))
}

// SaveRun 将一次运行写入 stats.json。
func (s *Store) SaveRun(run model.StatsRun) error {
	runs := map[string]model.StatsRun{}
	if err := storage.MutateJSON(s.Path, &runs, func() (bool, error) {
		runID := uniqueRunID(runs)
		runs[runID] = run
		return true, nil
	}); err != nil {
		return err
	}
	s.invalidateCache()
	return nil
}

// LoadRuns 读取全部统计记录。
func (s *Store) LoadRuns() (map[string]model.StatsRun, error) {
	cache, err := s.cachedStats()
	if err != nil {
		return map[string]model.StatsRun{}, err
	}
	return cloneRuns(cache.runs), nil
}

// LoadRecords 返回倒序历史记录，供 Web 历史和 CLI 展示使用。
func (s *Store) LoadRecords() ([]model.HistoryRecord, error) {
	cache, err := s.cachedStats()
	if err != nil {
		return nil, err
	}
	records := recordsFromCache(cache, "", 0)
	return records, nil
}

// SummaryByConfig 聚合每个配置的历史总流量与下载数。
func (s *Store) SummaryByConfig() (map[string]model.PlanSummary, error) {
	cache, err := s.cachedStats()
	if err != nil {
		return nil, err
	}
	return cloneSummaryMap(cache.summary), nil
}

// HistoryForConfig 合并指定配置的持久化历史。
func (s *Store) HistoryForConfig(name string, limit int) ([]model.HistoryRecord, error) {
	name = storage.NormalizeConfigName(name)
	cache, err := s.cachedStats()
	if err != nil {
		return nil, err
	}
	return recordsFromCache(cache, name, limit), nil
}

func (s *Store) cachedStats() (statsCache, error) {
	state, err := currentStatsFileState(s.Path)
	if err != nil {
		return statsCache{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cache.valid && s.cache.state == state {
		return s.cache, nil
	}

	runs := map[string]model.StatsRun{}
	if err := storage.ReadJSON(s.Path, &runs); err != nil {
		return statsCache{}, err
	}
	s.cache = buildStatsCache(state, runs)
	return s.cache, nil
}

func (s *Store) invalidateCache() {
	s.mu.Lock()
	s.cache.valid = false
	s.mu.Unlock()
}

func currentStatsFileState(path string) (statsFileState, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return statsFileState{}, nil
		}
		return statsFileState{}, err
	}
	return statsFileState{exists: true, size: info.Size(), modTime: info.ModTime()}, nil
}

func buildStatsCache(state statsFileState, runs map[string]model.StatsRun) statsCache {
	keys := make([]string, 0, len(runs))
	for key := range runs {
		keys = append(keys, key)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	summary := map[string]model.PlanSummary{}
	for _, key := range keys {
		run := runs[key]
		name := storage.NormalizeConfigName(firstNonEmpty(run.ConfigName, "default"))
		item := summary[name]
		item.Name = name
		item.TotalBytesRaw += run.TotalBytes
		item.DownloadCount += run.DownloadCount
		item.TotalBytes = FormatBytes(float64(item.TotalBytesRaw))
		if item.LastRun == nil {
			record := recordFromRun(run)
			item.LastRun = &record
		}
		summary[name] = item
	}
	return statsCache{
		valid:   true,
		state:   state,
		runs:    cloneRuns(runs),
		keys:    keys,
		summary: summary,
	}
}

func recordsFromCache(cache statsCache, configName string, limit int) []model.HistoryRecord {
	records := make([]model.HistoryRecord, 0)
	for _, key := range cache.keys {
		run := cache.runs[key]
		if configName != "" && storage.NormalizeConfigName(run.ConfigName) != configName {
			continue
		}
		records = append(records, recordFromRun(run))
		if limit > 0 && len(records) >= limit {
			break
		}
	}
	return records
}

func cloneRuns(runs map[string]model.StatsRun) map[string]model.StatsRun {
	result := make(map[string]model.StatsRun, len(runs))
	for key, run := range runs {
		result[key] = cloneRun(run)
	}
	return result
}

func cloneRun(run model.StatsRun) model.StatsRun {
	if run.URLs != nil {
		run.URLs = append([]string{}, run.URLs...)
	}
	if run.URLUsage != nil {
		usage := make(map[string]int64, len(run.URLUsage))
		for url, count := range run.URLUsage {
			usage[url] = count
		}
		run.URLUsage = usage
	}
	if run.History != nil {
		run.History = append([]model.HistoryRecord{}, run.History...)
	}
	return run
}

func cloneSummaryMap(summary map[string]model.PlanSummary) map[string]model.PlanSummary {
	result := make(map[string]model.PlanSummary, len(summary))
	for name, item := range summary {
		result[name] = clonePlanSummary(item)
	}
	return result
}

func clonePlanSummary(item model.PlanSummary) model.PlanSummary {
	if item.LastRun != nil {
		record := *item.LastRun
		item.LastRun = &record
	}
	if item.StatsHistory != nil {
		item.StatsHistory = append([]model.HistoryRecord{}, item.StatsHistory...)
	}
	return item
}

func recordFromRun(run model.StatsRun) model.HistoryRecord {
	return model.HistoryRecord{
		Timestamp:      firstNonEmpty(run.EndTime, run.StartTime),
		Result:         firstNonEmpty(run.Result, "成功"),
		BytesConsumed:  FormatBytes(float64(run.TotalBytes)),
		DownloadCount:  run.DownloadCount,
		Bytes:          run.TotalBytes,
		ElapsedSeconds: run.ElapsedSeconds,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func uniqueRunID(runs map[string]model.StatsRun) string {
	base := time.Now().Format("20060102150405.000000000")
	runID := base
	for suffix := 1; ; suffix++ {
		if _, exists := runs[runID]; !exists {
			return runID
		}
		runID = fmt.Sprintf("%s-%d", base, suffix)
	}
}
