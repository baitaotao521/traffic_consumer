# Frontend UI Optimization Design

## Boundaries

The maintained frontend flow stays unchanged:

```text
Go Huma API -> generated SDK -> apiClient -> Vue Query -> useDashboard
  -> domain projections -> dashboard components
```

This task only changes source under `frontend/src/**`, generated `frontend/dist/**` from the build, and the task planning files. API contracts, generated SDK files, and backend code are out of scope.

## Technical Design

### Font Loading

Remove the Google Fonts `@import` from `frontend/src/style.css`. The current compiled CSS already resolves `--font-sans` to the Tailwind/system stack, so removing the import avoids an external request without requiring a replacement font asset.

### Lazy Chart Loading

Keep `DashboardChart.vue` as the single chart wrapper, but stop statically importing it from first-render dashboard panels. `SpeedPanel.vue` will use Vue `defineAsyncComponent` for `DashboardChart.vue` and render stable skeleton placeholders while the chart module loads.

Expected build effect:

- `app.js` no longer statically imports `charts.js`.
- `dist/index.html` no longer preloads the chart chunk.
- Chart rendering remains isolated in `DashboardChart.vue`.

### Automatic Bundle Splitting

Remove all manual `codeSplitting.groups` entries from `frontend/vite.config.ts`. Chunk boundaries should come from Vite/Rolldown's automatic strategy plus the explicit dynamic import for `DashboardChart.vue`, not from hand-maintained `vue-vendor`, `data-vendor`, `ui-vendor`, or `vendor` groups.

Non-first-screen dialogs in `App.vue` should also use async component imports gated by their open state. The configuration editor, plan detail dialog, and confirmation dialog are user-triggered surfaces, so they should not force their shadcn-vue form/dialog dependencies into the first dashboard render.

### Responsive Operational Panels

Desktop layouts should remain dense tables. Narrow screens should get stacked card-like row summaries inside the same product components:

- `PlanPanel.vue`: show a mobile row list with plan name, status, key metrics, schedule, last run, and action buttons; hide the wide table below `md`.
- `ThreadPanel.vue`: show mobile thread cards with status, URL, response time, and progress; keep table for `md+`.
- `LogHistoryPanel.vue`: keep realtime log stream unchanged, but render mobile history rows instead of forcing a table on small screens.

These are product-level compositions, not new base primitives.

### Status Semantics

`TopBar.vue` should compute a small status-dot class from `statusVariant` or an explicit state-like value. Existing `Badge` variant continues to own the label container; the dot should no longer always use `bg-destructive`.

### Accessibility And Component Rules

`ConfigSheet.vue` should assign stable ids to `NumberFieldInput` elements and matching `for` attributes on labels. Icon-only/loader compositions should continue using shadcn-vue `Button`, lucide icons, `Spinner`, `sr-only`, and `data-icon`.

### Cleanup

Remove dead source files only after confirming no imports:

- `frontend/src/components/dashboard/UrlUsagePanel.vue`
- `frontend/src/assets/vue.svg`
- `frontend/src/assets/vite.svg`
- `frontend/src/assets/hero.png`

### Success Color Token

Add a semantic success token in `frontend/src/style.css` and expose it through Tailwind `@theme inline`. Update `domain/dashboard.ts` to return the semantic class for success logs. This keeps the existing domain projection owner while removing raw green styling from the domain layer.

## Trade-Offs

- Lazy loading charts may show a short placeholder before charts appear. This is acceptable because KPIs, controls, plans, threads, and logs remain immediately useful.
- Mobile row summaries duplicate some table template markup. A shared abstraction is not justified yet; the repeated markup is localized and more readable than forcing a generic table/card adapter.
- Removing Google Fonts changes nothing obvious if the font was not actually applied, but it removes a network dependency. If a specific brand font is later required, it should be self-hosted and declared through the theme.

## Rollback

Each change is source-local and can be reverted independently:

- Restore `style.css` import for external font behavior.
- Replace async `DashboardChart` with static import if lazy loading causes runtime issues.
- Replace async dialog imports with static imports if open-state lifecycle bugs appear.
- Restore manual `codeSplitting.groups` only if a measured production regression proves automatic chunking is insufficient.
- Remove mobile-only sections if layout regressions appear.
- Restore removed dead files from git if a hidden consumer is discovered.
