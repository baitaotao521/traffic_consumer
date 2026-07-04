# 前后端 API 契约重写

## Goal

完全重写 Traffic Consumer Web 控制台的前后端 API 交互边界，让配置、运行态、调度、日志和事件的请求/响应契约集中、稳定、可验证。前端组件不再直接理解后端零散字段和异常细节，后端接口不再靠“前端刚好这样写”维持行为。

## Background

- 主后端是 Go HTTP 服务，Vue dashboard 从 `frontend/` 构建到 `static/app/` 并由 Go API 提供服务；Python Web UI 已废弃，只能作为兼容参考，不应承接新行为。证据：`.trellis/spec/backend/index.md:9`、`.trellis/spec/backend/index.md:13`、`.trellis/spec/backend/index.md:61`。
- 页面 shell `templates/index.html` 已直接加载 Vite 构建产物 `static/app/assets/app.js` 和 `static/app/assets/app.css`，老 `static/js/main.js` 不是主入口。
- 当前 API 路由集中注册在 `internal/httpapi/server.go`，已有资源覆盖 cron 预览、配置 CRUD、运行控制、运行状态、调度状态、计划详情、日志和事件。证据：`internal/httpapi/server.go:47`。
- Go 侧跨层 payload 主要定义在 `internal/model/model.go`，包含配置、状态、计划、历史、日志和事件结构。证据：`internal/model/model.go:65`、`internal/model/model.go:156`、`internal/model/model.go:190`、`internal/model/model.go:222`、`internal/model/model.go:282`、`internal/model/model.go:298`。
- 前端在 `frontend/src/types.ts` 手写了对应 TypeScript 类型，和 Go model 形成重复契约。证据：`frontend/src/types.ts:3`、`frontend/src/types.ts:24`、`frontend/src/types.ts:57`、`frontend/src/types.ts:100`。
- 前端请求封装只有 `apiFetch<T>`，直接 `JSON.parse` 后按泛型强转；没有统一处理非 JSON 响应、网络异常、响应 envelope、业务错误类型或契约校验。证据：`frontend/src/api.ts:10`。
- 前端 dashboard 业务编排集中在 `useDashboard.ts`：它负责表单转换、状态归一化、轮询、事件分派、错误 toast、操作后刷新等多个职责。证据：`frontend/src/composables/useDashboard.ts:207`、`frontend/src/composables/useDashboard.ts:245`、`frontend/src/composables/useDashboard.ts:274`、`frontend/src/composables/useDashboard.ts:310`、`frontend/src/composables/useDashboard.ts:337`、`frontend/src/composables/useDashboard.ts:365`。
- 后端现有错误响应约定为 JSON，兼容字段同时写 `error` 和 `message`；新 API 可破坏该兼容约束，但必须继续提供稳定 JSON 错误。证据：`.trellis/spec/backend/error-handling.md:34`、`.trellis/spec/backend/error-handling.md:43`。
- 项目跨层指南明确要求在边界定义精确输入、输出和错误，并避免每个消费者私自解析同一 payload。证据：`.trellis/spec/guides/cross-layer-thinking-guide.md:44`、`.trellis/spec/guides/cross-layer-thinking-guide.md:93`。
- 用户已明确允许破坏现有 `/api/*` URL 和 JSON 字段兼容性，按完全重写处理。
- 用户已明确要求接口全部重新设计，包括与前端的交互方式，而不是只规范旧接口；旧 `/api/runtime/*`、`/api/preview_cron`、`/api/runtime/logs`、`/api/runtime/events` 等接口不得保留。
- 用户已确认使用现成包作为重写基线，不为请求状态、契约生成、类型生成或客户端调用重复造轮子。实施时采用 Huma + `@hey-api/openapi-ts` 生成 fetch SDK + `@tanstack/vue-query`；原定 `openapi-typescript` 与项目 TypeScript 6 peer dependency 冲突，已替换为支持 TS6 的 Hey API 生成器。

## Requirements

- R1: 保持 Go 后端为唯一维护实现；不得把新 API 行为加到 `web_ui.py`、`main.py`、`traffic_consumer.py` 或 `app/**`。
- R2: 定义清晰的 API 分层：HTTP transport、OpenAPI 契约、生成 SDK、server-state 查询/变更、dashboard UI orchestration 分开，避免组件或 composable 到处拼 URL、解析错误和修补字段。
- R3: 配置契约必须保留现有业务语义：`urls` 显式空列表、旧 `url` 单链接兼容、`request_headers` 多格式兼容、`name/config_name` 兼容。
- R4: 运行态契约必须稳定表达 `running | scheduled | stopped`、线程状态、URL 使用统计、操作消息，并明确哪些字段允许缺省、哪些字段由前端补默认值。
- R5: 日志和事件契约必须有单一前端入口进行增量游标、事件类型判断、toast 投影和后续刷新动作；组件不能重复判断 `event.type`、`event_type`、`variant` 这些后端细节。
- R6: 错误处理使用新 API 的统一错误模型，优先采用 Huma/OpenAPI 标准 Problem Details；前端必须映射为稳定错误对象，避免只有裸 `Error(message)` 这种贫血信息。
- R7: API 文案必须遵守现有 UI 文案规则：界面只显示最终用户可理解的操作结果，不解释设计原因；同一信息不换措辞重复展示。
- R8: 涉及后端接口或 payload 变化时，必须同步更新 Go model 注释、HTTP handler 测试、前端类型/decoder、前端调用层和构建产物。
- R9: API 契约和前端调用层优先使用成熟包：后端 OpenAPI 生成与校验用 Huma，前端 SDK 生成用 `@hey-api/openapi-ts`，轮询、缓存和 mutation 状态用 `@tanstack/vue-query`；项目代码只保留业务投影和 UI 状态。
- R10: 前端主界面读取使用聚合 dashboard 快照，日志和事件使用统一 activity 增量接口，用户操作使用 command 接口并返回操作后的 dashboard；不得延续旧的状态、调度、日志、事件多接口散乱轮询模型。

## Acceptance Criteria

- [ ] 后端 API 改为 OpenAPI-first 契约，OpenAPI schema 可由 Go 代码生成并纳入测试或构建校验。
- [ ] 前端 API SDK 由 OpenAPI schema 生成，不再手写后端 payload 镜像类型。
- [ ] 前端请求使用生成的类型安全 SDK，不再手写 URL 字符串和 `apiFetch<T>` 泛型强转。
- [ ] 旧 `/api/runtime/*`、`/api/preview_cron`、`/api/runtime/logs`、`/api/runtime/events` 接口不可用，OpenAPI schema 中不再出现这些旧路径。
- [ ] 前端轮询、缓存、mutation pending/error/success 状态使用 Vue Query，不再在 composable 中手写定时器状态机。
- [ ] `frontend/src/composables/useDashboard.ts` 不再直接承担 URL 拼接、接口错误解析或运行事件底层契约判断；这些逻辑迁移到 API/client、query hooks 或 domain projection 层。
- [ ] 前端对配置、状态、调度、计划详情、日志、事件的读取都经过生成类型和业务投影入口。
- [ ] Go HTTP API 返回统一 JSON 错误，HTTP status 约定保持可测试。
- [ ] 配置 CRUD、启动/停止、临时测试、计划停止、cron 预览、日志轮询、事件轮询都有契约级测试或等价构建校验覆盖。
- [ ] `go test ./...` 通过。
- [ ] `npm --prefix frontend run build` 通过并更新 `static/app/` 构建产物。

## Technical Direction

- 采用破坏式新 API，前端同步迁移到新路由和新响应结构。
- 使用 Huma 生成 OpenAPI 并承接请求/响应校验；前端用 `@hey-api/openapi-ts` 从 schema 生成 fetch SDK，用 `@tanstack/vue-query` 管理 dashboard/activity 轮询和 command mutation。

## Out Of Scope

- 不重启 Python Web UI、Socket.IO 或 PyInstaller 维护路径。
- 不自研 API 框架、自定义 OpenAPI 生成器、自定义 fetch 状态机或自定义缓存层。
- 不把这次重设计扩大成 UI 视觉改版、运行调度语义重写或存储格式大迁移。

## Open Questions

- None.
