package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port        string
	DatabaseURL string
	AnomalyThresholdSeconds float64
	AnomalyIntervalSeconds  int
}

func Load() Config {
	port := os.Getenv("ANALYTICS_PORT")
	if port == "" {
		port = "8084"
	}

	return Config{
		Port:        port,
		DatabaseURL: os.Getenv("DB_DSN"),
		AnomalyThresholdSeconds: readFloat("ANOMALY_WAIT_THRESHOLD_SECONDS", 1800),
		AnomalyIntervalSeconds: readInt("ANOMALY_INTERVAL_SECONDS", 300),
	}
}

func readFloat(key string, fallback float64) float64 {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return value
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
