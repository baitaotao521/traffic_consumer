# 浅色专业控制台重设计

## Goal
完全重新设计 Traffic Consumer 前端，使运行状态更易读、常用操作更易发现。

## Confirmed facts
- 用户要求完全重设计，并于 2026-09-08 选择浅色专业控制台。
- 现有 Vue 3 / Vite / shadcn-vue 前端为单页长列表；业务由 useDashboard 编排。
- 原 Docker 部署任务独立保留，本任务只涉及前端与对应规范。

## Requirements
- 浅色中性背景、青绿色强调、统一排版与层级。
- 响应式侧边导航：总览、计划管理、线程监控、日志与历史。
- 总览优先展示指标、流量趋势、线程活跃和 URL 分布。
- 计划支持名称搜索、运行/调度/空闲筛选和完整操作。
- 重组配置编辑为基础配置、执行限制、调度与请求；保留所有字段与校验。
- 保留真实 API、轮询、图表、日志、调度及确认流程。
- 手机导航可用，表格局部滚动，键盘焦点可辨识。

## Acceptance criteria
- [x] 四个导航入口可切换，刷新与浏览器前进后退保留页面。
- [x] 首屏指标及图表真实，空状态和加载状态清楚。
- [x] 搜索/筛选可组合，清除筛选恢复列表。
- [x] 创建/编辑/详情和危险动作确认可达。
- [x] 390px 手机与桌面均无页面横向溢出。
- [x] 生产构建、差异检查和浏览器交互验证通过。

## Out of scope
后端契约重写、远程部署、新增身份认证。

## Validation notes

- `npm --prefix frontend run build`: passed (vue-tsc and Vite).
- `git diff --check`: passed.
- Browser QA at desktop and 390px: navigation drawer, plan name search, state filters, empty results, reset filters, config save, cross-tab error positioning, delete confirmation/cancel, thread empty state, and log pause state.
- QA used an isolated local data directory at `/tmp/traffic-redesign-qa-20260908`; remote deployment data was not modified.
- Connection error state is explicit, and the previous snapshot is retained only when available.
