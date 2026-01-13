# Secrets Rotation Plan

## Scope
- Database credentials
- JWT secrets
- Webhook tokens
- SMTP credentials

## Rotation Steps
1. Add new secret in secret manager.
2. Deploy services with dual-read if supported.
3. Rotate consumers and validate logs/metrics.
4. Remove old secret after verification.

## Cadence
- Rotate quarterly or after incidents.
