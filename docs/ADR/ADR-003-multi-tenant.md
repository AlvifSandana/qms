# ADR-003: Multi-tenant Strategy

## Context
The platform is multi-tenant and must prevent cross-tenant access at all layers.

## Decision
- Every table includes `tenant_id` (and `branch_id` where applicable).
- All queries are scoped by `tenant_id` (and `branch_id` as needed).
- Optional future: Postgres RLS policies per tenant.

## Alternatives Considered
- Separate DB per tenant (strong isolation, high ops overhead)
- Schema per tenant (complex migrations)

## Consequences
- Requires strict query discipline and review.
- Enables centralized analytics and simpler ops.

## Links
- docs/specs.md
- AGENT_RULES.md
