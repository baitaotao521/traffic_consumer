# Backend Development Guidelines

> Project-specific backend rules for Traffic Consumer.

---

## Overview

Traffic Consumer's maintained backend is the Go HTTP service and CLI in `main.go`
and `internal/`. The Vue dashboard is built from `frontend/` into `static/app/`
and served by the Go HTTP API.

The Python implementation (`main.py`, `web_ui.py`, `traffic_consumer.py`,
`app/`, `requirements.txt`, and `build_config.py`) is deprecated. Keep it only
as a legacy compatibility reference while preserving migrated file formats and
user-visible behavior. New backend work belongs in Go, not Python. Do not add
new features, release plumbing, or tests that treat the Python backend as the
primary implementation.

---

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | Go package layout, frontend/static boundaries, and deprecated Python paths | Filled |
| [Database Guidelines](./database-guidelines.md) | JSON-file persistence, config/stat schema compatibility, and storage tests | Filled |
| [Error Handling](./error-handling.md) | Go error propagation, HTTP response shape, CLI exits, and cancellation | Filled |
| [Quality Guidelines](./quality-guidelines.md) | Required checks, testing expectations, comments, and forbidden legacy work | Filled |
| [Logging Guidelines](./logging-guidelines.md) | CLI/Web log callbacks, levels, event buffers, and sensitive data rules | Filled |
| [Web API Contracts](./web-api-contracts.md) | Huma Web API routes, OpenAPI SDK generation, Vue Query interaction model | Filled |

---

## Pre-Development Checklist

- Confirm the touched backend path is Go (`main.go` or `internal/**`) unless the
  task is explicitly about deleting or documenting the deprecated Python path.
- Read [Directory Structure](./directory-structure.md) before adding packages,
  handlers, runtime state, storage helpers, or CLI flags.
- Read [Database Guidelines](./database-guidelines.md) before changing
  `config.json`, `stats.json`, config normalization, or compatibility fields.
- Read [Error Handling](./error-handling.md) before changing API handlers, CLI
  commands, scheduler/runtime behavior, retries, or storage failures.
- Read [Web API Contracts](./web-api-contracts.md) before changing Web dashboard
  routes, OpenAPI schema, generated SDK usage, or frontend API interaction.
- Read [Logging Guidelines](./logging-guidelines.md) before changing download
  logs, Web runtime events, status messages, or notification payloads.
- Read [Quality Guidelines](./quality-guidelines.md) before committing any
  backend change.

## Quality Check

- Run `gofmt -w .` after Go edits and make sure `git diff` contains only
  intentional formatting changes.
- Run `go test ./...` for backend changes.
- Run `npm --prefix frontend run api:generate` when Web API signatures or
  payloads change.
- Run `npm --prefix frontend run build` when Go API changes affect the dashboard
  contract or when frontend/static assets are touched.
- Keep Python tests only as legacy compatibility checks. They must not become
  the acceptance gate for new backend behavior.

---

## Maintained Runtime Boundary

Primary runtime:

```text
main.go -> internal/cli or internal/httpapi -> internal/runtime/consumer/storage/stats/urlmanager
```

Deprecated runtime:

```text
main.py, web_ui.py, traffic_consumer.py, app/**, build_config.py
```

Legacy compatibility is still valuable when Go code intentionally preserves old
JSON fields, status strings, or CLI output. It is not a reason to extend the
Python backend. Seriously, adding new backend behavior to both stacks is how a
rewrite turns into a maintenance trap.
