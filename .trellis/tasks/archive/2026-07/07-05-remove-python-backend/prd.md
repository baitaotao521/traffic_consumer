# Remove Python backend

## Goal

Remove the deprecated product Python backend so Traffic Consumer is maintained as a Go HTTP service with a Vue/Vite dashboard.

The repository should no longer ship, document, or test the old Python runtime path as a usable product surface.

## Confirmed Facts

- The maintained backend is the Go entry point `main.go` plus `internal/**`; backend specs now describe the product Python backend as removed.
- CI in `.github/workflows/build-simple.yml` installs Node, builds the frontend, runs `gofmt`, and runs `go test ./...`; it does not install Python dependencies or run pytest.
- Product Python files tracked by git are `main.py`, `web_ui.py`, `traffic_consumer.py`, `app/**`, `requirements.txt`, `build_config.py`, `tests/test_frontend_theme.py`, and `tests/test_web_ui_issue22.py`.
- Documentation no longer presents product Python files as source code that remains in the repository.
- Project infrastructure also contains Python scripts under `.trellis/scripts/**` and `.codex/hooks/**`; these scripts power Trellis/Codex workflow and are not product backend code.
- User confirmed on 2026-07-06 that cleanup scope is the product Python backend, while Trellis/Codex workflow scripts stay in the repository.

## Requirements

- Delete the deprecated product Python runtime files:
  - `main.py`
  - `web_ui.py`
  - `traffic_consumer.py`
  - `app/**`
  - `requirements.txt`
  - `build_config.py`
- Delete Python tests that exercise the removed runtime or rely on Python as a repo quality path:
  - `tests/test_web_ui_issue22.py`
  - `tests/test_frontend_theme.py`
- Update user-facing documentation so installation, development, release, and directory structure describe only the Go backend, Vue frontend, Docker image, and Go executable artifacts.
- Update backend Trellis specs so future work treats the Python product path as removed, not deprecated-but-present.
- Preserve Go compatibility behavior for existing `config.json` and `stats.json` files. Compatibility comments in Go may still mention historical Python behavior where that explains persisted formats.
- Keep Trellis/Codex Python automation scripts out of deletion scope because they are project workflow infrastructure, not the product Python backend.

## Acceptance Criteria

- [ ] No tracked product Python runtime files remain at the repository root or under `app/**`.
- [ ] No tracked Python tests remain under `tests/**`.
- [ ] `README.md`, `BUILD_GUIDE.md`, and backend specs no longer instruct maintainers to keep or consult the Python backend files as present source code.
- [ ] Go runtime behavior, CLI entry points, Docker build, and Vue dashboard build remain unchanged.
- [ ] `go test ./...` passes.
- [ ] `npm --prefix frontend run build` passes if documentation/static/frontend boundaries are touched.

## Out of Scope

- Rewriting Go behavior solely to remove historical compatibility comments.
- Removing `.trellis/**` or `.codex/hooks/**` Python scripts.
- Cleaning IDE metadata such as `.idea/**` solely because it references a Python SDK.
- Changing persisted config/stat schemas.
