package delivery

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/queue"
	"github.com/josedab/waas/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestIsPrivateIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ip       net.IP
		expected bool
	}{
		// Private/blocked IPs
		{"loopback 127.0.0.1", net.ParseIP("127.0.0.1"), true},
		{"loopback 127.0.0.2", net.ParseIP("127.0.0.2"), true},
		{"10.0.0.0/8", net.ParseIP("10.0.0.1"), true},
		{"10.255.255.255", net.ParseIP("10.255.255.255"), true},
		{"172.16.0.0/12 low", net.ParseIP("172.16.0.1"), true},
		{"172.16.0.0/12 high", net.ParseIP("172.31.255.255"), true},
		{"192.168.0.0/16", net.ParseIP("192.168.1.1"), true},
		{"192.168.255.255", net.ParseIP("192.168.255.255"), true},
		{"link-local 169.254.0.0/16", net.ParseIP("169.254.169.254"), true},
		{"0.0.0.0", net.ParseIP("0.0.0.0"), true},
		{"IPv6 loopback ::1", net.IPv6loopback, true},

		// Public IPs (should be allowed)
		{"public 8.8.8.8", net.ParseIP("8.8.8.8"), false},
		{"public 1.1.1.1", net.ParseIP("1.1.1.1"), false},
		{"public 203.0.113.1", net.ParseIP("203.0.113.1"), false},
		{"172.15.255.255 (not in 172.16/12)", net.ParseIP("172.15.255.255"), false},
		{"172.32.0.0 (not in 172.16/12)", net.ParseIP("172.32.0.0"), false},
		{"public 100.64.0.1", net.ParseIP("100.64.0.1"), false},

		// Edge cases: octal/decimal representations
		// net.ParseIP normalizes these, so 0177.0.0.1 won't parse as octal
		// Testing the parsed IP directly
		{"127.0.0.1 as IPv4", net.IPv4(127, 0, 0, 1), true},
		{"0.0.0.0 as IPv4", net.IPv4(0, 0, 0, 0), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isPrivateIP(tt.ip)
			assert.Equal(t, tt.expected, result, "isPrivateIP(%s) = %v, want %v", tt.ip, result, tt.expected)
		})
	}
}

func TestIsPrivateIP_DecimalAndOctalEdgeCases(t *testing.T) {
	t.Parallel()

	// 2130706433 is the decimal representation of 127.0.0.1
	// Go's net.ParseIP doesn't handle decimal IPs, but the raw IP bytes would
	ip := net.IPv4(127, 0, 0, 1) // equivalent of decimal 2130706433
	assert.True(t, isPrivateIP(ip), "decimal equivalent of 127.0.0.1 should be private")

	// 0177.0.0.1 in octal = 127.0.0.1, test the actual IP
	ip = net.IPv4(0x7f, 0, 0, 1) // 0177 octal = 0x7f = 127
	assert.True(t, isPrivateIP(ip), "octal equivalent of 127.0.0.1 should be private")
}

func TestSsrfSafeDialContext_BlocksPrivateIPs(t *testing.T) {
	t.Parallel()

	dialer := &net.Dialer{Timeout: 1 * time.Second}
	dialFunc := ssrfSafeDialContext(dialer)

	// Attempting to connect to localhost should be blocked
	_, err := dialFunc(context.Background(), "tcp", "127.0.0.1:80")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SSRF protection")
}

func TestSsrfSafeDialContext_InvalidAddress(t *testing.T) {
	t.Parallel()

	dialer := &net.Dialer{Timeout: 1 * time.Second}
	dialFunc := ssrfSafeDialContext(dialer)

	// Missing port
	_, err := dialFunc(context.Background(), "tcp", "invalid-no-port")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid address")
}

func TestSsrfSafeDialContext_DNSResolutionFailure(t *testing.T) {
	t.Parallel()

	dialer := &net.Dialer{Timeout: 1 * time.Second}
	dialFunc := ssrfSafeDialContext(dialer)

	// Non-existent domain
	_, err := dialFunc(context.Background(), "tcp", "this-domain-does-not-exist-xyzzy.invalid:80")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DNS resolution failed")
}

// TestRouteToDeadLetter_NilHook verifies no panic when dlqHook is nil
func TestRouteToDeadLetter_NilHook(t *testing.T) {
	t.Parallel()

	engine := &DeliveryEngine{
		logger: utils.NewLogger("test"),
	}

	message := &queue.DeliveryMessage{
		DeliveryID: uuid.New(),
		EndpointID: uuid.New(),
		TenantID:   uuid.New(),
		Payload:    json.RawMessage(`{"event":"test"}`),
		Headers:    map[string]string{"X-Test": "value"},
	}
	errMsg := "max retries exceeded"
	result := &queue.DeliveryResult{
		DeliveryID:    message.DeliveryID,
		Status:        queue.StatusFailed,
		ErrorMessage:  &errMsg,
		AttemptNumber: 3,
	}

	// Should not panic with nil hook
	assert.NotPanics(t, func() {
		engine.routeToDeadLetter(context.Background(), message, result)
	})
}

// TestRouteToDeadLetter_HookInvoked verifies the DLQ hook receives correct data
func TestRouteToDeadLetter_HookInvoked(t *testing.T) {
	t.Parallel()

	engine := &DeliveryEngine{
		logger: utils.NewLogger("test"),
	}

	var capturedTenantID, capturedEndpointID, capturedDeliveryID, capturedFinalError string
	var capturedAttempts []DLQAttemptDetail
	var capturedPayload, capturedHeaders json.RawMessage

	engine.SetDLQHook(func(ctx context.Context, tenantID, endpointID, deliveryID string, payload json.RawMessage, headers json.RawMessage, attempts []DLQAttemptDetail, finalError string) {
		capturedTenantID = tenantID
		capturedEndpointID = endpointID
		capturedDeliveryID = deliveryID
		capturedPayload = payload
		capturedHeaders = headers
		capturedAttempts = attempts
		capturedFinalError = finalError
	})

	tenantID := uuid.New()
	endpointID := uuid.New()
	deliveryID := uuid.New()
	message := &queue.DeliveryMessage{
		DeliveryID: deliveryID,
		EndpointID: endpointID,
		TenantID:   tenantID,
		Payload:    json.RawMessage(`{"event":"test.failed"}`),
		Headers:    map[string]string{"X-Event": "test"},
	}
	httpStatus := 500
	errMsg := "HTTP 500: Internal Server Error"
	result := &queue.DeliveryResult{
		DeliveryID:    deliveryID,
		Status:        queue.StatusFailed,
		HTTPStatus:    &httpStatus,
		ErrorMessage:  &errMsg,
		AttemptNumber: 3,
	}

	engine.routeToDeadLetter(context.Background(), message, result)

	assert.Equal(t, tenantID.String(), capturedTenantID)
	assert.Equal(t, endpointID.String(), capturedEndpointID)
	assert.Equal(t, deliveryID.String(), capturedDeliveryID)
	assert.Equal(t, errMsg, capturedFinalError)
	assert.NotEmpty(t, capturedPayload)
	assert.NotEmpty(t, capturedHeaders)
	assert.Len(t, capturedAttempts, 1)
	assert.Equal(t, 3, capturedAttempts[0].AttemptNumber)
	assert.Equal(t, &httpStatus, capturedAttempts[0].HTTPStatus)
	assert.Equal(t, &errMsg, capturedAttempts[0].ErrorMessage)
}

// TestRouteToDeadLetter_NilErrorMessage verifies empty string when ErrorMessage is nil
func TestRouteToDeadLetter_NilErrorMessage(t *testing.T) {
	t.Parallel()

	engine := &DeliveryEngine{
		logger: utils.NewLogger("test"),
	}

	var capturedFinalError string
	engine.SetDLQHook(func(ctx context.Context, tenantID, endpointID, deliveryID string, payload json.RawMessage, headers json.RawMessage, attempts []DLQAttemptDetail, finalError string) {
		capturedFinalError = finalError
	})

	message := &queue.DeliveryMessage{
		DeliveryID: uuid.New(),
		EndpointID: uuid.New(),
		TenantID:   uuid.New(),
		Payload:    json.RawMessage(`{}`),
		Headers:    map[string]string{},
	}
	result := &queue.DeliveryResult{
		DeliveryID:    message.DeliveryID,
		Status:        queue.StatusFailed,
		ErrorMessage:  nil,
		AttemptNumber: 1,
	}

	engine.routeToDeadLetter(context.Background(), message, result)

	assert.Equal(t, "", capturedFinalError)
}
