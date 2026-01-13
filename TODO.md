# TODO.md — Omnichannel QMS (Production-Ready)
**Tanggal**: 12 Jan 2026 (Asia/Jakarta)  
**Repo goal**: Platform **multi-tenant** untuk **bank/klinik/layanan publik**: Kiosk + Web/Mobile + Agent App + Display + Notifications + Admin + Analytics + API.

> Referensi dokumen (letakkan di repo):
> - `docs/PRD_Omnichannel_QMS.md` (PRD)
> - `docs/QMS_Backlog_Epics_UserStories_AcceptanceTests.md` (Backlog epics/stories/AC)

---

## 0) Project structure (disarankan)
> Sesuaikan dengan stack yang kamu pilih. Struktur di bawah enak untuk kerja bareng AI coding agent (jelas boundary, mudah dites).

```
/
  apps/
    admin-web/
    agent-web/
    kiosk-app/
    display-app/
    public-web/        # optional: self-service ticketing web
  services/
    queue-service/
    realtime-service/  # ws/sse
    notification-service/
    analytics-service/
    auth-service/      # optional jika tidak pakai managed auth
  packages/
    shared-types/
    shared-utils/
    ui-kit/            # optional
    sdk/               # API client + webhook verifier
  infra/
    docker/
    k8s/               # optional
    terraform/         # optional
  docs/
    PRD_Omnichannel_QMS.md
    QMS_Backlog_Epics_UserStories_AcceptanceTests.md
    ADR/               # Architecture Decision Records
    RUNBOOKS/
  scripts/
  .github/workflows/
```

---

## 1) Milestones (prioritas)
### M0 — Bootstrap repo (wajib sebelum coding fitur)
- [x] Buat struktur repo + folder `docs/` + pindahkan PRD & backlog ke repo
- [x] Pilih tech stack + tulis ADR:
  - [x] ADR-001: Stack backend (mis. Go/Node/Java), DB (Postgres), cache (Redis), message bus (Kafka/Rabbit/NATS)
  - [x] ADR-002: Real-time (SSE/WebSocket) + fallback
  - [x] ADR-003: Multi-tenant strategy (tenant_id on every row + RLS optional)
  - [x] ADR-004: Event model (TicketEvent append-only vs state table + events)
- [x] Setup dev environment:
  - [x] `docker compose up` untuk local (db, cache, broker, mailhog)
  - [x] `.env.example` + secrets policy (jangan commit secrets)
- [x] CI baseline:
  - [x] Lint + unit test + typecheck
  - [x] Build artifacts (web apps + services)
  - [x] Dependency scan (SCA) + secret scan
- [x] Quality gates:
  - [x] Conventional Commits
  - [x] Pre-commit hook (format, lint, tests ringan)

### M1 — MVP Production (Phase 1)
Target: **Ticketing omnichannel + queue engine + agent app + display + basic notif + admin config + basic analytics**

- [x] Queue Engine MVP (E-QUEUE-* [MVP])
- [x] Agent App MVP (E-AGENT-* [MVP])
- [x] Kiosk MVP (E-KIOSK-* [MVP])
- [x] Display + Audio MVP (E-DISP-* [MVP])
- [x] Notifications MVP (E-NOTIF-* [MVP])
- [x] Admin Console MVP (E-ADMIN-* [MVP])
- [x] Analytics MVP (E-ANALYT-* [MVP])
- [x] Hardening: Observability + Security + HA/DR baseline (E-PLAT-* [MVP])

### M2 — Enterprise (Phase 2/3)
- [x] SSO (OIDC/SAML), approval workflow, fleet ops, WA/push, skill-based routing, advanced appointment blending
- [x] Scheduled reports + BI connectors, anomaly detection rules

---

## 2) Global TODO (lintas modul) — wajib untuk production

### 2.1 Coding standards
- [x] Tetapkan style guide:
  - [x] Backend: error format, logging format, idempotency pattern
  - [x] Frontend: component conventions, state management
- [x] Semua API response **terstruktur** (error code + message + request_id)
- [x] Semua command/action memiliki **idempotency key (`request_id`)**
- [x] Semua perubahan ticket menghasilkan **event** (untuk display/analytics/notif)

### 2.2 Security baseline
- [x] RBAC (Admin/Supervisor/Agent) + permission matrix di `docs/`
- [x] Rate limiting per tenant/branch + per IP
- [x] Input validation + protection dasar (CSRF untuk web, SSRF blocklist, SQLi guard via ORM/param)
- [x] TLS everywhere (prod) + encryption at rest (KMS/managed)
- [x] Audit log untuk:
  - [x] config change
  - [x] privileged actions (transfer/override)
- [x] Secrets management:
  - [x] tidak ada secret di repo
  - [x] rotation plan

### 2.3 Observability & ops
- [x] OpenTelemetry tracing (service-to-service)
- [x] Metrics minimal: latency p95, error rate, queue event lag, ws connections, notif failures
- [x] Structured logs + correlation id
- [x] Alert rules + runbooks minimal:
  - [x] queue lag tinggi
  - [x] notif failure spike
  - [x] db connection saturation
  - [x] device offline surge

### 2.4 Data & migrations
- [x] DB migrations tool + policy:
  - [x] forward-only migrations
  - [x] rollback plan via new migration (bukan revert manual)
- [x] Data retention + anonymization jobs (opsional di MVP, wajib sebelum tenant simpan PII)
- [x] Timezone policy: store UTC, render per tenant/branch

---

## 3) Module TODO — MVP vs Enterprise (ringkas & operasional)
> Checklist ini mapping ke backlog epics/stories yang sudah dibuat.

---

# 3A) services/queue-service (Queue Engine)

## MVP
- [x] Implement Ticket state machine + transitions:
  - [x] Enqueue
  - [x] Call next (concurrency-safe)
  - [x] Start serving
  - [x] Complete
  - [x] Cancel (waiting only)
- [x] Implement actions:
  - [x] Recall
  - [x] Hold/Unhold
  - [x] Skip/No-show manual
  - [x] Transfer service/counter
  - [x] Auto no-show via scheduler/grace timer
- [x] Implement routing:
  - [x] FIFO per service
  - [x] Priority basic + anti-starvation minimal
- [x] Idempotency:
  - [x] request_id stored + dedupe
- [x] Events:
  - [x] ticket.created/called/serving/done/transferred/no_show/recalled/held
- [x] API:
  - [x] POST /tickets
  - [x] GET /tickets/:id
  - [x] POST /tickets/:id/actions
  - [x] GET /queues (by branch/service)
- [x] Tests:
  - [x] unit tests untuk state machine
  - [x] integration tests untuk concurrency call next
  - [x] negative tests (invalid transitions)

## Enterprise
- [x] Skill-based routing (counter skills)
- [x] Advanced appointment blending (quota/weight)
- [x] Append-only TicketEvent store (audit-grade) + rehydrate

---

# 3B) apps/agent-web (Agent App)

## MVP
- [x] Auth + role gating (Agent only)
- [x] Queue view per service (filter, realtime update)
- [x] “My counter” panel:
  - [x] Call next
  - [x] Recall
  - [x] Start
  - [x] Complete
  - [x] Transfer
  - [x] Hold
  - [x] No-show
- [x] Presence status (Available/Break) mempengaruhi eligibility counter
- [x] UX hardening:
  - [x] empty state queue kosong
  - [x] conflict handling (409) + retry
- [x] Telemetry:
  - [x] action latency
  - [x] error reasons

## Enterprise
- [x] Supervisor mode (monitor semua counter + intervene)
- [x] Multi-counter per agent
- [x] SSO (jika diputuskan di auth layer)

---

# 3C) apps/kiosk-app (Kiosk)

## MVP
- [x] Home → pilih bahasa → pilih layanan → confirm → issue ticket
- [x] Print ticket + QR (fallback QR on-screen jika printer error)
- [x] Optional input (phone/customer ref) sesuai tenant policy
- [x] Accessibility basics (font besar, high contrast, timeout reset)
- [x] Device resilience:
  - [x] health check
  - [x] reconnect loop
- [x] Limited offline buffer + sync/reconcile

## Enterprise
- [x] Remote config / fleet management
- [x] Appointment check-in via QR scanner

---

# 3D) apps/display-app (Display & Audio)

## MVP
- [x] Display board:
  - [x] Now calling + last N
  - [x] Config: per area/service
  - [x] Reconnect + resync snapshot
- [x] Audio:
  - [x] TTS atau audio file
  - [x] Recall re-announce

## Enterprise
- [x] Quiet hours + audio rate limiting
- [x] Signage playlist + schedule + branding pack

---

# 3E) services/notification-service (Notifications)

## MVP
- [x] Template engine (multi-bahasa) + variable substitution
- [x] Trigger pipeline:
  - [x] ticket created
  - [x] called
  - [x] “X nomor lagi” (rule-based sederhana)
- [x] Provider adapters:
  - [x] Email
  - [x] SMS (pluggable)
- [x] Retry policy + DLQ
- [x] Delivery status model (sent/failed)

## Enterprise
- [x] WhatsApp template messaging
- [x] Push notifications (mobile/PWA)
- [x] Preference center per user (opt-in/out granular)

---

# 3F) apps/admin-web + services/admin-api (Admin Console)

## MVP
- [x] CRUD:
  - [x] Branch + Area
  - [x] Service catalog (SLA, jam layanan, priority)
  - [x] Counter/Room mapping
  - [x] Users + Roles (RBAC)
  - [x] Register devices (kiosk/display) + status
- [x] Policy config:
  - [x] no-show grace
  - [x] return-to-queue
  - [x] priority basic
- [x] Audit log viewer (filter basic)

## Enterprise
- [x] Approval workflow config changes
- [x] Holiday calendar + blackout dates (appointment)
- [x] Fleet ops (remote update, config versioning, rollback)

---

# 3G) services/analytics-service + dashboard (Analytics)

## MVP
- [x] Event capture dari queue engine (ticket lifecycle)
- [x] Aggregations:
  - [x] wait_time, service_time
  - [x] throughput, no-show rate, SLA compliance
- [x] Real-time dashboard (≤5s refresh):
  - [x] queue length
  - [x] counter load
- [x] Historical dashboard:
  - [x] filter range tanggal/branch/service/priority/channel
  - [x] export CSV

## Enterprise
- [x] Scheduled reports via email
- [x] BI connectors (API tokenized)
- [x] Anomaly detection rules (threshold-based)

---

## 4) Engineering TODO (repo-wide)

### 4.1 CI/CD
- [x] GitHub Actions:
  - [x] lint + unit tests (PR)
  - [x] build artifacts (PR)
  - [x] integration tests (nightly/merge)
  - [x] security scan (SCA + secret scan)
- [x] CD:
  - [x] staging auto deploy (merge to main)
  - [x] production deploy (manual approval)
- [x] Versioning:
  - [x] semantic version + changelog
  - [x] db migration gating

### 4.2 API documentation
- [x] OpenAPI spec untuk semua services public
- [x] Postman/Insomnia collection (optional)
- [x] Webhook docs + signature verification contoh

### 4.3 Testing strategy
- [x] Unit tests: state machine, routing, templates
- [x] Integration tests: DB + broker + real-time
- [x] E2E tests: kiosk flow, agent flow, admin config, display update
- [x] Load test (p95 targets):
  - [x] create ticket
  - [x] call -> display update
  - [x] notif enqueue

### 4.4 Release readiness
- [x] UAT checklist + sign-off template
- [x] Runbooks:
  - [x] incident triage
  - [x] restore backup
  - [x] provider notif outage
  - [x] device offline mass
- [x] DR drill evidence (log + report)

---

# 5) AGENT_RULES.md (untuk AI Coding Agent)
> Copy bagian ini ke `AGENT_RULES.md` di repo. Ini aturan kerja yang membuat agent stabil, aman, dan “production-minded”.

## 5.1 Mode kerja (wajib)
1. **Selalu baca** `docs/PRD_Omnichannel_QMS.md`, `docs/QMS_Backlog_*.md`, dan `TODO.md` sebelum mulai.
2. Kerjakan **1 user story** atau **1 subtask kecil** per PR/commit (diff kecil, mudah review).
3. **Jaga main branch tetap hijau**: jangan merge kalau test/lint gagal.
4. Jika ada ambiguity, buat **ADR kecil** atau catatan di PR; jangan bikin keputusan diam-diam.

## 5.2 Aturan perubahan kode
- Jangan commit secrets / token / password. Pakai `.env.example`.
- Semua endpoint/action harus:
  - validasi input
  - error terstruktur (code + message + request_id)
  - audit/event bila relevan
- Semua perubahan ticket harus memicu event (untuk display/notif/analytics).
- Semua command/action harus idempotent via `request_id`.
- Tambahkan tests untuk:
  - happy path
  - invalid transition
  - concurrency edge case (minimal untuk call next)

## 5.3 Branching & commit
- Branch name: `feat/<module>-<short>` / `fix/<module>-<short>` / `chore/<short>`
- Commit message: **Conventional Commits**
  - `feat(queue): implement call-next concurrency guard`
  - `fix(notif): handle missing template variable`
- Setiap PR harus menyebut ID story (mis. `US-QUEUE-002`).

## 5.4 PR checklist (wajib)
- [x] Mengacu ke story ID + link ke bagian TODO/backlog
- [x] AC lulus (ditulis ulang di PR)
- [x] Tests ditambah/diupdate
- [x] DB migration aman (jika ada)
- [x] Observability: log/metric/trace minimal
- [x] Tidak ada breaking change tanpa catatan migrasi

## 5.5 Gaya implementasi (best practices)
- Prefer **pure functions** untuk routing/selection logic → gampang dites.
- Hindari shared mutable state di realtime handler; gunakan cache yang aman.
- Gunakan **transaction**/locking strategy untuk “call next” agar concurrency aman.
- Pisahkan:
  - command handler (write)
  - query handler (read)
- Buat adapter pattern untuk provider notif (SMS/Email/WA).

## 5.6 Kebijakan “safe defaults”
- Default tanpa PII; bila tenant mengaktifkan PII, harus ada masking + retention.
- [x] Default rate limit aktif (minimal).
- Default audit log aktif untuk config + privileged actions.

## 5.7 Larangan (hard constraints)
- Jangan men-disable test/lint untuk “cepat selesai”.
- Jangan menambahkan dependency besar tanpa alasan + catatan.
- Jangan mengubah kontrak API tanpa bump versi/kompatibilitas.
- Jangan menulis “TODO: security later” untuk hal baseline (auth, validation, RBAC).

---

## 6) Immediate next steps (paling efektif)
1. [x] Commit dokumen: PRD + Backlog + TODO + AGENT_RULES
2. [x] Buat ADR-001..004 (pilih stack & arsitektur)
3. [x] Bootstrap CI + docker compose local
4. [x] Implement MVP slice end-to-end paling tipis:
   - Create ticket → agent call next → display update → (optional) notif called
