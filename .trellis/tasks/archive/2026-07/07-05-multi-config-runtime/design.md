# Multi-Config Runtime Design

## Summary

The Web dashboard will treat the configuration list as the primary runtime control surface. Backend dashboard snapshots will merge saved configurations, last-run stats, and current in-memory runtime items into one per-config list. Top KPI values will be aggregate live totals derived from the same backend runtime view.

This task stays as one Trellis task instead of parent/child tasks because the runtime, API, and frontend list are tightly coupled through one dashboard contract. Splitting them would create artificial handoff points without independently shippable behavior.

## Data Flow

```text
config.json + stats.json + runtime.Manager items
  -> runtime.Manager.PlanSummaries()
  -> httpapi DashboardSnapshot.plans / scheduler.plans / status
  -> Vue Query dashboard cache
  -> configuration list rows + top KPI
```

Command flow:

```text
row action -> apiClient command -> httpapi command route -> runtime.Manager / storage -> fresh dashboard snapshot
```

## Backend Runtime

### Runtime Item Ownership

`runtime.Manager` should own runtime items by normalized configuration name instead of treating manual runs as one `primary` item and auto-start runs as a separate list. The current `primary`/`auto` shape already supports multiple items accidentally, but using `auto` for manual items would make the model lie. That sort of cleverness is how future bugs quietly breed.

Planned shape:

- `items map[string]*Item` for all non-temporary Web runtime items.
- `temp []*Item` remains separate because temporary tests are not configuration rows.
- `Item.AutoStart` remains the per-item flag for auto-start origin.
- `snapshot()` iterates the map, sorts names for stable responses, and computes active/scheduled totals.

Behavior:

- `StartConsumer(cfg)` normalizes the config name and rejects duplicate active/scheduled names.
- `StartConsumer(cfg)` no longer stops unrelated active downloads or schedulers.
- Scheduled and immediate manual starts both add the item under its config name.
- `StopRuntimeConfig(name)` removes only that named item and stops its consumer/scheduler.
- `StopRuntimeConfig(name)` waits briefly for an active consumer to finish writing stats before returning the dashboard snapshot.
- `StopAll()` stops every item and temp test.
- `SaveConfig(name, cfg)` and `DeleteConfig(name)` keep the existing behavior of stopping the same named runtime item before writing persisted config data.
- Immediate non-scheduled runtime items remove themselves from the runtime map after their run finishes. Saved stopped rows remain visible through `config.json` plus `stats.json`, so current-run snapshots are cleared without losing history.

`StartTempTest` keeps its conservative runtime constraints for this task, but the frontend entry point lives in the task configuration dialog. It uses the current form payload and writes a normal execution record when the test completes.

## Dashboard Contract

Keep existing route names stable:

- `GET /api/dashboard`
- `GET /api/configurations/{name}`
- `PUT /api/configurations/{name}`
- `DELETE /api/configurations/{name}`
- `GET /api/plans/{name}`
- existing runtime command routes

Use the existing `DashboardSnapshot.plans` field as the unified configuration row list:

- It includes every saved config name from `config.json`.
- It includes active/scheduled runtime items even if the config file was deleted while the item existed.
- It includes stopped configs with zero live traffic/download metrics and their last completed run record.
- `scheduler.plans` mirrors the same list for frontend compatibility.

Extend `model.PlanSummary` with live speed fields:

- `Speed string json:"speed"` displays per-config current speed.
- `SpeedBytes float64 json:"speed_bytes"` supports aggregation/sorting without parsing display text.
- `LastRun *HistoryRecord json:"last_run,omitempty"` displays the last completed run result on the config row.

The existing fields keep their semantics:

- `total_bytes` and `download_count` are the current active run totals only; stopped rows reset them to zero.
- Completed or stopped runs are counted through `stats.json`; their in-memory snapshots are not added again after the run ends.
- `running` means the config is actively downloading.
- `scheduler_running` means the config has a waiting scheduler.
- `threads`, `url_count`, `cron_expr`, `interval`, and `url_strategy` describe the row's saved or runtime config.

`runtime.Manager.Status()` becomes aggregate:

- `speed` is the sum of live speed across all running items.
- `total_bytes` is the sum of every row's current `total_bytes_raw`.
- `download_count` is the sum of every row's current download count.
- `running` is true when any config is running or scheduled.
- `state` is `running` if any config is downloading, else `scheduled` if any config is waiting, else `stopped`.
- `thread_count` is the sum of active/scheduled row thread counts.
- `config` is an aggregate label used by the existing KPI card until the frontend label is renamed.
- `thread_status` and `url_usage_stats` remain focused on the first active item for now; the per-config list is the source of truth for multi-config comparison.

## Persistence Safety

`stats.json` currently has two concurrency hazards:

- `SaveRun` does read-modify-write without synchronization.
- Run IDs use second precision, so two completed runs in the same second can overwrite each other even after locking.

Fix:

- Add process-local per-path JSON mutation locking in `storage`.
- Use the locked mutation path for config write/delete/remove and stats write.
- Make `WriteJSON` write through a temporary file and rename it into place so readers do not observe partial files.
- Generate stats run IDs with nanosecond precision and add a collision suffix if needed.

This preserves the outer JSON shapes:

- `config.json` remains `{ "<name>": Config }`.
- `stats.json` remains `{ "<run-id>": StatsRun }`.

## Frontend

### Layout

Top bar:

- Keep status badge.
- Keep create config.
- Keep global stop actions.
- Remove the top-level start/edit/delete dependency on a selected combobox from the main workflow, because row actions own per-config operations.

KPI:

- Show aggregate speed.
- Show aggregate traffic.
- Show aggregate download count.
- Show active/scheduled configuration count instead of a single current config.

Configuration list:

- Replace or substantially reshape `PlanPanel.vue` into a configuration list panel.
- Use existing shadcn-vue components, primarily `Card`, `Table`, `Badge`, `Button`, `HoverCard`, and existing dialogs/sheets.
- Each row shows name, status, speed, traffic, download count, threads, URL count, and schedule.
- Each row shows the last run result from `last_run`.
- Each row has detail, edit, delete, start, and stop actions.
- Start is disabled when the row is running or scheduled.
- Stop is disabled when the row is stopped.
- The task configuration dialog has a temporary test action and displays the last run result for the current config name.
- Temporary test is disabled while any download is running.

Dialogs:

- Reuse `ConfigSheet` for create/edit.
- Reuse `PlanDetailDialog` for detail/history.
- Reuse `ConfirmActionDialog` for destructive delete/stop actions.

### Frontend State

`useDashboard` will expose name-based row actions:

- `startConfig(name)`
- `startTempTestFromEditor()`
- `openEditorForConfig(name)`
- `requestDeleteConfig(name)`
- `requestStopConfig(plan)`
- existing create/save/detail helpers where possible

The frontend should not recompute row metrics from raw history. It should render the `PlanSummary` rows returned by the backend.

## Compatibility

- CLI flags and behavior stay unchanged.
- Python backend stays untouched.
- Existing API route paths stay unchanged.
- Generated OpenAPI/SDK files must be regenerated after model changes.
- Persisted JSON readers remain tolerant of old files.

## Rollback

Rollback is a normal Git revert of this task's files. Persisted data remains compatible because no outer JSON shape migration is introduced. New `stats.json` run IDs are still plain object keys and older readers treat them as opaque strings.
