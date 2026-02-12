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
	LogLevel      string
	LogFormat     string
}

func LoadConfig() (*Config, error) {
	var missing []string
	requireEnv := func(key string) string {
		value := os.Getenv(key)
		if value == "" {
			missing = append(missing, key)
		}
		return value
	}

	cfg := &Config{
		DatabaseURL:   requireEnv("DATABASE_URL"),
		RedisURL:      requireEnv("REDIS_URL"),
		APIPort:       getEnv("API_PORT", "8080"),
		AnalyticsPort: getEnv("ANALYTICS_PORT", "8082"),
		JWTSecret:     requireEnv("JWT_SECRET"),
		Environment:   getEnv("ENVIRONMENT", "development"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		LogFormat:     getEnv("LOG_FORMAT", "json"),
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("❌ required environment variable(s) not set: %s. Run 'make ensure-env' first", joinStrings(missing, ", "))
	}

	return cfg, nil
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
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
