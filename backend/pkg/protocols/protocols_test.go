package protocols

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsValidProtocol(t *testing.T) {
	tests := []struct {
		protocol Protocol
		valid    bool
	}{
		{ProtocolHTTP, true},
		{ProtocolHTTPS, true},
		{ProtocolGRPC, true},
		{ProtocolGRPCS, true},
		{ProtocolWebSocket, true},
		{ProtocolMQTT, true},
		{Protocol("invalid"), false},
		{Protocol(""), false},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.protocol), func(t *testing.T) {
			if IsValidProtocol(tt.protocol) != tt.valid {
				t.Errorf("IsValidProtocol(%s) = %v, want %v", tt.protocol, !tt.valid, tt.valid)
			}
		})
	}
}

func TestSupportedProtocols(t *testing.T) {
	protocols := SupportedProtocols()
	
	if len(protocols) == 0 {
		t.Error("expected non-empty list of supported protocols")
	}
	
	// Check that required protocols are present
	requiredProtocols := []Protocol{ProtocolHTTP, ProtocolHTTPS, ProtocolGRPC}
	for _, required := range requiredProtocols {
		found := false
		for _, info := range protocols {
			if info.Name == required {
				found = true
				if info.DisplayName == "" {
					t.Errorf("protocol %s should have a display name", required)
				}
				if info.Description == "" {
					t.Errorf("protocol %s should have a description", required)
				}
				break
			}
		}
		if !found {
			t.Errorf("expected protocol %s in supported protocols", required)
		}
	}
}

func TestRegistry(t *testing.T) {
	registry := NewRegistry()
	
	// Register HTTP deliverer
	httpDeliverer := NewHTTPDeliverer()
	registry.Register(ProtocolHTTP, httpDeliverer)
	
	// Get the deliverer back
	deliverer, err := registry.Get(ProtocolHTTP)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deliverer == nil {
		t.Fatal("expected non-nil deliverer")
	}
	
	// Try to get unregistered protocol
	_, err = registry.Get(Protocol("unknown"))
	if err == nil {
		t.Error("expected error for unknown protocol")
	}
	
	// Check supported list
	supported := registry.Supported()
	if len(supported) != 1 {
		t.Errorf("expected 1 supported protocol, got %d", len(supported))
	}
}

func TestDefaultRegistry(t *testing.T) {
	registry := DefaultRegistry()
	
	// Should have all standard protocols registered
	protocols := []Protocol{ProtocolHTTP, ProtocolHTTPS, ProtocolGRPC, ProtocolGRPCS, ProtocolWebSocket, ProtocolMQTT}
	for _, p := range protocols {
		deliverer, err := registry.Get(p)
		if err != nil {
			t.Errorf("expected deliverer for protocol %s: %v", p, err)
		}
		if deliverer == nil {
			t.Errorf("expected non-nil deliverer for protocol %s", p)
		}
	}
}

func TestHTTPDelivererValidate(t *testing.T) {
	deliverer := NewHTTPDeliverer()
	
	tests := []struct {
		name    string
		config  *DeliveryConfig
		wantErr bool
	}{
		{
			name: "valid HTTP URL",
			config: &DeliveryConfig{
				Target:   "http://example.com/webhook",
				Protocol: ProtocolHTTP,
			},
			wantErr: false,
		},
		{
			name: "valid HTTPS URL",
			config: &DeliveryConfig{
				Target:   "https://example.com/webhook",
				Protocol: ProtocolHTTPS,
			},
			wantErr: false,
		},
		{
			name: "empty target",
			config: &DeliveryConfig{
				Target:   "",
				Protocol: ProtocolHTTP,
			},
			wantErr: true,
		},
		{
			name: "invalid URL",
			config: &DeliveryConfig{
				Target:   "not-a-url",
				Protocol: ProtocolHTTP,
			},
			wantErr: true,
		},
		{
			name: "wrong scheme",
			config: &DeliveryConfig{
				Target:   "ftp://example.com/file",
				Protocol: ProtocolHTTP,
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := deliverer.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHTTPDelivererDeliver(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected Content-Type header")
		}
		if r.Header.Get("X-Webhook-ID") == "" {
			t.Error("expected X-Webhook-ID header")
		}
		if r.Header.Get("X-Delivery-ID") == "" {
			t.Error("expected X-Delivery-ID header")
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()
	
	deliverer := NewHTTPDeliverer()
	
	config := &DeliveryConfig{
		Target:   server.URL,
		Protocol: ProtocolHTTP,
		Timeout:  30,
		Options: map[string]interface{}{
			"method": "POST",
		},
	}
	
	request := &DeliveryRequest{
		ID:            "delivery-123",
		WebhookID:     "webhook-456",
		EndpointID:    "endpoint-789",
		Payload:       []byte(`{"event": "test"}`),
		ContentType:   "application/json",
		Headers:       map[string]string{},
		AttemptNumber: 1,
	}
	
	ctx := context.Background()
	response, err := deliverer.Deliver(ctx, config, request)
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if response == nil {
		t.Fatal("expected non-nil response")
	}
	
	if !response.Success {
		t.Errorf("expected success, got error: %s", response.Error)
	}
	
	if response.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", response.StatusCode)
	}
	
	if response.Duration <= 0 {
		t.Error("expected positive duration")
	}
}

func TestHTTPDelivererWithAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	deliverer := NewHTTPDeliverer()
	
	config := &DeliveryConfig{
		Target:   server.URL,
		Protocol: ProtocolHTTP,
		Timeout:  30,
		Auth: &AuthConfig{
			Type: AuthBearer,
			Credentials: map[string]string{
				"token": "test-token",
			},
		},
	}
	
	request := &DeliveryRequest{
		ID:            "delivery-123",
		WebhookID:     "webhook-456",
		EndpointID:    "endpoint-789",
		Payload:       []byte(`{}`),
		ContentType:   "application/json",
		AttemptNumber: 1,
	}
	
	ctx := context.Background()
	response, _ := deliverer.Deliver(ctx, config, request)
	
	if !response.Success {
		t.Errorf("expected success with auth, got: %s", response.Error)
	}
}

func TestHTTPDelivererErrorHandling(t *testing.T) {
	deliverer := NewHTTPDeliverer()
	
	// Test with unreachable server
	config := &DeliveryConfig{
		Target:   "http://localhost:59999/nonexistent",
		Protocol: ProtocolHTTP,
		Timeout:  1,
	}
	
	request := &DeliveryRequest{
		ID:            "delivery-123",
		WebhookID:     "webhook-456",
		Payload:       []byte(`{}`),
		ContentType:   "application/json",
		AttemptNumber: 1,
	}
	
	ctx := context.Background()
	response, _ := deliverer.Deliver(ctx, config, request)
	
	if response.Success {
		t.Error("expected failure for unreachable server")
	}
	
	if response.Error == "" {
		t.Error("expected error message")
	}
	
	if response.ErrorType == "" {
		t.Error("expected error type")
	}
}

func TestHTTPDelivererStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantSuccess bool
		wantErrorType DeliveryErrorType
	}{
		{"200 OK", http.StatusOK, true, ""},
		{"201 Created", http.StatusCreated, true, ""},
		{"202 Accepted", http.StatusAccepted, true, ""},
		{"204 No Content", http.StatusNoContent, true, ""},
		{"400 Bad Request", http.StatusBadRequest, false, ErrorTypeClientError},
		{"401 Unauthorized", http.StatusUnauthorized, false, ErrorTypeClientError},
		{"403 Forbidden", http.StatusForbidden, false, ErrorTypeClientError},
		{"404 Not Found", http.StatusNotFound, false, ErrorTypeClientError},
		{"429 Too Many Requests", http.StatusTooManyRequests, false, ErrorTypeRateLimit},
		{"500 Internal Server Error", http.StatusInternalServerError, false, ErrorTypeServer},
		{"502 Bad Gateway", http.StatusBadGateway, false, ErrorTypeServer},
		{"503 Service Unavailable", http.StatusServiceUnavailable, false, ErrorTypeServer},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()
			
			deliverer := NewHTTPDeliverer()
			config := &DeliveryConfig{
				Target:   server.URL,
				Protocol: ProtocolHTTP,
				Timeout:  30,
			}
			request := &DeliveryRequest{
				ID:          "delivery-123",
				WebhookID:   "webhook-456",
				Payload:     []byte(`{}`),
				ContentType: "application/json",
			}
			
			ctx := context.Background()
			response, _ := deliverer.Deliver(ctx, config, request)
			
			if response.Success != tt.wantSuccess {
				t.Errorf("expected success=%v, got %v", tt.wantSuccess, response.Success)
			}
			
			if tt.wantErrorType != "" && response.ErrorType != tt.wantErrorType {
				t.Errorf("expected error type %s, got %s", tt.wantErrorType, response.ErrorType)
			}
		})
	}
}

func TestGRPCDelivererValidate(t *testing.T) {
	deliverer := NewGRPCDeliverer()
	
	tests := []struct {
		name    string
		config  *DeliveryConfig
		wantErr bool
	}{
		{
			name: "valid gRPC config",
			config: &DeliveryConfig{
				Target:   "localhost:50051",
				Protocol: ProtocolGRPC,
				Options: map[string]interface{}{
					"service": "webhook.WebhookService",
					"method":  "Deliver",
				},
			},
			wantErr: false,
		},
		{
			name: "missing target",
			config: &DeliveryConfig{
				Target:   "",
				Protocol: ProtocolGRPC,
				Options: map[string]interface{}{
					"service": "webhook.WebhookService",
					"method":  "Deliver",
				},
			},
			wantErr: true,
		},
		{
			name: "missing service",
			config: &DeliveryConfig{
				Target:   "localhost:50051",
				Protocol: ProtocolGRPC,
				Options: map[string]interface{}{
					"method": "Deliver",
				},
			},
			wantErr: true,
		},
		{
			name: "missing method",
			config: &DeliveryConfig{
				Target:   "localhost:50051",
				Protocol: ProtocolGRPC,
				Options: map[string]interface{}{
					"service": "webhook.WebhookService",
				},
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := deliverer.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWebSocketDelivererValidate(t *testing.T) {
	deliverer := NewWebSocketDeliverer()
	
	tests := []struct {
		name    string
		config  *DeliveryConfig
		wantErr bool
	}{
		{
			name: "valid ws URL",
			config: &DeliveryConfig{
				Target:   "ws://example.com/ws",
				Protocol: ProtocolWebSocket,
			},
			wantErr: false,
		},
		{
			name: "valid wss URL",
			config: &DeliveryConfig{
				Target:   "wss://example.com/ws",
				Protocol: ProtocolWebSocket,
			},
			wantErr: false,
		},
		{
			name: "invalid scheme",
			config: &DeliveryConfig{
				Target:   "http://example.com/ws",
				Protocol: ProtocolWebSocket,
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := deliverer.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMQTTDelivererValidate(t *testing.T) {
	deliverer := NewMQTTDeliverer()
	
	tests := []struct {
		name    string
		config  *DeliveryConfig
		wantErr bool
	}{
		{
			name: "valid MQTT config",
			config: &DeliveryConfig{
				Target:   "tcp://mqtt.example.com:1883",
				Protocol: ProtocolMQTT,
				Options: map[string]interface{}{
					"topic": "webhooks/events",
				},
			},
			wantErr: false,
		},
		{
			name: "missing topic",
			config: &DeliveryConfig{
				Target:   "tcp://mqtt.example.com:1883",
				Protocol: ProtocolMQTT,
				Options:  map[string]interface{}{},
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := deliverer.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeliveryConfig(t *testing.T) {
	config := &DeliveryConfig{
		ID:         "config-123",
		TenantID:   "tenant-456",
		EndpointID: "endpoint-789",
		Protocol:   ProtocolHTTP,
		Target:     "https://example.com/webhook",
		Headers: map[string]string{
			"X-Custom-Header": "value",
		},
		TLS: &TLSConfig{
			Enabled:            true,
			InsecureSkipVerify: false,
		},
		Auth: &AuthConfig{
			Type: AuthBearer,
			Credentials: map[string]string{
				"token": "secret",
			},
		},
		Timeout:   30,
		Retries:   3,
		Enabled:   true,
		CreatedAt: time.Now(),
	}
	
	if config.Protocol != ProtocolHTTP {
		t.Error("expected HTTP protocol")
	}
	if config.TLS.Enabled != true {
		t.Error("expected TLS enabled")
	}
	if config.Auth.Type != AuthBearer {
		t.Error("expected bearer auth")
	}
}

func TestDeliveryResponse(t *testing.T) {
	response := &DeliveryResponse{
		Success:    true,
		StatusCode: 200,
		Body:       []byte(`{"received": true}`),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Duration:     100 * time.Millisecond,
		ProtocolInfo: map[string]any{"version": "HTTP/2"},
	}
	
	if !response.Success {
		t.Error("expected success")
	}
	if response.Duration <= 0 {
		t.Error("expected positive duration")
	}
}
