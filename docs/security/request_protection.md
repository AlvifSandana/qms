# Request Protection Policy

## CSRF (Admin Web)
- Admin web requests that mutate state must include a CSRF token.
- Token is stored in a secure cookie and echoed in `X-CSRF-Token` header.
- Stateless services validate the token per request.

## SSRF (Outbound Webhooks)
- Only allow outbound webhook targets from an allowlist.
- Block private IP ranges by default (RFC1918, localhost, link-local).
- Enforce HTTPS for production webhook targets.

## Input Validation
- Validate all UUIDs on input.
- Enforce strict JSON decoding (`DisallowUnknownFields`).
- Sanitize user-provided strings in logs.
