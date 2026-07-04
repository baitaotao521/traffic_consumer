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
GET    /api/activity?after_log=<seq>&after_event=<seq>
GET    /api/configurations/{name}
PUT    /api/configurations/{name}
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
- Command endpoints return `{ dashboard, message }`; config save also returns `item`.
- Config save/start bodies use `model.Config` / generated `ConfigWritable`. Optional numeric/string fields are omitted when blank. `urls: []` is preserved as an explicit empty URL list.
- Dashboard numeric form fields must use the component-library number input pattern (`NumberField` / Reka `NumberFieldRoot` + `NumberFieldInput`), not a generic `Input type="number"`. The library's number field owns the numeric model state; form adapters should receive `number | undefined` and must not trim or parse these fields as strings.
- Cron preview request is `{ "expression": "*/5 * * * *" }`, not the old `{ "cron_expr": ... }`.
- Frontend server state belongs in Vue Query. Components must not handwrite API URLs or parse Huma Problem Details directly.

### 4. Validation & Error Matrix

| Condition | Status / Error |
| --- | --- |
| Invalid or empty cron expression | `400 Bad Request` Problem Details |
| Starting runtime while another run blocks it | `409 Conflict` Problem Details |
| Stopping inactive runtime or scheduler | `409 Conflict` Problem Details |
| Missing config or plan detail | `404 Not Found` Problem Details |
| Huma request validation failure | `422 Unprocessable Entity` Problem Details |
| Storage or unexpected runtime failure | `500 Internal Server Error` Problem Details |

### 5. Good/Base/Bad Cases

- Good: a save command writes the config, returns the saved `item`, and returns a fresh dashboard snapshot used to update Vue Query cache.
- Base: dashboard polling reads one aggregate snapshot; activity polling reads logs and events through one cursor pair; config numeric fields stay as numeric model values from the number-field component until request encoding.
- Bad: frontend code calls `/api/runtime/status`, `/api/runtime/logs`, `/api/preview_cron`, locally constructs an old route, or uses `Input type="number"` and then trims/parses the form value by hand.

### 6. Tests Required

- HTTP handler tests must assert new response shapes for dashboard, config commands, cron preview, activity cursors, and OpenAPI schema.
- Tests must assert old runtime routes are absent from OpenAPI and return `404`.
- Frontend contract changes require `npm --prefix frontend run api:generate` and `npm --prefix frontend run build`.
- Go API changes require `gofmt -w .` and `go test ./...`.
- Page-shell changes should assert `/` loads `/static/app/assets/app.js`, not retired static JS.

### 7. Wrong vs Correct

#### Wrong

```typescript
await fetch('/api/runtime/status')
await fetch('/api/preview_cron', { body: JSON.stringify({ cron_expr }) })
```

#### Correct

```typescript
const dashboard = await apiClient.dashboard()
const preview = await apiClient.previewCron(expression)
```

`apiClient` wraps the generated Hey API SDK, and Vue Query owns dashboard/activity polling plus command cache updates.
