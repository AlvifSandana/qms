# Backlog — Omnichannel QMS (Epics → User Stories + Acceptance Tests)
**Basis**: PRD Omnichannel Queue Management Platform (QMS) v1.0  
**Tanggal**: 12 Jan 2026 (Asia/Jakarta)

> Konvensi:
> - **[MVP]** = wajib untuk rilis production pertama (Core Omnichannel)
> - **[ENT]** = kebutuhan enterprise / fase lanjut
> - ID format: **E-<Modul>-##** (Epic), **US-<Modul>-###** (User Story)

---

## Definition of Ready (DoR) — minimal
- User story punya: persona, value, scope jelas, dependency disebut, AC (acceptance criteria) tertulis
- Mock/UX tersedia (untuk UI), atau fallback UI decision dicatat
- Telemetry minimal didefinisikan (event/metric yang harus muncul)

## Definition of Done (DoD) — minimal production
- Unit test + integration test lulus
- AC lulus (QA) + negative tests
- Audit log / telemetry sesuai story (jika relevan)
- Dokumentasi operasi singkat (README/runbook update) + migration note (jika ada DB change)

---

# 1) Modul: Queue Engine

## E-QUEUE-01 [MVP] Ticket lifecycle & state machine
**Tujuan**: ticket punya state jelas dan transisi aman (concurrency-safe).

### US-QUEUE-001 [MVP] Enqueue ticket (create ticket ke antrean)
**Sebagai** Customer/Front desk, **saya ingin** membuat ticket untuk layanan tertentu **agar** masuk antrean.
- **Acceptance tests**
  - Given service aktif dan branch valid, When create ticket, Then sistem mengembalikan `ticket_id`, `ticket_number`, `status=waiting`, `created_at`.
  - Given format nomor dikonfigurasi, Then `ticket_number` mengikuti konfigurasi.
  - Given request invalid (service tidak ada), Then 4xx dengan error terstruktur.

### US-QUEUE-002 [MVP] Call next ticket (dequeue ke called)
**Sebagai** Agent, **saya ingin** memanggil ticket berikutnya **agar** bisa mulai melayani.
- **Acceptance tests**
  - Given queue tidak kosong, When agent menekan "Call next", Then ticket berstatus `called` dan ter-assign `counter_id`.
  - Given 2 agent call bersamaan, Then hanya 1 yang sukses mendapat ticket yang sama; lainnya mendapat ticket berbeda atau 409/empty sesuai konfigurasi.
  - Then event `ticket.called` terbit (untuk display/notif).

### US-QUEUE-003 [MVP] Start serving (called → serving)
**Acceptance tests**
- Given ticket `called` di counter yang sama, When agent start, Then status menjadi `serving` dan `served_at` terisi.
- Given ticket called oleh counter A, When counter B start, Then ditolak (403/409).

### US-QUEUE-004 [MVP] Complete ticket (serving → done)
**Acceptance tests**
- Given ticket `serving`, When complete, Then status `done` dan `completed_at` terisi.
- Then service_time dihitung konsisten untuk analytics (via event).

### US-QUEUE-005 [MVP] Cancel ticket (waiting → cancelled)
**Acceptance tests**
- Given ticket `waiting`, When cancel oleh staff/admin, Then status `cancelled`.
- Given ticket sudah `serving`, When cancel, Then ditolak (kecuali policy mengizinkan).

---

## E-QUEUE-02 [MVP] Queue actions: recall, hold, skip, transfer, no-show
### US-QUEUE-006 [MVP] Recall ticket
**Acceptance tests**
- Given ticket `called`, When recall, Then event `ticket.recalled` terbit dan display menerima update.
- Given ticket bukan called, Then recall ditolak.

### US-QUEUE-007 [MVP] Hold/Unhold ticket
**Acceptance tests**
- Given ticket `waiting`, When hold, Then status `held` dan tidak dipilih saat call next.
- When unhold, Then kembali `waiting` dengan posisi sesuai policy (config).

### US-QUEUE-008 [MVP] Transfer ticket antar layanan/counter
**Acceptance tests**
- Given ticket `called` atau `waiting`, When transfer ke service B, Then ticket masuk antrean service B (status `waiting`) dengan catatan transfer.
- Then audit/event menyimpan `from_service`, `to_service`, `reason` (optional).

### US-QUEUE-009 [MVP] Skip / mark no-show manual
**Acceptance tests**
- Given ticket `called`, When mark no-show, Then status `no-show`.
- Given policy return-to-queue aktif, When mark no-show, Then ticket kembali `waiting` dan flag `returned=true`.

### US-QUEUE-010 [MVP] Auto no-show setelah grace period
**Acceptance tests**
- Given ticket `called` dan melewati `called_grace_period`, Then sistem otomatis status `no-show` (atau return-to-queue sesuai config).
- Then event `ticket.no_show` terbit sekali (idempotent).

---

## E-QUEUE-03 [MVP] Routing, prioritas, fairness (baseline)
### US-QUEUE-011 [MVP] FIFO per service
**Acceptance tests**
- Given 3 ticket waiting, When call next berulang, Then urutan pemanggilan mengikuti FIFO.

### US-QUEUE-012 [MVP] Priority class basic (Regular vs Priority)
**Acceptance tests**
- Given priority enabled, When ada ticket Priority & Regular, Then Priority dipilih dulu sesuai policy.
- Given anti-starvation minimal, Then setelah N priority, sistem wajib melayani 1 regular (configurable).

### US-QUEUE-013 [ENT] Skill-based routing (counter skills)
**Acceptance tests**
- Given counter memiliki skill layanan X/Y, When call next untuk counter, Then hanya ticket yang kompatibel dipilih.
- Given tidak ada ticket kompatibel, Then return empty dengan reason.

### US-QUEUE-014 [ENT] Advanced blending appointment vs walk-in (quota/weight)
**Acceptance tests**
- Given quota 60% appointment/jam, Then pemanggilan menjaga proporsi dalam window waktu.
- Given appointment mendekati slot, Then weight meningkat dan diprioritaskan (config).

---

## E-QUEUE-04 [MVP] Consistency, idempotency, eventing
### US-QUEUE-015 [MVP] Idempotent command (request_id)
**Acceptance tests**
- Given action dengan `request_id` yang sama dikirim dua kali, Then hanya 1 event dicatat dan response konsisten.

### US-QUEUE-016 [MVP] Event stream untuk real-time (ticket.*)
**Acceptance tests**
- Given ticket action terjadi, Then event sesuai topik dikirim (created/called/serving/done/transfer/no-show).
- Given subscriber reconnect, Then bisa resync state via snapshot API.

### US-QUEUE-017 [ENT] Append-only TicketEvent store (audit-grade)
**Acceptance tests**
- Given setiap perubahan status, Then tercatat di `TicketEvent` (append-only) dengan hash/sequence.
- Then rehydrate state dari event log menghasilkan status yang sama.

---

# 2) Modul: Agent App (Counter Web)

## E-AGENT-01 [MVP] Authentication & session
### US-AGENT-001 [MVP] Login + role gating
**Acceptance tests**
- Given user role Agent, When login, Then hanya melihat branch/service yang diizinkan.
- Given user tidak punya akses branch, Then akses ditolak.

### US-AGENT-002 [ENT] SSO OIDC/SAML
**Acceptance tests**
- Given tenant SSO configured, When login via IdP, Then user masuk tanpa password lokal.
- Then mapping role via claim berjalan.

---

## E-AGENT-02 [MVP] Queue view & “my counter”
### US-AGENT-003 [MVP] Lihat antrean per service + filter
**Acceptance tests**
- Given agent assigned service, When open queue view, Then tampil daftar ticket waiting + ringkas SLA/ETA.
- When filter berubah, Then list ter-update tanpa refresh penuh.

### US-AGENT-004 [MVP] Panel “Now Serving” (ticket aktif)
**Acceptance tests**
- Given agent memanggil ticket, Then panel menampilkan ticket aktif (called/serving), tombol aksi sesuai state.

---

## E-AGENT-03 [MVP] Actions (call/recall/start/complete/skip/transfer/hold)
### US-AGENT-005 [MVP] Call next + error handling
**Acceptance tests**
- Given queue kosong, When call next, Then tampil state “no tickets” (tanpa error crash).
- Given konflik concurrency (409), Then agent mendapat pesan yang jelas dan bisa retry.

### US-AGENT-006 [MVP] Recall, Start, Complete
**Acceptance tests**
- Given called, When recall, Then display menerima update.
- Given called, When start, Then status serving.
- Given serving, When complete, Then status done dan ticket berikutnya bisa dipanggil.

### US-AGENT-007 [MVP] Transfer & Hold
**Acceptance tests**
- Given ticket aktif, When transfer, Then wajib pilih target service/counter (sesuai policy) dan ticket berpindah.
- Given hold, Then ticket tidak akan dipanggil sampai unhold.

### US-AGENT-008 [ENT] Supervisor mode (monitor + intervene)
**Acceptance tests**
- Given role Supervisor, Then dapat melihat semua counter status dan melakukan intervene (reassign/force transfer) dengan audit log.

---

## E-AGENT-04 [MVP] Presence & availability
### US-AGENT-009 [MVP] Status Available/Busy/Break
**Acceptance tests**
- When agent set Break, Then engine tidak assign ticket baru ke counter itu.
- When kembali Available, Then counter kembali eligible.

### US-AGENT-010 [ENT] Multi-counter (1 agent handle banyak counter)
**Acceptance tests**
- Given agent memiliki beberapa counter, Then UI bisa switch counter dan state engine konsisten.

---

# 3) Modul: Kiosk (On-site Self Service)

## E-KIOSK-01 [MVP] Service selection & ticket issuance
### US-KIOSK-001 [MVP] Pilih bahasa + layanan + cetak ticket
**Acceptance tests**
- Given kiosk online, When user pilih service, Then ticket tercipta dan tiket tercetak dengan QR + info (nomor, layanan, waktu).
- When printer error, Then UI menampilkan fallback (QR on-screen) + log error.

### US-KIOSK-002 [MVP] Input opsional (phone/customer ref)
**Acceptance tests**
- Given tenant mengaktifkan phone input, Then kiosk bisa input nomor (validasi basic) dan tersimpan sesuai policy masking.

### US-KIOSK-003 [MVP] Accessibility basics
**Acceptance tests**
- Font besar mode, high-contrast mode, timeout reset ke home, multi-bahasa minimal ID/EN.

---

## E-KIOSK-02 [MVP] Device resilience & offline buffer
### US-KIOSK-004 [MVP] Health check + auto-reconnect
**Acceptance tests**
- Given koneksi putus, Then kiosk menampilkan status “offline” dan retry otomatis.

### US-KIOSK-005 [MVP] Limited offline ticket buffer
**Acceptance tests**
- Given offline mode enabled, When offline dan user ambil nomor, Then kiosk membuat ticket lokal (temporary id) dan mencetak.
- When online kembali, Then kiosk sync ke server dan menerima `ticket_id` final (reconcile) tanpa duplikasi.

### US-KIOSK-006 [ENT] Remote config & content (device fleet)
**Acceptance tests**
- Given admin update config, Then kiosk mengambil config baru via polling/push sesuai interval.
- Then versioning config mencegah partial update.

---

## E-KIOSK-03 [ENT] Appointment check-in via QR
### US-KIOSK-007 [ENT] Scan QR appointment → check-in
**Acceptance tests**
- Given QR valid, When scan, Then appointment status checked-in dan ticket masuk antrean sesuai policy.
- Given QR invalid/expired, Then kiosk menampilkan pesan yang aman.

---

# 4) Modul: Display & Audio

## E-DISP-01 [MVP] Display board real-time (now calling)
### US-DISP-001 [MVP] Render “Now Calling” + last N
**Acceptance tests**
- Given event `ticket.called`, Then display menampilkan ticket_number + counter/room dalam ≤1 detik.
- When reconnect, Then display resync via snapshot.

### US-DISP-002 [MVP] Multi-board per area/service
**Acceptance tests**
- Given display dikonfigurasi area A, Then hanya menampilkan ticket untuk area/service terkait.

---

## E-DISP-02 [MVP] Audio announcement
### US-DISP-003 [MVP] TTS / audio file panggilan
**Acceptance tests**
- Given ticket called, When audio enabled, Then speaker mengumumkan nomor + counter dengan bahasa yang dipilih.
- Given recall, Then audio mengulang.

### US-DISP-004 [ENT] Queue announcement rules (rate limit/quiet hours)
**Acceptance tests**
- Given quiet hours, Then audio tidak diputar, tapi display tetap update.
- Given burst calls, Then audio di-rate-limit (queue audio) tanpa kehilangan announcement.

---

## E-DISP-03 [ENT] Signage playlist & branding
### US-DISP-005 [ENT] Playlist konten (promo/edukasi) + schedule
**Acceptance tests**
- Given playlist configured, Then konten berputar di sela-sela panggilan tanpa menutupi info utama.

---

# 5) Modul: Notifications

## E-NOTIF-01 [MVP] Template engine + variable substitution
### US-NOTIF-001 [MVP] Template multi-bahasa + variables
**Acceptance tests**
- Given template `ticket_created`, When ticket dibuat, Then pesan berisi ticket_number, branch, service.
- Given variable missing, Then fallback default tanpa crash (dan log warning).

### US-NOTIF-002 [MVP] Preference minimal (opt-in by tenant policy)
**Acceptance tests**
- Given tenant disable notif, Then tidak ada notif yang dikirim.

---

## E-NOTIF-02 [MVP] Trigger & dispatch pipeline
### US-NOTIF-003 [MVP] Trigger: ticket created / X nomor lagi / called
**Acceptance tests**
- Given user punya phone/email, Then notif terkirim pada event yang relevan.
- Given user tanpa contact, Then event dicatat tapi delivery dilewati.

### US-NOTIF-004 [MVP] Retry + DLQ + delivery status
**Acceptance tests**
- Given provider gagal transient, Then retry sesuai policy.
- Given gagal permanen, Then masuk DLQ dan status “failed” tercatat.

---

## E-NOTIF-03 [MVP] Provider integrations baseline
### US-NOTIF-005 [MVP] Email provider integration
**Acceptance tests**
- Given email valid, Then pesan terkirim dan status “sent/delivered” tercatat (sesuai callback yang tersedia).

### US-NOTIF-006 [MVP] SMS provider integration (pluggable)
**Acceptance tests**
- Given provider A aktif, Then SMS terkirim.
- Given pindah provider, Then tidak perlu perubahan di queue engine (adapter pattern).

### US-NOTIF-007 [ENT] WhatsApp template messaging
**Acceptance tests**
- Given WA templates approved, Then sistem mengirim template message (ticket created/called/reminder) sesuai aturan template.
- Given template tidak approved, Then fallback ke SMS/email (configurable).

### US-NOTIF-008 [ENT] Push notifications (mobile/PWA)
**Acceptance tests**
- Given device token valid, Then push terkirim pada event called.

---

# 6) Modul: Admin Console

## E-ADMIN-01 [MVP] Tenant/Branch/Service/Counter CRUD
### US-ADMIN-001 [MVP] Manage Branch + Area
**Acceptance tests**
- Admin dapat membuat/ubah branch, jam operasional, area/floor.
- Validasi: tidak bisa hapus branch jika masih punya service aktif (atau soft-delete).

### US-ADMIN-002 [MVP] Manage Service catalog
**Acceptance tests**
- Admin dapat set SLA target, priority default, jam layanan, kapasitas.
- Perubahan service langsung berlaku (dengan versioning/config reload).

### US-ADMIN-003 [MVP] Manage Counter/Room + skills (basic)
**Acceptance tests**
- Admin dapat membuat counter/room dan mapping ke service.
- Counter tidak bisa melayani service yang tidak dipetakan.

---

## E-ADMIN-02 [MVP] User management + RBAC + audit
### US-ADMIN-004 [MVP] Roles & permissions
**Acceptance tests**
- Role Admin/Supervisor/Agent terbatas sesuai permission matrix.
- Agent tidak bisa akses halaman konfigurasi.

### US-ADMIN-005 [MVP] Audit log viewer (minimum)
**Acceptance tests**
- Semua perubahan config dan action privileged tercatat (who/what/when/from where).
- Admin bisa filter by date, user, action type.

### US-ADMIN-006 [ENT] Approval workflow untuk config changes
**Acceptance tests**
- Given tenant enable approvals, Then perubahan config butuh approve supervisor sebelum active.

---

## E-ADMIN-03 [MVP] Policy configuration (routing/no-show/priority/appointment)
### US-ADMIN-007 [MVP] Configure no-show grace period + return-to-queue
**Acceptance tests**
- Admin dapat set grace period per service/branch.
- Engine menerapkan policy baru tanpa restart (atau rolling).

### US-ADMIN-008 [ENT] Holiday calendar & blackout dates
**Acceptance tests**
- Admin dapat mengatur holiday; appointment slot tidak muncul pada tanggal tersebut.

---

## E-ADMIN-04 [MVP] Device management (kiosk/display) baseline
### US-ADMIN-009 [MVP] Register device + assign to branch/area
**Acceptance tests**
- Device punya `device_id` dan token, bisa dipasangkan ke branch/area.
- Status online/offline terlihat di admin.

### US-ADMIN-010 [ENT] Fleet ops (remote update, config versioning)
**Acceptance tests**
- Admin push config version; device ack dan rollback jika gagal.

---

# 7) Modul: Analytics & Reporting

## E-ANALYT-01 [MVP] Event capture & aggregation
### US-ANALYT-001 [MVP] Capture ticket timestamps & compute metrics
**Acceptance tests**
- Given ticket lifecycle lengkap, Then sistem menghitung wait_time dan service_time secara konsisten.
- Missing timestamp tidak membuat pipeline gagal (graceful handling).

### US-ANALYT-002 [MVP] KPI per service/branch/day
**Acceptance tests**
- Dashboard menampilkan avg/median wait, avg service, throughput, no-show/abandonment per filter.

---

## E-ANALYT-02 [MVP] Real-time dashboard
### US-ANALYT-003 [MVP] Real-time queue length & counter load
**Acceptance tests**
- Update real-time ≤ 5 detik (target) dan tidak membebani engine (cache/stream).
- Graf/angka sesuai data aktual.

---

## E-ANALYT-03 [MVP] Historical reports & export
### US-ANALYT-004 [MVP] Filter & drilldown
**Acceptance tests**
- Filter: range tanggal, branch, service, priority class, channel.
- Drilldown ke daftar ticket (anonymized jika PII off).

### US-ANALYT-005 [MVP] Export CSV
**Acceptance tests**
- Export menghasilkan CSV sesuai filter, dengan header stabil & timezone konsisten.

### US-ANALYT-006 [ENT] Scheduled reports (email) + BI connectors
**Acceptance tests**
- Admin menjadwalkan laporan mingguan; sistem mengirim file/link sesuai jadwal.
- BI connector API tokenized dan rate-limited.

### US-ANALYT-007 [ENT] Anomaly detection (ops insight)
**Acceptance tests**
- Sistem mendeteksi lonjakan wait time dan memberi alert (threshold/rule-based awal).

---

# 8) Backlog “Hardening” lintas modul (recommended)
> Walau bukan modul khusus, item ini sebaiknya masuk sprint awal karena mempengaruhi kesiapan production.

## E-PLAT-01 [MVP] Observability & alerting baseline
- Metrics: latency, error rate, queue lag, websocket connections, notification failure
- Logging terstruktur + correlation id
- Dashboards + alert critical

## E-PLAT-02 [MVP] Security baseline
- TLS, encryption at rest, secrets management
- Rate limiting per tenant
- SAST/DAST + dependency scanning

## E-PLAT-03 [MVP] HA/DR baseline
- Backup policy + restore drill
- Runbook incident

---

## 9) Prioritas ringkas (phase mapping)
- **MVP Production (Phase 1)**: semua epic berlabel **[MVP]** pada modul Queue Engine, Agent App, Kiosk, Display, Notifications, Admin, Analytics + PLAT-01/02/03
- **Enterprise (Phase 2/3)**: epic/stories berlabel **[ENT]** (SSO, skill-based routing, advanced appointment blending, fleet ops, scheduled reports, WA/push, anomaly detection, approvals)

