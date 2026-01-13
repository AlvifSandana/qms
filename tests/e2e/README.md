# E2E Smoke Tests

## Manual flow
1. Create ticket via `/api/tickets`.
2. Call next via `/api/tickets/actions/call-next`.
3. Confirm display receives `ticket.called`.

## Scripted smoke
Use `scripts/e2e-smoke.sh` for basic API validation.
