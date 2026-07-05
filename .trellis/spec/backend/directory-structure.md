# Directory Structure

> Backend organization and ownership boundaries for Traffic Consumer.

---

## Overview

The maintained backend is the Go module `github.com/baitaotao521/traffic_consumer`.
`main.go` is the only maintained runtime entry point. It parses CLI flags first,
runs CLI-only commands through `internal/cli`, and otherwise starts the Go HTTP
server in `internal/httpapi`.

The product Python backend has been removed. Trellis and Codex Python scripts
may still exist under their own workflow directories; they are development
automation, not Traffic Consumer runtime code.

---

## Directory Layout

```text
main.go                         Go entry point: Web UI by default, CLI with --no-gui or CLI-only flags
internal/
  cli/                          pflag parsing, config commands, stats display, scheduled CLI runs
  config/                       default URLs, chunk size, and config/stat file paths
  consumer/                     download loop, retries, rate limiting, failed-URL removal
  httpapi/                      chi router, page shell, JSON API endpoints
  model/                        shared config, status, plan, history, log, and event payload structs
  runtime/                      Web-mode manager, running items, schedulers, logs, event buffers
  stats/                        stats.json read/write, formatting, and per-config aggregation
  storage/                      config.json read/write, config CRUD, URL removal helpers
  urlmanager/                   URL selection, thread assignment, invalid URL state, usage stats
frontend/                       Vue/Vite dashboard source and dist runtime assets
frontend/dist/                  compiled dashboard bundle served by Go
scripts/package.sh              Shared local and GitHub Actions release packaging entry point
```

---

## Release Packaging Contract

### 1. Scope / Trigger

Use `scripts/package.sh` whenever a local or GitHub Actions flow needs release
archives for the Go HTTP service. The Web server reads all frontend runtime
assets from `frontend/dist` under the process run directory, so release archives
must package that directory beside the executable instead of publishing a bare
binary.

### 2. Signatures

```bash
scripts/package.sh
scripts/package.sh --all
scripts/package.sh --dist <dir>
scripts/package.sh --skip-frontend
TARGET_OS=linux TARGET_ARCH=amd64 PLATFORM_LABEL=Linux scripts/package.sh
```

### 3. Contracts

- Single-target builds read `TARGET_OS` / `TARGET_ARCH`, then `GOOS` /
  `GOARCH`, then the host OS/arch.
- `PLATFORM_LABEL` controls the archive suffix, such as
  `traffic_consumer_Linux.tar.gz`.
- `DIST_DIR` or `--dist` controls where archives are written.
- `BINARY_BASE` controls the binary name inside the archive; Windows builds add
  `.exe` when needed.
- `CGO_ENABLED` defaults to `0`; `LDFLAGS` defaults to `-s -w`.
- `SKIP_FRONTEND=1` or `--skip-frontend` reuses the current `frontend/dist`
  output.
- `SKIP_FRONTEND_INSTALL=1` or `--skip-frontend-install` skips `npm ci`.

### 4. Validation & Error Matrix

- Missing `go` -> fail before packaging.
- Missing `npm` while building frontend -> fail before Go build.
- Missing `frontend/dist/index.html` with `--skip-frontend` -> fail and tell the
  caller to build frontend first.
- Missing `zip` for Windows archives -> fail before creating the `.zip`.
- Native target build -> run `traffic_consumer --help` as a smoke test.

### 5. Good/Base/Bad Cases

- Good: GitHub Actions matrix calls `scripts/package.sh` with
  `TARGET_OS`, `TARGET_ARCH`, and `PLATFORM_LABEL`.
- Base: Local release runs `scripts/package.sh` and creates one host-platform
  archive under `dist/`.
- Bad: A workflow hand-writes `go build` and uploads only the executable; the
  Web UI then lacks `frontend/dist` at runtime.

### 6. Tests Required

- `bash -n scripts/package.sh` for shell syntax.
- `scripts/package.sh --skip-frontend --all --dist <tmp>` to verify all archive
  formats and cross-platform `go build` targets.
- `scripts/package.sh --dist <tmp>` to verify the local frontend build path.
- `go test ./...` after release plumbing changes that can affect backend build
  behavior.

### 7. Wrong vs Correct

Wrong:

```yaml
run: go build -trimpath -ldflags="-s -w" -o dist/traffic_consumer .
```

Correct:

```yaml
run: scripts/package.sh
```

---

## Module Organization

Keep transport, runtime orchestration, persistence, and business execution in
separate Go packages.

- `main.go` wires the process together and should stay thin. It calls
  `cli.Parse`, chooses CLI/Web mode, constructs `runtime.New`, then serves
  `httpapi.New(...).Router()`.
- `internal/httpapi` owns HTTP route registration, request decoding, status
  codes, and response JSON. It delegates work to `runtime.Manager` and
  `storage`.
- `internal/runtime` owns Web-mode state such as active downloads, schedulers,
  temporary tests, auto-start instances, logs, and events. It should not parse
  HTTP requests.
- `internal/consumer` owns download execution, retries, rate limiting, stats
  writeback, and failed URL removal. It receives callbacks for logs/events
  rather than importing HTTP code.
- `internal/storage` and `internal/stats` are the only packages that should
  write `config.json` and `stats.json` directly.
- `internal/model` is the shared API/storage contract. Add fields here when the
  same payload must cross storage, HTTP, runtime, and frontend boundaries.

Example from `main.go`:

```go
args, _, err := cli.Parse(os.Args[1:])
if args.IsCLIMode() {
    if err := cli.Run(args, paths); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
    return
}
manager := appruntime.New(paths)
server := httpapi.New(manager, baseDir)
```

Example from `internal/httpapi/server.go`:

```go
r.Route("/api", func(r chi.Router) {
    r.Get("/configs", s.getConfigs)
    r.Post("/runtime/start", s.startConsumer)
    r.Get("/runtime/status", s.status)
})
```

---

## Removed Python Product Boundary

Do not recreate these removed product runtime paths:

- `main.py`
- `web_ui.py`
- `traffic_consumer.py`
- `app/**`
- `build_config.py`
- `requirements.txt`

Historical behavior that Go still preserves must be documented next to the Go
implementation, such as old `request_headers` formats or the `stats.json` time
strings preserved in `internal/model/model.go`.

When a compatibility behavior is preserved in Go, document it next to the Go
implementation. Existing examples:

```go
// MarshalJSON keeps the legacy Web UI object format for saved configs.
func (h HeaderMap) MarshalJSON() ([]byte, error)

// ReadJSON treats missing or corrupt JSON as empty, matching legacy tolerance.
func ReadJSON(path string, target any) error
```

---

## Naming Conventions

- Go files and packages use lowercase package names and conventional Go file
  names: `server.go`, `runtime.go`, `storage.go`, `consumer.go`.
- HTTP handler methods on `Server` use lower camel case (`saveConfig`,
  `startConsumer`, `previewCron`) and stay close to route registration.
- Runtime manager methods use business verbs (`StartConsumer`,
  `StartTempTest`, `StopRuntimeConfig`, `LaunchAutoStartConfigs`).
- JSON struct fields in `internal/model` keep snake_case tags because persisted
  config/stat files and frontend payloads already use that contract.
- Tests live next to the package under test as `*_test.go` and use `t.TempDir`
  for file-backed storage.

---

## Examples To Follow

- `internal/model/model.go` centralizes cross-layer payload structs and documents
  fields with business meaning.
- `internal/storage/storage_test.go` verifies config CRUD, legacy single-URL
  normalization, header array compatibility, and corrupt JSON tolerance.
- `internal/consumer/consumer_test.go` verifies download behavior with
  `httptest.NewServer` instead of real network dependencies.
- `internal/httpapi/server_test.go` uses `httptest` and a temp config directory
  to exercise HTTP handlers without starting a real server.
