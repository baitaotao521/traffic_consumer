# Redeployment Plan

- [x] Confirm source revision and clean working tree.
- [x] Inspect the prior deployment record and current remote container configuration.
- [x] Enable Netcatty External MCP and verify the connected target session.
- [x] Activate the Trellis task after plan review.
- [x] Run backend tests and the frontend production build.
- [x] Repair and verify the cross-platform npm lockfile after a clean Docker `npm ci` exposed missing optional dependency entries.
- [x] Build and inspect a `linux/amd64` Docker image from revision `816a042`.
- [x] Smoke-test `/` and `/api/dashboard` locally using isolated runtime state.
- [x] Export, checksum, and upload the image; verify the checksum before loading it remotely.
- [x] Back up persistent data and replace the container using an automatic rollback script.
- [x] Verify HTTP, container state, restart count, logs, and persisted-file checksums.
- [x] Delete temporary transfer files and record the deployed artifact and rollback commands.
