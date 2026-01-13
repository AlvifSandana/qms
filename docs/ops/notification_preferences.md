# Notification Preferences

Store preferences in a JSON file referenced by `NOTIF_PREFS_PATH`.

Example:
```json
{
  "sms": ["+62811111111"],
  "email": ["blocked@example.com"],
  "whatsapp": [],
  "push": []
}
```

Recipients listed per channel will be skipped by the notification worker.
