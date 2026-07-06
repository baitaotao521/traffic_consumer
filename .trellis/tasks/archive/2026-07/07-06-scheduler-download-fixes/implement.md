# 修复调度与下载流程问题 - Implementation Plan

## Checklist

- [x] Update `URLMgr.ResetRuntimeState` to clear transient invalid URL state and reset weights.
- [x] Add `URLMgr.IsInvalid` and short-circuit retry loops once another thread has invalidated the same URL.
- [x] Make `Consumer.RunOnce` publish a cancellable per-run context and clear it safely on finish.
- [x] Add context-aware rate limiter acquisition and use it from streaming downloads.
- [x] Track and persist business finish reasons instead of hard-coding `成功`.
- [x] Aggregate URL usage across all active runtime items.
- [x] Create and maintain root `记录文档.md`.
- [x] Add focused tests for each fixed behavior.
- [x] Run `gofmt -w .` and `go test ./...`.
- [x] Re-audit scheduler/download flow for remaining issues; if new issues remain, append them to this plan or create the next Trellis task.

## Validation Results

- `go test ./internal/consumer` passed.
- `go test ./internal/urlmanager` passed.
- `go test ./internal/runtime` passed.
- `go test ./...` passed.
- `go test -race ./internal/consumer ./internal/runtime ./internal/urlmanager` passed.
- Finish-reason regression coverage includes user stop, duration limit, traffic limit, count limit, no available URL, and all-URL-invalid results.
- Parent-context cancellation coverage includes pre-canceled `RunOnce` and mid-run cancellation.
- Temp-test isolation coverage rejects normal runtime starts while a temp test is active.
- Scheduler-stop isolation coverage keeps immediate downloads running while stopping scheduled plans.
- Scheduler-stop coverage cancels an active scheduled `RunOnce` and records the run as `已停止`.
- Scheduler-stop coverage cancels paused plans stored behind temporary tests without stopping the temp test itself.
- HTTP command coverage verifies `stop-scheduler` cancels paused plans stored behind temporary tests.
- Temp-test cleanup coverage confirms natural completion restores the paused scheduler, while explicit stop does not restore it.
- Temp-test exclusivity coverage rejects normal starts and new temp tests while a temp test is active or still waiting for cleanup.
- Stop-all coverage treats pending temp-test cleanup as active runtime state and reports success after clearing it.
- `TestStopSchedulersCancelsActiveScheduledRun` passed and covers an active scheduled round canceled by scheduler stop.
- `go test -count=1 ./internal/runtime` passed after the active scheduled-run coverage was added.
- `go test -race -count=1 ./internal/runtime` passed after the active scheduled-run coverage was added.
- `go test -race ./internal/consumer ./internal/runtime ./internal/urlmanager ./internal/httpapi` passed.
- `go test -count=1 ./...` passed.
- `go test -race -count=1 ./internal/consumer ./internal/runtime ./internal/urlmanager ./internal/httpapi` passed.
- `go test -race -count=1 ./...` passed.
- `go vet ./...` passed.
- `git diff --check` passed.
- `go run ./cmd/openapi > /tmp/traffic-consumer-openapi.json` passed, and generated OpenAPI output matched `frontend/src/api/openapi.json`.

## Validation Commands

```bash
gofmt -w .
go test ./...
```

## Risky Files

- `internal/consumer/consumer.go`: shared lifecycle and history semantics.
- `internal/consumer/limiter.go`: shared rate limiter API.
- `internal/urlmanager/urlmanager.go`: URL selection and invalidation rules.
- `internal/runtime/runtime.go`: status aggregation and scheduler interaction.

## Review Gates

- Every behavior change must have a targeted test.
- `记录文档.md` must be updated after each completed fix, not only at the end.
- The final pass must compare remaining code against the PRD acceptance criteria.
