# ADR-002: Realtime Strategy

## Context
Realtime updates are required for display boards, agent consoles, and public queue tracking with fallbacks for legacy browsers.

## Decision
- Primary: WebSocket via SockJS
- Fallback: polling snapshots

## Alternatives Considered
- SSE only (simpler, but limited browser support for bidirectional needs)
- Native WebSocket only (no legacy fallback)

## Consequences
- Requires SockJS server integration and polling fallback endpoints.
- Clients must handle reconnect + resync.

## Links
- docs/specs.md
