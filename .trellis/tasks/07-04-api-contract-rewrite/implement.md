# Implementation Plan

## Checklist

- [ ] Add backend dependency `github.com/danielgtaylor/huma/v2` and use the `humachi` adapter with the existing `chi` router.
- [ ] Replace current `/api/*` handlers with redesigned `/api` Huma operations: dashboard, configurations, commands, activity, plan detail, cron preview.
- [ ] Add request/response wrapper structs with business comments near non-obvious compatibility behavior.
- [ ] Add OpenAPI schema generation path, either as a committed schema artifact or a repeatable Go generator command.
- [ ] Add HTTP tests for new routes, status codes, command responses, cursor responses, and error responses.
- [ ] Add frontend dependency `@tanstack/vue-query`; add dev dependency `@hey-api/openapi-ts`.
- [ ] Add `npm` script to generate frontend API SDK from the Go-produced OpenAPI schema.
- [ ] Replace `frontend/src/api.ts` with generated SDK adapter modules.
- [ ] Refactor dashboard server state into Vue Query hooks and mutation invalidation.
- [ ] Move config/form conversion, thread rows, charts, event-to-toast projection, and date formatting into domain helpers where appropriate.
- [ ] Remove direct URL string calls, old split status/scheduler/log/event polling, direct `event.type` branching, and hand-rolled polling timers from `useDashboard.ts`.
- [ ] Rebuild `static/app/` from Vue source.

## Validation

- `gofmt -w .`
- `go test ./...`
- `npm --prefix frontend run build`
- Manual smoke test in browser after implementation:
  - load dashboard
  - list configs
  - create/update config
  - start runtime
  - stop runtime
  - preview cron
  - view logs/events without duplicate toasts

## Risk Points

- `internal/httpapi/server.go`: route replacement changes every dashboard API call.
- `internal/model/model.go`: payload comments must stay aligned with generated OpenAPI behavior.
- `frontend/src/types.ts`: backend payload mirrors should be deleted or reduced to UI-only types after generation.
- `frontend/src/composables/useDashboard.ts`: largest refactor; keep UI state while moving server-state machinery out.
- `static/app/`: generated bundle will change substantially after build.

## Rollback

The breaking rewrite should stay on its task branch until the dashboard smoke test passes. If Huma integration creates too much route churn, keep `chi` page/static serving intact and isolate Huma only under `/api/v2` before deleting old handlers.
