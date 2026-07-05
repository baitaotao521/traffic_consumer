# Fix same-name config save hang

## Goal

Fix the regression where saving a configuration without changing its name can appear to hang or leave the confirmation dialog stuck. The save flow must complete normally for same-name edits while rejecting duplicate create/rename targets without opening another confirmation dialog.

## User Value

Users can edit an existing task configuration and press save without renaming it. The dashboard should save, close the editor, refresh state, and block duplicate names with a direct warning instead of stacking confirmation dialogs.

## Known Context

- Recent guard work added `mode=create|edit` and `force` query support to `PUT /api/configurations/{name}`.
- Frontend save confirmation uses `ConfirmActionDialog`; pending state is driven by `confirmDialogPendingKey`.
- Same-name edit should not be treated as duplicating another configuration.
- Runtime stop confirmation is still required when the same-name config is running or scheduled.
- The fix must stay decoupled: API/storage/runtime rules stay in backend packages; editor validation/projection stays in `frontend/src/domain/configEditor.ts`; UI components should remain display-only.

## Requirements

- Same-name edit save must not require overwrite `force=true`.
- Same-name edit save that does not need a runtime-stop confirmation must complete without opening or sticking a confirmation dialog.
- Same-name edit save that does need runtime-stop confirmation must close the confirmation dialog after the save completes.
- Create/rename-to-existing must be rejected directly without opening overwrite confirmation.
- Backend save semantics must reject duplicate create/rename targets even when `force=true`, while permitting normal same-name edit.
- Add regression coverage for the stuck confirmation or same-name save path.
- URL 列表编辑改为输入表格添加，保存/测试仍复用现有 URL 投影与校验规则。

## Acceptance Criteria

- [x] Editing config `A` and saving as `A` completes successfully without overwrite confirmation.
- [x] Editing config `A` and saving as `A` while `A` is running or scheduled asks for stop confirmation, then completes and closes the dialog.
- [x] Creating config `A` when `A` exists is rejected without an overwrite confirmation.
- [x] Renaming `A` to existing `B` is rejected without an overwrite confirmation.
- [x] URL 列表通过输入表格逐条添加和删除。
- [x] Frontend build passes.
- [x] Targeted/backend tests pass.

## Out Of Scope

- Reworking unrelated editor validation rules.
- Changing deprecated Python backend behavior.
- Changing persisted config/stat JSON outer shapes.
