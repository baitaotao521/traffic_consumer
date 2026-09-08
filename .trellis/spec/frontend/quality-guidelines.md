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
- Keep Vite/Rolldown bundle splitting automatic. Use dynamic imports such as
  `defineAsyncComponent(() => import(...))` for heavy charts and user-triggered
  dialogs instead of maintaining `output.codeSplitting.groups`.
- Gate async dialog components by their open state (`v-if="dialogOpen"`). An
  async component rendered unconditionally will load immediately, so the split
  will not reduce the first dashboard render.
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
- Reintroducing hand-maintained vendor chunks such as `vue-vendor`,
  `data-vendor`, `ui-vendor`, or a catch-all manual `vendor` group. Prefer
  dynamic-import boundaries and let the bundler choose the generated chunk
  layout.
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

## Scenario: Reproducible Frontend Dependency Installation in Docker

### 1. Scope / Trigger

- Trigger: `frontend/package.json`, `frontend/package-lock.json`, the Node base
  image, or frontend build dependencies change before a Docker release.
- A local build that reuses `frontend/node_modules` is insufficient evidence
  that the Dockerfile's clean install can be reproduced.

### 2. Signatures

The release boundary is the existing Dockerfile command:

```dockerfile
FROM node:22-alpine AS frontend-builder
RUN npm ci
```

The matching validation command is:

```bash
docker buildx build --platform linux/amd64 --load -t traffic_consumer:verify .
```

### 3. Contracts

- `frontend/package.json` and `frontend/package-lock.json` must be accepted by
  the npm version bundled in `node:22-alpine`.
- The lockfile must contain transitive optional packages for the Docker target,
  including packages omitted by a host-specific install.
- Regenerate the lockfile with Node 22/npm 10 in a clean directory when the
  container reports missing lock entries; do not hand-edit integrity data.

### 4. Validation & Error Matrix

- Local `npm run build` passes but Docker `npm ci` reports `EUSAGE` and
  `Missing: <package> from lock file` -> regenerate the lockfile using the
  Dockerfile's Node/npm environment, then retry the clean install and image build.
- Clean `npm ci` fails with a peer-dependency conflict -> resolve the declared
  dependency versions; do not add `--force` to the Dockerfile.
- Docker build passes only with an existing `node_modules` directory -> the
  validation is invalid because `.dockerignore` and `npm ci` require a clean install.

### 5. Good/Base/Bad Cases

- Good: Node 22/npm 10 accepts the lockfile in a clean Linux/AMD64 directory,
  then the complete Docker image builds.
- Base: local TypeScript/Vite build and Docker build both pass without changing
  the lockfile.
- Bad: local build passes from cached dependencies while Docker release fails
  because the lockfile lacks target-specific optional packages.

### 6. Tests Required

- Run `npm --prefix frontend run build` and assert TypeScript and Vite succeed.
- Run a clean Node 22/npm 10 `npm ci` when repairing the lockfile.
- Run the complete `docker buildx build --platform linux/amd64` command and
  assert the frontend builder, Go builder, and final image export all succeed.

### 7. Wrong vs Correct

#### Wrong

```bash
# Reuses whatever is already installed on the developer machine.
npm --prefix frontend run build
# Assume the Docker release will therefore build.
```

#### Correct

```bash
npm --prefix frontend run build
docker buildx build --platform linux/amd64 --load -t traffic_consumer:verify .
```
