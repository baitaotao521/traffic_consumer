package stats

import (
	"path/filepath"
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
