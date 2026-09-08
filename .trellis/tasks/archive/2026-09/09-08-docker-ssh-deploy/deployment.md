# Deployment Record

## Target and Artifact

- Deployed at: 2026-09-08 17:16 Asia/Taipei
- Target: `root@192.168.5.251`, reached through the Netcatty session labeled `docker`
- Source revision: `4e6c69e33cbf`
- Image tag: `traffic_consumer:deploy-20260908-4e6c69e33cbf`
- Image platform: `linux/amd64`
- Image ID: `sha256:b6b83e436ea55f1cc079e6a04051674bdf79b402f3e8710c3d9d788e323b5442`
- Transfer archive SHA-256: `de32d6ac9c906083022aa9069b7cc82e03952480431e246dc0aa0c2537add60c`

## Runtime Configuration

- Container: `traffic_consumer`
- Entrypoint: `traffic_consumer`
- Published port: `5001:5001`
- Persistent bind: `/root/.traffic_consumer_data:/root/.traffic_consumer`
- Restart policy: `unless-stopped`
- Service URL: `http://192.168.5.251:5001/`

## Verification

- Existing `config.json` and `stats.json` were backed up before the switch.
- Their SHA-256 values remained unchanged after startup.
- Local image smoke test returned HTTP 200 for `/` and `/api/dashboard`.
- Remote and LAN checks returned HTTP 200 for `/` and `/api/dashboard`.
- Dashboard loaded one existing configuration and one plan.
- Container state was `running` with zero restarts after deployment.
- Runtime logs show the existing configuration started. Some configured download URLs currently fail with DNS, HTTP 403, or HTTP 404 responses; this is upstream/configuration behavior rather than a container startup failure.

## Rollback

- Previous container: `traffic_consumer_rollback_20260908_1718`
- Pre-deployment data backup: `/root/traffic_consumer_data_backup_20260908_1718.tar.gz`

Container rollback commands:

```bash
docker stop traffic_consumer
docker rename traffic_consumer traffic_consumer_failed_20260908_1718
docker rename traffic_consumer_rollback_20260908_1718 traffic_consumer
docker start traffic_consumer
```

If persisted data must also be restored, stop both containers first and restore the pre-deployment archive before starting the rollback container.

The temporary image transfer archive was deleted after verification. The loaded image, previous container, and data backup were retained.
