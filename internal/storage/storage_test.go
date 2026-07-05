package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/baitaotao521/traffic_consumer/internal/model"
)

func TestConfigCRUDAndRemoveURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := model.Config{
		URLs:                []string{"https://example.com/a", "https://example.com/b"},
		URLStrategy:         "round_robin",
		Threads:             3,
		LimitSpeed:          1.5,
		AutoRemoveFailedURL: true,
		RequestHeaders:      model.HeaderMap{"Referer": "https://example.com"},
	}

	if err := SaveConfigEntry(path, " demo ", cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	loaded, ok, err := LoadConfigEntry(path, "demo")
	if err != nil || !ok {
		t.Fatalf("load config ok=%v err=%v", ok, err)
	}
	if loaded.Threads != 3 || loaded.URLStrategy != "round_robin" {
		t.Fatalf("unexpected loaded config: %+v", loaded)
	}

	changed, err := RemoveURLFromConfig(path, "demo", "https://example.com/a")
	if err != nil || !changed {
		t.Fatalf("remove url changed=%v err=%v", changed, err)
	}
	loaded, _, _ = LoadConfigEntry(path, "demo")
	if len(loaded.URLs) != 1 || loaded.URLs[0] != "https://example.com/b" {
		t.Fatalf("url was not removed: %+v", loaded.URLs)
	}

	deleted, err := DeleteConfigEntry(path, "demo")
	if err != nil || !deleted {
		t.Fatalf("delete config deleted=%v err=%v", deleted, err)
	}
}

func TestLoadConfigAcceptsLegacySingleURLAndHeaderArray(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	raw := []byte(`{
	  "legacy": {
	    "url": "https://example.com/one",
	    "threads": 2,
	    "request_headers": ["Referer: https://example.com", "Origin: https://example.com"]
	  }
	}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	cfg, ok, err := LoadConfigEntry(path, "legacy")
	if err != nil || !ok {
		t.Fatalf("load legacy ok=%v err=%v", ok, err)
	}
	normalized := cfg.Normalize([]string{"fallback"}, "legacy", false)
	if len(normalized.URLs) != 1 || normalized.URLs[0] != "https://example.com/one" {
		t.Fatalf("legacy URL was not normalized: %+v", normalized.URLs)
	}
	if normalized.RequestHeaders["Referer"] != "https://example.com" {
		t.Fatalf("legacy headers were not parsed: %+v", normalized.RequestHeaders)
	}
}

func TestSaveConfigEntryAsRenamesAndRequiresForceForOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := SaveConfigEntry(path, "one", model.Config{URLs: []string{"https://example.com/one"}, Threads: 1}); err != nil {
		t.Fatalf("save one: %v", err)
	}
	if err := SaveConfigEntry(path, "two", model.Config{URLs: []string{"https://example.com/two"}, Threads: 2}); err != nil {
		t.Fatalf("save two: %v", err)
	}

	if _, err := SaveConfigEntryAs(path, SaveConfigEntryOptions{
		SourceName: "one",
		TargetName: "two",
		Config:     model.Config{URLs: []string{"https://example.com/new"}, Threads: 3},
	}); !errors.Is(err, ErrConfigConflict) {
		t.Fatalf("expected overwrite conflict, got %v", err)
	}
	if cfg, ok, _ := LoadConfigEntry(path, "one"); !ok || cfg.Threads != 1 {
		t.Fatalf("source should remain after rejected overwrite: ok=%v cfg=%+v", ok, cfg)
	}
	if cfg, ok, _ := LoadConfigEntry(path, "two"); !ok || cfg.Threads != 2 {
		t.Fatalf("target should remain after rejected overwrite: ok=%v cfg=%+v", ok, cfg)
	}

	result, err := SaveConfigEntryAs(path, SaveConfigEntryOptions{
		SourceName: "one",
		TargetName: "two",
		Config:     model.Config{URLs: []string{"https://example.com/new"}, Threads: 3},
		Force:      true,
	})
	if err != nil {
		t.Fatalf("force overwrite: %v", err)
	}
	if !result.SourceExisted || !result.TargetExisted || !result.Renamed || !result.Overwritten {
		t.Fatalf("unexpected result: %+v", result)
	}
	if _, ok, _ := LoadConfigEntry(path, "one"); ok {
		t.Fatalf("source should be removed after force rename overwrite")
	}
	if cfg, ok, _ := LoadConfigEntry(path, "two"); !ok || cfg.Threads != 3 {
		t.Fatalf("target should contain new config: ok=%v cfg=%+v", ok, cfg)
	}
}

func TestSaveConfigEntryAsRequiresForceForCreateExistingTarget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := SaveConfigEntry(path, "demo", model.Config{URLs: []string{"https://example.com/old"}, Threads: 1}); err != nil {
		t.Fatalf("save demo: %v", err)
	}

	if _, err := SaveConfigEntryAs(path, SaveConfigEntryOptions{
		SourceName: "demo",
		TargetName: "demo",
		Config:     model.Config{URLs: []string{"https://example.com/new"}, Threads: 2},
		RequireNew: true,
	}); !errors.Is(err, ErrConfigConflict) {
		t.Fatalf("expected create conflict, got %v", err)
	}
	if cfg, ok, _ := LoadConfigEntry(path, "demo"); !ok || cfg.Threads != 1 {
		t.Fatalf("target should remain unchanged after rejected create overwrite: ok=%v cfg=%+v", ok, cfg)
	}

	result, err := SaveConfigEntryAs(path, SaveConfigEntryOptions{
		SourceName: "demo",
		TargetName: "demo",
		Config:     model.Config{URLs: []string{"https://example.com/new"}, Threads: 2},
		Force:      true,
		RequireNew: true,
	})
	if err != nil {
		t.Fatalf("force create overwrite: %v", err)
	}
	if !result.SourceExisted || !result.TargetExisted || result.Renamed || !result.Overwritten {
		t.Fatalf("unexpected result: %+v", result)
	}
	if cfg, ok, _ := LoadConfigEntry(path, "demo"); !ok || cfg.Threads != 2 {
		t.Fatalf("target should update after forced create overwrite: ok=%v cfg=%+v", ok, cfg)
	}
}

func TestReadJSONTreatsCorruptFileAsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{broken"), 0o644); err != nil {
		t.Fatalf("write corrupt config: %v", err)
	}

	configs, err := LoadConfigs(path)
	if err != nil {
		t.Fatalf("corrupt config should be ignored: %v", err)
	}
	if len(configs) != 0 {
		t.Fatalf("corrupt config should load as empty, got %+v", configs)
	}
}
