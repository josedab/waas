package httputil

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"
)

// Default transport settings for SSRF-safe HTTP clients.
const (
	DefaultDialTimeout    = 30 * time.Second
	DefaultDialKeepAlive  = 30 * time.Second
	DefaultRequestTimeout = 30 * time.Second
)

// IsPrivateIP returns true if the IP is in a private/internal range that
// should not be reachable via user-provided URLs (SSRF protection).
func IsPrivateIP(ip net.IP) bool {
	// Check IPv4-mapped IPv6 addresses (e.g., ::ffff:10.0.0.1)
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}

	privateRanges := []net.IPNet{
		{IP: net.IPv4(127, 0, 0, 0), Mask: net.CIDRMask(8, 32)},    // 127.0.0.0/8
		{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},     // 10.0.0.0/8
		{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},  // 172.16.0.0/12
		{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)}, // 192.168.0.0/16
		{IP: net.IPv4(169, 254, 0, 0), Mask: net.CIDRMask(16, 32)}, // 169.254.0.0/16
		{IP: net.IPv4(0, 0, 0, 0), Mask: net.CIDRMask(8, 32)},      // 0.0.0.0/8
	}

	ipv6PrivateRanges := []net.IPNet{
		{IP: net.ParseIP("fc00::"), Mask: net.CIDRMask(7, 128)},  // fc00::/7 (IPv6 ULA)
		{IP: net.ParseIP("fe80::"), Mask: net.CIDRMask(10, 128)}, // fe80::/10 (IPv6 link-local)
	}

	for _, cidr := range privateRanges {
		if cidr.Contains(ip) {
			return true
		}
	}
	for _, cidr := range ipv6PrivateRanges {
		if cidr.Contains(ip) {
			return true
		}
	}
	if ip.Equal(net.IPv6loopback) {
		return true
	}
	return false
}

// SSRFSafeDialContext wraps a net.Dialer to reject connections to private IPs.
func SSRFSafeDialContext(dialer *net.Dialer) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid address %q: %w", addr, err)
		}

		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("DNS resolution failed for %q: %w", host, err)
		}

		for _, ip := range ips {
			if IsPrivateIP(ip.IP) {
				return nil, fmt.Errorf("connections to private/internal IP %s are not allowed (SSRF protection)", ip.IP)
			}
		}

		return dialer.DialContext(ctx, network, net.JoinHostPort(host, port))
	}
}

// NewSSRFSafeTransport creates an http.Transport with SSRF protection.
func NewSSRFSafeTransport() *http.Transport {
	dialer := &net.Dialer{
		Timeout:   DefaultDialTimeout,
		KeepAlive: DefaultDialKeepAlive,
	}
	return &http.Transport{
		DialContext:         SSRFSafeDialContext(dialer),
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: false},
	}
}

// NewSSRFSafeClient creates an http.Client with SSRF protection and the given timeout.
func NewSSRFSafeClient(timeout time.Duration) *http.Client {
	if timeout == 0 {
		timeout = DefaultRequestTimeout
	}
	return &http.Client{
		Transport: NewSSRFSafeTransport(),
		Timeout:   timeout,
	}
}
