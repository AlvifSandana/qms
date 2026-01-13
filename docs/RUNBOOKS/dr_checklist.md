# DR Drill Checklist

## Preparation
- Confirm latest backup integrity and storage location.
- Record current version, migration level, and environment config.

## Restore Steps
- Bring up DB container and restore from backup.
- Run `scripts/migrate.sh` to align schema.
- Start services and verify `/healthz`.

## Validation
- Create a test ticket via `/api/tickets` and confirm realtime delivery.
- Verify admin login and report export.

## Post-Drill
- Document downtime and issues encountered.
- Update runbook with any missing steps.
