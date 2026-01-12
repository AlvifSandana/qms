# PRD — Omnichannel Queue Management Platform (QMS)

**Tanggal**: 12 Jan 2026 (Asia/Jakarta)  
**Versi**: v1.0 (Production baseline)  
**Target industri**: Bank, Klinik/RS, Layanan publik  
**Channel**: Kiosk + Web + Mobile/PWA + Display + API

---

## 0) Ringkasan
Platform ini mengelola antrian **omnichannel** (walk-in + virtual queue + appointment) untuk banyak tenant dan banyak cabang (multi-tenant), dengan modul:
- Ticketing omnichannel (kiosk/web/mobile/staff/API)
- Queue engine (routing, prioritas, fairness, no-show)
- Agent/Counter app (pemanggilan dan eksekusi layanan)
- Display & audio announcement
- Notifications (SMS/WhatsApp/email/push)
- Admin console (konfigurasi & user/role)
- Analytics & reporting
- Observability + security + HA/DR baseline

---

## 1) Latar belakang & masalah
Masalah yang ingin diselesaikan:
- Waktu tunggu tinggi, crowding, dan pengalaman pelanggan/pasien yang buruk
- Ketidakmerataan beban antar counter/ruang
- Minim visibilitas real-time dan data historis untuk perbaikan proses
- Walk-in + appointment sering tidak sinkron (klinik) dan menimbulkan konflik prioritas

---

## 2) Tujuan & KPI
### 2.1 Tujuan bisnis
- Menurunkan waktu tunggu & kepadatan
- Meningkatkan kepuasan pelanggan/pasien
- Meningkatkan produktivitas petugas dan utilisasi counter/ruang
- Menyediakan data operasional untuk staffing & SLA

### 2.2 KPI inti (minimum)
- Avg/Median Wait Time, Max Wait Time
- Avg Service Time
- Abandonment / No-show Rate
- Throughput (ticket served/jam)
- SLA compliance per layanan (mis. 80% ticket dilayani < X menit)

---

## 3) Ruang lingkup
### 3.1 In-scope (v1 production)
- Multi-tenant + multi-branch + multi-area
- Ticketing omnichannel: kiosk/web/mobile/PWA/staff/API
- Appointment + walk-in hybrid
- Agent app: call/recall/skip/hold/transfer/complete
- Display board + audio announcement (TTS/rekaman)
- Notifications & template per bahasa (SMS/WhatsApp/email/push)
- Admin konfigurasi: layanan, counter/ruang, jadwal, prioritas, user/role
- Analytics real-time & historis + export
- Security baseline + audit log + observability + HA/DR

### 3.2 Out-of-scope (v1)
- Pemrosesan pembayaran/claim (BPJS/core banking)
- EMR/HIS penuh (hanya integrasi)
- Face recognition/biometric
- AI triage/decisioning (future)

---

## 4) Persona
1. **Customer/Patient**: ambil nomor/booking, lihat estimasi, dapat notifikasi, dipanggil tepat waktu  
2. **Front desk/Agent**: buat & kelola ticket, pemanggilan, transfer, catatan kunjungan  
3. **Supervisor/Manager**: monitoring real-time, staffing, SLA, laporan, konfigurasi operasional  
4. **Tenant Admin**: konfigurasi layanan/cabang, user & role, template notif & display, audit  
5. **IT/Implementer**: setup device, integrasi API/webhook, SSO, observability

---

## 5) User journeys (end-to-end)
### A) Walk-in via kiosk
1. Pilih bahasa → pilih layanan → (opsional) input nomor HP/ID
2. Dapat ticket (print + QR) + estimasi waktu tunggu
3. Monitor display/HP → dipanggil → dilayani → selesai → feedback

### B) Virtual queue (pre-arrival via web/mobile)
1. Pilih cabang + layanan → ambil ticket virtual
2. Terima notifikasi “X nomor lagi / mendekati giliran”
3. Datang ke lokasi → check-in (QR/link) → dipanggil → dilayani

### C) Appointment (hybrid)
1. Pilih layanan + slot waktu → booking
2. Reminder otomatis → check-in
3. Masuk antrean appointment sesuai aturan blending (quota/weight)

### D) Operasional petugas
- Login → set status (available/break) → call next → start → complete
- Recall / Transfer / Skip / No-show sesuai kebijakan

---

## 6) Functional Requirements (FR)

### FR-01 Multi-tenant & branch management
- Struktur: **Tenant → Branch → Area/Floor → Service → Counter/Room**
- Branding per tenant: logo, warna, layout display, bahasa
- Konfigurasi berbeda per cabang (jam operasional, layanan, counter)

### FR-02 Service catalog & routing rules
- Service attributes:
  - kode/nama, SLA target, jam layanan, kapasitas, prioritas default
- Routing:
  - FIFO per service
  - skill-based routing (counter A bisa layanan X/Y)
  - load balancing antar counter
- Priority classes (konfigurable): Regular / Priority / VIP / Emergency
- Anti-starvation: aturan agar queue regular tidak “kelamaan” kalah oleh priority

### FR-03 Ticketing omnichannel
**Channel supported**
- Kiosk (touch UI + printer + QR)
- Web (responsive)
- Mobile App / PWA (status real-time + push)
- Staff create (front desk)
- API (integrasi eksternal)

**Ticket fields minimum**
- `ticket_id` (UUID)
- `ticket_number` (human readable, per service/branch, configurable format)
- `tenant_id`, `branch_id`, `service_id`, `area_id` (opsional)
- `status`: waiting/called/serving/done/no-show/cancelled/held
- `channel`: kiosk/web/mobile/staff/api
- timestamps: created_at, called_at, served_at, completed_at
- optional customer_ref (PII optional, configurable): phone/patient_id/cif (harus bisa anonymized)

### FR-04 Appointment scheduling (hybrid)
- Create appointment slot time + kuota per slot/service
- Check-in via QR/link
- Kebijakan:
  - grace period terlambat
  - no-show handling
  - blending appointment vs walk-in:
    - quota (mis 60% appointment per jam), atau
    - weight/priority (mis appointment diutamakan saat dekat slot)
- Strategy configurable:
  - appointment masuk antrean saat check-in, atau
  - auto-enqueue H-XX menit sebelum slot

### FR-05 Queue engine (core actions & state machine)
**Actions**
- Enqueue, Call/Dequeue, Recall, Hold/Unhold, Skip, Transfer, Cancel, Complete
- Idempotency: semua action menerima `request_id` untuk mencegah double submit

**State machine minimum**
- waiting → called → serving → done
- waiting/called → no-show (auto/manual)
- waiting → cancelled
- called/serving → transfer → waiting (pada service baru)

**No-show handling**
- `called_grace_period` (mis 2–5 menit) configurable
- opsi:
  - mark no-show, atau
  - return-to-queue (kembali antre dengan aturan tertentu)

### FR-06 Agent/Counter app (web)
- Auth + role
- View queue per service + “my counter”
- One-click actions:
  - Call next, Recall, Start service, Complete
  - Skip/No-show, Transfer service/counter, Hold
- Presence:
  - Available / Busy / Break / Offline
- Tagging & notes (optional) untuk reason / disposition

### FR-07 Display & audio announcement
- Display modes:
  - Lobby (multi-service)
  - Per-area/per-floor
  - Per-service board
- Display content:
  - Now calling (ticket + counter/room)
  - Recently called (last N)
  - Next up (optional)
  - Estimasi wait time (optional)
- Audio:
  - TTS multi-bahasa atau audio file
  - Recall support
- Digital signage playlist integration (optional)

### FR-08 Notifications (SMS/WhatsApp/email/push)
- Trigger events:
  - ticket created
  - “X nomor lagi”
  - called
  - appointment reminder
  - cancel/reschedule
- Template engine:
  - multi-bahasa
  - variable substitution (ticket_number, branch_name, counter, ETA)
- Preference center: opt-in/opt-out per channel
- Retry policy + dead-letter queue (DLQ) untuk kegagalan provider

### FR-09 Admin console
- CRUD: branch, area, service, counter/room, device, user, role
- Konfigurasi:
  - jam operasional + holiday calendar
  - SLA target per service
  - priority rules, blending appointment/walk-in
  - notification templates
  - display layout & content playlist
- Audit log viewer (filterable)

### FR-10 Analytics & reporting
- Real-time dashboard:
  - queue length, wait time, counter load, event lag
- Historical analytics:
  - per jam/hari/minggu, per service, per branch, per counter/agent
- KPI:
  - avg/median wait, avg service, abandonment/no-show, SLA compliance
- Export:
  - CSV/XLSX (minimal CSV)
  - API endpoints untuk BI tools (optional)
- Data retention configurable

### FR-11 Integrations
- REST API (minimal):
  - `POST /tickets`
  - `GET /tickets/{id}`
  - `POST /tickets/{id}/actions` (call/recall/transfer/complete/cancel/hold)
  - `POST /appointments`
  - `POST /appointments/{id}/checkin`
  - `GET /queues?branch_id=&service_id=`
- Webhooks outgoing:
  - ticket.created, ticket.called, ticket.completed, appointment.booked, notification.sent
  - Signed (HMAC) + retry + DLQ
- SSO (enterprise optional): OIDC/SAML

---

## 7) Non-Functional Requirements (NFR) — Production Ready

### NFR-01 Availability & DR
- SLO baseline: 99.9% monthly (target minimal)
- Backup + restore:
  - RPO ≤ 15 menit
  - RTO ≤ 60 menit
- DR drill: minimal 1x sebelum go-live

### NFR-02 Performance targets (p95)
- Create ticket (API): ≤ 300 ms
- Call → update display (real-time): ≤ 1 detik
- Dashboard initial load: ≤ 2 detik
- Notification enqueue: ≤ 1 detik (delivery tergantung provider)

### NFR-03 Scalability & capacity
- Multi-tenant horizontal scale
- Burst handling di jam sibuk:
  - ticket creation spikes
  - websocket connections spikes
- Rate limiting per tenant/branch

### NFR-04 Security baseline
- RBAC (least privilege)
- MFA untuk admin (recommended)
- TLS in-transit + encryption at-rest (KMS-managed keys)
- Secrets management + rotation
- Input validation + protections (CSRF, SSRF, injection)
- Audit log append-only (immutable storage recommended)

### NFR-05 Privacy & retention
- Default mode tanpa PII
- PII optional by tenant policy:
  - masking/anonymization
  - retention (auto purge/anonymize setelah N hari)
- Export/delete (jika tenant menyimpan PII)

### NFR-06 Observability & ops
- Metrics: RPS, error rate, latency, queue event lag, websocket conn count, notif failures
- Logs terstruktur + correlation id
- Tracing (OpenTelemetry recommended)
- Alerting: SLO burn, queue lag, notification failure spike, device offline surge
- Runbook operasional

### NFR-07 Offline/Degraded mode
- Kiosk:
  - limited offline ticket issuance (local buffer)
  - sync ketika koneksi pulih
  - reconciliation berbasis event log
- Display:
  - cache last state + reconnect loop
- Agent app:
  - read-only degraded mode (optional), atau hard-fail dengan banner

---

## 8) Data model (ringkas)
Entities:
- Tenant, Branch, Area
- Service, Counter, User, Role
- Ticket, TicketEvent (append-only), Appointment
- Notification, NotificationTemplate
- Device (kiosk/display), DeviceHealth
- AuditLog

---

## 9) Real-time architecture (produk level)
- Event-driven:
  - semua perubahan ticket sebagai event (`TicketEvent`)
- Real-time delivery:
  - WebSocket/SSE untuk agent app & display
- Idempotency:
  - semua command menerima `request_id`
- Consistency:
  - server authority, device cache untuk offline/degraded

---

## 10) Acceptance criteria (Definition of Done)
Wajib lulus sebelum production:
- [ ] Semua journeys A/B/C berjalan end-to-end
- [ ] Role & permission: agent tidak bisa akses admin config
- [ ] Audit log mencatat semua action ticket & perubahan config
- [ ] Load test memenuhi p95 target
- [ ] Observability aktif + alert critical
- [ ] DR drill (restore dari backup + verifikasi)
- [ ] Security test: SAST/DAST baseline + dependency scan
- [ ] Retention job berjalan sesuai setting

---

## 11) Release plan (recommended)
### Phase 1 — Core Omnichannel (MVP Production)
- Kiosk + Web + Agent app + Display + Queue engine + basic analytics
- Notifications minimal (SMS/email), WhatsApp menyusul jika perlu

### Phase 2 — Appointment + Advanced Routing
- Appointment blending, skill-based routing, supervisor staffing tools

### Phase 3 — Enterprise
- SSO, multi-region, advanced BI export, device fleet management, custom integrations

---

## 12) Lampiran — Checklist go-live (ops)
- [ ] Device inventory & provisioning (kiosk/display)
- [ ] Network readiness (wired/WiFi), captive portal avoidance
- [ ] Provider notif (SMS/WA/email) siap + template approved
- [ ] Monitoring dashboard + alert routing
- [ ] SOP petugas & supervisor + training
- [ ] UAT sign-off + rollback plan
