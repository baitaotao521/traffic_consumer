# Multi-Config Runtime Implementation Plan

## Preconditions

- Task must be started with `python3 ./.trellis/scripts/task.py start .trellis/tasks/07-05-multi-config-runtime` after planning approval.
- Before editing implementation files, load `trellis-before-dev` for the touched backend/frontend specs.
- Inline Codex mode is active, so no `implement.jsonl` / `check.jsonl` curation is required.

## Implementation Checklist

1. Persistence safety
   - Add per-path locked JSON mutation support in `internal/storage`.
   - Make JSON writes atomic through temp-file write plus rename.
   - Update `SaveConfigEntry`, `DeleteConfigEntry`, and `RemoveURLFromConfig` to use the locked mutation path.
   - Update `stats.Store.SaveRun` to use locked mutation and generate nanosecond/collision-safe run IDs.
   - Add tests for concurrent stats saves preserving all runs.

2. Runtime multi-config model
   - Refactor `runtime.Manager` from `primary`/`auto` storage into a name-keyed non-temp item map plus existing temp slice.
   - Preserve `Item.AutoStart` as item metadata.
   - Update `LaunchAutoStartConfigs`, `StartConsumer`, `StopAll`, `StopRuntimeConfig`, `items`, and `snapshot`.
   - Reject duplicate start for an already running or scheduled config name.
   - Keep save/delete stopping only the same named runtime before writing config data.
   - Remove completed immediate runtime items from memory after stats are written so saved rows clear current-run snapshots.
   - Keep scheduled runtime items registered after a scheduled run completes.
   - Add runtime tests for multi-start, duplicate rejection, concurrent duplicate rejection, named stop isolation, immediate cleanup, scheduled retention, stop record persistence, temp-test result persistence, and save/delete isolation.

3. Backend dashboard aggregation
   - Extend `model.PlanSummary` with `speed`, `speed_bytes`, and `last_run`.
   - Update `PlanSummaries` to merge all saved config names, last-run stats, and runtime items into one sorted list.
   - Ensure completed runs are shown through `last_run` only, while stopped row traffic/download metrics reset to zero.
   - Update `Status` to aggregate speed, traffic, download count, thread count, and global running/scheduled state from the unified row set.
   - Keep `scheduler.plans` synchronized with `dashboard.plans`.
   - Add HTTP tests that stopped saved configs appear in `plans` and active configs expose row metrics.

4. API generation
   - Run `npm --prefix frontend run api:generate` after model/API changes.
   - Review generated SDK/type diffs for expected `PlanSummary` additions only.

5. Frontend component planning checks
   - Run `npx shadcn-vue@latest docs table button badge card dialog alert-dialog sheet tooltip hover-card` before editing component usage.
   - Use existing installed shadcn-vue components; do not add a new component unless the existing set is insufficient.

6. Frontend dashboard update
   - Replace the current runtime-only `PlanPanel` behavior with a configuration list panel.
   - Add row actions for detail, edit, delete, start, and stop.
   - Add a last-run result column that renders `PlanSummary.last_run`.
   - Add temporary test action and last-run result display to the task configuration dialog.
   - Add name-based actions in `useDashboard`.
   - Adjust `TopBar` so per-config start/edit/delete no longer depends on one selected combobox as the main workflow.
   - Update `KpiGrid` to show aggregate active/scheduled configuration count instead of one current config.
   - Keep user-facing UI text concise and non-duplicated.

7. Validation
   - Run `gofmt -w .`.
   - Run `go test ./...`.
   - Run `npm --prefix frontend run build`.
   - If frontend/static build output changes, review `static/app` diff before reporting completion.

## Risk Areas

- `internal/runtime/runtime.go`: broadest behavioral change; preserve cancellation and scheduler stop semantics.
- `internal/storage/storage.go` and `internal/stats/stats.go`: concurrent mutation changes must preserve legacy corrupt/missing-file tolerance.
- `internal/model/model.go`: new API fields require comments and generated frontend types.
- `frontend/src/composables/useDashboard.ts`: row actions should reuse existing mutation/cache patterns instead of hand-writing fetch calls.
- `frontend/src/components/dashboard/PlanPanel.vue`: likely large UI rewrite; keep layout dense and operational.
- `frontend/src/App.vue` and `frontend/src/components/dashboard/TopBar.vue`: top-level action changes must not remove global stop access.

## Rollback Points

- After persistence changes pass Go tests.
- After runtime/API aggregation passes Go tests.
- After API generation compiles.
- After frontend build passes.

If any stage regresses broadly, revert the current stage before continuing rather than stacking more fixes on an unstable contract.
