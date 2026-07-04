# API 契约重写设计

## Decision

采用破坏式新 API。现有 `/api/*` 路由、手写前端 payload 类型和 `apiFetch<T>` 薄封装不作为兼容目标。Vue dashboard 会同步迁移到新契约。

包选型基线：

- 后端：`github.com/danielgtaylor/huma/v2`，继续使用现有 `chi` router，通过 `humachi` 适配器注册 API。Huma 文档显示它支持 Chi adapter、Go struct 输入/输出、OpenAPI 生成和请求校验。
- 前端 SDK：`@hey-api/openapi-ts`，从 OpenAPI 3.x schema 生成 TypeScript 类型和 fetch SDK；它支持当前 TypeScript 6 依赖树。
- 前端 server state：`@tanstack/vue-query`，管理状态查询、轮询、mutation pending/error/success 和 mutation 后 query invalidation。

## Architecture

```text
internal/model
  -> Go 业务 payload 与兼容字段

internal/httpapi
  -> Huma operation registration
  -> request/response wrapper structs
  -> OpenAPI schema generation
  -> HTTP status / Problem Details mapping

frontend/src/api/openapi.json
  -> Go 路由生成的 OpenAPI schema

frontend/src/api/generated/*
  -> @hey-api/openapi-ts generated fetch SDK and types

frontend/src/api/queries.ts
  -> Vue Query keys, queries, mutations, invalidation rules

frontend/src/domain/dashboard.ts
  -> runtime event projection, thread row projection, form/config conversion

frontend/src/composables/useDashboard.ts
  -> UI-local state only: selected config, dialogs, toast orchestration
```

## API Shape

New routes live under `/api`.旧 `/api/runtime/*`、`/api/preview_cron`、`/api/runtime/logs`、`/api/runtime/events` 不保留。

Dashboard:

- `GET /api/dashboard` returns `{ dashboard }`, where dashboard includes configs, runtime status, scheduler status, and plans.

Configuration:

- `GET /api/configurations/{name}` returns `{ item: ConfigResource }`.
- `PUT /api/configurations/{name}` saves and returns `{ item: ConfigResource, dashboard, message }`.
- `DELETE /api/configurations/{name}` returns `{ dashboard, message }`.

Commands:

- `POST /api/commands/start-runtime` accepts `ConfigPayload` and returns `{ dashboard, message }`.
- `POST /api/commands/start-temp-test` accepts `ConfigPayload` and returns `{ dashboard, message }`.
- `POST /api/commands/stop-runtime` returns `{ dashboard, message }`.
- `POST /api/commands/stop-scheduler` returns `{ dashboard, message }`.
- `POST /api/commands/stop-plan/{name}` returns `{ dashboard, message }`.

Plans:

- `GET /api/plans/{name}` returns `{ item: PlanDetail }`.

Activity by polling:

- `GET /api/activity?after_log=<seq>&after_event=<seq>` returns `{ logs, events, log_cursor, event_cursor }`.

Utilities:

- `POST /api/cron/preview` accepts `{ expression: string }` and returns `{ items: string[] }`.

## Data Flow

Config edit:

```text
ConfigSheet form -> dashboard domain formToConfig -> generated SDK PUT
  -> Huma validates body -> runtime.Manager.SaveConfig
  -> command response returns fresh dashboard
  -> Vue Query updates dashboard cache
```

Runtime polling:

```text
Vue Query useQuery(refetchInterval)
  -> generated SDK GET /dashboard
  -> generated type
  -> domain projection creates KPI, thread rows, chart values
  -> components render projected values
```

Events/logs:

```text
Vue Query useQuery(refetchInterval, cursor)
  -> generated SDK GET /activity
  -> domain event projector maps backend event to toast/update actions
  -> cursor stored as query/domain state, not scattered timers
```

## Error Handling

New API uses Huma/OpenAPI-compatible JSON error responses instead of the old `{ error, message }` compatibility pair. Frontend maps any non-2xx OpenAPI error into an app-level `ApiError` with:

- `status`
- `title`
- `message`
- optional `details`

UI displays only the user-facing message once. Duplicate frontend success toasts must be removed when the backend command response already supplies a message.

## Compatibility

Breaking compatibility is accepted for the Web dashboard API. Persisted config/stat files remain compatible because storage schema migration is out of scope. Legacy Python Web UI and old static JS remain deprecated references only.

## Package Notes

- Huma docs show `humachi.New(router, huma.DefaultConfig(...))` and route group support for `/api` style base paths: https://github.com/danielgtaylor/huma/blob/main/docs/docs/tutorial/your-first-api.md and https://github.com/danielgtaylor/huma/blob/main/docs/docs/features/bring-your-own-router.md.
- TanStack Query docs show Vue `useQuery`, `useMutation`, `useQueryClient`, and mutation success invalidation: https://github.com/TanStack/query/blob/main/docs/framework/vue/quick-start.md.
- `@hey-api/openapi-ts` npm metadata identifies it as an OpenAPI to TypeScript SDK generator with TypeScript 6 support: https://www.npmjs.com/package/@hey-api/openapi-ts.

## Risks

- Huma response wrapper structs can initially feel more verbose than raw `http.Handler`; keep wrappers close to route registration and do not duplicate business structs unless OpenAPI tags require it.
- Generated TypeScript must not be edited by hand. Add a repeatable script and build check.
- Vue Query will own server state. UI-local state must not shadow it unless the value is genuinely client-only, such as dialog open state or selected config name.
