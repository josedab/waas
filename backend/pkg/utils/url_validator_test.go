package utils

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateWebhookURL(t *testing.T) {
	validator := NewURLValidator()

	tests := []struct {
		name        string
		url         string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid HTTPS URL",
			url:         "https://api.example.com/webhook",
			expectError: false,
		},
		{
			name:        "valid HTTPS URL with path",
			url:         "https://api.example.com/webhooks/receive",
			expectError: false,
		},
		{
			name:        "valid HTTPS URL with query params",
			url:         "https://api.example.com/webhook?token=abc123",
			expectError: false,
		},
		{
			name:        "valid HTTPS URL with port",
			url:         "https://api.example.com:8443/webhook",
			expectError: false,
		},
		{
			name:        "HTTP URL (not allowed)",
			url:         "http://api.example.com/webhook",
			expectError: true,
			errorMsg:    "must use HTTPS",
		},
		{
			name:        "localhost URL (not allowed)",
			url:         "https://localhost:8080/webhook",
			expectError: true,
			errorMsg:    "localhost",
		},
		{
			name:        "127.0.0.1 URL (not allowed)",
			url:         "https://127.0.0.1:8080/webhook",
			expectError: true,
			errorMsg:    "loopback",
		},
		{
			name:        "0.0.0.0 URL (not allowed)",
			url:         "https://0.0.0.0:8080/webhook",
			expectError: true,
			errorMsg:    "loopback",
		},
		{
			name:        "IPv6 loopback (not allowed)",
			url:         "https://[::1]:8080/webhook",
			expectError: true,
			errorMsg:    "loopback",
		},
		{
			name:        "invalid URL format",
			url:         "not-a-url",
			expectError: true,
			errorMsg:    "must use HTTPS",
		},
		{
			name:        "empty URL",
			url:         "",
			expectError: true,
			errorMsg:    "must use HTTPS",
		},
		{
			name:        "URL without host",
			url:         "https:///webhook",
			expectError: true,
			errorMsg:    "must have a valid host",
		},
		{
			name:        "FTP URL (not allowed)",
			url:         "ftp://example.com/webhook",
			expectError: true,
			errorMsg:    "must use HTTPS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateWebhookURL(tt.url)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckURLAccessibility(t *testing.T) {
	validator := NewURLValidator()
	ctx := context.Background()

	tests := []struct {
		name        string
		url         string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "non-existent domain",
			url:         "https://this-domain-does-not-exist-12345.com/webhook",
			expectError: true,
			errorMsg:    "not accessible",
		},
		{
			name:        "invalid URL format",
			url:         "not-a-url",
			expectError: true,
			errorMsg:    "must use HTTPS",
		},
		{
			name:        "localhost URL",
			url:         "https://localhost:8080/webhook",
			expectError: true,
			errorMsg:    "localhost",
		},
		{
			name:        "127.0.0.1 URL",
			url:         "https://127.0.0.1:8080/webhook",
			expectError: true,
			errorMsg:    "loopback",
		},
		{
			name:        "HTTP URL",
			url:         "http://example.com/webhook",
			expectError: true,
			errorMsg:    "must use HTTPS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set a shorter timeout for tests
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			err := validator.CheckURLAccessibility(ctx, tt.url)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	validator := NewURLValidator()

	tests := []struct {
		name      string
		ip        string
		isPrivate bool
	}{
		// Public IPs
		{name: "Google DNS", ip: "8.8.8.8", isPrivate: false},
		{name: "Cloudflare DNS", ip: "1.1.1.1", isPrivate: false},
		{name: "Public IPv6", ip: "2001:4860:4860::8888", isPrivate: false},
		
		// Private IPv4 ranges
		{name: "10.x.x.x", ip: "10.0.0.1", isPrivate: true},
		{name: "172.16.x.x", ip: "172.16.0.1", isPrivate: true},
		{name: "172.31.x.x", ip: "172.31.255.255", isPrivate: true},
		{name: "192.168.x.x", ip: "192.168.1.1", isPrivate: true},
		
		// Loopback
		{name: "IPv4 loopback", ip: "127.0.0.1", isPrivate: true},
		{name: "IPv6 loopback", ip: "::1", isPrivate: true},
		
		// Link-local
		{name: "IPv4 link-local", ip: "169.254.1.1", isPrivate: true},
		{name: "IPv6 link-local", ip: "fe80::1", isPrivate: true},
		
		// Unique local IPv6
		{name: "IPv6 unique local", ip: "fc00::1", isPrivate: true},
		{name: "IPv6 unique local fd", ip: "fd00::1", isPrivate: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := parseIP(t, tt.ip)
			result := validator.isPrivateIP(ip)
			assert.Equal(t, tt.isPrivate, result, "IP %s should be private=%v", tt.ip, tt.isPrivate)
		})
	}
}

func TestURLValidatorTimeout(t *testing.T) {
	validator := NewURLValidator()
	
	// Set a very short timeout and try to access a real but slow domain
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// This should timeout due to the very short context timeout
	err := validator.CheckURLAccessibility(ctx, "https://httpbin.org/delay/1")
	require.Error(t, err)
	// The error might be timeout or context canceled
	assert.True(t, 
		strings.Contains(err.Error(), "timeout") || 
		strings.Contains(err.Error(), "context") ||
		strings.Contains(err.Error(), "canceled"),
		"Expected timeout or context error, got: %s", err.Error())
}

// Helper function to parse IP addresses for testing
func parseIP(t *testing.T, ipStr string) net.IP {
	ip := net.ParseIP(ipStr)
	require.NotNil(t, ip, "Failed to parse IP: %s", ipStr)
	return ip
}