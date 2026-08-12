package config

import "os"

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	LXCBinary   string
}

func Load() Config {
	return Config{
		HTTPAddr:    env("SUDA_HTTP_ADDR", ":8080"),
		DatabaseURL: env("DATABASE_URL", "postgres://suda:suda@localhost:5432/suda_forge?sslmode=disable"),
		LXCBinary:   env("LXC_BINARY", "lxc"),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
