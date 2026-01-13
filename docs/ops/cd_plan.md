# CD Plan

## Staging
- Auto-deploy on merge to `main`.
- Run migrations before service rollout.

## Production
- Manual approval required.
- Backup DB before migration.
- Roll forward only; rollback via new migration.
