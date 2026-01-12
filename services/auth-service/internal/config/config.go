package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port        string
	DatabaseURL string
	RateLimitPerMinute int
	RateLimitBurst int
	TenantRateLimitPerMinute int
	TenantRateLimitBurst int
}

func Load() Config {
	port := os.Getenv("AUTH_PORT")
	if port == "" {
		port = "8081"
	}

	return Config{
		Port:        port,
		DatabaseURL: os.Getenv("DB_DSN"),
		RateLimitPerMinute: readInt("AUTH_RATE_LIMIT_PER_MIN", 120),
		RateLimitBurst: readInt("AUTH_RATE_LIMIT_BURST", 30),
		TenantRateLimitPerMinute: readInt("AUTH_TENANT_RATE_LIMIT_PER_MIN", 300),
		TenantRateLimitBurst: readInt("AUTH_TENANT_RATE_LIMIT_BURST", 60),
	}
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
