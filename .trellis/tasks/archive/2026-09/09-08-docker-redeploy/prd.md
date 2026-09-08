# 重新打包并部署 Traffic Consumer

## Goal

Rebuild the current Traffic Consumer source and redeploy it to 192.168.5.251 through the connected Netcatty SSH session.

## Requirements

- Build the image from clean source revision `816a042`, including the redesigned Vue dashboard and the dashboard-response guard fix.
- Target the remote host's `linux/amd64` architecture while building on the local ARM64 Docker host.
- Preserve `/root/.traffic_consumer_data`, port `5001:5001`, and restart policy `unless-stopped`.
- Preserve the currently running Go container and create a fresh data backup before switching.
- Automatically restore the current container if the replacement fails to start or fails HTTP checks.
- Verify the dashboard and API from both the remote host and the local LAN client.

## Acceptance Criteria

- [x] `go test ./...` and `npm --prefix frontend run build` pass for revision `816a042`.
- [x] A `linux/amd64` image builds and passes local HTTP smoke tests for `/` and `/api/dashboard`.
- [x] The transferred archive checksum matches before the image is loaded on 192.168.5.251.
- [x] The replacement container runs with the existing bind mount, published port, and restart policy.
- [x] Persisted `config.json` and `stats.json` remain intact across the switch.
- [x] `/` and `/api/dashboard` return HTTP 200 over `http://192.168.5.251:5001`.
- [x] The deployment record contains artifact identifiers, validation evidence, and tested rollback commands.

## Background

- The previous deployment used image `traffic_consumer:deploy-20260908-4e6c69e33cbf` with image ID `sha256:b6b83e436ea55f1cc079e6a04051674bdf79b402f3e8710c3d9d788e323b5442`.
- The current remote container is healthy with zero restarts and uses `/root/.traffic_consumer_data:/root/.traffic_consumer`.
- The previous Python container remains available as `traffic_consumer_rollback_20260908_1718`.
- Netcatty External MCP is enabled and exposes the connected `root@192.168.5.251` session labeled `docker`.
- The current working tree is clean except for this Trellis task directory.
- The first Docker build exposed missing cross-platform optional dependency entries in `package-lock.json`; the lockfile was regenerated with the Dockerfile's Node 22/npm 10 environment before deployment.
