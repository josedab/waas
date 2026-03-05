package utils

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// URLValidator provides URL validation and accessibility checking
type URLValidator struct {
	client *http.Client
}

// NewURLValidator creates a new URL validator with a configured HTTP client
func NewURLValidator() *URLValidator {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}

	return &URLValidator{
		client: client,
	}
}

// ValidateWebhookURL validates a webhook URL format and basic security checks
func (v *URLValidator) ValidateWebhookURL(webhookURL string) error {
	// Parse URL
	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Must use HTTPS
	if parsedURL.Scheme != "https" {
		return fmt.Errorf("webhook URL must use HTTPS, got: %s", parsedURL.Scheme)
	}

	// Must have a host
	if parsedURL.Host == "" {
		return fmt.Errorf("webhook URL must have a valid host")
	}

	// Check for localhost/private IPs (basic security check)
	if err := v.checkForPrivateAddresses(parsedURL.Host); err != nil {
		return err
	}

	return nil
}

// CheckURLAccessibility performs an actual HTTP request to verify the URL is accessible
func (v *URLValidator) CheckURLAccessibility(ctx context.Context, webhookURL string) error {
	// First validate the URL format
	if err := v.ValidateWebhookURL(webhookURL); err != nil {
		return err
	}

	// Create a test request (HEAD request to minimize data transfer)
	req, err := http.NewRequestWithContext(ctx, "HEAD", webhookURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	// Set a user agent to identify our validation request
	req.Header.Set("User-Agent", "WebhookPlatform-URLValidator/1.0")

	// Perform the request
	resp, err := v.client.Do(req)
	if err != nil {
		// Check if it's a network error
		var netErr net.Error
		if errors.As(err, &netErr) {
			if netErr.Timeout() {
				return fmt.Errorf("webhook URL is not accessible: connection timeout")
			}
		}
		return fmt.Errorf("webhook URL is not accessible: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response indicates the endpoint can receive webhooks
	// We accept any response that's not a client error (4xx) or server error (5xx)
	// Some endpoints might return 405 Method Not Allowed for HEAD requests, which is acceptable
	if resp.StatusCode >= 400 && resp.StatusCode != 405 {
		return fmt.Errorf("webhook URL returned error status: %d %s", resp.StatusCode, resp.Status)
	}

	return nil
}

// checkForPrivateAddresses checks if the host points to private/local addresses
func (v *URLValidator) checkForPrivateAddresses(host string) error {
	// Extract hostname without port
	hostname := host
	if strings.Contains(host, ":") {
		var err error
		hostname, _, err = net.SplitHostPort(host)
		if err != nil {
			return fmt.Errorf("invalid host format: %w", err)
		}
	}

	// Check for obvious localhost patterns
	if strings.Contains(hostname, "localhost") ||
		hostname == "127.0.0.1" ||
		hostname == "0.0.0.0" ||
		hostname == "::1" {
		return fmt.Errorf("webhook URL cannot point to localhost or loopback addresses")
	}

	// Resolve the hostname to check for private IP ranges
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return fmt.Errorf("failed to resolve hostname %q: %w", hostname, err)
	}

	// Check each resolved IP
	for _, ip := range ips {
		if v.isPrivateIP(ip) {
			return fmt.Errorf("webhook URL resolves to private IP address: %s", ip.String())
		}
	}

	return nil
}

// isPrivateIP checks if an IP address is in a private range
func (v *URLValidator) isPrivateIP(ip net.IP) bool {
	// Private IPv4 ranges:
	// 10.0.0.0/8
	// 172.16.0.0/12
	// 192.168.0.0/16
	// 127.0.0.0/8 (loopback)

	// Private IPv6 ranges:
	// ::1/128 (loopback)
	// fc00::/7 (unique local)
	// fe80::/10 (link-local)

	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Check IPv4 private ranges
	if ipv4 := ip.To4(); ipv4 != nil {
		// 10.0.0.0/8
		if ipv4[0] == 10 {
			return true
		}
		// 172.16.0.0/12
		if ipv4[0] == 172 && ipv4[1] >= 16 && ipv4[1] <= 31 {
			return true
		}
		// 192.168.0.0/16
		if ipv4[0] == 192 && ipv4[1] == 168 {
			return true
		}
	} else {
		// Check IPv6 private ranges
		// fc00::/7 (unique local)
		if ip[0] >= 0xfc && ip[0] <= 0xfd {
			return true
		}
	}

	return false
}
