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

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"a,b,c", []string{"a", "b", "c"}},
		{" a , b , c ", []string{"a", "b", "c"}},
		{"single", []string{"single"}},
		{"a,,b", []string{"a", "b"}},
		{",", nil},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitCSV(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("splitCSV(%q) = %v (len %d), want %v (len %d)", tt.input, got, len(got), tt.expected, len(tt.expected))
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("splitCSV(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestLoadConfig_RequiredVars(t *testing.T) {
	// All required vars missing → error
	_, err := LoadConfig()
	if err == nil {
		t.Fatal("LoadConfig() should fail when required vars are missing")
	}
}

func TestLoadConfig_Success(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/waas")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("JWT_SECRET", "test-secret-for-unit-tests")
	t.Setenv("ADMIN_TENANT_IDS", "tid-1, tid-2")
	t.Setenv("DELIVERY_HEALTH_PORT", "8081")
	t.Setenv("ALLOW_INSECURE_TLS", "true")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.DeliveryHealthPort != "8081" {
		t.Errorf("DeliveryHealthPort = %q, want %q", cfg.DeliveryHealthPort, "8081")
	}
	if len(cfg.AdminTenantIDs) != 2 || cfg.AdminTenantIDs[0] != "tid-1" {
		t.Errorf("AdminTenantIDs = %v, want [tid-1 tid-2]", cfg.AdminTenantIDs)
	}
	if !cfg.AllowInsecureTLS {
		t.Error("AllowInsecureTLS should be true")
	}
}

func TestConfig_IsProd(t *testing.T) {
	c := &Config{Environment: "production"}
	if !c.IsProd() {
		t.Error("IsProd() should be true for production")
	}
	c.Environment = "development"
	if c.IsProd() {
		t.Error("IsProd() should be false for development")
	}
}
