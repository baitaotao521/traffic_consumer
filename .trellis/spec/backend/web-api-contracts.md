# Web API Contracts

## Scenario: Dashboard API Contract Rewrite

### 1. Scope / Trigger

- Trigger: Web dashboard API signatures or frontend API interaction changes.
- Scope: Go `internal/httpapi` routes, Huma OpenAPI schema, generated frontend SDK under `frontend/src/api/generated`, Vue Query server-state hooks, and `static/app` build output.
- Python Web UI files remain deprecated references only; do not add new API behavior there.

### 2. Signatures

The maintained Web API is registered under `/api` with Huma:

```text
GET    /api/dashboard
GET    /api/activity?after_log=<seq>&after_event=<seq>&include_logs=<bool>&log_sources=<csv>&log_levels=<csv>
GET    /api/configurations/{name}
PUT    /api/configurations/{name}?force=<bool>&mode=create|edit
DELETE /api/configurations/{name}
GET    /api/plans/{name}
POST   /api/cron/preview
POST   /api/commands/start-runtime
POST   /api/commands/start-temp-test
POST   /api/commands/stop-runtime
POST   /api/commands/stop-scheduler
POST   /api/commands/stop-plan/{name}
```

OpenAPI schema generation is the contract source for the frontend:

```bash
npm --prefix frontend run api:generate
```

This runs `go run ../cmd/openapi > src/api/openapi.json` and regenerates `frontend/src/api/generated`.

### 3. Contracts

- `GET /api/dashboard` returns `{ dashboard }`; the dashboard contains config summaries, runtime status, scheduler status, and plan summaries.
- `GET /api/activity` returns `{ logs, events, log_cursor, event_cursor }`; clients send the last consumed log/event cursors as `after_log` and `after_event`.
- `GET /api/activity` log reads support three log-only controls:
  `include_logs`, `log_sources`, and `log_levels`.
  `include_logs=false` means “do not advance the log cursor yet”.
- `LogEntry` includes `source` in addition to `level`, `message`, and `seq`.
  `source` is the backend-owned dimension for frontend log filtering.
- Command endpoints return `{ dashboard, message }`; config save also returns `item`.
- Config save/start bodies use `model.Config` / generated `Config` through the
  frontend `ConfigPayload` alias. Optional numeric/string fields are omitted
  when blank. `urls: []` is preserved as an explicit empty URL list. Repeated
  strings inside `urls` are valid user input: guards trim and validate each
  non-empty entry, but must not reject or collapse duplicates before storage or
  runtime startup.
- Huma schema-link metadata stays disabled for the Web API: responses and
  `frontend/src/api/openapi.json` must not include `$schema`, and
  `/api/schemas/*` must return `404`. `/api/openapi.json` remains the only
  generated frontend contract source.
- `PUT /api/configurations/{name}` treats the path name as the source/original
  config and the body `name` / `config_name` as the target. `mode=create`
  rejects an existing target directly. `mode=edit` allows saving the same name,
  but renaming to an existing different target also returns `409 Conflict`.
  The legacy `force` query parameter may be accepted for compatibility, but it
  must not override duplicate-name protection.
- Save/rename must stop only affected runtime items before mutating config
  data: source name and target name when they are active or scheduled. Rejected
  duplicate-name creates/renames must return before stopping runtime items. The
  final response item name is always the saved target name.
- Web config validation belongs in a shared backend guard before runtime or
  storage mutation. The Vue editor may pre-validate for feedback, but direct API
  callers must still get the same 400/409 behavior from the Go API.
- `DashboardSnapshot.plans` is the configuration list contract.
  `PlanSummary.total_bytes`, `total_bytes_raw`, and `download_count` describe
  the current task-session totals. That session starts when a runtime item is
  started, includes completed scheduled runs while the scheduler waits for the
  next trigger, includes the current active run snapshot, and clears to `0 B` /
  `0` once the task or scheduler is stopped and the runtime item is removed.
  Persisted `stats.json` history must not be folded back into these row totals.
- `GET /api/plans/{name}.summary` reuses `PlanSummary` and follows the same
  task-session metric window as the configuration list. The detail history
  array remains the persisted `stats.json` history, so ended tasks show cleared
  summary totals while still listing historical runs.
- `StatusPayload.total_bytes` and `download_count` aggregate the same
  task-session values shown in `DashboardSnapshot.plans`. They are live
  runtime KPIs, not historical lifetime totals. Historical records stay in
  `stats.json` and are surfaced through `last_run` / plan detail history.
- `StatusPayload.url_usage_stats` is the dashboard URL distribution source.
  The URL distribution chart must display both `count` and `percentage` for
  each URL; chart hover text and any visible URL breakdown must not reduce this
  payload to only one of the two fields.
- Dashboard numeric form fields must use the component-library number input pattern (`NumberField` / Reka `NumberFieldRoot` + `NumberFieldInput`), not a generic `Input type="number"`. The library's number field owns the numeric model state; form adapters should receive `number | undefined` and must not trim or parse these fields as strings.
- Dashboard config editor payloads must be projected through the shared
  frontend editor guard before save, start, or temporary test. Do not rebuild a
  separate `Config` serializer inside a component or composable.
- Dashboard URL table rows are an editor affordance only. Rows must write back
  to `ConfigForm.urls`, and save/start/temp-test must continue to let the shared
  frontend projection produce `ConfigPayload.urls` so URL parsing, duplicate
  preservation, and empty-list handling stay in one path.
- Dashboard command confirmations that run async mutations must keep close
  timing in the composable. Do not wire mutation confirms through
  `AlertDialogAction` / `DialogClose`, because those primitives close before the
  save/stop/delete request resolves and can leave pending/retry state out of
  sync. Keep the confirm action available until success or explicit cancel.
- Global command confirmations must render after primary dashboard dialogs in
  `App.vue`. Dialog and alert-dialog overlays share the same component-library
  z-index, so source order is the project-owned stacking contract.
- Cron preview request is `{ "expression": "*/5 * * * *" }`, not the old `{ "cron_expr": ... }`.
- Frontend server state belongs in Vue Query. Components must not handwrite API URLs or parse Huma Problem Details directly.
- Dashboard production builds use Vite 8 / Rolldown configuration. Put bundle
  output and warning hooks under `build.rolldownOptions`; `build.rollupOptions`
  is only a deprecated compatibility alias.
- The compiled dashboard must keep `/static/app/assets/app.js` as the HTML entry
  script for the Go static page contract, but production JavaScript may be split
  into additional `modulepreload` chunks.
- Build-log filtering must be narrowly scoped by warning code and dependency.
  Do not lower the global log level or drop every warning just to keep the
  production build output clean.

### 4. Validation & Error Matrix

| Condition | Status / Error |
| --- | --- |
| Invalid or empty cron expression | `400 Bad Request` Problem Details |
| Invalid Web config name, URL, header, schedule, strategy, or numeric bound | `400 Bad Request` Problem Details |
| Create or rename targets an existing different config, even with `force=true` | `409 Conflict` Problem Details |
| Starting runtime while another run blocks it | `409 Conflict` Problem Details |
| Stopping inactive runtime or scheduler | `409 Conflict` Problem Details |
| Missing config or plan detail | `404 Not Found` Problem Details |
| Huma request validation failure | `422 Unprocessable Entity` Problem Details |
| Storage or unexpected runtime failure | `500 Internal Server Error` Problem Details |

### 5. Good/Base/Bad Cases

- Good: a save command writes the config, returns the saved `item`, and returns a fresh dashboard snapshot used to update Vue Query cache.
- Good: editing config `A` with unused body target `B` renames the stored entry,
  stops affected `A`/`B` runtime items only, and returns `item.name = "B"`.
- Good: creating or renaming to an existing target shows a duplicate-name
  warning and does not open an overwrite confirmation dialog.
- Good: saving or starting with `urls: ["https://example.com/a", "https://example.com/a"]` preserves both entries;
  runtime URL failure checks use remaining availability, while dashboard URL
  usage stats display each repeated address once with the shared usage count.
- Base: dashboard polling reads one aggregate snapshot; activity polling reads logs and events through one cursor pair; config numeric fields stay as numeric model values from the number-field component until request encoding.
- Base: pausing实时日志 must keep event polling active without advancing `after_log`; changing source/level filters should restart the log view from the buffered log set, not from a swallowed cursor.
- Bad: frontend code calls `/api/runtime/status`, `/api/runtime/logs`, `/api/preview_cron`, locally constructs an old route, bypasses the config editor guard, advances the log cursor while `include_logs=false`, uses `Input type="number"` and then trims/parses the form value by hand, lets an async command confirmation auto-close before the mutation result is known, hides all production build warnings through a broad logger override, or raises the chunk warning threshold instead of splitting an oversized dashboard bundle.

### 6. Tests Required

- HTTP handler tests must assert new response shapes for dashboard, config commands, cron preview, activity cursors, activity log filters, and OpenAPI schema.
- HTTP/storage/runtime tests must cover config validation, create conflict,
  forced create conflict, rename to unused target, forced rename conflict, and
  source/target runtime stop isolation for successful saves.
- Runtime tests must assert that stopped immediate runs clear
  `PlanSummary.total_bytes_raw` and `download_count` while preserving
  `last_run`, and that waiting scheduled runtime items keep task-session totals
  across completed triggers until the task/scheduler is stopped.
- Runtime tests must assert plan detail summary totals follow the same
  task-session window as the list row, while `PlanDetailPayload.history` keeps
  persisted historical runs after the task is stopped.
- Config guard tests must assert repeated `urls` entries are accepted and
  preserved after trimming. URL manager tests must assert a repeated address
  marked invalid leaves no available URL when every configured entry is the same
  string. Runtime status tests must assert URL usage stats deduplicate display
  rows for repeated addresses.
- Tests must assert old runtime routes are absent from OpenAPI and return `404`.
- Tests must assert Huma schema metadata stays disabled: no `$schema` field in
  generated OpenAPI or runtime JSON responses, no `describedby` response link,
  and no `/api/schemas/*` route.
- Frontend contract changes require `npm --prefix frontend run api:generate` and `npm --prefix frontend run build`.
- Frontend build-configuration changes require `npm --prefix frontend run build`
  and a log assertion for any warning class the task claims to remove.
- Frontend command-confirmation changes must verify the dialog stays pending
  while the mutation is in flight, closes on success, and leaves a retry/cancel
  path on failure.
- Go API changes require `gofmt -w .` and `go test ./...`.
- Page-shell changes should assert `/` loads `/static/app/assets/app.js`, not retired static JS.

### 7. Wrong vs Correct

#### Wrong

```typescript
await fetch('/api/activity?after_log=10&after_event=10')
await fetch('/api/preview_cron', { body: JSON.stringify({ cron_expr }) })
```

#### Correct

```typescript
const dashboard = await apiClient.dashboard()
const activity = await apiClient.activity(10, 10, {
  includeLogs: true,
  logSources: ['scheduler', 'runtime'],
  logLevels: ['info', 'error'],
})
```

`apiClient` wraps the generated Hey API SDK, and Vue Query owns dashboard/activity polling plus command cache updates.

#### Wrong

```typescript
await apiClient.saveConfig(form.name, form as Config)
```

#### Correct

```typescript
const projection = projectConfigForm(form, 'save')
if (projection.ok) {
  await apiClient.saveConfig(sourceName, projection.payload, { mode })
}
```

#### Wrong

```go
plan.TotalBytesRaw = history.TotalBytesRaw + currentTotal
plan.DownloadCount = history.DownloadCount + currentCount
```

#### Correct

```go
sessionTotal, sessionCount := itemSessionTotals(item, currentTotal, currentCount, active)
plan.TotalBytesRaw = sessionTotal
plan.DownloadCount = sessionCount
plan.LastRun = history.LastRun
```
