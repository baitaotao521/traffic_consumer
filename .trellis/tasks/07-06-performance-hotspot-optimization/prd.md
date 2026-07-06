# 性能优化与热点排查

## Goal

Reduce avoidable CPU, disk I/O, and lock pressure in the dashboard/runtime hot path while preserving the current Web console behavior and API contracts.

The user asked to fix the identified performance issues and continue checking for other performance problems.

## Initial Confirmed Facts

- The Web dashboard polls a full snapshot every second during active work and every five seconds while idle: `frontend/src/api/queries.ts:21`.
- `dashboardSnapshot()` computes `plans := s.Manager.PlanSummaries()` and then calls `s.Manager.Status()`: `internal/httpapi/server.go:500`.
- `Status()` calls `PlanSummaries()` again, causing duplicate config/stat reads and duplicate aggregation in one dashboard request: `internal/runtime/runtime.go:489`.
- `PlanSummaries()` reads full historical stats through `Stats.SummaryByConfig()` and reads all saved configs on each call: `internal/runtime/runtime.go:597`.
- `SummaryByConfig()` and `HistoryForConfig()` both load the entire `stats.json`, sort every run key, and then aggregate/filter: `internal/stats/stats.go:76`, `internal/stats/stats.go:104`.
- `SaveRun()` currently reads and rewrites the full stats map; `WriteJSON()` uses indented JSON for all persisted files: `internal/stats/stats.go:38`, `internal/storage/storage.go:75`.
- The download loop updates thread progress and total bytes on every chunk, which can create lock pressure with high thread counts: `internal/consumer/consumer.go:408`, `internal/urlmanager/urlmanager.go:153`.
- Weighted random URL selection performs nested URL lookup when computing candidate weights: `internal/urlmanager/urlmanager.go:346`.

## Requirements

- R1: Remove duplicate dashboard aggregation inside one `/api/dashboard` response.
- R2: Reduce repeated full-file stats processing for dashboard and plan detail paths without changing response fields.
- R3: Keep stats persistence backward compatible with the existing `stats.json` map format.
- R4: Preserve current runtime semantics for running, scheduled, temporary, and stopped plans.
- R5: Continue scanning for additional performance issues and apply only safe, scoped fixes in this task.
- R6: Keep UI behavior and generated API contracts compatible unless a change is required for performance and covered by tests.

## Acceptance Criteria

- [x] A dashboard snapshot computes plan summaries once and reuses them for status aggregation.
- [x] Stats summary reads are cached or otherwise avoid reloading and sorting `stats.json` on every dashboard poll when the file has not changed.
- [x] Plan detail avoids redundant stats summary work and still returns the selected plan summary plus recent history.
- [x] At least one additional safe performance issue beyond duplicate dashboard aggregation is investigated and either fixed or documented as intentionally deferred.
- [x] Existing Go tests pass with `go test ./...`.
- [x] Frontend production build passes with `npm run build`.
- [x] No unrelated files or generated build artifacts are left changed unless they are required by the implementation.

## Out Of Scope

- Replacing `stats.json` with a different durable storage backend.
- Replacing polling with SSE/WebSocket streaming.
- Visual redesign of dashboard components.
- Benchmark harnesses beyond targeted tests unless implementation risk requires them.

## Open Questions

None blocking. Default product decision: favor low-risk hot-path fixes over storage/API redesign.
