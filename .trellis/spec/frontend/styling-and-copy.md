# Styling And Copy

> Tailwind v4 theme ownership, semantic tokens, chart colors, layout, and UI text.

---

## Overview

The frontend uses Tailwind v4 through `@tailwindcss/vite`. Theme tokens live in
`frontend/src/style.css`, not in scattered component classes.

The dashboard should remain a work-focused operational surface: compact, clear,
and built for scanning repeated runtime state.

---

## Theme Ownership

`frontend/src/style.css` owns:

- Tailwind imports.
- `@custom-variant dark`.
- `@theme inline` token mapping.
- light/dark CSS variables.
- base body styles.
- chart color variables.

Reference: `frontend/src/style.css`.

Do not create a second global CSS theme file. Extend `src/style.css` when a new
semantic token is truly needed.

---

## Tailwind Class Rules

Follow shadcn-vue/Tailwind conventions:

- Do not add `class` to a shadcn-vue component unless the class is necessary for
  layout, size constraints, overflow behavior, accessibility, or a documented
  local composition pattern.
- Do not use `class` to override component color, typography, radius, border,
  shadow, padding, or interaction state when a variant/size prop or semantic
  token exists.
- Use semantic tokens such as `bg-background`, `text-foreground`,
  `text-muted-foreground`, `border-border`, `text-destructive`, and
  `bg-card`.
- Use component variants before custom color or border classes.
- Use `gap-*` for spacing; avoid `space-x-*` and `space-y-*`.
- Use `size-*` when width and height are equal.
- Use `truncate` instead of manually combining overflow/text ellipsis classes.
- Use `cn()` from `frontend/src/lib/utils.ts` for conditional class merging.
- Do not add broad manual `z-index` classes to overlay components.

Raw Tailwind colors are acceptable only when the project already has a specific
local convention and no semantic token exists. For new status colors, prefer a
component variant or add a named CSS variable.

Current exception to notice: `domain/dashboard.ts` maps success log text to
`text-emerald-600`. Do not multiply that pattern casually; it is a candidate
for a future semantic success token if success styling spreads.

Allowed component classes:

- layout constraints: `class="w-[220px] justify-between"` on a combobox trigger
- stable dimensions: `class="h-[200px]"` on `ScrollArea`
- overflow handling: `class="truncate"` on text wrappers
- accessibility-only helpers: `class="sr-only"`

Questionable or forbidden component classes:

- `class="bg-blue-500 text-white"` on `Button`
- `class="rounded-2xl shadow-xl"` on `Card`
- `class="text-lg font-bold"` on form controls
- `class="border-red-500"` instead of validation `data-invalid`/`aria-invalid`

If the needed class changes how the component looks rather than how it fits in
the layout, check the component docs and variants first.

---

## Layout Patterns

The dashboard uses constrained wide content:

- sticky top bar
- `max-w-[1600px]`
- `px-6 py-6`
- `flex flex-col gap-*`
- responsive grids such as `md:grid-cols-2`, `xl:grid-cols-4`, and custom
  `minmax(0, ...)` tracks

Reference files:
- `frontend/src/App.vue`
- `frontend/src/components/dashboard/TopBar.vue`
- `frontend/src/components/dashboard/SpeedPanel.vue`
- `frontend/src/components/dashboard/ConfigSheet.vue`

Use stable dimensions for fixed-format dashboard surfaces: cards, charts,
scroll areas, tables, icon buttons, and loading skeletons should not resize
just because data changes.

Do not turn dashboard sections into marketing-style hero layouts. This app is a
runtime control surface.

---

## Chart Colors

Charts use `--chart-1` through `--chart-5` from `src/style.css`.

`DashboardChart.vue` accepts token names such as `--chart-1` and resolves them
to CSS variables. Keep chart color choices in tokens or component props; do not
hardcode arbitrary color values in Unovis config.

Reference files:
- `frontend/src/style.css`
- `frontend/src/components/DashboardChart.vue`
- `frontend/src/components/dashboard/SpeedPanel.vue`

---

## User-Facing Copy

UI copy must be final text the user sees. It is not a design note.

Rules from the project instructions:

- Do not explain why the UI is designed a certain way inside the UI itself.
- Do not repeat the same information with slightly different wording.
- Do not use teaching-style copy like "if you do X, then Y".
- Present the state and the available action directly.

Good:

```text
删除配置
配置 "demo" 会从当前列表中移除。
```

Bad:

```text
如果你删除这个配置，它将被移除，因为删除功能用于管理配置。
```

Existing examples to follow:

- `useDashboard.ts` de-duplicates toast text by title and description.
- `domain/dashboard.ts` suppresses duplicate status-update success toasts when
  command responses already show the result.
- `ConfirmActionDialog.vue` receives title, description, and action label from
  one composable owner.

---

## Chinese UI Text

The current dashboard UI text is Simplified Chinese for product actions,
runtime state, validation errors, and empty states.

Keep new product-facing strings consistent with that style:

- short action labels: `保存配置`, `停止`, `删除`
- direct state labels: `运行中`, `已停止`, `暂无日志`
- precise validation messages: `URL 必须是有效的 http 或 https 地址。`

Avoid mixing English labels into product UI unless the term is a protocol,
format, or product name such as `Cron`, `URL`, or `User-Agent`.

---

## Accessibility And Text Fit

- Icon-only buttons need `aria-label` or `sr-only` text.
- Dialogs need titles even when the visual layout is compact.
- Use `truncate` plus `HoverCard`/`Tooltip` for long config names, URLs, and log
  messages.
- Keep button labels short enough for mobile and dense table action columns.
- Do not let text overlap previous or following content.

Reference files:
- `frontend/src/components/dashboard/TopBar.vue`
- `frontend/src/components/dashboard/PlanPanel.vue`
- `frontend/src/components/dashboard/ThreadPanel.vue`
- `frontend/src/components/dashboard/LogHistoryPanel.vue`
