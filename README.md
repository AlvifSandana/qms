# Omnichannel QMS

Omnichannel **Queue Management System (QMS)** untuk **bank/klinik/layanan publik** dengan dukungan:
- Walk-in ticketing (kiosk/staff)
- Virtual queue (public web/PWA)
- Agent/Counter app
- Display board + audio announcement
- Notifications (Email/SMS; WhatsApp/PWA push optional)
- Admin console (multi-tenant config)
- Analytics (real-time + historis)

## Stack default (dipilih)
- **Backend**: Go (monolith modular) + Outbox worker
- **DB**: PostgreSQL
- **Realtime**: SockJS (WebSocket primary + fallback transports) + snapshot polling fallback
- **Frontend**: Web apps (Admin/Agent/Kiosk/Display/Public Web) — build via Vite (legacy build optional)
- **Edge**: Caddy
- **Deploy**: Docker Compose (cloud VPS / on-prem mini PC)

Implementasi repo saat ini memakai **multi-service** (queue/auth/admin/notification/analytics/realtime) agar modul terpisah sejak awal, namun kontrak mengikuti specs monolith.

## Dokumentasi
- PRD: `docs/PRD_Omnichannel_QMS.md`
- Backlog epics/stories/AC: `docs/QMS_Backlog_Epics_UserStories_AcceptanceTests.md`
- Technical specs: `docs/specs.md`
- Docs index: `docs/README.md`
- Project TODO: `TODO.md`
- AI agent rules: `AGENT_RULES.md`

## Quick start (Local / On-Prem / Cloud VPS)
### Prerequisites
- Docker + Docker Compose (plugin)
- (Opsional) Make

### 1) Setup env
1. Copy env template:
   - `cp .env.example .env`
2. Sesuaikan variabel minimum (DB password, admin bootstrap, dsb.)

### 2) Run
```bash
docker compose -f infra/docker/compose.yml up -d
```
> Compose sekarang build image dari `services/*/Dockerfile` untuk setiap service.

### 3) Open apps (default)
> Sesuaikan port dengan `infra/docker/compose.yml`

- Admin: `https://localhost/admin`
- Agent: `apps/agent-web/index.html` (static prototype, set `Queue Base` + `Realtime Base`, needs session token)
- Kiosk: `apps/kiosk-app/index.html`
- Display: `apps/display-app/index.html` (set `Queue Base` + `Realtime Base`, needs session token)
- Public web: `https://localhost/` (optional, tracking requires session token)

### 4) Health checks
- API health: `GET /healthz`
- Realtime: `GET /realtime/info` (SockJS info endpoint)
- DB: check container logs / psql

## Services (local ports)
- Queue service: `:8080`
- Auth service: `:8081`
- Notification worker: `:8082`
- Admin service: `:8083`
- Analytics service: `:8084`
- Realtime service: `:8085` (`/realtime/info`, `/realtime/`)

## Deployment modes
### On-prem branch (single-node)
- 1 mini PC/NUC menjalankan compose stack (API + DB + Caddy)
- Kiosk/Display idealnya pakai Chromium kiosk mode (lebih stabil dari browser legacy)

### Cloud central
- 1 VPS untuk MVP (API + DB + Caddy)
- Branch akses via browser

## Contributing
- Ikuti `AGENT_RULES.md`
- Semua perubahan harus terkait story (mis. `US-QUEUE-002`) dan punya acceptance test
- PR harus hijau (lint + tests)

## Security
- Jangan commit secrets
- Default tanpa PII; jika PII diaktifkan tenant, wajib masking + retention policy

## License
TBD
