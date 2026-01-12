package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port        string
	DatabaseURL string
	PollInterval time.Duration
	BatchSize   int
}

func Load() Config {
	port := os.Getenv("REALTIME_PORT")
	if port == "" {
		port = "8085"
	}

	return Config{
		Port:        port,
		DatabaseURL: os.Getenv("DB_DSN"),
		PollInterval: readDurationSeconds("REALTIME_POLL_SECONDS", 1),
		BatchSize:   readInt("REALTIME_BATCH_SIZE", 100),
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
