# Repository Guidelines

## Project Structure & Module Organization
- `docs/` holds PRD, backlog, specs, ADRs, and runbooks. Start here for requirements.
- `services/` contains Go services (queue/auth/admin/notification/analytics/realtime). Each service has its own `go.mod` and `migrations/`.
- `apps/` contains web apps (admin/agent/kiosk/display/public). These are static prototypes served by the stack.
- `packages/` hosts shared UI and utility modules.
- `infra/docker/compose.yml` defines the local stack; `scripts/migrate.sh` applies DB migrations.

## Build, Test, and Development Commands
- `cp .env.example .env`: create local environment configuration.
- `docker compose -f infra/docker/compose.yml up -d`: run the full stack locally.
- `scripts/migrate.sh`: apply Postgres migrations via the migrate container.
- `docker compose -f infra/docker/compose.yml logs -f`: follow service logs.
- Health checks: `GET /healthz` (services), `GET /realtime/info` (SockJS).
- Tests per service: `cd services/queue-service && go test ./...` (repeat for other services).

## Coding Style & Naming Conventions
- Go code must be `gofmt`-formatted; keep handlers and store logic separated by package boundaries.
- Frontend files follow existing formatting (2-space indentation in `apps/*/app.js`).
- Migrations use numeric prefixes: `services/queue-service/migrations/001_init.sql`.
- Follow `AGENT_RULES.md`: require `request_id`, outbox events for ticket changes, tenant scoping, and structured error format.

## Testing Guidelines
- Use standard Go `testing` via `go test ./...` inside each service.
- Add tests alongside story work; include queue state transitions, outbox events, and realtime resync when touched.
- Update docs/specs when tests reveal contract changes.

## Commit & Pull Request Guidelines
- Commit messages follow story or scope prefixes from history: `US-ADMIN-004: ...`, `SEC-BASELINE: ...`, `UI: ...`, `docs: ...`.
- PRs should reference a story ID, list acceptance criteria, and include verification steps.
- Keep changes scoped; update docs when behavior or contracts change.
- If adding dependencies or changing APIs, add an ADR or update OpenAPI specs.

## Security & Configuration Tips
- Do not commit secrets; use `.env` from `.env.example`.
- Default to minimal PII; enable masking/retention if tenant PII is introduced.
- Keep rate limiting and audit logging enabled for privileged actions.

## Local Troubleshooting
- Services not healthy: check `docker compose -f infra/docker/compose.yml logs -f` and confirm `GET /healthz`.
- Realtime not updating: verify `GET /realtime/info`, then reload clients to force resync.
- DB errors: re-run `scripts/migrate.sh` and confirm the Postgres container is up.
