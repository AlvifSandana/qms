# ADR-004: Event Model (Outbox)

## Context
We need reliable event emission for realtime, notifications, and analytics without distributed transactions.

## Decision
- Use an outbox table in the primary DB.
- Write domain changes and outbox events in the same transaction.
- Background worker publishes outbox events to downstream systems.

## Alternatives Considered
- Direct publish (risk of lost events)
- Message bus as primary source of truth (higher complexity)

## Consequences
- Outbox table must be monitored and cleaned up.
- Publishing worker is required for eventual consistency.

## Links
- docs/specs.md
- AGENT_RULES.md
