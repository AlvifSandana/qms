# specs.md — Omnichannel QMS (Default Stack)
**Date**: 12 Jan 2026 (Asia/Jakarta)  
**Status**: Draft v1 (implementation-ready)  
**Target**: Easy deploy (cloud/on‑prem), runs on old devices, minimal infra cost

---

## 1) Summary
This document defines **system specifications** for an **Omnichannel Queue Management System (QMS)** that supports:
- Walk‑in ticketing (kiosk/staff)
- Virtual queue (web/PWA)
- Agent/Counter app
- Display board + audio announcements
- Notifications (Email/SMS; WA later)
- Admin console (multi-tenant configuration)
- Analytics (real-time + historical)

Primary design goals:
- **Single-node deploy** (branch mini-PC) and **cloud deploy** with the same artifacts
- **Old device compatibility** (IE11 possible) via WebSocket + fallback (SockJS) and optional legacy build
- **Low operational cost**: start with **Postgres-only** + optional Redis; avoid Kafka initially

---

## 2) Default Stack (Chosen)
### Backend
- Language: **Go**
- Service shape: **Monolith modular** (single binary/container) for MVP, with clean module boundaries for later extraction
- API style: REST + WebSocket (SockJS)
- Job processing: background worker inside the same service (can be split later)

### Data
- DB: **PostgreSQL**
- Optional cache: Redis (phase 2 if needed for rate limit/locks)

### Realtime
- **SockJS** endpoint (WebSocket primary + transport fallbacks)
- Message format: JSON events + versioned schema

### Frontend
- Web apps: **Admin**, **Agent**, **Kiosk**, **Display**, optional **Public Web** (virtual queue)
- Build: Vite
- **Legacy build optional** for older browsers

### Edge
- Reverse proxy + TLS: **Caddy**
- Deployment: **Docker Compose** (cloud VPS / on‑prem mini-PC)

---

## 3) Deployment Modes
### Mode A — Branch On‑Prem (Single-node)
**Use case**: Internet unreliable, branch must operate autonomously.

Runs on 1 mini-PC (Intel NUC / similar):
- `qms-api` (Go monolith: API + realtime + worker)
- `postgres`
- `caddy`

Benefits:
- Works offline from cloud
- Lowest failure surface

Tradeoffs:
- Central reporting across branches requires periodic sync (optional later)

### Mode B — Cloud Central (Multi-branch)
**Use case**: Internet stable, centralized management.

Runs in 1 VPS (MVP) or multi-node (later):
- `qms-api`
- `postgres`
- `caddy`

Branches access via browser devices.

### Mode C — Hybrid (Later)
Branch keeps local queue engine + sync to cloud analytics.

---

## 4) High-Level Architecture
### Components
1. **QMS API (monolith)**
   - Auth & RBAC
   - Tenant/branch/service config
   - Queue Engine
   - Ticket lifecycle + actions
   - Realtime gateway (SockJS)
   - Notification dispatcher (email/sms)
   - Analytics aggregator (basic)
   - Audit logging

2. **PostgreSQL**
   - Source of truth for tickets, events, config, audit, notification status, analytics aggregates

3. **Caddy**
   - TLS termination + reverse proxy
   - Static asset hosting (optional)

### Data Flow (core)
- Ticket create/action → DB transaction → enqueue event in **outbox** → realtime + notif + analytics consume outbox
- Display/Agent subscribe to realtime → update UI

---

## 5) Repository Layout (Recommended)
```
/
  apps/
    admin-web/
    agent-web/
    kiosk-app/
    display-app/
    public-web/        # optional
  services/
    qms-api/           # Go monolith (modules inside)
  packages/
    shared-types/      # shared DTO schemas (jsonschema or TS types)
    sdk/               # client + webhook verifier (optional)
  infra/
    docker/
      compose.yml
      caddy/
  docs/
    PRD_Omnichannel_QMS.md
    QMS_Backlog_Epics_UserStories_AcceptanceTests.md
    TODO.md
    AGENT_RULES.md
    ADR/
  .github/workflows/
```

---

## 6) Core Domain Model
### Entities
- **Tenant**: `tenant_id`, name, branding, feature flags, pii_policy
- **Branch**: `branch_id`, tenant_id, name, address(optional), tz, hours
- **Area**: `area_id`, branch_id, name (floor/zone)
- **Service**: `service_id`, branch_id, name, code, sla_minutes, priority_policy, hours, capacity
- **Counter/Room**: `counter_id`, branch_id, area_id, name, mapped services, status
- **User**: `user_id`, tenant_id, role_id, branch scope, credentials/SSO mapping (later)
- **Role**: `role_id`, permissions
- **Device**: `device_id`, type(kiosk/display), branch_id, area_id, status, last_seen, config_version
- **Ticket**
- **TicketEvent** (append-only recommended; optional for MVP but schema prepared)
- **Appointment** (MVP minimal or phase 2)
- **Notification**: record + delivery status
- **AuditLog**

### Ticket Fields (Minimum)
- `ticket_id` (UUID)
- `ticket_number` (human-readable, per branch/service)
- `tenant_id`, `branch_id`, `service_id`, `area_id` (optional)
- `status`: `waiting | called | serving | done | no_show | cancelled | held`
- `channel`: `kiosk | web | mobile | staff | api`
- `priority_class`: `regular | priority | vip | emergency` (configurable)
- timestamps: `created_at`, `called_at`, `served_at`, `completed_at`
- assignment: `counter_id` (nullable)
- customer_ref (optional): `phone_hash` / `external_id` (PII off by default)
- `version` (optimistic concurrency) or use row lock in critical ops

---

## 7) Queue Engine Specification
### State Machine
Allowed transitions:
- `waiting → called → serving → done`
- `waiting → cancelled`
- `called → no_show` (manual or auto)
- `waiting → held → waiting`
- `called/serving → transfer → waiting` (new service/counter)

### Call Next (Concurrency-safe)
**Goal**: two agents pressing “Call next” cannot get the same ticket.

Implementation options (pick 1; recommended A):
- A) SQL row lock: `SELECT ... FOR UPDATE SKIP LOCKED` on waiting tickets by service/priority
- B) Optimistic concurrency with `version` + retry loop
- C) Redis lock (phase 2)

Recommended:
- Use **A** within a DB transaction to keep infra minimal.

### Routing (MVP)
- FIFO per service
- Priority basic + anti-starvation (configurable ratio, e.g. serve 1 regular after N priority)

### No‑show
- Config: `called_grace_period_seconds`
- Background job checks `called_at + grace` and marks `no_show` (or return-to-queue)

### Idempotency
All commands accept:
- `request_id` (UUID) — stored and deduped
- repeated requests return same response (no double events)

---

## 8) Eventing & Outbox
### Why outbox
Avoid distributed transactions while ensuring:
- event emission reliable
- notifications/analytics/realtime are eventually consistent

### Tables
- `outbox_events`:
  - `event_id`, `tenant_id`, `type`, `payload_json`, `created_at`, `processed_at`, `attempts`, `last_error`
- `ticket_events` (optional MVP, recommended):
  - `seq`, `ticket_id`, `type`, `payload`, `created_at`

### Worker
- Poll `outbox_events` in batches
- Publish to:
  - realtime broadcaster
  - notification pipeline
  - analytics aggregation

---

## 9) Realtime Protocol (SockJS)
### Endpoint
- `/realtime` (SockJS)
- Auth: cookie session or bearer token
- Subscription model: client sends `subscribe` message to topics

### Topics
- `branch:{branch_id}:display:{area_id|all}`
- `branch:{branch_id}:service:{service_id}`
- `counter:{counter_id}`
- `admin:{tenant_id}` (config updates)

### Message Envelope
```json
{
  "v": 1,
  "type": "ticket.called",
  "ts": "2026-01-12T10:00:00Z",
  "tenant_id": "...",
  "branch_id": "...",
  "payload": { }
}
```

### Client fallbacks
SockJS handles transport fallbacks. If a device cannot use SockJS (rare), provide **polling** endpoint:
- `GET /queues/snapshot?...` (every 3–5s)

---

## 10) Public REST API (High-Level)
> OpenAPI spec MUST be generated from this contract.

### Authentication
- MVP: username/password or magic link (tenant configurable)
- Tokens: JWT (short-lived) + refresh (optional)
- Enterprise: OIDC/SAML (phase 2/3)

### Endpoints (MVP)
- `POST /api/tickets`
- `GET /api/tickets/{ticket_id}`
- `POST /api/tickets/{ticket_id}/actions`
  - actions: `call_next`, `recall`, `start`, `complete`, `hold`, `unhold`, `transfer`, `cancel`, `mark_no_show`
- `GET /api/queues?branch_id=&service_id=`
- `GET /api/display/snapshot?branch_id=&area_id=`
- Admin:
  - `POST/GET /api/branches`
  - `POST/GET /api/services`
  - `POST/GET /api/counters`
  - `POST/GET /api/devices`
  - `POST/GET /api/users`
  - `GET /api/audit-logs`
- Analytics:
  - `GET /api/analytics/kpi?...`
  - `GET /api/analytics/export?...`

### Error format
```json
{
  "request_id": "uuid",
  "error": { "code": "INVALID_TRANSITION", "message": "..." }
}
```

---

## 11) Frontend Apps Specification
### 11.1 Agent App
- Purpose: call/serve tickets
- Realtime: subscribe to assigned services/counter updates
- Key screens:
  - Login
  - Queue list + filter
  - “Now Serving” panel
  - Actions (call/recall/start/complete/transfer/hold/no-show)
  - Presence status

### 11.2 Admin Console
- Purpose: configure tenant/branch/services/counters/devices/users
- Needs:
  - RBAC gating
  - audit log viewer
  - config changes should be versioned

### 11.3 Kiosk App
- Purpose: self-service ticket issuance
- Requirements:
  - large touch targets
  - multi-language
  - print ticket + QR (fallback QR on screen)
  - device health + auto reconnect
  - limited offline buffer + reconcile

### 11.4 Display App
- Purpose: show now calling + last N, play audio
- Realtime subscribe + resync snapshot
- Audio:
  - TTS or pre-recorded assets
  - recall support
  - quiet hours (enterprise)

### 11.5 Public Web (Optional MVP)
- Purpose: virtual queue join + status tracking
- PWA-friendly, minimal data, no PII by default

---

## 12) Notifications Specification
### Providers
- MVP: Email + SMS (adapter pattern)
- Enterprise: WhatsApp template messaging, push notifications

### Pipeline
1. Trigger event (ticket created/called/X remaining)
2. Render template (multi-language)
3. Send via provider adapter
4. Record status: `queued → sent → delivered/failed`
5. Retry transient failures + DLQ for permanent errors

### Templates
- Stored per tenant, versioned
- Variables: `{ticket_number}`, `{branch_name}`, `{service_name}`, `{counter_name}`, `{eta_minutes}`

---

## 13) Analytics Specification (MVP)
### Metrics
- wait_time = called_at - created_at
- service_time = completed_at - served_at
- throughput by service/branch/time bucket
- no_show_rate, cancellation_rate
- SLA compliance (<= sla_minutes)

### Implementation
- Consume outbox events
- Write aggregates to `analytics_daily` / `analytics_hourly` tables
- Real-time dashboard can query cached counts (or compute quickly)

### Export
- CSV generation server-side with streaming response

---

## 14) Multi-Tenancy, RBAC, Audit
### Multi-tenant rules
- Every table includes `tenant_id`
- Every query is scoped by `tenant_id` (enforced in DAL)
- Optional later: Postgres Row Level Security (RLS)

### RBAC
Roles (MVP):
- Admin (tenant-wide)
- Supervisor (branch-wide)
- Agent (counter/service scope)

Permission matrix must live in `docs/permissions.md`.

### Audit log
Record for:
- config changes
- privileged ticket actions (transfer/override)
Fields:
- actor_user_id, role, action_type, target_id, before/after (optional), timestamp, ip/user-agent

---

## 15) Security Requirements (Baseline)
- TLS required in production (Caddy)
- Password hashing (Argon2/bcrypt), MFA optional
- Rate limiting per IP + per tenant
- Input validation for all APIs
- CSRF protection for cookie-based sessions
- Secrets via environment variables, never in repo
- Log redaction: avoid PII in logs by default

---

## 16) Observability Requirements (Baseline)
- Structured logs (JSON) with `request_id`, `tenant_id`, `branch_id`
- Metrics:
  - HTTP latency p95, error rate
  - queue lag (outbox backlog)
  - ws connections count
  - notification failures
  - DB pool saturation
- Tracing: OpenTelemetry (recommended)

---

## 17) Performance Targets (MVP)
- Create ticket p95 ≤ 300ms (local network)
- Call → display update ≤ 1s (local)
- Dashboard load ≤ 2s
- Notification enqueue ≤ 1s (delivery depends on provider)

---

## 18) Offline & Degraded Mode (Branch On-Prem)
### Kiosk offline buffer
- If API unreachable:
  - generate temporary local ticket number (prefixed, e.g. `L-...`)
  - print ticket
  - queue locally
- On reconnect:
  - sync to API with idempotency key
  - reconcile ticket_number if needed (policy: keep printed number as “display number”)

### Display degraded
- Cache last snapshot
- Reconnect loop; show “Connection lost” banner

---

## 19) CI/CD & Testing
### CI
- Lint, unit tests, build on PR
- Integration tests nightly (compose stack)
- Security scans (SCA + secret scan)

### Testing
- Unit: queue selection, state machine, template rendering
- Integration: call-next concurrency, outbox processing
- E2E: kiosk issue ticket, agent call/serve, display update, admin config, analytics export
- Load tests for create ticket + call next + realtime fanout

---

## 20) Upgrade Path (When scale grows)
- Extract worker into separate service
- Add Redis for rate limit + ephemeral cache
- Add message bus (NATS) when outbox polling becomes bottleneck
- Add SSO (OIDC/SAML), approvals, fleet management, scheduled reports

---

## 21) Deliverables Checklist
- [ ] `OpenAPI` spec generated & versioned
- [ ] `docs/permissions.md`
- [ ] `docs/adr/*` for key decisions
- [ ] `docker-compose.yml` for cloud + on-prem variants
- [ ] `RUNBOOKS/` (backup restore, incident triage, provider outage)
- [ ] `AGENT_RULES.md` in repo root
