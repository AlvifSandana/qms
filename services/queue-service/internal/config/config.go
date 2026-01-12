package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port        string
	DatabaseURL string
	NoShowGrace time.Duration
	NoShowInterval time.Duration
	NoShowBatchSize int
	NoShowReturnToQueue bool
	PriorityStreakLimit int
	RateLimitPerMinute int
	RateLimitBurst int
	TenantRateLimitPerMinute int
	TenantRateLimitBurst int
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return Config{
		Port:        port,
		DatabaseURL: os.Getenv("DB_DSN"),
		NoShowGrace: readDurationSeconds("NO_SHOW_GRACE_SECONDS", 300),
		NoShowInterval: readDurationSeconds("NO_SHOW_SCAN_INTERVAL_SECONDS", 30),
		NoShowBatchSize: readInt("NO_SHOW_BATCH_SIZE", 100),
		NoShowReturnToQueue: readBool("NO_SHOW_RETURN_TO_QUEUE", false),
		PriorityStreakLimit: readInt("PRIORITY_STREAK_LIMIT", 3),
		RateLimitPerMinute: readInt("RATE_LIMIT_PER_MIN", 120),
		RateLimitBurst: readInt("RATE_LIMIT_BURST", 30),
		TenantRateLimitPerMinute: readInt("TENANT_RATE_LIMIT_PER_MIN", 600),
		TenantRateLimitBurst: readInt("TENANT_RATE_LIMIT_BURST", 120),
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

func readBool(key string, fallback bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}
