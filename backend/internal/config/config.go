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

	CORSAllowedOrigin string

	UploadDir      string
	WorkerPoolSize int

	ProviderMinDelay      time.Duration
	ProviderMaxDelay      time.Duration
	ProviderApproveChance float64
	ProviderFailureChance float64

	ProviderRateLimitRPS       float64
	ProviderRateLimitBurst     int
	ProviderCBFailureThreshold int
	ProviderCBCooldown         time.Duration
}

func Load() Config {
	return Config{
		HTTPPort:     getEnv("HTTP_PORT", "8080"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/kyc_verify?sslmode=disable"),
		JWTSecret:    getEnv("JWT_SECRET", "dev-secret-change-me"),
		JWTTTL:       15 * time.Minute,
		RefreshTTL:   30 * 24 * time.Hour,
		CookieSecure: getEnv("COOKIE_SECURE", "false") == "true",

		CORSAllowedOrigin: getEnv("CORS_ALLOWED_ORIGIN", "http://localhost:5173"),

		UploadDir:      getEnv("UPLOAD_DIR", "./uploads"),
		WorkerPoolSize: getEnvInt("WORKER_POOL_SIZE", 4),

		ProviderMinDelay:      getEnvDuration("PROVIDER_MIN_DELAY", 500*time.Millisecond),
		ProviderMaxDelay:      getEnvDuration("PROVIDER_MAX_DELAY", 3*time.Second),
		ProviderApproveChance: getEnvFloat("PROVIDER_APPROVE_CHANCE", 0.8),
		ProviderFailureChance: getEnvFloat("PROVIDER_FAILURE_CHANCE", 0),

		ProviderRateLimitRPS:       getEnvFloat("PROVIDER_RATE_LIMIT_RPS", 5),
		ProviderRateLimitBurst:     getEnvInt("PROVIDER_RATE_LIMIT_BURST", 10),
		ProviderCBFailureThreshold: getEnvInt("PROVIDER_CB_FAILURE_THRESHOLD", 3),
		ProviderCBCooldown:         getEnvDuration("PROVIDER_CB_COOLDOWN", 15*time.Second),
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
