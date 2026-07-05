# 前端体验与性能优化

## Goal

提升 Traffic Consumer 前端 dashboard 的首屏稳定性、移动端可用性、可访问性和维护整洁度，同时保持现有 Vue Query / domain / shadcn-vue 组件边界不漂移。

本次优化不改变后端 API、运行态业务规则或配置保存语义，只收敛前端展示、加载和维护问题。

## Confirmed Facts

- 初始生产构建输出包含 `charts.js`、`ui-vendor.js`、`data-vendor.js`、`vue-vendor.js` 和 `app.js`。
- 初始 `frontend/dist/index.html` 会 `modulepreload` `/assets/charts.js`，`frontend/dist/assets/app.js` 顶部也静态导入 `./charts.js`。
- 初始 `charts.js` 是最大 JS chunk，构建输出约 `317.54 kB` 原始、`107.82 kB` gzip。
- 改为自动拆包后，生产构建输出由动态导入生成 `DashboardChart.js`、`ConfigSheet.js`、`PlanDetailDialog.js`、`ConfirmActionDialog.js` 等 chunk，不再依赖手动 vendor 分组。
- `frontend/src/style.css` 顶部通过 `@import url('https://fonts.googleapis.com/...')` 请求 Google Fonts，但 Tailwind 输出仍使用系统 `--font-sans`。
- `frontend/src/components/dashboard/PlanPanel.vue` 使用 `Table class="min-w-[1220px]"`，适合桌面扫描，但在窄屏上主要依赖横向滚动。
- `frontend/src/components/dashboard/ThreadPanel.vue` 和 `LogHistoryPanel.vue` 的表格也更偏桌面布局。
- `frontend/src/components/dashboard/TopBar.vue` 的状态圆点固定为 `bg-destructive`，会让运行中/等待调度也显示危险色。
- `frontend/src/components/dashboard/ConfigSheet.vue` 中多个 `NumberField` 控件缺少稳定 `id` 与 `FieldLabel for` 的关联。
- `frontend/src/components/dashboard/UrlUsagePanel.vue` 未被入口引用。
- `frontend/src/assets/vue.svg`、`frontend/src/assets/vite.svg`、`frontend/src/assets/hero.png` 未被源码入口引用。
- `frontend/src/domain/dashboard.ts` 的 `logLevelClass('success')` 直接返回 `text-emerald-600`，项目规范已标注它是未来语义成功色 token 的候选项。

## Requirements

- R1: 移除无效外部字体请求，避免本地 dashboard 首屏依赖 Google Fonts；字体表现应继续使用系统字体栈或项目已有 token。
- R2: 图表能力应从首屏静态依赖中拆出，生产构建入口不应在初始 HTML 中 preload 图表 chunk。
- R3: 保留桌面端密集表格体验，同时给配置列表、线程列表、调度历史提供窄屏可用布局，避免主要操作只能靠超宽横向滚动完成。
- R4: TopBar 状态指示应按运行状态展示语义视觉，不能固定危险色。
- R5: 配置表单的数字字段和图标按钮应补齐必要的无障碍关联，不降低现有 shadcn-vue 组合规范。
- R6: 清理未使用的业务组件和默认静态资产，减少后续维护误导。
- R7: 将日志成功色收敛为项目语义 token，避免继续扩散 raw Tailwind 颜色。
- R8: 所有改动必须继续遵守现有前端边界：组件只渲染 props/emit 事件，API 仍走 `apiClient`、Vue Query、`useDashboard` 和 `domain/**`。
- R9: UI 文案保持简体中文、直接表达状态和操作，不添加解释式或教学式文案。
- R10: Vite/Rolldown 构建不保留任何手动分包组，chunk 拆分交给动态导入和构建工具自动策略处理。

## Acceptance Criteria

- [ ] `frontend/src/style.css` 不再请求 Google Fonts，生产 CSS 中不再出现 `fonts.googleapis.com`。
- [ ] `npm --prefix frontend run build` 通过。
- [ ] 构建后的 `frontend/dist/index.html` 不再 `modulepreload` 图表 chunk；图表相关代码仍能在 dashboard 渲染图表时加载。
- [ ] `frontend/vite.config.ts` 不再配置 `codeSplitting.groups`、`vue-vendor`、`data-vendor`、`ui-vendor` 或手动 `vendor` 分包。
- [ ] 窄屏下配置列表、线程状态、调度历史提供不依赖超宽表格的主要信息和操作入口；桌面端保留现有表格密度。
- [ ] TopBar 运行中、等待调度、已停止三类状态不再共用危险色圆点。
- [ ] `ConfigSheet` 数字字段具备稳定 label/control 关联；确认弹窗 loading 按钮图标符合本项目 Button/Icon 规则。
- [ ] 未引用的 `UrlUsagePanel.vue` 和默认 Vite/hero 静态资产被移除，且没有残留导入。
- [ ] 成功日志颜色通过 `src/style.css` 语义 token 暴露，并由 domain 投影使用。
- [ ] `git diff --check` 通过。

## Out Of Scope

- 不改 Go API、OpenAPI schema、生成 SDK 或 runtime 行为。
- 不新增前端路由、全局状态库或单元测试框架。
- 不替换 shadcn-vue 基础组件，不手写新的基础 UI primitive。
- 不做完整视觉重设计；这次只处理已识别的优化点。
