# Redeployment Record

## Target and Artifact

- Deployed at: 2026-09-08 18:48 Asia/Taipei
- Target: `root@192.168.5.251`, reached through the Netcatty session labeled `docker`
- Source revision: `816a042a0dd10aa0452e3bde4cee5ae4a64a569b`
- Build fix: regenerated `frontend/package-lock.json` with Node 22/npm 10 so the Dockerfile's clean `npm ci` is reproducible
- Image tag: `traffic_consumer:deploy-20260908-816a042`
- Image platform: `linux/amd64`
- Image ID: `sha256:af082c09acd6280b490d49ab7c2663dc52db8b18da6bbad53eb154f78d0ec645`
- Image size: 18,193,769 bytes
- Transfer archive SHA-256: `79f1b3722111665126562a0fddef640c6b94900e0ab272b66a5ca12b2afd158c`

## Runtime Configuration

- Container: `traffic_consumer`
- Container ID: `58bd4ff70165d9189308d89a2062cb61ac85e0bf3f8eb6735e0bff1c578434bb`
- Entrypoint: `traffic_consumer`
- Published port: `5001:5001`
- Persistent bind: `/root/.traffic_consumer_data:/root/.traffic_consumer`
- Restart policy: `unless-stopped`
- Service URL: `http://192.168.5.251:5001/`

## Verification

- `go test ./...` passed.
- `npm --prefix frontend run build` passed.
- The first Docker build correctly failed because the committed lockfile lacked three optional cross-platform packages required by Node 22/npm 10. The regenerated lockfile passed a clean Linux/AMD64 `npm ci`, and the next Docker build passed.
- The local Linux/AMD64 image returned HTTP 200 for `/` and `/api/dashboard`.
- The uploaded archive checksum matched before `docker load`.
- Remote in-container checks returned HTTP 200 for `/` and `/api/dashboard`.
- LAN checks returned HTTP 200 for `/` and `/api/dashboard`.
- Dashboard returned one configuration, one plan, and running state.
- Container state was `running` with zero restarts after deployment.
- `config.json` remained at SHA-256 `bb227f298a4554f26e2a133e0eee56e0a4ea5f2dfe3f120f50fcc121347c1b22`.
- `stats.json` remained at SHA-256 `2acab067075f148cfb6e95c53c20756b34c9a1b5a417ac1d36c297a84873c2e2`.
- Both persisted files passed JSON parsing after the switch.
- Runtime logs show configuration `123` resumed automatically. Some configured download targets timed out; this is upstream network behavior rather than an application health failure.

## Rollback

- Immediately previous Go container: `traffic_consumer_rollback_20260908_1848`
- Earlier Python container: `traffic_consumer_rollback_20260908_1718`
- Pre-deployment data backup: `/root/traffic_consumer_data_backup_20260908_1848.tar.gz`

Container rollback commands:

```bash
docker stop traffic_consumer
docker rename traffic_consumer traffic_consumer_failed_20260908_1848
docker rename traffic_consumer_rollback_20260908_1848 traffic_consumer
docker start traffic_consumer
```

If persisted data also needs restoration, stop the containers first and extract `/root/traffic_consumer_data_backup_20260908_1848.tar.gz` from `/root` before starting the rollback container.

The temporary local and remote image archives were deleted after verification. The loaded image, both rollback containers, and data backup were retained.
