package stats

import (
	"fmt"
	"sort"
	"time"

	"github.com/baitaotao521/traffic_consumer/internal/model"
	"github.com/baitaotao521/traffic_consumer/internal/storage"
)

// Store 封装 stats.json 的读写和聚合。
type Store struct {
	// Path 是 stats.json 的完整路径。
	Path string
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
	return storage.MutateJSON(s.Path, &runs, func() (bool, error) {
		runID := uniqueRunID(runs)
		runs[runID] = run
		return true, nil
	})
}

// LoadRuns 读取全部统计记录。
func (s *Store) LoadRuns() (map[string]model.StatsRun, error) {
	runs := map[string]model.StatsRun{}
	if err := storage.ReadJSON(s.Path, &runs); err != nil {
		return map[string]model.StatsRun{}, err
	}
	return runs, nil
}

// LoadRecords 返回倒序历史记录，供 Web 历史和 CLI 展示使用。
func (s *Store) LoadRecords() ([]model.HistoryRecord, error) {
	runs, err := s.LoadRuns()
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(runs))
	for key := range runs {
		keys = append(keys, key)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	records := make([]model.HistoryRecord, 0, len(keys))
	for _, key := range keys {
		run := runs[key]
		records = append(records, recordFromRun(run))
	}
	return records, nil
}

// SummaryByConfig 聚合每个配置的历史总流量与下载数。
func (s *Store) SummaryByConfig() (map[string]model.PlanSummary, error) {
	runs, err := s.LoadRuns()
	if err != nil {
		return nil, err
	}
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
	return summary, nil
}

// HistoryForConfig 合并指定配置的持久化历史。
func (s *Store) HistoryForConfig(name string, limit int) ([]model.HistoryRecord, error) {
	name = storage.NormalizeConfigName(name)
	runs, err := s.LoadRuns()
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(runs))
	for key := range runs {
		keys = append(keys, key)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	records := make([]model.HistoryRecord, 0)
	for _, key := range keys {
		run := runs[key]
		if storage.NormalizeConfigName(run.ConfigName) != name {
			continue
		}
		records = append(records, recordFromRun(run))
		if limit > 0 && len(records) >= limit {
			break
		}
	}
	return records, nil
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
