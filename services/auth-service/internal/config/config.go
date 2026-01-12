package config

import "os"

type Config struct {
	Port        string
	DatabaseURL string
}

func Load() Config {
	port := os.Getenv("AUTH_PORT")
	if port == "" {
		port = "8081"
	}

	return Config{
		Port:        port,
		DatabaseURL: os.Getenv("DB_DSN"),
	}
}
