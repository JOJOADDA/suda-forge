package config

import "os"

type Config struct {
	HTTPAddr          string
	DatabaseURL       string
	LXCBinary         string
	OllamaURL         string
	VLLMURL           string
	LlamaCPPURL       string
	CaddyAdminURL     string
	DeployStorageRoot string
}

func Load() Config {
	return Config{
		HTTPAddr:          env("SUDA_HTTP_ADDR", ":8080"),
		DatabaseURL:       env("DATABASE_URL", "postgres://suda:suda@localhost:5432/suda_forge?sslmode=disable"),
		LXCBinary:         env("LXC_BINARY", "lxc"),
		OllamaURL:         env("OLLAMA_URL", ""),
		VLLMURL:           env("VLLM_URL", ""),
		LlamaCPPURL:       env("LLAMACPP_URL", ""),
		CaddyAdminURL:     env("CADDY_ADMIN_URL", ""),
		DeployStorageRoot: env("SUDA_DEPLOY_STORAGE_ROOT", "/var/lib/suda-forge/deployments"),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
