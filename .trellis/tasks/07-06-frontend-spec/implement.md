# 编写前端 Trellis 规范 - Implementation Plan

## Checklist

1. Create `.trellis/spec/frontend/index.md`.
   - Mirror the backend spec index style.
   - Include overview, guide table, Pre-Development Checklist, and Quality Check.

2. Create `.trellis/spec/frontend/directory-structure.md`.
   - Document `frontend/`, `src/api/generated`, `src/api`, `src/domain`, `src/composables`, `src/components/dashboard`, `src/components/ui`, `src/style.css`, `frontend/dist`.
   - Call out generated code and dist assets as non-hand-edit targets.

3. Create `.trellis/spec/frontend/api-state-guidelines.md`.
   - Document generated SDK wrapping through `apiClient`.
   - Document Vue Query query keys, command mutation cache updates, dashboard orchestration, activity/log cursor rules, and toast dedupe.

4. Create `.trellis/spec/frontend/ui-component-guidelines.md`.
   - Document mandatory shadcn-vue skill/CLI lookup before UI work.
   - Document the ban on hand-written base UI components.
   - Document installed component checks, composition patterns, form validation attributes, dialogs/sheets, tables, empty states, icons, and loading states.

5. Create `.trellis/spec/frontend/styling-and-copy.md`.
   - Document Tailwind v4 theme ownership in `src/style.css`.
   - Document semantic token usage, chart variables, responsive dashboard layout, `cn()`, the ban on non-essential component `class`, and user-facing copy rules.

6. Create `.trellis/spec/frontend/quality-guidelines.md`.
   - Document build and generation commands.
   - Document comment expectations from AGENTS.
   - Document forbidden patterns and review checklist.

7. Verify Trellis discovery and placeholder cleanup.
   - `python3 ./.trellis/scripts/get_context.py --mode packages`
   - `grep -R "TBD\\|TODO: fill\\|placeholder" .trellis/spec/frontend`
   - `git diff --check`

8. Run frontend quality command if spec changes still warrant a full gate.
   - `npm --prefix frontend run build`
   - If build fails because of pre-existing product code, report the failure without fixing product source in this task.

## Rollback Points

- If the spec split feels too granular after drafting, merge topic files before task completion and update `index.md`.
- If `get_context.py --mode packages` does not discover `.trellis/spec/frontend/index.md`, inspect `.trellis/scripts/common/packages_context.py` before changing Trellis config.
- If frontend build fails on existing product code, keep spec changes and record the failure as out of scope unless the user expands scope.

## Review Gate

Do not run `python3 ./.trellis/scripts/task.py start` until the user approves this planning set or explicitly asks to proceed with implementation.
