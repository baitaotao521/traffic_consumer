# Docker SSH Deployment Design

Use the existing multistage Dockerfile to build the frontend and Go binary. Inspect the remote CPU architecture and Docker environment before selecting the image platform. Tag the image with the source revision and deployment timestamp, then transfer it through SSH using docker save/load.

Inspect existing containers and mounts before replacing the service. Reuse existing data and configuration; for a new installation use port 5001, persistent storage mounted at /root/.traffic_consumer, and restart policy unless-stopped. Preserve the previous image and container configuration for rollback. Verify HTTP responses and container logs after startup.

Use the connected Netcatty SSH session labeled `docker` for remote inspection and deployment. Resolve port conflicts and existing deployment details from remote inspection before execution.
