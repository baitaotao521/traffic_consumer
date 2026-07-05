# Directory Structure

> Frontend organization and ownership boundaries for Traffic Consumer.

---

## Overview

The frontend is a Vite SPA in `frontend/`. It is intentionally separate from
the Go source tree while still sharing the Go Web API contract through OpenAPI
generation.

`frontend/dist/` is compiled output. Treat it like a release artifact: build it
from source, do not hand-edit it.

---

## Directory Layout

```text
frontend/
  package.json                  npm scripts and frontend dependency contract
  components.json               shadcn-vue config: reka-nova, lucide, @ alias
  vite.config.ts                Vite/Tailwind plugin setup and bundle names
  tailwind.config.ts            Legacy Tailwind config shell; v4 theme lives in CSS
  src/
    main.ts                     Vue app mount and Vue Query provider
    App.vue                     Dashboard composition root
    style.css                   Tailwind v4 imports, theme tokens, base layer
    api/
      openapi.json              Generated schema snapshot
      generated/                Generated Hey API SDK
      client.ts                 Hand-written generated-SDK wrapper and ApiError
      queries.ts                Vue Query keys, polling hooks, command mutations
    domain/
      dashboard.ts              Dashboard payload normalization and view projection
      configEditor.ts           Config editor form validation and payload projection
    composables/
      useDashboard.ts           Dashboard orchestration, command flow, activity stream
    components/
      DashboardChart.vue        Shared Unovis chart wrapper
      dashboard/                Product dashboard panels and dialogs
      ui/                       shadcn-vue component source
    types.ts                    UI-only aliases and view model types
  dist/                         Compiled dashboard bundle served by Go
```

Reference files:
- `frontend/package.json`
- `frontend/components.json`
- `frontend/src/App.vue`
- `frontend/src/api/client.ts`
- `frontend/src/api/queries.ts`
- `frontend/src/domain/configEditor.ts`
- `frontend/src/composables/useDashboard.ts`
- `frontend/vite.config.ts`

---

## Ownership Rules

### Generated API Files

`frontend/src/api/generated/**` and `frontend/src/api/openapi.json` are generated
from the Go API:

```bash
npm --prefix frontend run api:generate
```

Do not patch generated files by hand. Fix the Go schema or the hand-written
frontend adapter, then regenerate.

The hand-written API boundary is:

- `frontend/src/api/client.ts` for wrapping generated SDK calls and converting
  errors into `ApiError`.
- `frontend/src/api/queries.ts` for Vue Query keys, polling hooks, cache writes,
  and command mutations.

### Domain Projection

Put payload normalization and business projection in `frontend/src/domain/**`,
not directly inside components.

Current examples:

- `domain/dashboard.ts` normalizes dashboard snapshots, status labels, thread
  rows, plan labels, log source labels, and activity-event toast projection.
- `domain/configEditor.ts` owns config form parsing, save/start/temp-test
  validation, URL/header parsing, optional numeric semantics, duplicate-name
  helpers, and dirty-form snapshots.

If a rule has to be shared by save, start, temporary test, or multiple panels,
it belongs in `domain/**` first.

### Composables

`frontend/src/composables/useDashboard.ts` owns dashboard state orchestration:

- selected config and loaded config detail
- log/event cursors
- editor and confirmation dialog state
- Vue Query mutations
- toast de-duplication
- activity stream consumption
- cron preview debounce

Components may call emitted actions exposed by `useDashboard`, but they should
not duplicate orchestration state locally.

### Components

Use these component boundaries:

- `frontend/src/App.vue` wires the page shell, providers, panels, dialogs, and
  top-level loading state.
- `frontend/src/components/dashboard/**` contains product panels and dialogs.
- `frontend/src/components/DashboardChart.vue` wraps Unovis and the local chart
  component primitives.
- `frontend/src/components/ui/**` contains shadcn-vue source components.

Do not put product business rules into `components/ui/**`. Those files are UI
primitives and should stay reusable.

### Static Build Output

`frontend/dist/` must be produced by:

```bash
npm --prefix frontend run build
```

The Go release path packages this directory beside the executable. Hand-edited
dist assets are brittle because the next build overwrites them.

---

## Examples To Follow

- `frontend/src/api/client.ts` centralizes generated SDK unwrapping and error
  conversion instead of making components parse Huma Problem Details.
- `frontend/src/api/queries.ts` centralizes dashboard polling and command cache
  writes.
- `frontend/src/domain/configEditor.ts` keeps config validation and payload
  projection in one path for save, start, and temporary test.
- `frontend/src/components/dashboard/ConfigSheet.vue` renders editor controls
  while leaving payload validation in `domain/configEditor.ts`.
- `frontend/src/components/dashboard/LogHistoryPanel.vue` uses shadcn-vue
  `Tabs`, `ToggleGroup`, `Table`, `Empty`, and `ScrollArea` primitives for a
  dense operational panel.

---

## Common Mistakes

- Editing `frontend/src/api/generated/**` directly.
- Adding `fetch('/api/...')` inside a component.
- Recreating config validation inside a Vue component.
- Putting product dashboard behavior into `components/ui/**`.
- Changing `frontend/dist/**` without rebuilding from source.
- Adding a second chart wrapper when `DashboardChart.vue` already owns the
  chart lifecycle and theme-token mapping.
