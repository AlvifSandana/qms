# Observability Overview

## Tracing
- OpenTelemetry tracing is enabled when `OTEL_EXPORTER_OTLP_ENDPOINT` is set.
- Set `OTEL_EXPORTER_OTLP_INSECURE=true` for local collectors.

## Metrics
- Each service exposes expvar metrics on `/metrics`:
  - `requests_total`
  - `requests_errors_total`

## Logging
- HTTP middleware logs method, path, status, duration, tenant, request ID.
- Logs should include `request_id` and tenant identifiers when available.
