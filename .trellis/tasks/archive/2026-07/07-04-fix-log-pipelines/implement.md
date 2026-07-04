# 实施计划

## Checklist

- [ ] 梳理现有 `runtime.Manager`、`consumer.Consumer`、`/api/activity` 与前端 `useDashboard()` 的日志链路，确认需要补充的调度日志节点。
- [ ] 修改运行态日志写入逻辑，让 `AddLog()` 同时写入后端控制台和 Web 缓冲。
- [ ] 重新设计 Go 端日志埋点，在调度建立、启动、触发、结束、下次执行时间刷新、停止/替换等节点补充业务日志，并补齐贴近实现的注释。
- [ ] 扩展 activity 接口查询参数，支持按日志来源和日志级别筛选 `logs`。
- [ ] 修改前端日志面板状态与查询逻辑，用来源/级别请求替代当前“开关+本地丢弃日志”行为。
- [ ] 增加或更新 Go 测试，覆盖调度日志写入与 activity 日志筛选。
- [ ] 视前端改动影响更新 API 生成/构建产物，并执行质量检查。

## Validation

- `gofmt -w .`
- `go test ./...`
- `npm --prefix frontend run build`

## Risky Files

- `internal/runtime/runtime.go`
- `internal/httpapi/server.go`
- `internal/httpapi/server_test.go`
- `frontend/src/composables/useDashboard.ts`
- `frontend/src/components/dashboard/LogHistoryPanel.vue`
- `frontend/src/api/queries.ts`

## Pre-Start Review Focus

- 前端日志来源与级别双筛选的默认值与交互形式是否足够简单。
