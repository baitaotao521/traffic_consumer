# 性能优化与热点排查 Implementation Plan

## Checklist

- [x] Load backend/frontend coding guidelines before editing.
- [x] Refactor runtime dashboard aggregation so `/api/dashboard` computes plan summaries once.
- [x] Add stats store cache for loaded runs, sorted keys, and per-config summary.
- [x] Update plan detail to reuse summary work and avoid duplicate stats aggregation.
- [x] Inspect additional hotspots after the primary fixes and apply safe scoped improvements.
- [x] Add or update focused tests for cache invalidation, dashboard aggregation behavior, and any additional performance fix.
- [x] Run `go test ./...`.
- [x] Run `npm run build` in `frontend`.
- [x] Check git diff for unrelated churn.

## Validation Commands

```bash
go test ./...
cd frontend && npm run build
git status --short
```

## Risky Files

- `internal/runtime/runtime.go`: status/plan aggregation semantics are user-facing.
- `internal/stats/stats.go`: cache correctness affects dashboard history and summaries.
- `internal/httpapi/server.go`: dashboard response shape must remain stable.
- `internal/consumer/consumer.go`: hot-loop changes must not break stop, limit, or count semantics.
- `internal/urlmanager/urlmanager.go`: URL selection changes must preserve fairness behavior.

## Review Gates

- Public API structs and JSON field names remain unchanged.
- Runtime tests cover running, scheduled, and stopped status after refactor.
- Stats tests prove cache invalidates after `SaveRun()` and after external file writes.
- Any deferred performance issue is named with the reason it is not safe to fix in this task.
