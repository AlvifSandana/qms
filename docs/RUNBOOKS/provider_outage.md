# Provider Outage (Notifications)

## Symptoms
- Spike in notification failures or DLQ entries.
- Provider API returns 5xx/timeout.

## Immediate Actions
- Confirm provider status page.
- Switch provider to `log` or backup webhook if available.
- Pause notification dispatch if rate limits trigger.

## Recovery
- Retry failed notifications once provider is stable.
- Review DLQ and requeue if needed.

## Post-Incident
- Document failure window and affected tenants.
- Update retry thresholds or provider config.
