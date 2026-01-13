# Style Guide

## Backend (Go)
- Error responses must include `request_id`, `error.code`, and `error.message`.
- Command endpoints require `request_id` (UUID) for idempotency.
- Log in structured key-value pairs: `request_id`, `tenant_id`, `branch_id`, `path`, `status`, `duration_ms`.
- Keep command (write) handlers separate from query (read) handlers.

## Frontend (Web apps)
- Use 2-space indentation in `apps/*/app.js` files.
- Keep UI state in a single `state` object per app.
- Avoid heavy animations; ensure reconnection and offline states are visible.
- Follow existing naming: `handleX`, `renderX`, `setX`.

## Migrations
- Use forward-only migrations with numeric prefixes.
- Any rollback is a new migration; never edit applied migrations.
