package config

import "os"

type Config struct {
	Port        string
	DatabaseURL string
}

func Load() Config {
	port := os.Getenv("ADMIN_PORT")
	if port == "" {
		port = "8083"
	}

	return Config{
		Port:        port,
		DatabaseURL: os.Getenv("DB_DSN"),
	}
}
