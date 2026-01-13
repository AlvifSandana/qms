# Postgres Backup and Restore

## Backup
- Identify target DB container: `docker compose -f infra/docker/compose.yml ps`.
- Run dump: `docker exec -t qms-postgres pg_dump -U qms qms > backup.sql`.
- Store backups in a secure location (encrypted storage).

## Restore
- Stop writers if possible (pause API containers).
- Restore: `cat backup.sql | docker exec -i qms-postgres psql -U qms qms`.
- Re-run migrations: `scripts/migrate.sh`.

## Validation
- Check `/healthz` for all services.
- Sample query: `SELECT count(*) FROM tickets;`.
