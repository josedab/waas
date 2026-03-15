package utils

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL        string
	RedisURL           string
	APIPort            string
	AnalyticsPort      string
	JWTSecret          string
	Environment        string
	LogLevel           string
	LogFormat          string
	CORSAllowedOrigins string
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
		DatabaseURL:        requireEnv("DATABASE_URL"),
		RedisURL:           requireEnv("REDIS_URL"),
		APIPort:            getEnv("API_PORT", "8080"),
		AnalyticsPort:      getEnv("ANALYTICS_PORT", "8082"),
		JWTSecret:          requireEnv("JWT_SECRET"),
		Environment:        getEnvWithFallback("ENVIRONMENT", "APP_ENV", "development"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		LogFormat:          getEnv("LOG_FORMAT", "json"),
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", ""),
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("❌ required environment variable(s) not set: %s. Run 'make ensure-env' first", joinStrings(missing, ", "))
	}

	// Validate port values are in valid range
	var configErrors []string
	if err := validatePort(cfg.APIPort, "API_PORT"); err != nil {
		configErrors = append(configErrors, err.Error())
	}
	if err := validatePort(cfg.AnalyticsPort, "ANALYTICS_PORT"); err != nil {
		configErrors = append(configErrors, err.Error())
	}

	// Validate URL formats
	if err := validateDatabaseURL(cfg.DatabaseURL); err != nil {
		configErrors = append(configErrors, err.Error())
	}
	if err := validateRedisURL(cfg.RedisURL); err != nil {
		configErrors = append(configErrors, err.Error())
	}

	if len(configErrors) > 0 {
		return nil, fmt.Errorf("❌ configuration error(s): %s", joinStrings(configErrors, "; "))
	}

	// Reject known-weak default values that indicate unmodified .env.example
	if cfg.Environment == "production" {
		weakDefaults := map[string][]string{
			"JWT_SECRET": {
				"change-me-in-production",
				"REPLACE_ME_OR_STARTUP_FAILS",
				"secret",
				"jwt-secret",
			},
		}
		var insecure []string
		for key, weakValues := range weakDefaults {
			val := os.Getenv(key)
			for _, weak := range weakValues {
				if val == weak {
					insecure = append(insecure, fmt.Sprintf("%s=%s", key, weak))
					break
				}
			}
		}
		if pw := os.Getenv("POSTGRES_PASSWORD"); pw == "password" || pw == "REPLACE_ME_OR_STARTUP_FAILS" {
			insecure = append(insecure, "POSTGRES_PASSWORD="+pw)
		}
		if len(insecure) > 0 {
			return nil, fmt.Errorf("❌ insecure default value(s) detected in production: %s. Update these values before deploying", joinStrings(insecure, ", "))
		}
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

// getEnvWithFallback tries the primary key first, then falls back to the
// alternate key, and finally returns the default value.
func getEnvWithFallback(key, fallbackKey, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	if value := os.Getenv(fallbackKey); value != "" {
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

func validatePort(port, name string) error {
	p, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("%s=%q is not a valid integer", name, port)
	}
	if p < 1 || p > 65535 {
		return fmt.Errorf("%s=%d is out of range (must be 1-65535)", name, p)
	}
	return nil
}

func validateDatabaseURL(rawURL string) error {
	if !strings.HasPrefix(rawURL, "postgres://") && !strings.HasPrefix(rawURL, "postgresql://") {
		return fmt.Errorf("DATABASE_URL must start with postgres:// or postgresql://")
	}
	if _, err := url.Parse(rawURL); err != nil {
		return fmt.Errorf("DATABASE_URL is not a valid URL: %v", err)
	}
	return nil
}

func validateRedisURL(rawURL string) error {
	if !strings.HasPrefix(rawURL, "redis://") && !strings.HasPrefix(rawURL, "rediss://") {
		return fmt.Errorf("REDIS_URL must start with redis:// or rediss://")
	}
	if _, err := url.Parse(rawURL); err != nil {
		return fmt.Errorf("REDIS_URL is not a valid URL: %v", err)
	}
	return nil
}
