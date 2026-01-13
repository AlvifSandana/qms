package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port           string
	DatabaseURL    string
	PollInterval   time.Duration
	BatchSize      int
	MaxAttempts    int
	SMSProvider    string
	EmailProvider  string
	WAProvider     string
	PushProvider   string
	ReminderThreshold int
}

func Load() Config {
	port := os.Getenv("NOTIF_PORT")
	if port == "" {
		port = "8082"
	}

	return Config{
		Port:         port,
		DatabaseURL:  os.Getenv("DB_DSN"),
		PollInterval: readDurationSeconds("NOTIF_POLL_SECONDS", 5),
		BatchSize:    readInt("NOTIF_BATCH_SIZE", 50),
		MaxAttempts:  readInt("NOTIF_MAX_ATTEMPTS", 3),
		SMSProvider:  os.Getenv("NOTIF_SMS_PROVIDER"),
		EmailProvider: os.Getenv("NOTIF_EMAIL_PROVIDER"),
		WAProvider:    os.Getenv("NOTIF_WA_PROVIDER"),
		PushProvider:  os.Getenv("NOTIF_PUSH_PROVIDER"),
		ReminderThreshold: readInt("NOTIF_REMINDER_THRESHOLD", 3),
	}
}

func readDurationSeconds(key string, fallback int) time.Duration {
	value := readInt(key, fallback)
	if value <= 0 {
		return 0
	}
	return time.Duration(value) * time.Second
}

func readInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
