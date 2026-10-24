package config

import "os"

type Config struct {
	Port   string
	IsProd bool
}

func Load() Config {
	return Config{
		Port:   env("PORT", "8021"),
		IsProd: env("APP_ENV", "development") == "production",
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
