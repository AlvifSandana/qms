# Webhook Integration

## Payload
All outbound webhooks are JSON and include:
- `channel`
- `recipient`
- `message`

## Signature Verification (Example)
If you add signing, verify using HMAC SHA256:

```go
mac := hmac.New(sha256.New, []byte(secret))
mac.Write(body)
expected := hex.EncodeToString(mac.Sum(nil))
if !hmac.Equal([]byte(signature), []byte(expected)) {
  // reject
}
```

## Recommendations
- Require HTTPS endpoints.
- Block private IP ranges for outbound hooks.
- Rotate tokens regularly.
