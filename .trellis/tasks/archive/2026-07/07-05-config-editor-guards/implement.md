# Config Editor Guard Implementation Plan

## Preconditions

- Before editing implementation files, load `trellis-before-dev`.
- Inline Codex mode is active, so `implement.jsonl` / `check.jsonl` curation is skipped.
- Do not start implementation until this plan is reviewed and the task is moved to `in_progress` with `task.py start`.

## Checklist

1. Backend validation foundation
   - Add shared Web config validation helpers with mode-specific rules for save, start, and temp-test.
   - Validate names, URLs, headers, schedules, numeric bounds, and User-Agent conflicts.
   - Add focused Go tests for valid save-empty-URLs and invalid start/test-empty-URLs.

2. Backend save/rename/overwrite semantics
   - Extend `PUT /api/configurations/{name}` with explicit force query handling.
   - Add runtime save method that accepts source name, target name, config, and force.
   - Add storage locked mutation helper for rename/overwrite in one config-file mutation.
   - Map validation failures to `400`, overwrite conflicts to `409`, missing source cases as appropriate, and storage failures to `500`.
   - Add tests for create conflict, force overwrite, rename to unused, rename conflict, force rename overwrite, and runtime stop isolation.

3. Backend start/temp-test validation
   - Apply start-mode validation before `Manager.StartConsumer`.
   - Apply temp-test validation before `Manager.StartTempTest`.
   - Keep existing runtime conflict messages for active/scheduled temp-test constraints.
   - Add HTTP/runtime tests for invalid Cron, Cron+interval, empty URLs, zero/overflow-ish numeric limits, and temp-test scheduled-plan blockers.

4. API generation
   - Run `npm --prefix frontend run api:generate`.
   - Review generated SDK/type diffs for expected force query support and schema changes.

5. Frontend form validation and payload normalization
   - Add client-side validation/projection helpers instead of silently dropping invalid URL/header data.
   - Normalize optional zero numeric fields to unset before payload submission.
   - Track editor mode, original name, and initial snapshot for dirty-state and rename detection.
   - Clear stale Cron preview when preview validation fails.

6. Frontend confirmations
   - Reuse `ConfirmActionDialog` for overwrite, stop-on-save, delete-stop, reset dirty, and close dirty flows.
   - Ensure save confirmation can combine overwrite and stop effects into one dialog.
   - Pass force only after explicit overwrite confirmation.

7. Frontend temp-test guards
   - Validate the form in temp-test mode before submission.
   - Pre-check active download, multiple scheduled plans, and scheduled-name mismatch.
   - Do not change selected saved config when testing unsaved form data.
   - Keep last-run result visible in the editor after activity history updates.

8. Frontend UI polish
   - Render validation errors with existing field components.
   - Disable or surface mutually exclusive Cron/interval state clearly.
   - Keep UI copy concise and non-duplicated.

9. Validation
   - Run `gofmt -w .`.
   - Run `go test ./...`.
   - Run `npm --prefix frontend run build`.
   - Review `static/app` diffs if the build updates compiled assets.

## Risk Areas

- `frontend/src/composables/useDashboard.ts`: editor mode, confirmation state, and save force retry need careful state transitions.
- `frontend/src/domain/dashboard.ts`: payload normalization must preserve explicit empty URL list for save.
- `internal/httpapi/server.go`: Huma query input changes affect OpenAPI generation.
- `internal/runtime/runtime.go`: stopping source and target during rename/overwrite must not stop unrelated configs.
- `internal/storage/storage.go`: config rename/overwrite should be one locked mutation to avoid split writes.

## Rollback Points

- After backend validation tests pass.
- After save/rename/overwrite API tests pass.
- After frontend form validation compiles.
- After full build and static asset review.
