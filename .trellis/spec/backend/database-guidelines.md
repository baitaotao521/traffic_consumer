# Database Guidelines

> Persistence conventions for Traffic Consumer.

---

## Overview

This project does not use a database, ORM, migrations, or SQL query layer.
Persistence is JSON-file based and intentionally simple:

- `~/.traffic_consumer/config.json` stores named runtime configurations.
- `~/.traffic_consumer/stats.json` stores run history keyed by timestamp IDs.
- `TRAFFIC_CONSUMER_CONFIG_DIR` overrides the default config directory for
  Docker, NAS, tests, and local isolation.

All maintained persistence code lives in Go:

- `internal/config/config.go` resolves file locations and default constants.
- `internal/storage/storage.go` owns config JSON CRUD.
- `internal/stats/stats.go` owns stats JSON history and aggregation.
- `internal/model/model.go` owns the serialized schema.

The removed Python storage path is not a source for new persistence work.
Historical compatibility rules must live in Go tests and comments.

---

## File Access Patterns

Use the existing storage helpers instead of opening config/stat files in feature
code. This keeps corrupt-file tolerance, directory creation, JSON formatting,
and config-name normalization consistent.

Example from `internal/storage/storage.go`:

```go
func SaveConfigEntry(path, name string, cfg model.Config) error {
    configs, err := LoadConfigs(path)
    if err != nil {
        return err
    }
    name = NormalizeConfigName(name)
    cfg.ConfigName = ""
    cfg.Name = ""
    configs[name] = cfg
    return WriteJSON(path, configs)
}
```

Example from `internal/stats/stats.go`:

```go
runID := time.Now().Format("20060102150405")
runs[runID] = run
return storage.WriteJSON(s.Path, runs)
```

Business rules to preserve:

- Missing config/stat files load as empty data.
- Corrupt JSON currently loads as empty data to preserve legacy tolerance.
- Saved configs are stored under normalized names; empty config names become
  `default`.
- `SaveConfigEntry` clears transient `ConfigName` and `Name` before writing a
  named config because the outer JSON key already identifies it.
- Parent directories are created automatically before JSON writes.
- JSON files are indented with two spaces and end with a newline.

---

## Schema Compatibility

`internal/model.Config` is shared by CLI, HTTP API, frontend payloads, and
`config.json`.

Compatibility examples:

```go
// URL supports old single-link config files; Normalize moves it into URLs.
URL string `json:"url,omitempty"`

// HeaderMap accepts object, string-array, and newline-string header formats.
type HeaderMap map[string]string
```

Keep these rules intact:

- `urls: []` means the user explicitly saved an empty URL list. It must not
  fall back to default URLs.
- Missing `urls` may fall back to `config.DefaultURLs` only when the caller
  explicitly opts into default URLs.
- `request_headers` accepts old arrays like
  `["Referer: https://example.com"]` and the current object format.
- `user_agent` overrides a `User-Agent` value inside `request_headers` so the UI
  has one clear source of truth.
- `stats.json` keeps string times in `2006-01-02 15:04:05` form for historical
  runs, while Web event timestamps use RFC3339.

---

## No Migrations

There is no migration framework. Schema evolution is handled with tolerant
readers and normalization in `internal/model` and `internal/storage`.

When adding a field:

- Add the field to `internal/model.Config` or the appropriate payload struct.
- Keep `omitempty` unless zero values must be serialized for compatibility.
- Update `Normalize` when the field has defaulting or legacy translation rules.
- Update CLI parsing in `internal/cli/cli.go` and HTTP/frontend payload handling
  only when the field is user-configurable.
- Add tests with raw JSON fixtures when old files need compatibility coverage.

The current pattern is shown in `internal/storage/storage_test.go`, where the
test writes a raw legacy JSON file and verifies Go normalization.

---

## Testing Requirements

Persistence tests must use `t.TempDir()` and explicit `config.Paths` values.
Never write to the developer's real `~/.traffic_consumer` directory in tests.

Example from `internal/consumer/consumer_test.go`:

```go
dir := t.TempDir()
paths := config.Paths{
    ConfigDir:  dir,
    ConfigFile: filepath.Join(dir, "config.json"),
    StatsFile:  filepath.Join(dir, "stats.json"),
}
```

Run `go test ./...` after changes to persistence, config normalization, stats
aggregation, failed-URL removal, or CLI config commands.

---

## Common Mistakes

- Writing JSON directly from HTTP handlers or consumers instead of using
  `storage` and `stats` helpers.
- Treating `nil` URLs and an empty URL slice as the same value. They have
  different business meaning.
- Adding a config field only to CLI or only to Web payloads, creating split
  behavior.
- Reintroducing product Python persistence. Go owns schema evolution now.
