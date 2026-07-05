# Implementation Plan

## Checklist

1. Remove external Google Fonts import from `frontend/src/style.css`.
2. Add semantic success color tokens in `style.css` and update `domain/dashboard.ts` success log class.
3. Convert `SpeedPanel.vue` chart usage to async `DashboardChart` with stable `Skeleton` placeholders.
4. Remove all manual Vite/Rolldown `codeSplitting.groups` so bundle splitting is automatic.
5. Lazy-load user-triggered dialog surfaces from `App.vue` so automatic chunking keeps them out of the first render.
6. Add mobile layouts to:
   - `PlanPanel.vue`
   - `ThreadPanel.vue`
   - `LogHistoryPanel.vue` history tab
7. Update `TopBar.vue` status dot semantics.
8. Add stable ids / label associations for `ConfigSheet.vue` number fields and align `ConfirmActionDialog.vue` spinner icon usage.
9. Remove unused source files:
   - `frontend/src/components/dashboard/UrlUsagePanel.vue`
   - `frontend/src/assets/vue.svg`
   - `frontend/src/assets/vite.svg`
   - `frontend/src/assets/hero.png`
10. Run validation:
   - `npm --prefix frontend run build`
   - inspect `frontend/vite.config.ts` for absence of `codeSplitting.groups` and manual vendor groups
   - inspect `frontend/dist/index.html` for absence of chart chunk preload
   - `rg "fonts.googleapis.com|UrlUsagePanel|src/assets/(vue|vite)|hero\\.png" frontend`
   - `git diff --check`

## Risk Points

- `SpeedPanel.vue` async component must preserve prop names and chart dimensions so layout does not jump.
- Dialog async imports must be gated by open state; otherwise Vue will load them immediately and the split will not reduce the first-render entry.
- Removing manual vendor groups will change generated chunk names and cache boundaries; keep only entry/chunk file naming rules, because they do not define split ownership.
- Mobile layouts must not remove desktop table behavior or table empty states.
- `style.css` token additions should use semantic names and avoid scattering raw colors into components.
- `frontend/dist/**` must be produced by the build, not hand-edited.

## Review Gate

Before implementation starts:

- User approves this plan or explicitly asks to proceed.
- Task status is moved from `planning` to `in_progress` with `task.py start`.
