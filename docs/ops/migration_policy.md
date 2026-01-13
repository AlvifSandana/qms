# Migration Policy

## Forward-Only
- Migrations are append-only; never edit or delete applied files.
- Each migration uses a numeric prefix and descriptive suffix.

## Rollback
- Rollbacks are new migrations that restore previous behavior.
- Document rollback steps in PR description when schema changes.

## Deployment
- Apply migrations before deploying services that depend on them.
- Validate with `scripts/migrate.sh` in staging first.
