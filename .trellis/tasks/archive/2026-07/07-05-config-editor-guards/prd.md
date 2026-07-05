# Harden task configuration editor guards

## Goal

Make the task configuration editor reject unsafe or ambiguous submissions before they mutate saved configuration data or start runtime work. Every guard identified in the review must be implemented with user-visible feedback in the Vue dashboard and server-side enforcement in the Go Web API/runtime path where the same invalid state can be submitted directly.

## User Value

Users can edit, rename, test, start, and delete configurations without accidentally overwriting another configuration, silently stopping the wrong runtime item, saving invalid runtime values, or launching a task that can only fail after it has already entered the runtime.

## Confirmed Facts

- The maintained Web backend is the Go service under `internal/**`; Python paths are deprecated and stay out of scope.
- The task configuration editor is `frontend/src/components/dashboard/ConfigSheet.vue`.
- `ConfigSheet.vue` renders inputs and emits `save` / `test`, but field validation currently lives outside the component. The save button only checks `saving` state and emits immediately (`frontend/src/components/dashboard/ConfigSheet.vue:371`).
- `saveEditorConfig` only checks for a non-empty trimmed name before calling `saveConfig`; it does not know whether the editor is creating, editing, renaming, overwriting, or stopping a running plan (`frontend/src/composables/useDashboard.ts:391`).
- `openPlanEditor` loads the row config and writes the row name into the form, but the original name is not retained separately once the user edits `form.name` (`frontend/src/composables/useDashboard.ts:385`).
- `configFromForm` silently filters URL lines, sends every finite numeric value, and silently drops malformed request header lines (`frontend/src/domain/dashboard.ts:148`, `frontend/src/domain/dashboard.ts:170`).
- Cron preview validates through `/api/cron/preview`, but failed preview does not clear stale preview output (`frontend/src/composables/useDashboard.ts:570`).
- `PUT /api/configurations/{name}` currently treats the path name as the only save target and maps all save failures to `500` (`internal/httpapi/server.go:258`).
- `Manager.SaveConfig` stops only the normalized save target before writing; it cannot stop the original runtime item when a user edits config `A` and renames it to `B` because the original name is not part of the save contract (`internal/runtime/runtime.go:604`).
- `StartTempTest` has backend guards for active downloads, multiple scheduled plans, and scheduled-plan name mismatch, but the frontend only pre-checks active downloads before submitting (`frontend/src/composables/useDashboard.ts:341`, `internal/runtime/runtime.go:274`).
- Runtime scheduling currently uses Cron when both `cron_expr` and `interval` are present, leaving interval silently ignored (`internal/runtime/runtime.go:632`).

## Decisions

- User confirmed that all listed missing guards are required.
- The UI must catch invalid input before submitting, but direct API calls must still be rejected server-side for data loss or runtime-safety conditions.
- Saved configs may intentionally contain an empty URL list, preserving the existing explicit-empty-URLs compatibility rule. Starting or temporary-testing a config with no URLs must be rejected.
- Creating or renaming to an existing configuration name requires explicit overwrite confirmation in the UI and an explicit force signal to the API. A direct API request without that force signal must return a conflict instead of overwriting.
- Renaming a configuration uses the existing `PUT /api/configurations/{name}` route: the path name is the original config name and the body `name` / `config_name` is the target name. This keeps route names stable.
- When a save affects a running or scheduled source config, target config, or overwritten config, those runtime items must be stopped before config data is mutated.
- Temporary tests run from the current form values and may be launched for unsaved form data, but they must not switch the dashboard selection to a non-existent saved config.

## Requirements

### Editor State And Save Semantics

- Track whether the editor was opened for create or edit, and retain the original config name during edit.
- Support renaming a saved config by saving the form under the target name and removing the old config entry.
- Prevent accidental overwrite when the target name already exists and differs from the original name. The first save attempt must ask for confirmation; the confirmed attempt must call the API with an explicit overwrite/force signal.
- When overwrite is confirmed, stop affected runtime items for both the original and target names before writing the final config data.
- Saving a running or scheduled config must show a confirmation that the affected runtime item will stop.
- Deleting a running or scheduled config must show that the selected runtime item will stop before deletion.
- Closing the editor or clearing the form with unsaved changes must ask for confirmation.

### Name, URL, Header, And Schedule Validation

- Config names must be trimmed, non-empty, length-limited, and reject control characters or path separators that cannot safely round-trip through route paths.
- Saved config names must remain user-readable and may contain Unicode text outside the rejected character set.
- URL lines must be trimmed and de-duplicated.
- Saving may preserve an explicit empty URL list; starting or testing requires at least one URL.
- Non-empty URL lines must be valid `http` or `https` URLs and length-limited.
- Request headers must reject malformed lines instead of silently dropping them.
- Duplicate request headers must be rejected or surfaced as validation errors instead of silently overwriting prior values.
- `User-Agent` must be configured in one place only. If the dedicated User-Agent field is set, a custom `User-Agent` header must be rejected.
- Cron expressions must be validated before save/start/test, not only during preview.
- Invalid Cron preview must clear stale preview results.
- Cron and interval scheduling are mutually exclusive in the editor and Web API. Submissions containing both must be rejected with a clear validation error.

### Numeric Validation

- Numeric fields must use shared bounds and shared zero semantics across frontend payload building and backend validation.
- `threads` must be a positive integer within a safe upper bound.
- Optional limits (`traffic_limit`, `duration`, `count`, `interval`, `url_switch_interval`, `thread_start_delay`) must treat blank as unset.
- For optional limit fields, `0` must not be sent as a meaningful limit from the editor; backend Web validation must reject non-positive submitted limits where they would produce confusing runtime behavior.
- Numeric upper bounds must prevent runaway local resource use, overflow-prone traffic conversion, and nonsensical scheduler intervals.

### Temporary Test Guards

- Temporary test must be disabled while any download is active.
- Temporary test must reject multiple scheduled plans before submission.
- Temporary test must reject a scheduled-plan name mismatch before submission.
- Temporary test for unsaved form data must leave the saved-config selection unchanged and still surface the last-run result in the editor after completion.
- Temporary test must apply the same name, URL, header, schedule, and numeric validation as save/start, with the existing test-specific runtime behavior preserved.

### Backend And API Contract

- Add a backend validation owner for Web config payloads so save, start, and temp-test routes do not duplicate ad hoc validation.
- Keep existing route paths stable unless a query parameter is required for explicit overwrite confirmation.
- Save/rename/overwrite conflicts must return `409 Conflict`; validation failures must return `400 Bad Request` or Huma validation errors; storage failures remain `500`.
- OpenAPI and generated frontend SDK must be regenerated when API signatures change.
- Existing CLI behavior and persisted `config.json` / `stats.json` outer shapes must remain compatible.

## Acceptance Criteria

- [ ] Creating a config with an existing name does not overwrite until the user confirms overwrite.
- [ ] Editing config `A` and renaming it to unused `B` removes `A`, saves `B`, and stops `A` if it was running or scheduled.
- [ ] Editing config `A` and renaming it to existing `B` returns a conflict without force; confirmed overwrite stops affected `A`/`B` runtime items, saves `B`, and does not leave duplicate rows.
- [ ] Saving a running or scheduled config requires confirmation and stops only the affected config names before mutation.
- [ ] Deleting a running or scheduled config makes the stop effect explicit and stops only that config before deletion.
- [ ] Closing or resetting a dirty editor asks for confirmation before discarding changes.
- [ ] Empty config name, too-long name, path separators, and control characters are rejected before save/test/start.
- [ ] Invalid, duplicate, or non-http(s) URLs are rejected; empty URL lists can be saved but cannot be started or temp-tested.
- [ ] Invalid header lines, duplicate header names, and duplicate `User-Agent` configuration are rejected.
- [ ] Invalid Cron expressions cannot be saved, started, or temp-tested; failed preview clears old preview values.
- [ ] Cron and interval cannot both be submitted from the editor or Web API.
- [ ] Numeric fields reject unsafe lower/upper bound values; `0` no longer means an accidental immediate count/traffic stop in editor submissions.
- [ ] Temp test pre-checks active downloads, multiple scheduled plans, and scheduled-name mismatch before calling the API.
- [ ] Temp test for unsaved form data does not set the selected saved config to a missing name and still shows the last-run result in the editor.
- [ ] Backend tests cover validation, rename, overwrite conflict/force, runtime stop isolation, and temp-test guard behavior.
- [ ] Frontend build passes after SDK generation.
- [ ] `go test ./...` passes.

## Out Of Scope

- Changing the deprecated Python backend.
- Changing CLI command semantics unless required to keep shared model compatibility.
- Adding account-level authorization or multi-user ownership checks.
- Changing the persisted outer shape of `config.json` or `stats.json`.
