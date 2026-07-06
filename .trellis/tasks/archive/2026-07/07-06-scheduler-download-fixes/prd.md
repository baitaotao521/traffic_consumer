# 修复调度与下载流程问题

## Goal

修复当前调度模式和下载调度流程中已经发现的可靠性、可停止性、运行态污染和观测问题，形成“发现问题 -> 修复 -> 记录 -> 复查”的闭环。每个完成的修复必须记录到仓库根目录 `记录文档.md`。

## Confirmed Facts

- Web 运行态允许不同配置并行运行，但同名配置只能有一个下载或等待调度实例，证据：`internal/runtime/runtime.go:244-282`。
- Web 调度器为每个计划创建独立 `cron.Cron`，触发时调用 `runScheduledItem(item, context.Background())`，证据：`internal/runtime/runtime.go:716-727`。
- `Consumer.RunOnce` 执行同步下载轮次，但只有 `StartAsync` 会预先设置可取消 context；Web 调度触发的同步轮次没有把 `Stop()` 连接到正在执行的请求 context，证据：`internal/consumer/consumer.go:91-114`、`internal/consumer/consumer.go:151-160`、`internal/runtime/runtime.go:862-865`。
- 每轮下载开始会调用 `URLMgr.ResetRuntimeState()`，但该方法只重置线程状态和使用统计，没有清理本轮失效 URL 标记，证据：`internal/urlmanager/urlmanager.go:64-74`。
- 单条 URL 失败会在同一线程内最多重试 5 次；并发线程拿到同一坏链后不会提前感知其他线程已经判定该 URL 失效，证据：`internal/consumer/consumer.go:314-343`。
- 多配置运行时，顶部线程状态会聚合全部活动任务，但 URL 使用占比只读取第一个活动任务，证据：`internal/runtime/runtime.go:460-467`。
- 历史记录和持久化运行结果目前固定写入 `成功`，无法区分用户停止、达到限制、无可用链接或全部链接失效，证据：`internal/consumer/consumer.go:206-244`。

## Requirements

- R1: 调度轮次之间不能共享临时失效 URL 状态；未开启失败自动移除时，下一轮调度必须重新尝试配置中的 URL。
- R2: Web 调度触发的同步下载轮次必须可被 `StopRuntimeConfig` / `StopAll` 取消，正在阻塞的 HTTP 请求和限速等待应尽快退出。
- R3: 当任一线程已经将 URL 判定为失效，其他线程后续重试同一 URL 时必须短路，避免重复放大坏链重试。
- R4: 多配置并行运行时，状态接口的 URL 使用统计必须覆盖全部活动任务，而不是只展示第一个活动任务。
- R5: 运行历史必须记录业务结束原因，包括成功、用户停止、达到流量限制、达到下载次数限制、达到时长限制、无可用链接和全部链接失效。
- R6: 每个完成修复项必须追加到 `记录文档.md`，包含问题、改动、验证证据和结果。
- R7: 修复后必须重新执行自动化测试并复查剩余问题；若仍有未解决问题，继续补充计划并迭代修复。

## Acceptance Criteria

- [x] AC1: 存在测试证明 URL 被标记失效后，调用 `ResetRuntimeState` 的下一轮仍能重新选择该 URL。
- [x] AC2: 存在测试证明直接调用 `RunOnce(context.Background())` 的下载轮次能被 `Consumer.Stop()` 取消，并且阻塞请求会结束。
- [x] AC3: 下载重试逻辑在 URL 已失效时提前退出，并保留现有失败通知和自动移除行为。
- [x] AC4: 存在测试证明多个活动配置的 URL 使用统计会合并输出。
- [x] AC5: 存在测试证明达到限制或停止原因会写入历史结果，不再全部显示为 `成功`。
- [x] AC6: `go test ./...` 通过。
- [x] AC7: `记录文档.md` 逐项记录本任务已修复问题和验证结果。

## Out Of Scope

- 不在本轮新增 UI 控件或公开配置项。
- 不改变默认的多配置并行能力；全局并发预算作为复查项记录，只有在当前代码证据显示必须落地时再继续拆分修复。
