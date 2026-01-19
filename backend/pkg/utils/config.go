package utils

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL    string
	RedisURL       string
	APIPort        string
	AnalyticsPort  string
	JWTSecret      string
	Environment    string
}

func LoadConfig() *Config {
	return &Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/webhook_platform?sslmode=disable"),
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6379"),
		APIPort:       getEnv("API_PORT", "8080"),
		AnalyticsPort: getEnv("ANALYTICS_PORT", "8082"),
		JWTSecret:     getEnv("JWT_SECRET", "your-secret-key"),
		Environment:   getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}