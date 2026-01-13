# TLS & Encryption Baseline

## TLS
- All public endpoints must use HTTPS in production.
- Terminate TLS at the edge (Caddy/ingress) with automatic renewal.

## Encryption at Rest
- Use managed database encryption where possible.
- Rotate database keys per provider guidance.

## Certificates
- Track certificate expiration and alert at 14 days.
