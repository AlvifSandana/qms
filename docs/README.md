# Docs Index

Dokumen utama untuk project Omnichannel QMS.

## Product & Requirements
- **PRD**: `PRD_Omnichannel_QMS.md`
- **Backlog**: `QMS_Backlog_Epics_UserStories_AcceptanceTests.md`

## Engineering
- **Specs (Architecture/Contracts)**: `specs.md`
- **OpenAPI**: `openapi/queue-service.yaml`, `openapi/admin-service.yaml`, `openapi/auth-service.yaml`, `openapi/analytics-service.yaml`, `openapi/realtime-service.yaml`
- **Style Guide**: `style-guide.md`
- **Postman Collection**: `postman_collection.json`
- **Project TODO**: `../TODO.md`
- **AI Agent Rules**: `../AGENT_RULES.md`

## Architecture Decision Records (ADR)
Simpan keputusan penting di `docs/ADR/`:
- ADR-001: Backend stack + DB
- ADR-002: Realtime strategy (SockJS + fallback)
- ADR-003: Multi-tenant strategy (tenant_id scoping, optional RLS)
- ADR-004: Event model (outbox + eventual consistency)

Template ADR (saran):
- Context
- Decision
- Alternatives considered
- Consequences
- Links (PR, issue, diagram)

## Runbooks
Simpan runbook di `docs/RUNBOOKS/`:
- Backup/Restore Postgres
- Incident triage (queue lag, notif outage, device offline surge)
- DR drill checklist
- Security baseline
- Provider outage
- Device offline surge

## Security & Ops
- **RBAC matrix**: `security/rbac_matrix.md`
- **Request protection**: `security/request_protection.md`
- **Secrets rotation**: `security/secrets_rotation.md`
- **TLS & encryption**: `security/tls_encryption.md`
- **Migration policy**: `ops/migration_policy.md`
- **Observability**: `ops/observability.md`
- **Alert rules**: `ops/alerts.md`
- **Retention policy**: `ops/retention_policy.md`
- **Timezone policy**: `ops/timezone_policy.md`
- **CD plan**: `ops/cd_plan.md`
- **Versioning**: `ops/versioning.md`
- **Testing strategy**: `ops/testing_strategy.md`
- **UAT checklist**: `ops/uat_checklist.md`
- **DR evidence**: `ops/dr_evidence.md`
- **Webhook docs**: `webhooks.md`
- **Notification preferences**: `ops/notification_preferences.md`

## Testing Assets
- **E2E smoke**: `../tests/e2e/README.md`
- **Smoke script**: `../scripts/e2e-smoke.sh`
- **Load test script**: `../scripts/load-test.sh`

## Diagrams
Taruh diagram di `docs/diagrams/` (png/svg/mermaid) dan referensikan dari `specs.md`/ADR.
