package utils

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL   string
	RedisURL      string
	APIPort       string
	AnalyticsPort string
	JWTSecret     string
	Environment   string
}

func LoadConfig() *Config {
	return &Config{
		DatabaseURL:   requireEnv("DATABASE_URL"),
		RedisURL:      requireEnv("REDIS_URL"),
		APIPort:       getEnv("API_PORT", "8080"),
		AnalyticsPort: getEnv("ANALYTICS_PORT", "8082"),
		JWTSecret:     requireEnv("JWT_SECRET"),
		Environment:   getEnv("ENVIRONMENT", "development"),
	}
}

func requireEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return value
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
