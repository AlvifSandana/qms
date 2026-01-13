# RBAC Matrix

Roles used across the system: `admin`, `supervisor`, `agent`.

## Admin
- Manage tenants, branches, services, counters, users, roles.
- Configure service policies and approval workflows.
- View audit logs and device status.

## Supervisor
- Monitor counters, intervene on calls/transfers.
- View queue metrics and realtime status.
- No access to tenant-level configuration.

## Agent
- Operate assigned counters: call, recall, start, complete, transfer, hold, no-show.
- View own queue scope only.
- No access to admin configuration.

## Notes
- All API endpoints must enforce tenant scoping.
- Admin-only endpoints live under `/api/admin/*`.
