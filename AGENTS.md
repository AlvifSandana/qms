# Repository Guidelines

## Project Structure & Module Organization
- `docs/` holds PRD, specs, and backlog. Start here before coding.
- `services/` contains Go services (queue/auth/admin/notification/analytics/realtime), each with its own `go.mod` and `migrations/`.
- `apps/` contains web apps (admin/agent/kiosk/display/public) as static prototypes.
- `packages/` holds shared UI and utility packages.
- `infra/docker/compose.yml` defines local stack; `scripts/migrate.sh` runs DB migrations.
- `AGENT_RULES.md` and `TODO.md` define workflow, constraints, and priorities.

## Build, Test, and Development Commands
- `cp .env.example .env`: create local environment configuration.
- `docker compose -f infra/docker/compose.yml up -d`: run the full stack locally.
- `scripts/migrate.sh`: apply DB migrations via the migrate container.
- `docker compose -f infra/docker/compose.yml logs -f`: tail service logs.
- Health checks: `GET /healthz` (services), `GET /realtime/info` (SockJS).
- Tests per service: `cd services/queue-service && go test ./...` (repeat for other services).

## Coding Style & Naming Conventions
- Follow `AGENT_RULES.md` contracts (idempotency, outbox, multi-tenant scope, error format).
- Go code uses `gofmt`; prefer clear package names and keep command (write) vs query (read) handlers separate.
- Use structured JSON logging including `request_id`, `tenant_id`, and `branch_id` when applicable.
- Migrations use numeric prefixes (example: `services/queue-service/migrations/001_init.sql`).
- Branch naming: `feat/<module>-<short>`, `fix/<module>-<short>`, `chore/<short>`.

## Testing Guidelines
- Required minimums are in `AGENT_RULES.md` (unit + integration tests by area).
- Queue engine tests must cover state transitions and concurrency (“call next”).
- Realtime tests must cover subscribe + resync; notifications must cover template render + retry/DLQ.
- Add tests with each story change and keep AC coverage explicit.

## Commit & Pull Request Guidelines
- Git history is minimal; use story IDs in commits (example: `US-QUEUE-002: add call-next`).
- PRs must reference a story ID, list acceptance criteria, and include verification steps.
- Keep changes small, include tests, and update docs when behavior changes.
- Never commit secrets or disable lint/test gates.

## Security & Configuration Tips
- Default to no PII; if enabled, implement masking and retention jobs.
- Keep rate limiting and audit logging enabled for privileged actions.
