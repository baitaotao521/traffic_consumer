# Config Editor Guard Design

## Summary

The editor will move from "serialize form and submit" to an explicit command model:

```text
open create/edit -> keep original editor target -> validate form -> detect runtime/overwrite effects -> confirm when destructive -> submit command -> refresh dashboard
```

The Go Web API will own the same validation rules and save semantics so direct API calls cannot bypass the dashboard guards. The existing route names stay stable. Rename and overwrite are handled by extending the meaning of the existing `PUT /api/configurations/{name}` route and adding an explicit force signal.

## Contracts

### Save Configuration

Route remains:

```text
PUT /api/configurations/{name}
```

The path `{name}` is the source/original name for edit operations. For create operations it is the requested target name. The body `name` / `config_name` is the normalized target name.

Add a query flag:

```text
?force=true
```

Behavior:

- Source name defaults to normalized path name.
- Target name defaults to body `config_name`, then body `name`, then source name.
- If source does not exist and target exists, `force=false` returns `409`.
- If source exists and target differs, save becomes a rename.
- If target exists and target differs from source, `force=false` returns `409`.
- If `force=true`, the target entry may be overwritten after stopping affected runtime items.
- Source and target runtime items are both stopped before mutation when they exist.
- Response `item.name` is always the final target name.

### Validation

Create a backend validation owner near the HTTP/runtime boundary, for example `internal/httpapi/config_validation.go` or `internal/runtime` if the runtime owns startability. It should return business errors that handlers map to `400` or `409`.

Validation modes:

- `save`: allows explicit empty URL list.
- `start`: requires at least one URL.
- `temp-test`: requires at least one URL and applies the same form constraints before test-specific runtime changes.

Name rules:

- Trim whitespace.
- Require non-empty.
- Reject `/`, `\`, ASCII control characters, and newline-like characters.
- Cap display name length to a fixed value such as 80 runes.
- Preserve Unicode display text that passes these constraints.

URL rules:

- Trim and de-duplicate exact URL strings.
- Allow empty only in save mode.
- Require `http` or `https`.
- Reject missing host.
- Cap each URL length to a fixed value such as 4096 characters.

Header rules:

- Reject malformed lines in the frontend parser instead of dropping them.
- Backend validation checks normalized `request_headers`.
- Header names must be non-empty, contain no colon/control characters, and be unique case-insensitively.
- Header values must be single-line and non-empty.
- Dedicated `user_agent` and custom `User-Agent` header are mutually exclusive.

Schedule rules:

- Validate Cron with `cron.ParseStandard` when `cron_expr` is present.
- Reject body payloads that include both valid `cron_expr` and positive `interval`.
- Frontend clears interval when Cron is used or blocks save with one validation error. Backend remains authoritative.

Numeric rules:

- Frontend converts blank/zero optional limits to `undefined` instead of serializing `0` as a runtime limit.
- Backend rejects submitted non-positive optional limits in Web payloads where the field is meaningful only when positive.
- Backend validates positive integer/float bounds before runtime conversion.
- Suggested constants:
  - config name: max 80 runes
  - URLs: max 500 entries, max 4096 characters each
  - threads: 1 to 512
  - limit speed: 0 to 10240 MB/s
  - traffic limit: 1 to 1073741824 MB when set
  - duration: 1 to 31536000 seconds when set
  - count: 1 to 1000000000 when set
  - interval: 1 to 525600 minutes when set
  - URL switch interval: 0.001 to 86400 seconds when set
  - thread start delay: 0 to 86400 seconds when set

## Frontend State

Add editor metadata in `useDashboard`:

- `editorMode: 'create' | 'edit'`
- `editorOriginalName`
- `editorInitialSnapshot`
- `pendingSavePayload` when a confirmation is needed

Derived checks:

- Dirty state compares current form projection to `editorInitialSnapshot`.
- Existing config names come from dashboard rows/configs.
- Affected runtime names include original name, target name, and overwrite target when different.
- Temp-test blockers derive from current `plans`.

The editor component should receive validation errors or disabled state through props and render them near fields using existing `Field` / `FieldError` patterns.

## Confirmation Flow

Reuse `ConfirmActionDialog`.

Confirmation cases:

- Save stops running/scheduled source or target.
- Save overwrites an existing target.
- Delete stops running/scheduled target.
- Reset/close discards dirty edits.

Combined confirmation text should list the concrete affected config names once. Avoid repeating the same warning in both title and description; UI text should stay direct.

## Temporary Test

Frontend pre-check order:

1. Validate form for temp-test mode.
2. Reject active downloads.
3. Reject multiple scheduled plans.
4. If one scheduled plan exists, require it to match the form target name.
5. Submit temp-test without changing selected saved config.

Activity `history_update` already carries `config` and `history_record`; keep using `lastRunByConfig` so unsaved form tests can show the result while the dialog remains open.

## Backend Runtime Save

Add a runtime method such as:

```go
SaveConfigAs(sourceName string, targetName string, cfg model.Config, force bool) (savedName string, overwritten bool, err error)
```

Business behavior:

- Validate names before mutation.
- Load configs under storage lock when checking source/target existence.
- Stop source and target runtime items before writing.
- Save target config with transient `Name` / `ConfigName` cleared by storage.
- Delete source entry when source and target differ.

Storage may need a locked helper that can rename/delete/write in one `config.json` mutation so source removal and target save cannot split under concurrent writes.

## Compatibility

- No persisted outer JSON shape changes.
- Existing saved configs with values outside new Web validation still load and render. The user must fix invalid values before saving or starting them from the Web dashboard.
- CLI remains compatible with existing parsing and config files.
- API route paths remain stable; OpenAPI changes are limited to the optional force query parameter and any response/error documentation changes.

## Rollback

Rollback is a normal Git revert. Because no persisted schema migration is introduced, previously saved `config.json` and `stats.json` files remain readable.
