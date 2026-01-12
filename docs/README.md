# Docs Index

Dokumen utama untuk project Omnichannel QMS.

## Product & Requirements
- **PRD**: `PRD_Omnichannel_QMS.md`
- **Backlog**: `QMS_Backlog_Epics_UserStories_AcceptanceTests.md`

## Engineering
- **Specs (Architecture/Contracts)**: `specs.md`
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

## Diagrams
Taruh diagram di `docs/diagrams/` (png/svg/mermaid) dan referensikan dari `specs.md`/ADR.
