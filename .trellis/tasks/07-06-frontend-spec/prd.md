# 编写前端 Trellis 规范

## Goal

为 Traffic Consumer 的 Vue/Vite dashboard 编写项目级前端 Trellis 规范，让后续前端实现、检查和评审能按真实代码结构、shadcn-vue 组件约束、API 状态流和 UI 文案规则执行。

## Confirmed Facts

- 当前 Trellis 规范只有 `backend` 层，`python3 ./.trellis/scripts/get_context.py --mode packages` 只列出 `Spec layers: backend`。
- 前端是单独的 `frontend/` Vite 应用，使用 Vue 3、TypeScript、Tailwind v4、shadcn-vue/reka-nova、Vue Query、Unovis 和 lucide。证据：`frontend/package.json:6`, `frontend/package.json:13`, `frontend/package.json:27`。
- shadcn-vue 配置使用 `src/style.css`、`@` alias、`@/components/ui`、`@/lib/utils`、`@/composables` 和 lucide icon library。证据：`frontend/components.json:3`, `frontend/components.json:6`, `frontend/components.json:13`, `frontend/components.json:16`。
- 页面组合入口是 `frontend/src/App.vue`，它把 `useDashboard()` 暴露的状态和动作传给 dashboard panels、dialogs、toast、tooltip provider。证据：`frontend/src/App.vue:18`, `frontend/src/App.vue:60`, `frontend/src/App.vue:126`, `frontend/src/App.vue:137`。
- OpenAPI SDK 由 `npm run api:generate` 生成到 `frontend/src/api/generated`；手写 API 边界是 `frontend/src/api/client.ts`，Vue Query 边界是 `frontend/src/api/queries.ts`。证据：`frontend/package.json:8`, `frontend/src/api/client.ts:1`, `frontend/src/api/queries.ts:21`。
- 表单和业务投影集中在 `frontend/src/domain/configEditor.ts`，包括保存、启动和临时测试三种校验模式。证据：`frontend/src/domain/configEditor.ts:3`, `frontend/src/domain/configEditor.ts:26`, `frontend/src/domain/configEditor.ts:307`。
- dashboard orchestration 集中在 `frontend/src/composables/useDashboard.ts`，负责 toast 去重、确认弹窗、配置加载、运行态命令、活动流消费和日志游标。证据：`frontend/src/composables/useDashboard.ts:69`, `frontend/src/composables/useDashboard.ts:182`, `frontend/src/composables/useDashboard.ts:305`, `frontend/src/composables/useDashboard.ts:679`。
- shadcn-vue 组件组合已经在 `ConfigSheet`、`LogHistoryPanel`、`DashboardChart` 等文件中形成项目模式：`FieldGroup`/`Field`、`Table`/`Empty`、`TabsList`/`TabsTrigger`、`ChartContainer`、semantic CSS variables。证据：`frontend/src/components/dashboard/ConfigSheet.vue:138`, `frontend/src/components/dashboard/ConfigSheet.vue:175`, `frontend/src/components/dashboard/LogHistoryPanel.vue:55`, `frontend/src/components/dashboard/LogHistoryPanel.vue:89`, `frontend/src/components/DashboardChart.vue:56`, `frontend/src/style.css:12`。
- 用户提供的 UI 文案规则要求界面文字只呈现最终用户可见内容，不解释设计原因，不重复同一信息，不使用“如果你...就...”式教导语气。
- 用户提供的注释原则要求业务规则、状态流转和异常分支的注释贴近实现，避免空泛注释，并在修改逻辑时同步更新。
- 用户确认前端编写最重要的硬规则：写 UI 前必须先使用 shadcn-vue skill/CLI 查询组件用法，禁止手写基础 UI 组件，禁止给组件添加非必要 `class`。
- 本任务实现阶段已查询 shadcn-vue 组件文档：`npx shadcn-vue@latest docs button card dialog field table toggle-group`。

## Acceptance Criteria

- [x] 新增 `.trellis/spec/frontend/`，并包含可从 `get_context.py --mode packages` 发现的 `index.md`。
- [x] 前端规范覆盖目录边界、API/状态流、UI 组件组合、样式主题、构建质量和文案/注释规则。
- [x] 每条关键规则都引用真实源码、配置或项目约束，不留下模板占位内容。
- [x] `index.md` 的 Pre-Development Checklist 指向正确的前端规范文件。
- [x] `index.md` 的 Quality Check 包含本项目可靠命令：`npm --prefix frontend run build`，以及 API 合同变化时的 `npm --prefix frontend run api:generate`。
- [x] 规范明确 `frontend/src/api/generated/**` 是生成代码，手写代码通过 `api/client.ts` 和 `api/queries.ts` 访问后端。
- [x] 规范明确 shadcn-vue 组件优先、semantic tokens、lucide `data-icon`、`Field`/`FieldGroup`、`ToggleGroup`、`TableEmpty`/`Empty` 等规则。
- [x] 规范明确前端 UI 编写前必须先调用 shadcn-vue skill/CLI 查询组件用法，至少确认项目配置、已安装组件和目标组件 docs/search 结果。
- [x] 规范明确禁止手写基础 UI 组件；业务组件只能组合已有 shadcn-vue primitives，缺组件时先查询/添加/征求确认。
- [x] 规范明确禁止给组件添加非必要 `class`；组件视觉样式优先使用 variant/size/semantic token，`class` 只保留布局、尺寸、溢出和可访问性必要场景。
- [x] 规范明确 UI 文案去重和最终用户文案规则，并以当前 toast/activity 去重模式作证据。
- [x] 规范不修改产品前端源码，不修复业务 bug，不重写现有组件。
- [x] `grep -R "TBD\\|TODO: fill\\|placeholder" .trellis/spec/frontend` 无命中。
- [x] `git diff --check` 通过。

## Out Of Scope

- 修复或重构 `frontend/src/**` 产品代码。
- 新增前端测试框架、lint 工具或依赖。
- 重新组织 Trellis 为 monorepo packages。
- 修改 backend 规范，除非发现前端 index 链接必须和 backend 交叉引用。

## Review Notes

- 规划阶段发现 `frontend/src/composables/useDashboard.ts` 的历史输出曾出现重复对象属性风险；本任务只写规范，不把该点纳入产品代码修复。
- Codex 当前为 inline 模式，Phase 2 前不需要维护 `implement.jsonl` / `check.jsonl` 的 sub-agent 注入清单。
