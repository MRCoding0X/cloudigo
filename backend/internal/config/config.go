package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv      string
	Port        string
	DatabaseURL string
	StoragePath string
	FrontendURL string

	JWTAccessSecret    string
	JWTRefreshSecret   string
	AccessTokenTTLMin  int
	RefreshTokenTTLDay int

	// SMTP is admin-editable at runtime via the settings table (Fas 4), not
	// static env config — see internal/service/email_service.go.

	CleanupIntervalMin int
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		StoragePath: getEnv("STORAGE_PATH", "./storage"),
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),

		JWTAccessSecret:    getEnv("JWT_ACCESS_SECRET", "dev-access-secret-change-me"),
		JWTRefreshSecret:   getEnv("JWT_REFRESH_SECRET", "dev-refresh-secret-change-me"),
		AccessTokenTTLMin:  getEnvInt("ACCESS_TOKEN_TTL_MIN", 15),
		RefreshTokenTTLDay: getEnvInt("REFRESH_TOKEN_TTL_DAYS", 30),

		CleanupIntervalMin: getEnvInt("CLEANUP_INTERVAL_MIN", 15),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
