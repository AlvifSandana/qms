# Device Offline Surge

## Symptoms
- Many kiosks/displays report offline.
- Admin device status view shows rapid disconnects.

## Immediate Actions
- Verify network connectivity at the branch.
- Check realtime-service and admin-service health.
- Confirm DNS and certificate validity.

## Mitigation
- Enable fallback polling on display/kiosk clients.
- Reduce reconnect interval if backlog spikes.

## Post-Incident
- Collect device IDs and timestamps for root cause analysis.
- Update device firmware/config if needed.
