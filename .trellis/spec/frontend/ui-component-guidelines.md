# UI Component Guidelines

> shadcn-vue, dashboard composition, forms, dialogs, tables, icons, and charts.

---

## Overview

The dashboard is an operational UI. It should be dense, scannable, and built
from existing shadcn-vue primitives before custom markup.

The project config is shadcn-vue `reka-nova`, Tailwind v4, lucide icons, and
the `@` alias. Confirm the installed component exists under
`frontend/src/components/ui/**` before importing it.

Reference files:
- `frontend/components.json`
- `frontend/src/components/ui`
- `frontend/src/components/dashboard`

---

## Component Selection

Before writing frontend UI code, query shadcn-vue first. This is mandatory.

Minimum lookup flow from `frontend/`:

```bash
npx shadcn-vue@latest info --json
npx shadcn-vue@latest docs <component>
npx shadcn-vue@latest search @shadcn -q "<need>"
```

Use the `info` result for aliases, icon library, Tailwind file, framework, and
installed component paths. Use `docs` for the target component API. Use `search`
when no installed component obviously fits.

Do not guess component APIs from memory. shadcn-vue changes, and stale memory is
how a UI gets an imaginary prop. Frontend UI changes without this lookup should
be rejected in review.

Use the existing UI primitives first:

- Actions: `Button`
- Status: `Badge`
- Panels: `Card`
- Forms: `Field`, `FieldGroup`, `Input`, `Textarea`, `NumberField`, `Select`,
  `Switch`, `ToggleGroup`
- Tables: `Table`, `TableHeader`, `TableBody`, `TableEmpty`
- Empty states: `Empty`
- Overlays: `Dialog`, `AlertDialog`, `Sheet`, `Popover`, `HoverCard`,
  `Tooltip`
- Loading: `Skeleton`, `Spinner`
- Scrolling: `ScrollArea`
- Tabs: `Tabs`, `TabsList`, `TabsTrigger`, `TabsContent`
- Charts: `DashboardChart` and `components/ui/chart`

Bad pattern: styled `div` blocks that recreate `Alert`, `Empty`, `Badge`,
`Card`, `Table`, or `Skeleton`.

---

## No Hand-Written Base UI Components

Do not hand-write base UI components. New business components may compose
installed shadcn-vue primitives, but they must not reinvent primitive behavior,
styling, accessibility, or interaction.

Allowed:

- A product panel such as `SpeedPanel.vue` that composes `Card`, `Button`,
  `Badge`, `Tooltip`, and `DashboardChart`.
- A product dialog that composes `Dialog`, `Field`, `Input`, `NumberField`,
  `Select`, `Table`, and `Button`.
- A thin wrapper around a domain-specific library only when the project already
  has that pattern, such as `DashboardChart.vue` wrapping Unovis.

Forbidden:

- A custom button, select, switch, tab, dialog, dropdown, table, badge, alert,
  empty state, skeleton, tooltip, or form field built from raw HTML and Tailwind.
- A local "mini component" that duplicates an installed `components/ui/**`
  primitive because adding the correct primitive feels slower.
- Copying registry component code by hand instead of using the shadcn-vue CLI.

When no shadcn-vue component fits, stop and ask for approval before custom UI.
The usual path is to add an official/community registry component with the CLI,
then review and adapt imports.

---

## Dashboard Panels

`App.vue` is the composition root. Product panels under
`frontend/src/components/dashboard/**` should stay focused on rendering props
and emitting events.

Local pattern:

- `App.vue` gets state/actions from `useDashboard`.
- Panels receive computed values and callbacks through props/events.
- Panels do not import `apiClient`, Vue Query hooks, or generated SDK types
  unless the panel is a deliberately narrow adapter.

Examples:
- `PlanPanel.vue` renders plan rows and emits `startPlan`, `stopPlan`,
  `deletePlan`, `editPlan`, and `openPlanDetail`.
- `SpeedPanel.vue` receives chart labels/values and emits `refresh`.
- `ThreadPanel.vue` receives projected `ThreadRow[]` and `ThreadCounts`.

---

## Forms

Use shadcn-vue form composition:

- Wrap related controls in `FieldGroup`.
- Use `Field` for each form field.
- Use `FieldLabel`, `FieldDescription`, and `FieldError` instead of loose label
  markup.
- Put `data-invalid` on `Field`.
- Put `aria-invalid` on the actual control.
- Use `NumberField` for numeric values.
- Use `SelectGroup` around `SelectItem`.
- Use `Switch` for boolean settings.
- Use `ToggleGroup` for 2-7 option sets.

Reference: `frontend/src/components/dashboard/ConfigSheet.vue`.

The URL table in `ConfigSheet.vue` is only an editing affordance. It writes
back to `form.urls`; `projectConfigForm` remains the owner for URL parsing,
duplicate preservation, and empty-list behavior.

Bad:

```vue
<Input type="number" v-model="threads" />
```

Good:

```vue
<NumberField v-model="form.threads">
  <NumberFieldContent>
    <NumberFieldDecrement />
    <NumberFieldInput />
    <NumberFieldIncrement />
  </NumberFieldContent>
</NumberField>
```

---

## Dialogs And Confirmations

Use `Dialog` for focused editing and details, and `AlertDialog` for destructive
confirmation.

Required rules:

- `DialogContent` must include `DialogTitle`.
- `AlertDialogContent` must include `AlertDialogTitle` and
  `AlertDialogDescription`.
- Async command confirmations should stay controlled by `useDashboard`, not by
  primitives that close before the mutation finishes.
- Keep the global `ConfirmActionDialog` rendered after primary dashboard
  dialogs in `App.vue` so overlay source order stays predictable.

Examples:
- `ConfigSheet.vue` uses `Dialog` for config editing.
- `PlanDetailDialog.vue` uses `Dialog` for plan details.
- `ConfirmActionDialog.vue` uses `AlertDialog` for delete/stop/save
  confirmation.

---

## Tables And Empty States

Use `Table` for dense operational rows. Use `TableEmpty` with `Empty` when the
table has no rows.

Reference examples:
- `PlanPanel.vue`
- `ThreadPanel.vue`
- `LogHistoryPanel.vue`
- `PlanDetailDialog.vue`

For long URLs, config names, or log messages:

- Render a truncated cell for the table.
- Use `HoverCard` or `Tooltip` to expose the full value.
- Keep IDs or numeric columns aligned with fixed-width classes when the table
  needs stable scanning.

---

## Icons And Buttons

Use lucide icons from `@lucide/vue`, matching `components.json`.

Rules:

- Icons inside `Button` use `data-icon="inline-start"` or
  `data-icon="inline-end"`.
- Use built-in button `variant` and `size` props before custom classes.
- Loading buttons compose `Spinner` plus `disabled`; `Button` has no
  `isLoading` prop.
- Icon-only buttons need an accessible label, usually an `aria-label` or
  `sr-only` text.

Examples:
- `TopBar.vue`
- `ConfigSheet.vue`
- `SpeedPanel.vue`
- `ConfirmActionDialog.vue`

---

## Loading States

Use `Skeleton` for initial dashboard loading and `Spinner` for command-level
loading.

Current pattern:

- `App.vue` renders a full dashboard skeleton while the first dashboard query
  is pending.
- Save/test/confirm buttons render `Spinner` while mutations are pending.
- Refresh buttons switch between `Spinner` and the action icon.

Avoid custom `animate-pulse` blocks.

---

## Charts

Use `DashboardChart.vue` for dashboard charts. It wraps Unovis and the local
chart primitives, maps chart colors to CSS variables, and owns empty-data
behavior.

Rules:

- Pass stable `labels` and `values`.
- Use `--chart-*` tokens for chart colors.
- Put value formatting in the panel or chart wrapper, not in raw template
  expressions repeated across panels.
- Prefer the existing wrapper before creating another third-party chart
  lifecycle.

Reference files:
- `frontend/src/components/DashboardChart.vue`
- `frontend/src/components/dashboard/SpeedPanel.vue`
- `frontend/src/style.css`

---

## shadcn-vue CLI Use

When adding or updating UI primitives, use the project package runner from
`frontend/`:

```bash
cd frontend
npx shadcn-vue@latest info --json
npx shadcn-vue@latest docs button dialog select
npx shadcn-vue@latest add <component>
```

Before adding a component, check whether it already exists under
`frontend/src/components/ui`. After adding a registry component, read the added
files and fix imports to match this project's aliases and icon library.

Adding or updating a primitive without this lookup/review step is not an
acceptable frontend change.
