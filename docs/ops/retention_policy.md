# Data Retention & Anonymization

## Default
- No PII stored by default.
- If PII is enabled, define retention period per tenant.

## Anonymization
- Hash or mask phone/email after retention period.
- Ticket analytics should keep aggregated metrics only.

## Operations
- Run a nightly retention job.
- Log deletion counts and affected tenants.
