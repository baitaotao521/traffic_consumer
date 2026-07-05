# Remove Python Backend Implementation Plan

## Checklist

1. Delete product Python runtime files: `main.py`, `web_ui.py`, `traffic_consumer.py`, `app/**`, `requirements.txt`, and `build_config.py`.
2. Delete Python tests under `tests/**` because they either exercise removed Python runtime code or keep Python as a repository quality path.
3. Update `README.md` directory structure, local development, and release notes to describe only Go, Vue, Docker, and GitHub Actions paths.
4. Update `BUILD_GUIDE.md` to remove the deprecated Python build warning and keep Go/Docker release instructions.
5. Update `.trellis/spec/backend/index.md`, `directory-structure.md`, and `quality-guidelines.md` so future backend work assumes the Python product runtime is gone.
6. Search for stale product Python references with `rg -n --hidden "main\\.py|web_ui\\.py|traffic_consumer\\.py|requirements\\.txt|build_config\\.py|app/|pytest|PyInstaller|Flask|SocketIO"`.
7. Run `gofmt -w .` only if Go files are edited.
8. Run `go test ./...`.
9. Run `npm --prefix frontend run build` if docs/spec cleanup also changes frontend/static boundaries or if verification needs full CI parity.

## Risk Points

- Do not delete `.trellis/scripts/**` or `.codex/hooks/**`; those Python files are workflow infrastructure.
- Preserve Go comments that explain compatibility with old persisted file formats.
- The `tests/` directory may become empty; remove it if no tracked tests remain.

## Review Gate

Before starting implementation, review the PRD/design/implementation artifacts and get explicit approval to run `task.py start`.
