package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPPort     string
	DatabaseURL  string
	JWTSecret    string
	JWTTTL       time.Duration
	RefreshTTL   time.Duration
	CookieSecure bool

	UploadDir      string
	WorkerPoolSize int

	ProviderMinDelay      time.Duration
	ProviderMaxDelay      time.Duration
	ProviderApproveChance float64
}

func Load() Config {
	return Config{
		HTTPPort:     getEnv("HTTP_PORT", "8080"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/kyc_verify?sslmode=disable"),
		JWTSecret:    getEnv("JWT_SECRET", "dev-secret-change-me"),
		JWTTTL:       15 * time.Minute,
		RefreshTTL:   30 * 24 * time.Hour,
		CookieSecure: getEnv("COOKIE_SECURE", "false") == "true",

		UploadDir:      getEnv("UPLOAD_DIR", "./uploads"),
		WorkerPoolSize: getEnvInt("WORKER_POOL_SIZE", 4),

		ProviderMinDelay:      getEnvDuration("PROVIDER_MIN_DELAY", 500*time.Millisecond),
		ProviderMaxDelay:      getEnvDuration("PROVIDER_MAX_DELAY", 3*time.Second),
		ProviderApproveChance: getEnvFloat("PROVIDER_APPROVE_CHANCE", 0.8),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
