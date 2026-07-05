# Quality Guidelines

> Required standards for frontend changes.

---

## Overview

The maintained frontend quality gate is:

```bash
npm --prefix frontend run build
```

This runs `vue-tsc -b` and `vite build`, so it checks TypeScript and production
bundle output together.

For Web API contract changes, regenerate first:

```bash
npm --prefix frontend run api:generate
npm --prefix frontend run build
```

---

## Required Patterns

- Keep generated API files generated. Do not hand-edit
  `frontend/src/api/generated/**` or `frontend/src/api/openapi.json`.
- Route frontend API calls through `apiClient` and Vue Query hooks.
- Keep config validation and payload projection in `domain/configEditor.ts`.
- Keep dashboard payload normalization and display projections in
  `domain/dashboard.ts`.
- Keep runtime orchestration in `useDashboard.ts`; panels should render props
  and emit events.
- Use shadcn-vue primitives and installed `components/ui/**` files before
  custom markup.
- Before writing UI code, use the shadcn-vue skill/CLI to inspect project
  config, installed components, and the target component docs/search results.
- Do not hand-write base UI components. Business components may compose
  shadcn-vue primitives; primitives themselves must come from shadcn-vue or
  explicit user-approved custom work.
- Do not add non-essential `class` attributes to shadcn-vue components.
  Variants, sizes, semantic tokens, and existing component APIs are the first
  styling path; `class` is for necessary layout/size/overflow/accessibility.
- Use `Field` / `FieldGroup` / `FieldError` for forms and validation state.
- Use `NumberField` for numeric models.
- Use `TableEmpty` plus `Empty` for table empty states.
- Use `Skeleton` and `Spinner` for loading states.
- Use lucide icons from `@lucide/vue`, with `data-icon` inside buttons.
- Use semantic theme tokens from `src/style.css`.
- Add business comments near non-obvious state flow, compatibility behavior,
  async confirmation timing, cursor handling, and validation semantics.

Existing business-comment examples:

- `ConfigSheet.vue` documents that URL table rows are an editor affordance while
  `projectConfigForm` owns parsing and validation.
- `useDashboard.ts` documents why async confirmation actions are retained after
  failed mutations.
- `useDashboard.ts` documents why realtime logs restart from the current stream
  position.
- `DashboardChart.vue` documents why line chart hover uses the nearest sampling
  point.

---

## Comment Rules

Comments should describe the business rule, trigger, and consequence near the
implementation.

Good:

```typescript
// 实时日志重新开启后从当前时刻开始接收，不补拉关闭期间的旧日志。
```

Bad:

```typescript
// Handle data.
```

When changing a state transition, validation rule, or compatibility behavior,
update the nearby comment in the same edit. A stale comment is worse than no
comment because it lies with confidence, which is rude and technically ugly.

---

## Forbidden Patterns

- Calling backend routes with raw `fetch` in components.
- Importing generated SDK functions directly into dashboard panels.
- Rebuilding config serializers outside `domain/configEditor.ts`.
- Parsing raw activity/log payloads in multiple consumers.
- Using `Input type="number"` for config numeric fields.
- Creating custom styled empty states, alerts, badges, skeletons, or tables when
  a shadcn-vue primitive exists.
- Creating custom base UI primitives instead of querying shadcn-vue and using
  or adding the matching primitive.
- Styling a component by piling on classes before checking its docs, variants,
  size props, or semantic tokens.
- Using manual button loading props that the local `Button` component does not
  provide.
- Adding icon sizing classes inside shadcn-vue buttons.
- Adding broad `dark:` color overrides instead of semantic tokens.
- Hiding all Vite/Rolldown warnings to quiet one known warning.
- Raising chunk-size thresholds instead of splitting a genuinely oversized
  bundle.
- Hand-editing `frontend/dist/**`.

---

## Testing Requirements

Match checks to the blast radius:

- Frontend source, style, Vite config, generated SDK, or dist changes:
  `npm --prefix frontend run build`.
- Go API contract changes consumed by frontend:
  `npm --prefix frontend run api:generate`, `npm --prefix frontend run build`,
  and `go test ./...`.
- Vite build warning or chunking changes: verify the build output no longer
  contains the specific warning being addressed.
- Config editor behavior: cover save/start/temp-test projection through
  `domain/configEditor.ts`; do not test only the component shell.
- Activity/log behavior: verify cursor advancement, log-enable behavior, filter
  normalization, and toast de-duplication.
- Dialog confirmation behavior: verify pending state, success close, and
  failure retry/cancel path.

There is no dedicated frontend unit-test runner configured yet. Do not pretend
one exists. Use the production build as the baseline gate until a future task
adds focused frontend tests.

---

## Code Review Checklist

Before considering a frontend change done:

- The touched path is under `frontend/` or `.trellis/spec/frontend/`, unless
  the task explicitly includes cross-layer backend work.
- Generated SDK changes came from `npm --prefix frontend run api:generate`.
- `apiClient` remains the only hand-written backend gateway.
- Vue Query still owns server state.
- Components do not duplicate domain validation or raw payload parsing.
- shadcn-vue composition rules are followed.
- The frontend UI work records or clearly reflects that shadcn-vue usage was
  checked before implementation.
- No hand-written base UI component was introduced.
- Component `class` usage is necessary and limited to layout, dimensions,
  overflow, accessibility, or documented local composition.
- UI text is final user-facing copy and not repeated.
- Business comments near modified non-obvious behavior are still accurate.
- `npm --prefix frontend run build` has passed when frontend code or static
  assets changed.
- `git diff --check` passes.

---

## Common Mistakes

- Updating Go API payloads and forgetting to regenerate the frontend SDK.
- Updating generated SDK output and forgetting to adapt `apiClient`.
- Returning a command response with a fresh dashboard snapshot but then
  invalidating/refetching instead of setting the Vue Query cache.
- Letting components import `apiClient` for one special case.
- Adding a new status string in one panel while `domain/dashboard.ts` already
  owns status labels and variants.
- Showing the same backend message once as a command toast and again as an
  activity toast.
- Creating a custom mini form layout instead of using `Field` components.
- Making `frontend/dist` diffs without checking whether source changes and a
  build produced them.
