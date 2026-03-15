package utils

import (
	"testing"
)

func TestValidatePort(t *testing.T) {
	tests := []struct {
		port    string
		name    string
		wantErr bool
	}{
		{"8080", "API_PORT", false},
		{"1", "API_PORT", false},
		{"65535", "API_PORT", false},
		{"0", "API_PORT", true},
		{"65536", "API_PORT", true},
		{"-1", "API_PORT", true},
		{"abc", "API_PORT", true},
		{"", "API_PORT", true},
	}
	for _, tt := range tests {
		t.Run(tt.port, func(t *testing.T) {
			err := validatePort(tt.port, tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePort(%q) error = %v, wantErr %v", tt.port, err, tt.wantErr)
			}
		})
	}
}

func TestValidateDatabaseURL(t *testing.T) {
	tests := []struct {
		url     string
		wantErr bool
	}{
		{"postgres://user:pass@localhost:5432/waas", false},
		{"postgresql://user:pass@localhost:5432/waas?sslmode=disable", false},
		{"mysql://user:pass@localhost/waas", true},
		{"localhost:5432", true},
		{"", true},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			err := validateDatabaseURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDatabaseURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestValidateRedisURL(t *testing.T) {
	tests := []struct {
		url     string
		wantErr bool
	}{
		{"redis://localhost:6379", false},
		{"rediss://user:pass@redis.example.com:6380", false},
		{"http://localhost:6379", true},
		{"localhost:6379", true},
		{"", true},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			err := validateRedisURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRedisURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestGetEnvWithFallback(t *testing.T) {
	tests := []struct {
		name        string
		primary     string
		primaryVal  string
		fallback    string
		fallbackVal string
		defaultVal  string
		expected    string
	}{
		{
			name:    "primary set",
			primary: "TEST_PRIMARY_1", primaryVal: "primary-value",
			fallback: "TEST_FALLBACK_1", fallbackVal: "",
			defaultVal: "default", expected: "primary-value",
		},
		{
			name:    "primary empty, fallback set",
			primary: "TEST_PRIMARY_2", primaryVal: "",
			fallback: "TEST_FALLBACK_2", fallbackVal: "fallback-value",
			defaultVal: "default", expected: "fallback-value",
		},
		{
			name:    "both empty, use default",
			primary: "TEST_PRIMARY_3", primaryVal: "",
			fallback: "TEST_FALLBACK_3", fallbackVal: "",
			defaultVal: "default", expected: "default",
		},
		{
			name:    "both set, primary wins",
			primary: "TEST_PRIMARY_4", primaryVal: "primary-value",
			fallback: "TEST_FALLBACK_4", fallbackVal: "fallback-value",
			defaultVal: "default", expected: "primary-value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.primaryVal != "" {
				t.Setenv(tt.primary, tt.primaryVal)
			}
			if tt.fallbackVal != "" {
				t.Setenv(tt.fallback, tt.fallbackVal)
			}
			got := getEnvWithFallback(tt.primary, tt.fallback, tt.defaultVal)
			if got != tt.expected {
				t.Errorf("getEnvWithFallback() = %q, want %q", got, tt.expected)
			}
		})
	}
}
