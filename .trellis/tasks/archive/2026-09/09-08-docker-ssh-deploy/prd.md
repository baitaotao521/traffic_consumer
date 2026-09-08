# 打包 Docker 并通过 SSH 部署到 192.168.5.251

## Goal

Build the current project as a Docker image and deploy it over SSH to 192.168.5.251.

## Requirements

- Include the Go service and compiled Vue dashboard using the existing Dockerfile.
- Preserve any existing deployment configuration and persistent data.
- Verify the running container and HTTP service after deployment.

## Acceptance Criteria

- [x] Image builds for the remote host architecture.
- [x] Image is transferred and loaded on 192.168.5.251.
- [x] Container runs with persistent storage and an appropriate restart policy.
- [x] Dashboard and API respond successfully.

## Notes

- Existing Dockerfile exposes port 5001; BUILD_GUIDE.md documents persistence at /root/.traffic_consumer.
- Go tests passed on 2026-09-08.
- Netcatty External MCP is registered in Codex and exposes a connected root@192.168.5.251 SSH session labeled `docker`.
- Deployed image: `traffic_consumer:deploy-20260908-4e6c69e33cbf` (`linux/amd64`, `sha256:b6b83e436ea55f1cc079e6a04051674bdf79b402f3e8710c3d9d788e323b5442`).
- Service URL: `http://192.168.5.251:5001/`.
