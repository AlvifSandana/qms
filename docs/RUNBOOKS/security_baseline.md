# Security Baseline

## Access Control
- Enforce tenant scoping on every query.
- Require `request_id` for all command endpoints.
- Keep audit logging enabled for privileged actions.

## Data Protection
- Default to no PII; if enabled, mask and configure retention.
- Do not store secrets in the repo; use `.env` and secret managers.

## Network & Rate Limiting
- Keep rate limiting enabled on all public APIs.
- Restrict admin endpoints to trusted networks where possible.

## Operational Hygiene
- Rotate tokens and credentials regularly.
- Review logs for anomalies and failed login bursts.
