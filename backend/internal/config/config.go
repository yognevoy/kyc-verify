package config

import (
	"os"
	"time"
)

type Config struct {
	HTTPPort     string
	DatabaseURL  string
	JWTSecret    string
	JWTTTL       time.Duration
	RefreshTTL   time.Duration
	CookieSecure bool
}

func Load() Config {
	return Config{
		HTTPPort:     getEnv("HTTP_PORT", "8080"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/kyc_verify?sslmode=disable"),
		JWTSecret:    getEnv("JWT_SECRET", "dev-secret-change-me"),
		JWTTTL:       15 * time.Minute,
		RefreshTTL:   30 * 24 * time.Hour,
		CookieSecure: getEnv("COOKIE_SECURE", "false") == "true",
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
