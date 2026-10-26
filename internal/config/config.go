package config

import "os"

type Config struct {
	Port        string
	IsProd      bool
	DatabaseURL string
}

func Load() Config {
	return Config{
		Port:   env("PORT", "8021"),
		IsProd: env("APP_ENV", "development") == "production",
		// Default matches the Postgres service in compose.yaml.
		DatabaseURL: env("DATABASE_URL", "postgres://levireg:levireg@localhost:5432/levireg?sslmode=disable"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
