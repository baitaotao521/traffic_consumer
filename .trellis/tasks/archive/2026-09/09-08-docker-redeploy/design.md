# Docker Redeployment Design

Build the existing multistage Dockerfile with Buildx using `--platform linux/amd64 --load`. Tag the image with the source revision so the deployed artifact maps directly to Git history. Run it locally on an isolated loopback port with temporary persistent storage, and require HTTP 200 from `/` and `/api/dashboard` before transfer.

Export and gzip the image, calculate SHA-256 locally, upload it through the connected Netcatty SFTP session, and compare the remote checksum before `docker load`. Keep the uploaded archive only until deployment verification finishes.

Before switching, calculate checksums for persisted files and create a timestamped archive of `/root/.traffic_consumer_data`. Stop the current container and rename it to a unique rollback name. Start the replacement with the same bind mount, port mapping, and restart policy. If startup or either in-container HTTP check fails, remove the failed replacement, restore the renamed container, and start it.

After switching, verify container state, restart count, logs, persisted-file checksums, remote in-container HTTP, and LAN HTTP. Retain the loaded image, the immediately previous container, the older Python rollback container, and the new data backup.
