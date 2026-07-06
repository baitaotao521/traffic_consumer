# 性能优化与热点排查 Design

## Architecture And Boundaries

The task keeps the existing HTTP API and persisted JSON file formats stable. Optimizations stay inside the existing runtime, stats, storage, and URL manager boundaries.

The main dashboard hot path should be:

1. `httpapi.Server.dashboardSnapshot()` loads config names.
2. Runtime manager computes plan summaries once.
3. Runtime manager derives the top-level status from the same plan summary slice.
4. Scheduler status is computed independently because it depends on live cron entries rather than stats history.

## Data Flow And Contracts

`DashboardSnapshot` remains unchanged:

- `configs` still contains config names.
- `plans` still contains all saved and active non-temporary plan summaries.
- `status` still aggregates live totals, speed, threads, and URL usage across active plans.
- `scheduler` still describes the earliest scheduled run.

`stats.Store` remains compatible with the existing JSON map persisted in `stats.json`. A small in-process cache can sit behind `SummaryByConfig()` and `HistoryForConfig()` as long as:

- The cache invalidates when `SaveRun()` writes through the same store.
- The cache invalidates when file metadata changes, covering external writes.
- Corrupt or missing JSON keeps existing tolerant behavior.

## Implementation Choices

- Add an internal stats cache keyed by file `modTime` and file size, with explicit invalidation after successful `SaveRun()`. This avoids repeatedly unmarshalling and sorting unchanged stats during polling.
- Build sorted run keys once per cache refresh. Reuse that order for summary and per-config history.
- Add a status helper that accepts already-computed plans, avoiding `Status()` -> `PlanSummaries()` duplication. Keep `Status()` as a public compatibility method that calls the helper with freshly computed plans.
- Avoid broad frontend changes unless an obvious low-risk polling or rendering issue appears during the second pass.
- For lock pressure in the download loop, prefer targeted changes only when tests can cover behavior. Atomic counters are safe; throttled thread progress updates need more care because the UI expects responsive progress.

## Trade-Offs

- JSON map persistence is retained even though append-only storage would scale better. Replacing it would require migration, compatibility tests, and likely user-facing operational notes.
- Cache invalidation by file metadata is not a cross-process transaction mechanism, but it matches the existing tolerance model and avoids stale reads after normal writes.
- Dashboard polling remains in place. Removing polling would require a larger transport redesign and is out of scope for this task.

## Rollback Shape

All changes should be local to existing files. If a regression appears, revert the status helper/cache changes independently:

- Revert runtime status helper changes to restore old `Status()` behavior.
- Revert stats cache internals while preserving exported method signatures.
- Revert URL manager or consumer micro-optimizations independently.
