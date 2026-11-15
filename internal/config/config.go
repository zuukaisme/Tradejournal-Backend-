package config

import (
	"fmt"
	"os"
)

type Config struct {
	Environment string
	Port        string
	DatabaseURL string
}

func Load() (Config, error) {
	cfg := Config{
		Environment: valueOrDefault("APP_ENV", "development"),
		Port:        valueOrDefault("APP_PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
