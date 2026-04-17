package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	SeedAdminEmail    string
	SeedAdminPassword string
	SeedAdminName     string

	AppPort string

	OpenRouterAPIKey string
	OpenRouterBase   string
	AIModel          string
	AIMaxUsers       int
	AITimeout        time.Duration
}

func Load() *Config {
	accessTTL, _ := time.ParseDuration(getEnv("ACCESS_TOKEN_TTL", "15m"))
	refreshTTL, _ := time.ParseDuration(getEnv("REFRESH_TOKEN_TTL", "168h"))
	aiTimeout, _ := time.ParseDuration(getEnv("AI_TIMEOUT", "60s"))

	maxUsers, _ := strconv.Atoi(getEnv("AI_MAX_USERS", "100"))
	if maxUsers <= 0 {
		maxUsers = 100
	}

	return &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "admin"),
		DBPassword: getEnv("DB_PASSWORD", "secret"),
		DBName:     getEnv("DB_NAME", "miniadmin"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		JWTSecret:       getEnv("JWT_SECRET", "change-me-in-production-please"),
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,

		SeedAdminEmail:    getEnv("SEED_ADMIN_EMAIL", "admin@example.com"),
		SeedAdminPassword: getEnv("SEED_ADMIN_PASSWORD", "admin123"),
		SeedAdminName:     getEnv("SEED_ADMIN_NAME", "Administrator"),

		AppPort: getEnv("APP_PORT", "8080"),

		OpenRouterAPIKey: getEnv("OPENROUTER_API_KEY", ""),
		OpenRouterBase:   getEnv("OPENROUTER_BASE", "https://openrouter.ai/api/v1"),
		AIModel:          getEnv("AI_MODEL", "google/gemini-2.0-flash-001"),
		AIMaxUsers:       maxUsers,
		AITimeout:        aiTimeout,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
