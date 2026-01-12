# AGENT_RULES.md — AI Coding Agent Rules (Omnichannel QMS)

Tujuan aturan ini: membuat implementasi **stabil, aman, mudah di-review**, dan selaras dengan **PRD + specs**.

---

## 1) Golden rules (wajib)
1. **Baca dulu**:
   - `docs/specs.md`
   - `docs/PRD_Omnichannel_QMS.md`
   - `docs/QMS_Backlog_Epics_UserStories_AcceptanceTests.md`
   - `TODO.md`
2. Kerjakan **1 user story / 1 subtask kecil** per PR.
3. **Jangan merge** jika CI merah (lint/test/build).
4. Tidak ada **secret** di repo, tidak ada hardcode credentials.
5. Semua perubahan harus punya:
   - acceptance criteria (AC)
   - tests minimal (unit/integration sesuai area)
   - telemetry/log minimal jika relevan

---

## 2) Workflow standar (untuk setiap task)
1. Pilih story ID (contoh: `US-QUEUE-002`)
2. Buat branch:
   - `feat/<module>-<short>`
   - `fix/<module>-<short>`
   - `chore/<short>`
3. Implement dengan diff kecil:
   - update code
   - update tests
   - update docs (jika kontrak/behavior berubah)
4. Pastikan lulus:
   - lint
   - unit tests
   - integration tests (jika menyentuh DB/realtime/outbox)
5. Buat PR yang mencantumkan:
   - story ID
   - AC yang diuji
   - cara verifikasi manual (steps)

---

## 3) Kontrak teknis yang tidak boleh dilanggar
### 3.1 Idempotency
- Semua command/action harus menerima `request_id` (UUID)
- Duplicate request_id harus menghasilkan response yang **konsisten** dan tidak menambah event ganda

### 3.2 Eventing (Outbox)
- Semua perubahan ticket yang bermakna harus menghasilkan event (outbox):
  - `ticket.created`, `ticket.called`, `ticket.serving`, `ticket.done`,
  - `ticket.transferred`, `ticket.no_show`, `ticket.held`, `ticket.recalled`
- Jangan publish event langsung tanpa outbox untuk path utama (hindari lost event)

### 3.3 Concurrency safety: “Call next”
- “Call next” harus concurrency-safe (dua agent tidak boleh mendapat ticket yang sama)
- Gunakan transaction + row lock strategy (mis. `FOR UPDATE SKIP LOCKED`) sesuai `specs.md`

### 3.4 Multi-tenant scoping
- Semua query wajib scope `tenant_id` (dan `branch_id` bila perlu)
- Tidak boleh ada endpoint yang bisa “baca tenant lain”

### 3.5 Error format
- Error API wajib structured: `request_id` + `error.code` + `error.message`

---

## 4) Coding standards
### Backend (Go)
- Gunakan structured logging (JSON) dengan:
  - `request_id`, `tenant_id`, `branch_id` (jika ada)
- Pisahkan:
  - command handlers (write)
  - query handlers (read)
- Logic routing/selection dibuat pure function jika memungkinkan (mudah unit test)
- Hindari global mutable state untuk realtime broadcaster (gunakan channel/hub yang terkontrol)

### Frontend (Web apps)
- UI harus tetap usable pada device lambat:
  - minimize bundle
  - hindari animasi berat
- Untuk kebutuhan browser lama:
  - jangan pakai API browser modern tanpa polyfill/guard
  - pastikan fallback polling tersedia bila realtime gagal

---

## 5) Testing requirements (minimum)
### Queue Engine
- Unit tests:
  - state machine transitions (valid/invalid)
  - routing selection (FIFO, priority)
- Integration tests:
  - concurrency: 2+ call-next bersamaan (tidak boleh double-assign)
  - outbox event produced exactly once per action

### Realtime
- Integration tests:
  - subscriber receives `ticket.called` within expected time
  - reconnect → resync snapshot

### Notifications
- Unit tests:
  - template render variables + fallback
- Integration tests:
  - retry + DLQ behavior

---

## 6) PR checklist (wajib)
- [ ] Referensi story ID (mis. `US-QUEUE-002`)
- [ ] AC terpenuhi (ditulis di PR)
- [ ] Tests ditambah/diupdate
- [ ] Tidak ada secrets
- [ ] Observability minimal (log/metric) bila relevan
- [ ] Jika ada DB change:
  - [ ] migration forward-only
  - [ ] catatan dampak & langkah deploy

---

## 7) Safe defaults (wajib)
- Default **tanpa PII**; jika PII aktif, wajib masking + retention job
- Rate limiting aktif (minimal)
- Audit log aktif untuk config + privileged actions

---

## 8) Hard constraints (dilarang)
- Menonaktifkan test/lint untuk “cepat selesai”
- Mengubah kontrak API tanpa update OpenAPI + catatan migrasi
- Menambah dependency besar tanpa alasan + ADR
- Meninggalkan “TODO: security later” untuk hal baseline (auth/validation/RBAC)

---

## 9) When unsure
- Buat ADR singkat di `docs/ADR/ADR-<next>.md`
- Atau tulis keputusan & konsekuensi di PR description
