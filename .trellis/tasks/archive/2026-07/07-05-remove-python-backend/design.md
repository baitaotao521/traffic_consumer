# Remove Python Backend Design

## Boundary

The maintained product runtime remains:

```text
main.go -> internal/cli or internal/httpapi -> internal/runtime/consumer/storage/stats/urlmanager
frontend/ -> static/app/ -> templates/index.html
```

The removed product Python runtime is:

```text
main.py
web_ui.py
traffic_consumer.py
app/**
requirements.txt
build_config.py
tests/test_frontend_theme.py
tests/test_web_ui_issue22.py
```

Trellis and Codex Python scripts stay in place because they operate the development workflow, not the product.

## Documentation Contract

README and build documentation should present Go + Vue as the only runtime and release path. The old wording that says Python files are "deprecated and kept" must be removed because the files will no longer exist.

Backend specs should shift from "do not add to Python" to "Python product runtime has been removed." Go comments can still reference historical Python behavior when the comment explains a compatibility rule embedded in storage, config, or output formatting.

## Compatibility

Existing user data in `~/.traffic_consumer/config.json` and `~/.traffic_consumer/stats.json` must remain supported by the Go code. Removing Python files does not change JSON field names, config normalization, or stats formatting.

The compiled Go binary, Dockerfile, GitHub Actions workflow, and frontend build are already Go/Node based, so no runtime migration code is needed.

## Rollback

Rollback is a normal git revert of deleted Python files and documentation edits. No data migration is introduced.
