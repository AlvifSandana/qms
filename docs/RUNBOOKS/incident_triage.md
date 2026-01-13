# Incident Triage

## Symptoms and Checks
- Queue lag: check queue-service logs and DB connections.
- Realtime outage: verify `/realtime/info`, then check realtime-service logs.
- Notifications missing: check notification-worker logs and provider endpoints.
- Admin UI errors: verify admin-service `/healthz` and browser console errors.

## First Actions
- Confirm docker status: `docker compose -f infra/docker/compose.yml ps`.
- Tail logs: `docker compose -f infra/docker/compose.yml logs -f`.
- Verify migrations applied: `scripts/migrate.sh`.

## Escalation
- Capture timestamps, request IDs, and tenant IDs for affected flows.
- If DB is unstable, pause write traffic and take a backup.
