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
- [ ] Buat struktur repo + folder `docs/` + pindahkan PRD & backlog ke repo
- [ ] Pilih tech stack + tulis ADR:
  - [ ] ADR-001: Stack backend (mis. Go/Node/Java), DB (Postgres), cache (Redis), message bus (Kafka/Rabbit/NATS)
  - [ ] ADR-002: Real-time (SSE/WebSocket) + fallback
  - [ ] ADR-003: Multi-tenant strategy (tenant_id on every row + RLS optional)
  - [ ] ADR-004: Event model (TicketEvent append-only vs state table + events)
- [ ] Setup dev environment:
  - [ ] `docker compose up` untuk local (db, cache, broker, mailhog)
  - [ ] `.env.example` + secrets policy (jangan commit secrets)
- [ ] CI baseline:
  - [ ] Lint + unit test + typecheck
  - [ ] Build artifacts (web apps + services)
  - [ ] Dependency scan (SCA) + secret scan
- [ ] Quality gates:
  - [ ] Conventional Commits
  - [ ] Pre-commit hook (format, lint, tests ringan)

### M1 — MVP Production (Phase 1)
Target: **Ticketing omnichannel + queue engine + agent app + display + basic notif + admin config + basic analytics**

- [ ] Queue Engine MVP (E-QUEUE-* [MVP])
- [ ] Agent App MVP (E-AGENT-* [MVP])
- [ ] Kiosk MVP (E-KIOSK-* [MVP])
- [ ] Display + Audio MVP (E-DISP-* [MVP])
- [ ] Notifications MVP (E-NOTIF-* [MVP])
- [ ] Admin Console MVP (E-ADMIN-* [MVP])
- [ ] Analytics MVP (E-ANALYT-* [MVP])
- [ ] Hardening: Observability + Security + HA/DR baseline (E-PLAT-* [MVP])

### M2 — Enterprise (Phase 2/3)
- [ ] SSO (OIDC/SAML), approval workflow, fleet ops, WA/push, skill-based routing, advanced appointment blending
- [ ] Scheduled reports + BI connectors, anomaly detection rules

---

## 2) Global TODO (lintas modul) — wajib untuk production

### 2.1 Coding standards
- [ ] Tetapkan style guide:
  - [ ] Backend: error format, logging format, idempotency pattern
  - [ ] Frontend: component conventions, state management
- [ ] Semua API response **terstruktur** (error code + message + request_id)
- [ ] Semua command/action memiliki **idempotency key (`request_id`)**
- [ ] Semua perubahan ticket menghasilkan **event** (untuk display/analytics/notif)

### 2.2 Security baseline
- [ ] RBAC (Admin/Supervisor/Agent) + permission matrix di `docs/`
- [ ] Rate limiting per tenant/branch + per IP
- [ ] Input validation + protection dasar (CSRF untuk web, SSRF blocklist, SQLi guard via ORM/param)
- [ ] TLS everywhere (prod) + encryption at rest (KMS/managed)
- [ ] Audit log untuk:
  - [ ] config change
  - [ ] privileged actions (transfer/override)
- [ ] Secrets management:
  - [ ] tidak ada secret di repo
  - [ ] rotation plan

### 2.3 Observability & ops
- [ ] OpenTelemetry tracing (service-to-service)
- [ ] Metrics minimal: latency p95, error rate, queue event lag, ws connections, notif failures
- [ ] Structured logs + correlation id
- [ ] Alert rules + runbooks minimal:
  - [ ] queue lag tinggi
  - [ ] notif failure spike
  - [ ] db connection saturation
  - [ ] device offline surge

### 2.4 Data & migrations
- [ ] DB migrations tool + policy:
  - [ ] forward-only migrations
  - [ ] rollback plan via new migration (bukan revert manual)
- [ ] Data retention + anonymization jobs (opsional di MVP, wajib sebelum tenant simpan PII)
- [ ] Timezone policy: store UTC, render per tenant/branch

---

## 3) Module TODO — MVP vs Enterprise (ringkas & operasional)
> Checklist ini mapping ke backlog epics/stories yang sudah dibuat.

---

# 3A) services/queue-service (Queue Engine)

## MVP
- [ ] Implement Ticket state machine + transitions:
  - [ ] Enqueue
  - [ ] Call next (concurrency-safe)
  - [ ] Start serving
  - [ ] Complete
  - [ ] Cancel (waiting only)
- [ ] Implement actions:
  - [ ] Recall
  - [ ] Hold/Unhold
  - [ ] Skip/No-show manual
  - [ ] Transfer service/counter
  - [ ] Auto no-show via scheduler/grace timer
- [ ] Implement routing:
  - [ ] FIFO per service
  - [ ] Priority basic + anti-starvation minimal
- [ ] Idempotency:
  - [ ] request_id stored + dedupe
- [ ] Events:
  - [ ] ticket.created/called/serving/done/transferred/no_show/recalled/held
- [ ] API:
  - [ ] POST /tickets
  - [ ] GET /tickets/:id
  - [ ] POST /tickets/:id/actions
  - [ ] GET /queues (by branch/service)
- [ ] Tests:
  - [ ] unit tests untuk state machine
  - [ ] integration tests untuk concurrency call next
  - [ ] negative tests (invalid transitions)

## Enterprise
- [ ] Skill-based routing (counter skills)
- [ ] Advanced appointment blending (quota/weight)
- [ ] Append-only TicketEvent store (audit-grade) + rehydrate

---

# 3B) apps/agent-web (Agent App)

## MVP
- [ ] Auth + role gating (Agent only)
- [ ] Queue view per service (filter, realtime update)
- [ ] “My counter” panel:
  - [ ] Call next
  - [ ] Recall
  - [ ] Start
  - [ ] Complete
  - [ ] Transfer
  - [ ] Hold
  - [ ] No-show
- [ ] Presence status (Available/Break) mempengaruhi eligibility counter
- [ ] UX hardening:
  - [ ] empty state queue kosong
  - [ ] conflict handling (409) + retry
- [ ] Telemetry:
  - [ ] action latency
  - [ ] error reasons

## Enterprise
- [ ] Supervisor mode (monitor semua counter + intervene)
- [ ] Multi-counter per agent
- [ ] SSO (jika diputuskan di auth layer)

---

# 3C) apps/kiosk-app (Kiosk)

## MVP
- [ ] Home → pilih bahasa → pilih layanan → confirm → issue ticket
- [ ] Print ticket + QR (fallback QR on-screen jika printer error)
- [ ] Optional input (phone/customer ref) sesuai tenant policy
- [ ] Accessibility basics (font besar, high contrast, timeout reset)
- [ ] Device resilience:
  - [ ] health check
  - [ ] reconnect loop
- [ ] Limited offline buffer + sync/reconcile

## Enterprise
- [ ] Remote config / fleet management
- [ ] Appointment check-in via QR scanner

---

# 3D) apps/display-app (Display & Audio)

## MVP
- [ ] Display board:
  - [ ] Now calling + last N
  - [ ] Config: per area/service
  - [ ] Reconnect + resync snapshot
- [ ] Audio:
  - [ ] TTS atau audio file
  - [ ] Recall re-announce

## Enterprise
- [ ] Quiet hours + audio rate limiting
- [ ] Signage playlist + schedule + branding pack

---

# 3E) services/notification-service (Notifications)

## MVP
- [ ] Template engine (multi-bahasa) + variable substitution
- [ ] Trigger pipeline:
  - [ ] ticket created
  - [ ] called
  - [ ] “X nomor lagi” (rule-based sederhana)
- [ ] Provider adapters:
  - [ ] Email
  - [ ] SMS (pluggable)
- [ ] Retry policy + DLQ
- [ ] Delivery status model (sent/failed)

## Enterprise
- [ ] WhatsApp template messaging
- [ ] Push notifications (mobile/PWA)
- [ ] Preference center per user (opt-in/out granular)

---

# 3F) apps/admin-web + services/admin-api (Admin Console)

## MVP
- [ ] CRUD:
  - [ ] Branch + Area
  - [ ] Service catalog (SLA, jam layanan, priority)
  - [ ] Counter/Room mapping
  - [ ] Users + Roles (RBAC)
  - [ ] Register devices (kiosk/display) + status
- [ ] Policy config:
  - [ ] no-show grace
  - [ ] return-to-queue
  - [ ] priority basic
- [ ] Audit log viewer (filter basic)

## Enterprise
- [ ] Approval workflow config changes
- [ ] Holiday calendar + blackout dates (appointment)
- [ ] Fleet ops (remote update, config versioning, rollback)

---

# 3G) services/analytics-service + dashboard (Analytics)

## MVP
- [ ] Event capture dari queue engine (ticket lifecycle)
- [ ] Aggregations:
  - [ ] wait_time, service_time
  - [ ] throughput, no-show rate, SLA compliance
- [ ] Real-time dashboard (≤5s refresh):
  - [ ] queue length
  - [ ] counter load
- [ ] Historical dashboard:
  - [ ] filter range tanggal/branch/service/priority/channel
  - [ ] export CSV

## Enterprise
- [ ] Scheduled reports via email
- [ ] BI connectors (API tokenized)
- [ ] Anomaly detection rules (threshold-based)

---

## 4) Engineering TODO (repo-wide)

### 4.1 CI/CD
- [ ] GitHub Actions:
  - [ ] lint + unit tests (PR)
  - [ ] build artifacts (PR)
  - [ ] integration tests (nightly/merge)
  - [ ] security scan (SCA + secret scan)
- [ ] CD:
  - [ ] staging auto deploy (merge to main)
  - [ ] production deploy (manual approval)
- [ ] Versioning:
  - [ ] semantic version + changelog
  - [ ] db migration gating

### 4.2 API documentation
- [ ] OpenAPI spec untuk semua services public
- [ ] Postman/Insomnia collection (optional)
- [ ] Webhook docs + signature verification contoh

### 4.3 Testing strategy
- [ ] Unit tests: state machine, routing, templates
- [ ] Integration tests: DB + broker + real-time
- [ ] E2E tests: kiosk flow, agent flow, admin config, display update
- [ ] Load test (p95 targets):
  - [ ] create ticket
  - [ ] call -> display update
  - [ ] notif enqueue

### 4.4 Release readiness
- [ ] UAT checklist + sign-off template
- [ ] Runbooks:
  - [ ] incident triage
  - [ ] restore backup
  - [ ] provider notif outage
  - [ ] device offline mass
- [ ] DR drill evidence (log + report)

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
- [ ] Mengacu ke story ID + link ke bagian TODO/backlog
- [ ] AC lulus (ditulis ulang di PR)
- [ ] Tests ditambah/diupdate
- [ ] DB migration aman (jika ada)
- [ ] Observability: log/metric/trace minimal
- [ ] Tidak ada breaking change tanpa catatan migrasi

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
- Default rate limit aktif (minimal).
- Default audit log aktif untuk config + privileged actions.

## 5.7 Larangan (hard constraints)
- Jangan men-disable test/lint untuk “cepat selesai”.
- Jangan menambahkan dependency besar tanpa alasan + catatan.
- Jangan mengubah kontrak API tanpa bump versi/kompatibilitas.
- Jangan menulis “TODO: security later” untuk hal baseline (auth, validation, RBAC).

---

## 6) Immediate next steps (paling efektif)
1. [ ] Commit dokumen: PRD + Backlog + TODO + AGENT_RULES
2. [ ] Buat ADR-001..004 (pilih stack & arsitektur)
3. [ ] Bootstrap CI + docker compose local
4. [ ] Implement MVP slice end-to-end paling tipis:
   - Create ticket → agent call next → display update → (optional) notif called
