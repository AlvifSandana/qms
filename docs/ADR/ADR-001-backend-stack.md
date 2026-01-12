# ADR-001: Backend Stack + DB

## Context
We need a backend stack that supports multi-tenant queue operations, transactional integrity, and reliable outbox eventing. The PRD and specs target a modular monolith with PostgreSQL.

## Decision
- Backend: Go (modular monolith)
- Database: PostgreSQL
- Migrations: SQL files applied via migrate container

## Alternatives Considered
- Node.js + Postgres (faster iteration, but more runtime overhead)
- Java/Kotlin + Postgres (strong concurrency, heavier runtime)

## Consequences
- Go provides strong concurrency and static binaries.
- PostgreSQL supports row locks and SKIP LOCKED for queue safety.
- Must maintain explicit migrations for schema changes.

## Links
- docs/specs.md
- docs/PRD_Omnichannel_QMS.md
