package config

import (
	"os"
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
}

func Load() *Config {
	accessTTL, _ := time.ParseDuration(getEnv("ACCESS_TOKEN_TTL", "15m"))
	refreshTTL, _ := time.ParseDuration(getEnv("REFRESH_TOKEN_TTL", "168h"))

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
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
