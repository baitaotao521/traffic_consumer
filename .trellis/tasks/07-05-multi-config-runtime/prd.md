# Support multi-config runtime

## Goal

Allow users to manage and run multiple saved configurations from one Web dashboard list, while the top KPI area shows aggregate totals across configurations. Existing single-configuration CLI behavior and persisted `config.json` / `stats.json` compatibility must remain intact.

## User Value

Users can keep several traffic plans configured, start and stop them independently from the Web dashboard, and compare each configuration's live speed, total downloads, and total traffic without losing the aggregate system view.

## Confirmed Facts

- The maintained backend is the Go service under `main.go` and `internal/**`; Python files are deprecated compatibility references.
- Persisted named configurations already exist in `~/.traffic_consumer/config.json` through `internal/storage`, and both CLI and Web paths use the same file.
- Web runtime state already models multiple items for auto-start and scheduled plans through `runtime.Manager.auto`, `runtime.Manager.temp`, `PlanSummaries`, and `StopRuntimeConfig` (`internal/runtime/runtime.go:27`, `internal/runtime/runtime.go:161`, `internal/runtime/runtime.go:349`, `internal/runtime/runtime.go:450`).
- Manual Web start currently blocks any additional download when `snapshot.hasActiveDownload` is true in `runtime.Manager.StartConsumer` (`internal/runtime/runtime.go:220`).
- Manual Web start also stops any existing scheduler before switching configs (`internal/runtime/runtime.go:226`), which conflicts with the requested multi-config runtime.
- The Vue dashboard disables the primary Start button unless global status is `stopped`, so the UI currently prevents starting a second selected configuration even before the backend conflict path is hit (`frontend/src/App.vue:21`, `frontend/src/components/dashboard/TopBar.vue:117`).
- The existing runtime plan panel only lists active runtime items and offers detail/stop actions, not full saved-config CRUD/start/stop rows (`frontend/src/components/dashboard/PlanPanel.vue:12`, `frontend/src/components/dashboard/PlanPanel.vue:53`).
- `PlanSummary` already carries per-config total bytes, download count, thread count, URL count, schedule state, and strategy, but it does not carry per-config speed (`internal/model/model.go:190`).
- `runtime.Manager.Status` currently reports one primary active item rather than aggregated state across all active configs (`internal/runtime/runtime.go:391`).
- `runtime.Manager.PlanSummaries` currently aggregates historical bytes/downloads with the active item's current bytes/downloads, but only for items already in runtime memory (`internal/runtime/runtime.go:450`).
- `stats.Store.SaveRun` currently performs read-modify-write on `stats.json` without a lock; concurrent configurations finishing close together can overwrite each other's history unless this is made safe (`internal/stats/stats.go:37`).
- Auto-removing failed URLs writes back to `config.json` through the same unlocked JSON read-modify-write pattern (`internal/storage/storage.go:63`, `internal/storage/storage.go:114`).
- User clarified that Web must provide a configuration list where each row supports create/read/update/delete, start, and stop; each configuration's speed, download count, and traffic must be visible; the top KPI must show totals across configurations.
- User added that stopping a configuration must clear the current-run snapshot from the row while preserving the completed execution record.
- User added that temporary tests should run from the task configuration dialog and show the resulting last-run record.

## Decisions

- The configuration list and top KPI use current in-memory bytes/downloads for traffic and download count.
- Speed is a live-only value calculated from currently running configurations.
- CLI multi-config execution remains out of scope for this task.
- The configuration list is the row-level source of truth. The top status/KPI area is an aggregate summary.
- The result column shows the last completed run record; live traffic/download fields clear when the config is stopped.

## Requirements

- The Web dashboard provides one configuration list that includes saved configurations whether stopped, running, or waiting for schedule.
- Each configuration row supports view/detail, edit, delete, start, and stop actions.
- The task configuration dialog supports running a temporary test for the current form values.
- Users can start a saved Web configuration while other configurations are already running or waiting for schedule, provided the selected configuration name is not already active.
- Starting a configuration that is already active must not create a duplicate runtime item for the same name.
- Editing or deleting a configuration that is running or scheduled must stop that same configuration before mutating persisted config data.
- Row-level stop stops only the selected configuration; global stop continues to stop every active download and scheduler.
- Row-level stop waits for the consumer to write its execution record, then clears current-run speed and in-memory byte/download snapshots from that row.
- Each configuration row shows status, current speed, download count, traffic, thread count, URL count, last run result, and schedule summary.
- The top KPI area shows aggregate speed, download count, traffic, and active/scheduled configuration count across all rows.
- Stopped rows show zero traffic and download count while retaining the last completed execution record.
- Temporary test runs are started from the task configuration dialog and update the last run result after completion.
- Finished immediate runtime items are removed from runtime memory after stats are written; scheduled runtime items remain registered for their next trigger.
- Dashboard snapshots keep the configuration list as the source of truth for row-level state; top-level status is an aggregate summary only.
- Concurrent writes to `stats.json` and `config.json` must preserve records and config updates produced by separate running configurations.
- Existing CLI behavior remains single-process and compatible with `--config`, `--load-config`, `--save-config`, `--show-stats`, and existing config file format.
- Existing Web API route names remain stable unless a contract change is unavoidable.

## Acceptance Criteria

- [ ] The Web dashboard shows a configuration list containing stopped saved configs and active configs in one place.
- [ ] Each config row exposes actions to view/detail, edit, delete, start, and stop, with invalid actions disabled while the row is already in the opposite state.
- [ ] Starting configuration `one` and then starting configuration `two` leaves both visible as running rows.
- [ ] Starting an already-running configuration returns a clear conflict and does not duplicate the row.
- [ ] Stopping `one` does not stop `two`; global stop still stops both.
- [ ] Stopping `one` records the completed execution and clears `one`'s current-run speed/byte/download snapshot.
- [ ] Saving or deleting `one` stops only `one` before writing `config.json`.
- [ ] Each config row displays speed, download count, traffic, and last run result for that configuration.
- [ ] Temporary test can be launched from the task configuration dialog and its result appears as the last run result after completion.
- [ ] Finished immediate runtime items are cleared after completion, while scheduled runtime items remain waiting for their next trigger.
- [ ] The top KPI displays aggregate speed, aggregate download count, aggregate traffic, and aggregate active/scheduled config count.
- [ ] Stopped rows clear traffic/download totals, while running rows show the current in-memory totals.
- [ ] Concurrent run completion writes separate `stats.json` entries for each configuration.
- [ ] Tests cover runtime multi-start, duplicate-name rejection, named stop isolation, and concurrent stats persistence.
- [ ] Frontend build passes after API generation when response contracts change.
- [ ] `go test ./...` passes.

## Out of Scope

- Running multiple configurations from a single CLI process.
- Changing the persisted outer shape of `config.json` or `stats.json`.
- Extending the deprecated Python backend.
- Adding user/account-level quotas or global bandwidth scheduling.
