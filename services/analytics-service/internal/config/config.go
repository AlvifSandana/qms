package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                     string
	DatabaseURL              string
	AnomalyThresholdSeconds  float64
	AnomalyIntervalSeconds   int
	RateLimitPerMinute       int
	RateLimitBurst           int
	TenantRateLimitPerMinute int
	TenantRateLimitBurst     int
	ReportIntervalSeconds    int
	ReportWebhookURL         string
	ReportWebhookToken       string
	ReportEmailHost          string
	ReportEmailPort          int
	ReportEmailUser          string
	ReportEmailPass          string
	ReportEmailFrom          string
	ReportEmailSubject       string
	BIAPIToken               string
}

func Load() Config {
	port := os.Getenv("ANALYTICS_PORT")
	if port == "" {
		port = "8084"
	}

	return Config{
		Port:                     port,
		DatabaseURL:              os.Getenv("DB_DSN"),
		AnomalyThresholdSeconds:  readFloat("ANOMALY_WAIT_THRESHOLD_SECONDS", 1800),
		AnomalyIntervalSeconds:   readInt("ANOMALY_INTERVAL_SECONDS", 300),
		RateLimitPerMinute:       readInt("ANALYTICS_RATE_LIMIT_PER_MIN", 120),
		RateLimitBurst:           readInt("ANALYTICS_RATE_LIMIT_BURST", 30),
		TenantRateLimitPerMinute: readInt("ANALYTICS_TENANT_RATE_LIMIT_PER_MIN", 300),
		TenantRateLimitBurst:     readInt("ANALYTICS_TENANT_RATE_LIMIT_BURST", 60),
		ReportIntervalSeconds:    readInt("ANALYTICS_REPORT_INTERVAL_SECONDS", 300),
		ReportWebhookURL:         os.Getenv("ANALYTICS_REPORT_WEBHOOK_URL"),
		ReportWebhookToken:       os.Getenv("ANALYTICS_REPORT_WEBHOOK_TOKEN"),
		ReportEmailHost:          os.Getenv("ANALYTICS_REPORT_EMAIL_HOST"),
		ReportEmailPort:          readInt("ANALYTICS_REPORT_EMAIL_PORT", 587),
		ReportEmailUser:          os.Getenv("ANALYTICS_REPORT_EMAIL_USER"),
		ReportEmailPass:          os.Getenv("ANALYTICS_REPORT_EMAIL_PASS"),
		ReportEmailFrom:          os.Getenv("ANALYTICS_REPORT_EMAIL_FROM"),
		ReportEmailSubject:       os.Getenv("ANALYTICS_REPORT_EMAIL_SUBJECT"),
		BIAPIToken:               os.Getenv("ANALYTICS_BI_API_TOKEN"),
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
