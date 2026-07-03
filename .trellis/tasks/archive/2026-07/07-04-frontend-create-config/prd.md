# 前端新增配置入口

## Goal

在 Vue Web 控制台中提供明确的新建配置入口，让用户可以从当前页面创建一份新的任务配置，而不是只能通过现有“配置”按钮编辑当前选中配置。

## Background

- Web 控制台主运行路径是 Go HTTP 服务加载 `templates/index.html`，模板引用 `static/app/assets/app.js` 和 `static/app/assets/app.css`。
- 前端源码位于 `frontend/src`，Vite 构建输出到 `static/app`；前端变更需要通过 `npm --prefix frontend run build` 更新静态产物。
- 当前配置编辑弹窗由 `frontend/src/components/dashboard/ConfigSheet.vue` 提供，包含配置名称、URL、线程、执行限制、调度、请求头和运行行为字段。
- 顶部栏 `frontend/src/components/dashboard/TopBar.vue` 原本只有“配置”按钮；它触发 `useDashboard.openEditor()`。
- `useDashboard.openEditor()` 在有选中配置时会把当前配置填入表单，只有没有任何配置时才打开空表单。
- 保存逻辑已经存在：`useDashboard.saveEditorConfig()` 通过 `PUT /api/configs/{name}` 保存配置，保存后刷新配置列表并选中新配置。
- 后端已提供 `GET /api/configs`、`GET /api/configs/{name}`、`PUT /api/configs/{name}`、`DELETE /api/configs/{name}`，本需求不需要新增后端接口。
- 用户确认新建配置采用空白表单，不从当前选中配置复制字段。
- 测试约束要求前端源码和构建产物不能引入旧硬编码色值、默认色系或重复声明组件默认样式。

## Requirements

- R1: 在顶部操作区增加一个明确的新建配置入口，与现有编辑当前配置入口区分。
- R2: 点击新建配置入口时打开配置弹窗，并展示空白新建表单；新建配置不从当前选中配置复制字段。
- R3: 新建态保存时必须使用表单中的配置名称创建/保存配置，并在保存成功后选中新配置、刷新配置列表。
- R4: 编辑当前配置入口继续保留现有行为：有选中配置时预填当前配置，没有配置时可打开空表单。
- R5: 新增入口和弹窗文案必须是最终用户界面文字，不在界面中解释设计原因或重复表达同一信息。
- R6: 前端变更优先复用现有 `ConfigSheet`、`useDashboard` 表单转换和 `saveConfig` API，不新增重复配置模型。
- R7: 构建后的 `static/app` 静态资源必须与源码变更同步。

## Acceptance Criteria

- [x] 顶部栏存在可点击的新建配置入口。
- [x] 存在已选中配置时，点击新建入口打开空白表单，不带入当前配置名称、URL 或其他配置值。
- [x] 点击现有配置入口仍打开当前选中配置的编辑表单。
- [x] 新建配置保存成功后，配置列表刷新并选中新创建的配置。
- [x] 保存未填写配置名称的新建表单时，继续展示现有“请填写配置名称。”提示。
- [x] `npm --prefix frontend run build` 通过，并更新 `static/app` 构建产物。
- [x] 相关前端主题测试不因新增入口引入禁用颜色或重复默认样式。

## Out Of Scope

- 新增或修改后端配置存储接口。
- 调整配置字段含义、校验规则或运行调度行为。
- 复制配置、配置模板、批量导入导出。
