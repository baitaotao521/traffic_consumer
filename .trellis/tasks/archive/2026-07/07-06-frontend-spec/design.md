# 编写前端 Trellis 规范 - Design

## Scope And Boundary

This task adds a frontend spec layer under `.trellis/spec/frontend/` for the existing single-repo Trellis layout. It does not introduce package entries in `.trellis/config.yaml`, because the current repository is still a single product with one Go backend and one Vue dashboard rather than a monorepo package graph.

The frontend spec will describe `frontend/` as the Vue/Vite dashboard source and `frontend/dist/` as generated runtime assets served by the Go backend. Product source remains untouched.

## Proposed Spec Files

- `.trellis/spec/frontend/index.md`
  - Entry point, overview, Pre-Development Checklist, Quality Check.
- `.trellis/spec/frontend/directory-structure.md`
  - `src/api/generated` generated boundary, hand-written API wrappers, domain/composable/component ownership, dist assets.
- `.trellis/spec/frontend/api-state-guidelines.md`
  - OpenAPI generation, `ApiError`, Vue Query keys/mutations, dashboard cache updates, activity/log cursor behavior, toast/error handling.
- `.trellis/spec/frontend/ui-component-guidelines.md`
  - shadcn-vue composition, dashboard panel patterns, forms, dialogs, tables, empty states, icons, loading states.
- `.trellis/spec/frontend/styling-and-copy.md`
  - Tailwind v4 theme tokens, `src/style.css`, semantic colors, chart variables, responsive layout constraints, UI copy rules.
- `.trellis/spec/frontend/quality-guidelines.md`
  - Required checks, build contract, generated SDK policy, comment rules, forbidden patterns.

## Evidence Map

- Stack and scripts:
  - `frontend/package.json:6` through `frontend/package.json:11`
  - `frontend/package.json:13` through `frontend/package.json:37`
- shadcn-vue project config:
  - `frontend/components.json:3`
  - `frontend/components.json:6` through `frontend/components.json:21`
- Vite build contract:
  - `frontend/vite.config.ts:29` through `frontend/vite.config.ts:70`
- Tailwind v4 and theme variables:
  - `frontend/src/style.css:4`
  - `frontend/src/style.css:12` through `frontend/src/style.css:49`
  - `frontend/src/style.css:52` through `frontend/src/style.css:119`
- App orchestration:
  - `frontend/src/App.vue:18`
  - `frontend/src/App.vue:60` through `frontend/src/App.vue:168`
- API and cache:
  - `frontend/src/api/client.ts:33` through `frontend/src/api/client.ts:82`
  - `frontend/src/api/queries.ts:14` through `frontend/src/api/queries.ts:69`
- Domain validation and payload projection:
  - `frontend/src/domain/configEditor.ts:26` through `frontend/src/domain/configEditor.ts:38`
  - `frontend/src/domain/configEditor.ts:129` through `frontend/src/domain/configEditor.ts:161`
  - `frontend/src/domain/configEditor.ts:307` through `frontend/src/domain/configEditor.ts:340`
- UI composition:
  - `frontend/src/components/dashboard/ConfigSheet.vue:112` through `frontend/src/components/dashboard/ConfigSheet.vue:220`
  - `frontend/src/components/dashboard/LogHistoryPanel.vue:55` through `frontend/src/components/dashboard/LogHistoryPanel.vue:160`
  - `frontend/src/components/DashboardChart.vue:56` through `frontend/src/components/DashboardChart.vue:167`

## Design Decisions

1. Keep specs source-backed and descriptive, not aspirational.
   The spec should document the existing Vue dashboard architecture and only elevate rules that already come from code, shadcn-vue project configuration, or user-provided AGENTS instructions.

2. Split by developer lookup path.
   Future agents need different files when touching generated API, dashboard state, UI composition, styling, or quality checks. A single giant frontend spec would be searchable but not useful. Tiny artificial files would be annoying. Six files is the narrowest useful split.

3. Treat shadcn-vue as a project contract.
   Because `components.json` exists and `npx shadcn-vue@latest info --json` confirms Vite, Tailwind v4, `reka-nova`, lucide, and `@` alias, the frontend spec should explicitly require shadcn-vue component composition before custom markup.

4. Make shadcn-vue lookup mandatory before UI implementation.
   Frontend UI work must first use the shadcn-vue skill/CLI to inspect project
   config, installed components, and target component docs/search results.
   New business components may compose installed primitives, but new base UI
   primitives must come from shadcn-vue or explicit user approval.

5. Make non-essential component classes forbidden.
   Component props, variants, semantic tokens, and existing primitives are the
   default styling path. `class` on a component is allowed only when it is needed
   for layout, size constraints, overflow handling, accessibility, or a documented
   local composition pattern.

6. Keep UI copy rules in the frontend spec.
   The user's AGENTS rules are directly about user-facing frontend text, so `styling-and-copy.md` and `quality-guidelines.md` should capture them near the UI and review gates.

7. Keep product bug fixes out of this task.
   Any product code issue discovered during evidence gathering belongs in a separate task unless it prevents spec verification. This task's acceptance criteria are docs/spec-only.

## Compatibility

Adding `.trellis/spec/frontend/` makes `get_context.py --mode packages` list both `backend` and `frontend` spec layers in single-repo mode. Existing backend specs and active task routing remain valid.

No migration is required for existing tasks. Future tasks touching `frontend/**` should read `.trellis/spec/frontend/index.md` before implementation.
