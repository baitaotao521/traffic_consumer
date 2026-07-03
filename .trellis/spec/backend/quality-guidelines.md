# Quality Guidelines

> Required standards for backend changes.

---

## Overview

The maintained backend quality gate is Go-first:

```bash
gofmt -w .
go test ./...
```

Run the frontend build when API contracts, generated static assets, or Vue code
are touched:

```bash
npm --prefix frontend run build
```

Python tests and Python dependencies remain only for legacy compatibility checks.
They must not become the primary acceptance gate for new backend behavior.

---

## Required Patterns

- Keep `main.go` as wiring only. Put behavior into `internal/*` packages.
- Use `context.Context` for cancellation of long-running downloads and
  schedulers.
- Use `t.TempDir()` and explicit `config.Paths` in tests that touch persistence.
- Keep config/stat schema changes in `internal/model` and storage helpers.
- Preserve public JSON field names unless a migration strategy is explicitly
  implemented.
- Add business comments near non-obvious behavior, especially compatibility
  rules carried from the old Python implementation.
- Keep API response messages user-facing and non-duplicated. The frontend shows
  these strings directly.

Examples already in the codebase:

```go
// 显式空 URL 列表表示配置刻意禁用下载链接，不能退回默认测试 URL。
configHasExplicitURLs := loadedConfig && (cfg.URLs != nil || cfg.URL != "")
```

```go
// StartTempTest 使用当前配置执行最多 100MB 的立即验证。
testCfg.AutoRemoveFailedURL = false
```

---

## Forbidden Patterns

- Adding new backend features to `main.py`, `web_ui.py`, `traffic_consumer.py`,
  `app/**`, or `build_config.py`.
- Reintroducing Flask, Socket.IO, PyInstaller, or Python release paths as
  maintained behavior.
- Writing directly to `~/.traffic_consumer` in tests.
- Making real external network calls in unit tests. Use `httptest.NewServer`.
- Returning plain text from JSON API handlers.
- Creating new duplicate config fields instead of extending `internal/model`.
- Swallowing write errors from `storage.WriteJSON` or `stats.Store.SaveRun`.
- Treating `nil` and empty slices as interchangeable for config URLs.

---

## Testing Requirements

Match tests to the blast radius:

- CLI parsing/config behavior: update `internal/cli/cli_test.go`.
- Download loop, retries, rate limiting, headers, failed URL removal: update
  `internal/consumer/consumer_test.go`.
- HTTP handlers and response shapes: update `internal/httpapi/server_test.go`.
- Runtime scheduling, temp tests, plan details, logs/events: update
  `internal/runtime/runtime_test.go`.
- Config/stat file behavior: update `internal/storage` or `internal/stats`
  tests.
- URL selection, invalid URLs, thread status: update
  `internal/urlmanager/urlmanager_test.go`.

Use raw JSON fixtures when the behavior exists to preserve legacy compatibility.
`TestLoadConfigAcceptsLegacySingleURLAndHeaderArray` is the pattern to follow.

---

## Code Review Checklist

Before considering a backend change done:

- The changed path is in the maintained Go backend unless the task explicitly
  concerns deprecated Python documentation/removal.
- `gofmt -w .` has been run for Go edits.
- `go test ./...` passes.
- Frontend build has been run when API/frontend/static contracts changed.
- New fields flow across CLI, storage, HTTP, runtime, and frontend where
  applicable.
- User-facing API messages are not duplicated with slightly different wording.
- Comments explain business behavior rather than restating code.
- The Python backend was not extended with new maintained behavior.

---

## Common Mistakes

- Updating only the Web path and forgetting CLI still uses the same model.
- Updating only CLI flags and forgetting saved config JSON needs the same field.
- Letting a compatibility comment drift after changing the actual behavior.
- Using the old Python files as implementation targets because they look
  familiar. They are deprecated; yes, that means hands off for new features.
