# API And State Guidelines

> Frontend API, server-state, activity-stream, and dashboard orchestration rules.

---

## Overview

The dashboard API contract is Go-owned and OpenAPI-generated. Frontend code
should consume that contract through the generated SDK wrapper, then project it
through Vue Query, `useDashboard`, and `domain/**`.

Do not let components define their own version of the backend payload. That is
contract drift wearing a tiny hat.

---

## API Generation Contract

The generation command is:

```bash
npm --prefix frontend run api:generate
```

This runs `go run ../cmd/openapi > src/api/openapi.json` from `frontend/`, then
regenerates `frontend/src/api/generated`.

Run it whenever Web API routes, schemas, command payloads, error models, or
OpenAPI metadata change.

Reference files:
- `frontend/package.json`
- `cmd/openapi`
- `frontend/src/api/openapi.json`
- `frontend/src/api/generated`
- `.trellis/spec/backend/web-api-contracts.md`

---

## API Wrapper

Use `frontend/src/api/client.ts` as the only hand-written API gateway.

Local pattern:

- Import generated SDK functions from `./generated`.
- Convert generated `{ data, error, response }` results with `unwrap`.
- Convert backend errors into `ApiError`.
- Keep route-specific query/path/body encoding inside `apiClient` methods.

Reference: `frontend/src/api/client.ts`.

Good:

```typescript
const response = await apiClient.previewCron(expression)
```

Bad:

```typescript
await fetch('/api/cron/preview', { body: JSON.stringify({ expression }) })
```

Components and composables should handle `ApiError`, not raw Huma Problem
Details.

---

## Vue Query Ownership

Use `frontend/src/api/queries.ts` for server state.

Current query keys:

- `dashboardQueryKey`
- `activityQueryKey`
- `configurationQueryKey(name)`
- `planDetailQueryKey(name)`

Dashboard and activity polling currently use `refetchInterval: 1000`.
Command mutations should use `useDashboardCommandMutation` so successful
commands write the returned dashboard snapshot into the `dashboardQueryKey`
cache.

Reference: `frontend/src/api/queries.ts`.

Avoid these patterns:

- Duplicating query keys inside components.
- Calling `useQueryClient()` in a panel just to patch dashboard data.
- Refetching the dashboard after every command when the command response already
  includes a fresh snapshot.

---

## Dashboard Orchestration

`frontend/src/composables/useDashboard.ts` is the owner for dashboard runtime
state. It coordinates:

- selected config and config detail loading
- runtime command mutations
- save/delete/stop confirmations
- editor open/dirty state
- activity and log cursors
- recent toast de-duplication
- chart samples derived from dashboard snapshots
- cron preview debounce and error handling

Panels should receive plain props and emit typed events. `App.vue` wires these
props/events to `useDashboard`.

Reference files:
- `frontend/src/App.vue`
- `frontend/src/composables/useDashboard.ts`
- `frontend/src/components/dashboard/PlanPanel.vue`
- `frontend/src/components/dashboard/ConfigSheet.vue`

---

## Config Editor Projection

Config save, start, and temporary test must flow through
`domain/configEditor.ts`.

Local pattern:

- `projectConfigForm(form, mode)` validates the editor form and returns a
  `ConfigProjection`.
- `projectConfigPayload(name, payload, mode)` validates a saved config before
  start.
- Optional numeric fields treat blank/zero as omitted where the product uses
  "no limit" semantics.
- Duplicate URLs are preserved as user input.
- Empty URL lists are allowed for save but rejected for start/temp-test.
- `request_headers` parsing belongs in one shared parser.

Reference: `frontend/src/domain/configEditor.ts`.

Do not build a `Config` object inside `ConfigSheet.vue` or another panel. The
sheet edits `ConfigForm`; the domain layer produces `ConfigPayload`.

---

## Activity And Logs

Activity polling uses one endpoint with independent event/log cursor behavior.

Local rules from `useDashboard.ts`:

- `eventCursor` should keep advancing even when realtime logs are disabled.
- `include_logs=false` should not consume new log entries into the visible log
  list.
- Enabling logs resets the visible log view and starts from the current stream
  position.
- Source/level filter changes reset the visible log list and cursor before a
  refetch.
- Log filter selection cannot become empty, because an empty filter means the
  backend returns all logs.

Reference files:
- `frontend/src/api/client.ts`
- `frontend/src/api/queries.ts`
- `frontend/src/composables/useDashboard.ts`
- `frontend/src/domain/dashboard.ts`

---

## Toasts And User Feedback

Use `vue-sonner` through `useDashboard` for command and activity feedback.

Local pattern:

- `pushAlert` de-duplicates recent toasts by variant, title, and description.
- `pushError` converts `ApiError` into destructive toasts.
- Command success messages use the backend `message` field.
- `projectActivityEvent` suppresses routine `status_update` success toasts so
  command responses and activity events do not say the same thing twice.

Reference files:
- `frontend/src/composables/useDashboard.ts`
- `frontend/src/domain/dashboard.ts`
- `frontend/src/App.vue`

UI copy should be final user-facing text, not a design explanation. Do not
display the same message twice with slightly different wording.

---

## Cross-Layer API Changes

When a frontend change depends on Go API behavior:

1. Update the Go API/model/runtime contract first.
2. Run `npm --prefix frontend run api:generate`.
3. Update `apiClient` and Vue Query hooks.
4. Update domain projections.
5. Update panels only after the projected view model is stable.
6. Run `go test ./...` and `npm --prefix frontend run build`.

Read [Backend Web API Contracts](../backend/web-api-contracts.md) before making
these changes.
