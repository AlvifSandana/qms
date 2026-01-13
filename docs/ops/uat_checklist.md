# UAT Checklist

## Environment Ready
- All services healthy (`/healthz` or `/metrics`).
- Migrations applied (`scripts/migrate.sh`).
- Admin and agent credentials available.

## Core Flows
- Create ticket (kiosk or staff) and verify number format.
- Agent call-next, start, complete.
- Display updates within 5 seconds.
- Notifications delivered for created/called.

## Admin Configuration
- Create branch, area, service, counter.
- Assign counter skills and verify routing.
- Update service policy and audit log entry.

## Analytics
- Realtime dashboard updates.
- Export CSV works.
- Scheduled report delivery test (webhook/email).

## Sign-off
- Document issues, timestamps, and request IDs.
