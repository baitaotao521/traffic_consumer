# 修复调度日志缺失与实时日志返回链路

## Goal

修复 Web 控制台的日志链路，让调度相关日志能够被记录并复用，同时满足两条用户可见路径：

- 后端控制台始终打印完整日志。
- 前端按选中的日志类型请求后端返回对应日志，不再出现“打开实时日志后拿不到数据”的情况。

## Background

- Go 后端 `runtime.Manager` 维护 Web 运行态日志和事件内存缓冲，`GET /api/activity` 用 `after_log` / `after_event` 游标返回日志与事件增量。证据：`internal/runtime/runtime.go:82`、`internal/runtime/runtime.go:101`、`internal/httpapi/server.go:382`。
- 前端 `useDashboard()` 启动后会持续轮询 `useActivityQuery()`，即使 `logEnabled` 为 `false` 也会继续推进 `logCursor`，只是把日志丢弃不展示。证据：`frontend/src/api/queries.ts:28`、`frontend/src/composables/useDashboard.ts:94`、`frontend/src/composables/useDashboard.ts:341`。
- 当前前端“实时日志”只有一个总开关，没有日志类型筛选；开关打开后只能看到开启之后新增的日志，看不到此前已经被游标消费掉的日志。证据：`frontend/src/components/dashboard/LogHistoryPanel.vue:41`、`frontend/src/composables/useDashboard.ts:343`。
- Go 调度器启动链路 `startScheduler()` 只负责注册 cron 任务，没有记录清晰的调度业务日志，因此当前很难从日志判断“计划已建立 / 正在等待 / 已触发 / 已完成 / 已停止”。证据：`internal/runtime/runtime.go:477`。
- 用户已明确新的目标：日志必须全部打印到后端控制台；前端只负责设置需要返回的日志类型，由后端按类型返回日志。
- 用户已确认保留当前“调度历史”页签的历史摘要表，不改成真实日志列表。
- 用户已明确拒绝模仿 Python 端日志设计；Go 端日志类型和埋点需要重新设计，不能照着旧实现复制脏语义。
- 用户已确认前端日志筛选同时支持“来源”和“级别”两个维度，而不是只做其中一个。

## Requirements

- R1: 保持 Go 后端为唯一维护实现；不得把新日志行为扩展回 `web_ui.py`、`main.py`、`traffic_consumer.py` 或 `app/**`。
- R2: `runtime.Manager.AddLog` 写入内存缓冲时，必须同时把同一条日志输出到后端控制台，避免 Web 调试依赖前端面板。
- R3: 调度链路必须重新设计关键业务日志与埋点，不照搬 Python 端日志文本或结构；至少覆盖：调度配置建立、调度器启动、计划触发执行、计划执行结束、下一次执行时间、调度停止或替换。
- R4: `GET /api/activity` 必须支持按日志来源和日志级别筛选返回日志；日志筛选不能影响事件返回。
- R5: 前端实时日志不得在“未展示日志”的状态下提前吞掉历史游标；用户开启或调整日志筛选条件后，应该能收到符合条件的日志返回。
- R6: 前端日志筛选只负责声明需要的来源和级别，不在前端重复实现后端日志分类逻辑。
- R7: 日志来源至少要能区分调度、运行态控制、下载执行三类，让用户可以单独查看调度链路。
- R8: 现有 dashboard 事件刷新行为必须保留，`history_update` 等事件仍然要驱动调度历史/快照刷新。
- R9: 变更日志契约时，必须同步更新 Go 注释、HTTP handler 测试、运行态测试，以及前端 API 调用层。

## Acceptance Criteria

- [ ] 后端控制台能看到 Web 运行态写入的全部日志，包含调度相关业务日志。
- [ ] 启动带 cron 或 interval 的计划后，日志缓冲中会出现调度建立、触发执行、执行完成、下次执行时间等记录。
- [ ] `GET /api/activity` 支持按日志来源和日志级别筛选，并保持事件增量接口不受影响。
- [ ] 前端实时日志改为按所选来源和级别拉取日志；开启日志面板后不会因为旧游标已被吞掉而表现成“没有返回数据”。
- [ ] 调度执行完成后，原有事件驱动的 dashboard / 调度历史刷新仍然有效。
- [ ] `go test ./...` 通过。
- [ ] `npm --prefix frontend run build` 通过。

## Out Of Scope

- 不把这次修改扩大成 Python Web UI 复活或 Socket.IO 行为回滚。
- 不把“调度历史”页签改造成真实日志列表；它继续保留当前历史摘要语义。
- 不重做整套日志展示 UI，只修复返回链路并补齐必要的日志类型选择入口。

## Open Questions

- None.
