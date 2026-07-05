package configguard

import (
	"strings"
	"testing"

	"github.com/baitaotao521/traffic_consumer/internal/model"
)

func TestValidateAllowsEmptyURLsOnlyForSave(t *testing.T) {
	cfg := model.Config{ConfigName: "demo", URLs: []string{}, Threads: 1, URLStrategy: "random"}
	if _, _, err := Validate("demo", cfg, ModeSave); err != nil {
		t.Fatalf("save should allow explicit empty URLs: %v", err)
	}
	if _, _, err := Validate("demo", cfg, ModeStart); err == nil {
		t.Fatalf("start should reject explicit empty URLs")
	}
	if _, _, err := Validate("demo", cfg, ModeTempTest); err == nil {
		t.Fatalf("temp test should reject explicit empty URLs")
	}
}

func TestValidateRejectsUnsafeNamesAndURLs(t *testing.T) {
	for _, name := range []string{
		"",
		"bad/name",
		"bad\\name",
		"bad\nname",
		strings.Repeat("长", 81),
	} {
		cfg := model.Config{ConfigName: name, URLs: []string{"https://example.com"}, Threads: 1, URLStrategy: "random"}
		if _, _, err := Validate("", cfg, ModeSave); err == nil {
			t.Fatalf("expected unsafe name rejection for %q", name)
		}
	}

	cfg := model.Config{ConfigName: "demo", URLs: []string{"ftp://example.com/file"}, Threads: 1, URLStrategy: "random"}
	if _, _, err := Validate("", cfg, ModeSave); err == nil {
		t.Fatalf("expected non-http URL rejection")
	}

	cfg = model.Config{ConfigName: "demo", URLs: []string{"https://example.com/a", "https://example.com/a"}, Threads: 1, URLStrategy: "random"}
	if _, _, err := Validate("", cfg, ModeSave); err == nil {
		t.Fatalf("expected duplicate URL rejection")
	}
}

func TestValidateRejectsHeaderConflicts(t *testing.T) {
	cfg := model.Config{
		ConfigName:     "demo",
		URLs:           []string{"https://example.com"},
		Threads:        1,
		URLStrategy:    "random",
		UserAgent:      "TrafficConsumer",
		RequestHeaders: model.HeaderMap{"User-Agent": "Other"},
	}
	if _, _, err := Validate("", cfg, ModeSave); err == nil {
		t.Fatalf("expected User-Agent conflict")
	}

	cfg.UserAgent = ""
	cfg.RequestHeaders = model.HeaderMap{"X-Test": "a", "x-test": "b"}
	if _, _, err := Validate("", cfg, ModeSave); err == nil {
		t.Fatalf("expected duplicate case-insensitive header rejection")
	}

	cfg.RequestHeaders = model.HeaderMap{"Bad Header": "a"}
	if _, _, err := Validate("", cfg, ModeSave); err == nil {
		t.Fatalf("expected invalid header name rejection")
	}

	cfg.RequestHeaders = model.HeaderMap{"X-Test": ""}
	if _, _, err := Validate("", cfg, ModeSave); err == nil {
		t.Fatalf("expected empty header value rejection")
	}
}

func TestValidateRejectsScheduleAndNumericConflicts(t *testing.T) {
	interval := int64(5)
	cfg := model.Config{
		ConfigName:  "demo",
		URLs:        []string{"https://example.com"},
		Threads:     1,
		URLStrategy: "random",
		CronExpr:    "*/5 * * * *",
		Interval:    &interval,
	}
	if _, _, err := Validate("", cfg, ModeSave); err == nil {
		t.Fatalf("expected cron and interval conflict")
	}

	count := int64(0)
	cfg.CronExpr = ""
	cfg.Interval = nil
	cfg.Count = &count
	if _, _, err := Validate("", cfg, ModeSave); err == nil {
		t.Fatalf("expected zero count rejection")
	}

	cfg.Count = nil
	cfg.CronExpr = "bad cron"
	if _, _, err := Validate("", cfg, ModeSave); err == nil {
		t.Fatalf("expected invalid cron rejection")
	}
}
