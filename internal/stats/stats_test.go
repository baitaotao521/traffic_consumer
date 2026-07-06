package stats

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/baitaotao521/traffic_consumer/internal/model"
	"github.com/baitaotao521/traffic_consumer/internal/storage"
)

func TestStoreSavesAndAggregatesRuns(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "stats.json"))
	if err := store.SaveRun(model.StatsRun{
		ConfigName:     "demo",
		TotalBytes:     1024,
		DownloadCount:  2,
		StartTime:      "2026-07-04 00:00:00",
		EndTime:        "2026-07-04 00:00:01",
		ElapsedSeconds: 1,
		Result:         "成功",
	}); err != nil {
		t.Fatalf("save run: %v", err)
	}

	records, err := store.LoadRecords()
	if err != nil {
		t.Fatalf("load records: %v", err)
	}
	if len(records) != 1 || records[0].BytesConsumed != "1.00 KB" || records[0].DownloadCount != 2 {
		t.Fatalf("unexpected records: %+v", records)
	}

	summary, err := store.SummaryByConfig()
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if summary["demo"].TotalBytesRaw != 1024 || summary["demo"].DownloadCount != 2 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestStoreConcurrentSavesKeepEveryRun(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "stats.json"))
	const totalRuns = 40

	var wg sync.WaitGroup
	errs := make(chan error, totalRuns)
	for i := 0; i < totalRuns; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- store.SaveRun(model.StatsRun{
				ConfigName:     fmt.Sprintf("demo-%02d", i),
				TotalBytes:     int64(i + 1),
				DownloadCount:  1,
				StartTime:      "2026-07-04 00:00:00",
				EndTime:        "2026-07-04 00:00:01",
				ElapsedSeconds: 1,
				Result:         "成功",
			})
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("save run: %v", err)
		}
	}

	runs, err := store.LoadRuns()
	if err != nil {
		t.Fatalf("load runs: %v", err)
	}
	if len(runs) != totalRuns {
		t.Fatalf("expected %d runs, got %d: %+v", totalRuns, len(runs), runs)
	}
	var totalBytes int64
	for _, run := range runs {
		totalBytes += run.TotalBytes
	}
	expectedBytes := int64(totalRuns * (totalRuns + 1) / 2)
	if totalBytes != expectedBytes {
		t.Fatalf("expected total bytes %d, got %d", expectedBytes, totalBytes)
	}
}

func TestStoreSummaryCacheInvalidatesAfterSaveRun(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "stats.json"))
	if err := store.SaveRun(model.StatsRun{ConfigName: "demo", TotalBytes: 1, DownloadCount: 1, Result: "成功"}); err != nil {
		t.Fatalf("save first run: %v", err)
	}
	summary, err := store.SummaryByConfig()
	if err != nil {
		t.Fatalf("summary first run: %v", err)
	}
	if summary["demo"].TotalBytesRaw != 1 {
		t.Fatalf("expected first summary to include 1 byte, got %+v", summary["demo"])
	}

	if err := store.SaveRun(model.StatsRun{ConfigName: "demo", TotalBytes: 2, DownloadCount: 3, Result: "成功"}); err != nil {
		t.Fatalf("save second run: %v", err)
	}
	summary, err = store.SummaryByConfig()
	if err != nil {
		t.Fatalf("summary second run: %v", err)
	}
	if summary["demo"].TotalBytesRaw != 3 || summary["demo"].DownloadCount != 4 {
		t.Fatalf("expected cache to refresh after SaveRun, got %+v", summary["demo"])
	}
}

func TestStoreSummaryCacheInvalidatesAfterExternalWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.json")
	store := NewStore(path)
	storage.MustWriteJSON(path, map[string]model.StatsRun{
		"20260706000000.000000001": {
			ConfigName:    "demo",
			TotalBytes:    1,
			DownloadCount: 1,
			EndTime:       "2026-07-06 00:00:00",
			Result:        "成功",
		},
	})
	summary, err := store.SummaryByConfig()
	if err != nil {
		t.Fatalf("summary before external write: %v", err)
	}
	if summary["demo"].TotalBytesRaw != 1 {
		t.Fatalf("expected initial external summary, got %+v", summary["demo"])
	}

	// stats.json 可能被另一个运行态或用户手工替换；文件状态变化后 dashboard 历史汇总必须刷新。
	storage.MustWriteJSON(path, map[string]model.StatsRun{
		"20260706000000.000000001": {
			ConfigName:    "demo",
			TotalBytes:    1,
			DownloadCount: 1,
			EndTime:       "2026-07-06 00:00:00",
			Result:        "成功",
		},
		"20260706000001.000000001": {
			ConfigName:    "demo",
			TotalBytes:    2048,
			DownloadCount: 2,
			EndTime:       "2026-07-06 00:00:01",
			Result:        "成功",
		},
	})
	summary, err = store.SummaryByConfig()
	if err != nil {
		t.Fatalf("summary after external write: %v", err)
	}
	if summary["demo"].TotalBytesRaw != 2049 || summary["demo"].DownloadCount != 3 {
		t.Fatalf("expected refreshed external summary, got %+v", summary["demo"])
	}
}

func TestStoreLoadRunsReturnsCacheSafeCopy(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "stats.json"))
	if err := store.SaveRun(model.StatsRun{
		ConfigName: "demo",
		URLs:       []string{"https://example.com/a"},
		URLUsage:   map[string]int64{"https://example.com/a": 1},
		History:    []model.HistoryRecord{{Result: "成功", Bytes: 1}},
		Result:     "成功",
	}); err != nil {
		t.Fatalf("save run: %v", err)
	}

	runs, err := store.LoadRuns()
	if err != nil {
		t.Fatalf("load runs: %v", err)
	}
	for key, run := range runs {
		run.URLs[0] = "https://example.com/mutated"
		run.URLUsage["https://example.com/a"] = 99
		run.History[0].Bytes = 99
		runs[key] = run
	}

	runs, err = store.LoadRuns()
	if err != nil {
		t.Fatalf("load runs after mutation: %v", err)
	}
	for _, run := range runs {
		if run.URLs[0] != "https://example.com/a" || run.URLUsage["https://example.com/a"] != 1 || run.History[0].Bytes != 1 {
			t.Fatalf("LoadRuns returned data that could mutate cache: %+v", run)
		}
	}
}
