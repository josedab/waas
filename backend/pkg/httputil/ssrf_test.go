package httputil

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		private bool
	}{
		// Loopback
		{"loopback 127.0.0.1", "127.0.0.1", true},
		{"loopback 127.0.0.2", "127.0.0.2", true},

		// 10.x.x.x
		{"private 10.0.0.1", "10.0.0.1", true},
		{"private 10.255.255.255", "10.255.255.255", true},

		// 172.16-31.x.x
		{"private 172.16.0.1", "172.16.0.1", true},
		{"private 172.31.255.255", "172.31.255.255", true},
		{"public 172.15.0.1", "172.15.0.1", false},
		{"public 172.32.0.1", "172.32.0.1", false},

		// 192.168.x.x
		{"private 192.168.0.1", "192.168.0.1", true},
		{"private 192.168.255.255", "192.168.255.255", true},
		{"public 192.169.0.1", "192.169.0.1", false},

		// Link-local
		{"link-local 169.254.1.1", "169.254.1.1", true},

		// 0.0.0.0/8
		{"zero network 0.0.0.0", "0.0.0.0", true},
		{"zero network 0.1.2.3", "0.1.2.3", true},

		// Public IPs
		{"public 8.8.8.8", "8.8.8.8", false},
		{"public 1.1.1.1", "1.1.1.1", false},
		{"public 93.184.216.34", "93.184.216.34", false},

		// IPv6 loopback
		{"ipv6 loopback", "::1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP %q", tt.ip)
			}
			got := IsPrivateIP(ip)
			if got != tt.private {
				t.Errorf("IsPrivateIP(%s) = %v, want %v", tt.ip, got, tt.private)
			}
		})
	}
}

func TestSSRFSafeDialContext_BlocksPrivateIPs(t *testing.T) {
	dialer := &net.Dialer{Timeout: 2 * time.Second}
	dialFunc := SSRFSafeDialContext(dialer)
	ctx := context.Background()

	// Connecting to loopback should be blocked by SSRF protection
	_, err := dialFunc(ctx, "tcp", "127.0.0.1:80")
	if err == nil {
		t.Error("expected error when dialing private IP, got nil")
	}
}

func TestSSRFSafeDialContext_InvalidAddress(t *testing.T) {
	dialer := &net.Dialer{Timeout: 2 * time.Second}
	dialFunc := SSRFSafeDialContext(dialer)
	ctx := context.Background()

	_, err := dialFunc(ctx, "tcp", "no-port")
	if err == nil {
		t.Error("expected error for invalid address, got nil")
	}
}

func TestNewSSRFSafeTransport(t *testing.T) {
	transport := NewSSRFSafeTransport()
	if transport == nil {
		t.Fatal("expected non-nil transport")
	}
	if transport.DialContext == nil {
		t.Error("expected DialContext to be set")
	}
	if transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify to be false")
	}
}

func TestNewSSRFSafeClient(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		want    time.Duration
	}{
		{"zero uses default", 0, DefaultRequestTimeout},
		{"custom timeout", 10 * time.Second, 10 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewSSRFSafeClient(tt.timeout)
			if client == nil {
				t.Fatal("expected non-nil client")
			}
			if client.Timeout != tt.want {
				t.Errorf("client.Timeout = %v, want %v", client.Timeout, tt.want)
			}
		})
	}
}
