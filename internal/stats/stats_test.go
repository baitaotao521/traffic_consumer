package stats

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/baitaotao521/traffic_consumer/internal/model"
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
