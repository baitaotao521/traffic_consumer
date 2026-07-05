# Frontend Development Guidelines

> Project-specific frontend rules for Traffic Consumer.

---

## Overview

Traffic Consumer's maintained frontend is the Vue 3 / Vite dashboard under
`frontend/`. It is built into `frontend/dist/`, and the Go HTTP service serves
that compiled bundle at runtime.

The dashboard uses TypeScript, Tailwind v4, shadcn-vue with `reka-nova`, lucide
icons, Vue Query, and Unovis charts. New frontend work should preserve the
existing API and state boundaries instead of letting components hand-roll HTTP
calls, payload parsing, or config validation.

---

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | Frontend source layout, generated SDK boundary, dashboard components, and dist assets | Filled |
| [API And State Guidelines](./api-state-guidelines.md) | OpenAPI generation, API client wrapper, Vue Query cache updates, activity cursors, and toasts | Filled |
| [UI Component Guidelines](./ui-component-guidelines.md) | shadcn-vue composition, forms, dialogs, tables, empty states, icons, and loading states | Filled |
| [Styling And Copy](./styling-and-copy.md) | Tailwind v4 theme ownership, semantic tokens, chart colors, layout, and user-facing copy rules | Filled |
| [Quality Guidelines](./quality-guidelines.md) | Build checks, generated files, comments, review checklist, and forbidden frontend patterns | Filled |

---

## Pre-Development Checklist

- Read [Directory Structure](./directory-structure.md) before adding frontend
  files, generated API output, dashboard panels, or build assets.
- Read [API And State Guidelines](./api-state-guidelines.md) before changing
  OpenAPI generation, `frontend/src/api/**`, Vue Query keys, dashboard polling,
  activity/log handling, command mutations, or config validation flow.
- Read [UI Component Guidelines](./ui-component-guidelines.md) before changing
  `frontend/src/components/**`, dialogs, sheets, forms, tables, charts, loading
  UI, icons, or shadcn-vue components.
- Before any frontend UI implementation, use the shadcn-vue skill/CLI to inspect
  project config, installed components, and target component docs/search results.
  At minimum run from `frontend/`:
  `npx shadcn-vue@latest info --json` and
  `npx shadcn-vue@latest docs <component>`.
- Read [Styling And Copy](./styling-and-copy.md) before changing
  `frontend/src/style.css`, Tailwind classes, chart colors, responsive layout,
  or user-facing text.
- Read [Quality Guidelines](./quality-guidelines.md) before committing any
  frontend change.
- For cross-layer API work, also read
  [Backend Web API Contracts](../backend/web-api-contracts.md).

## Quality Check

- Run `npm --prefix frontend run build` for frontend source, theme, Vite config,
  generated SDK, or `frontend/dist` changes.
- Run `npm --prefix frontend run api:generate` when Go Web API signatures,
  payloads, or OpenAPI schema change, then review the generated SDK diff.
- Run `git diff --check` before finishing a frontend task.
- Run `go test ./...` as well when the frontend change depends on Go API,
  runtime, storage, or model behavior.

---

## Maintained Runtime Boundary

Primary dashboard flow:

```text
Go Huma API -> frontend/src/api/generated -> apiClient -> Vue Query hooks
  -> useDashboard -> domain projections -> dashboard components
```

Components should render projected state and emit typed events. They should not
parse raw backend payloads, call old routes, or rebuild config serializers. Yes,
even for "one quick field"; that is exactly how tiny UI edits grow extra
contracts nobody audits later.

Frontend UI work has an additional hard gate: no hand-written base UI component
and no casual `class` overrides on components. Query shadcn-vue first, compose
installed primitives, then add only the layout classes the composition truly
needs.
