# Repository Guidelines

## Project Structure & Module Organization
- `docs/` holds product docs (PRD, specs, backlog). Start here before coding.
- `TODO.md` lists near-term work and priorities.
- `AGENT_RULES.md` contains required workflow, architectural constraints, and testing expectations.
- Source code and tests are not yet present in this repo; follow `docs/specs.md` and add modules under conventional paths when implementation begins (for example, `backend/` for Go services, `frontend/` for Vite apps, `infra/` for Docker/Caddy configs).

## Build, Test, and Development Commands
- `cp .env.example .env`: create local environment configuration.
- `docker compose up -d`: run the full stack locally (API, DB, edge).
- `docker compose logs -f`: tail service logs during development.
- Health checks (when running): `GET /healthz` for API, `GET /realtime/info` for SockJS.

## Coding Style & Naming Conventions
- Follow the technical contracts in `AGENT_RULES.md` (idempotency, outbox events, multi-tenant scoping, error format).
- Backend is Go; keep handlers separated for commands (write) vs queries (read).
- Use structured JSON logging with `request_id`, `tenant_id`, and `branch_id` when applicable.
- Branch naming: `feat/<module>-<short>`, `fix/<module>-<short>`, `chore/<short>`.

## Testing Guidelines
- Required minimums are defined in `AGENT_RULES.md` (unit + integration tests by area).
- Queue engine tests must cover state transitions and concurrency (“call next”).
- Realtime tests must cover subscribe + resync. Notifications must cover template render + retries.
- Add tests alongside implementation once code structure exists.

## Commit & Pull Request Guidelines
- This repository does not expose Git history; follow `AGENT_RULES.md` for workflow.
- PRs must reference a story ID (e.g., `US-QUEUE-002`), list acceptance criteria, and include verification steps.
- Keep changes small, include tests, and update docs when behavior changes.
- Never commit secrets or disable lint/test gates.

## Security & Configuration Tips
- Default to no PII; if enabled, implement masking and retention jobs.
- Keep rate limiting and audit logging enabled for privileged actions.
