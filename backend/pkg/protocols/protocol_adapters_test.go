package protocols

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- Kafka Deliverer Tests ---

func TestKafkaDelivererProtocol(t *testing.T) {
	d := NewKafkaDeliverer()
	if d.Protocol() != ProtocolKafka {
		t.Errorf("expected protocol %s, got %s", ProtocolKafka, d.Protocol())
	}
}

func TestKafkaDelivererValidate(t *testing.T) {
	d := NewKafkaDeliverer()

	tests := []struct {
		name    string
		config  *DeliveryConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &DeliveryConfig{
				Target:   "localhost:9092",
				Protocol: ProtocolKafka,
				Options: map[string]interface{}{
					"topic": "webhooks",
				},
			},
			wantErr: false,
		},
		{
			name: "missing target",
			config: &DeliveryConfig{
				Target:   "",
				Protocol: ProtocolKafka,
				Options: map[string]interface{}{
					"topic": "webhooks",
				},
			},
			wantErr: true,
		},
		{
			name: "missing topic",
			config: &DeliveryConfig{
				Target:   "localhost:9092",
				Protocol: ProtocolKafka,
				Options:  map[string]interface{}{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := d.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKafkaDelivererClose(t *testing.T) {
	d := NewKafkaDeliverer()
	if err := d.Close(); err != nil {
		t.Errorf("Close() unexpected error: %v", err)
	}
}

func TestParseKafkaOptions(t *testing.T) {
	opts := parseKafkaOptions(map[string]interface{}{
		"topic":         "my-topic",
		"partition_key": "key1",
		"compression":   "gzip",
		"ack_mode":      "leader",
		"idempotent":    true,
		"max_retries":   float64(5),
		"brokers":       []interface{}{"broker1:9092", "broker2:9092"},
	})

	if opts.Topic != "my-topic" {
		t.Errorf("expected topic my-topic, got %s", opts.Topic)
	}
	if opts.PartitionKey != "key1" {
		t.Errorf("expected partition_key key1, got %s", opts.PartitionKey)
	}
	if opts.Compression != "gzip" {
		t.Errorf("expected compression gzip, got %s", opts.Compression)
	}
	if opts.AckMode != "leader" {
		t.Errorf("expected ack_mode leader, got %s", opts.AckMode)
	}
	if !opts.Idempotent {
		t.Error("expected idempotent true")
	}
	if opts.MaxRetries != 5 {
		t.Errorf("expected max_retries 5, got %d", opts.MaxRetries)
	}
	if len(opts.Brokers) != 2 {
		t.Errorf("expected 2 brokers, got %d", len(opts.Brokers))
	}
}

func TestParseKafkaOptionsDefaults(t *testing.T) {
	opts := parseKafkaOptions(nil)
	if opts.AckMode != "all" {
		t.Errorf("expected default ack_mode 'all', got %s", opts.AckMode)
	}
	if opts.MaxRetries != 3 {
		t.Errorf("expected default max_retries 3, got %d", opts.MaxRetries)
	}
}

func TestCategorizeKafkaError(t *testing.T) {
	tests := []struct {
		errMsg   string
		expected DeliveryErrorType
	}{
		{"connection refused", ErrorTypeConnection},
		{"request timeout", ErrorTypeTimeout},
		{"authorization failed", ErrorTypeAuth},
		{"unknown error", ErrorTypeServer},
	}
	for _, tt := range tests {
		got := categorizeKafkaError(fmt.Errorf("%s", tt.errMsg))
		if got != tt.expected {
			t.Errorf("categorizeKafkaError(%q) = %s, want %s", tt.errMsg, got, tt.expected)
		}
	}
}

// --- SNS Deliverer Tests ---

func TestSNSDelivererProtocol(t *testing.T) {
	d := NewSNSDeliverer()
	if d.Protocol() != ProtocolSNS {
		t.Errorf("expected protocol %s, got %s", ProtocolSNS, d.Protocol())
	}
}

func TestSNSDelivererValidate(t *testing.T) {
	d := NewSNSDeliverer()

	tests := []struct {
		name    string
		config  *DeliveryConfig
		wantErr bool
	}{
		{
			name: "valid SNS ARN",
			config: &DeliveryConfig{
				Target:   "arn:aws:sns:us-east-1:123456789012:my-topic",
				Protocol: ProtocolSNS,
			},
			wantErr: false,
		},
		{
			name: "empty target",
			config: &DeliveryConfig{
				Target:   "",
				Protocol: ProtocolSNS,
			},
			wantErr: true,
		},
		{
			name: "invalid ARN format",
			config: &DeliveryConfig{
				Target:   "not-an-arn",
				Protocol: ProtocolSNS,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := d.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSNSDelivererClose(t *testing.T) {
	d := NewSNSDeliverer()
	if err := d.Close(); err != nil {
		t.Errorf("Close() unexpected error: %v", err)
	}
}

func TestParseSNSOptions(t *testing.T) {
	opts := parseSNSOptions(map[string]interface{}{
		"topic_arn":        "arn:aws:sns:us-east-1:123456789012:my-topic",
		"message_group_id": "group-1",
		"region":           "us-west-2",
		"subject":          "Test Subject",
		"attributes": map[string]interface{}{
			"key1": "value1",
		},
	})

	if opts.TopicARN != "arn:aws:sns:us-east-1:123456789012:my-topic" {
		t.Errorf("unexpected topic ARN: %s", opts.TopicARN)
	}
	if opts.MessageGroupID != "group-1" {
		t.Errorf("unexpected message group ID: %s", opts.MessageGroupID)
	}
	if opts.Region != "us-west-2" {
		t.Errorf("unexpected region: %s", opts.Region)
	}
	if opts.Subject != "Test Subject" {
		t.Errorf("unexpected subject: %s", opts.Subject)
	}
	if opts.Attributes["key1"] != "value1" {
		t.Error("expected attribute key1=value1")
	}
}

// --- SQS Deliverer Tests ---

func TestSQSDelivererProtocol(t *testing.T) {
	d := NewSQSDeliverer()
	if d.Protocol() != ProtocolSQS {
		t.Errorf("expected protocol %s, got %s", ProtocolSQS, d.Protocol())
	}
}

func TestSQSDelivererValidate(t *testing.T) {
	d := NewSQSDeliverer()

	tests := []struct {
		name    string
		config  *DeliveryConfig
		wantErr bool
	}{
		{
			name: "valid SQS URL",
			config: &DeliveryConfig{
				Target:   "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue",
				Protocol: ProtocolSQS,
			},
			wantErr: false,
		},
		{
			name: "empty target",
			config: &DeliveryConfig{
				Target:   "",
				Protocol: ProtocolSQS,
			},
			wantErr: true,
		},
		{
			name: "invalid URL format",
			config: &DeliveryConfig{
				Target:   "https://example.com/not-sqs",
				Protocol: ProtocolSQS,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := d.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSQSDelivererClose(t *testing.T) {
	d := NewSQSDeliverer()
	if err := d.Close(); err != nil {
		t.Errorf("Close() unexpected error: %v", err)
	}
}

func TestParseSQSOptions(t *testing.T) {
	opts := parseSQSOptions(map[string]interface{}{
		"queue_url":        "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue",
		"message_group_id": "group-1",
		"delay_seconds":    float64(10),
		"region":           "us-east-1",
		"attributes": map[string]interface{}{
			"env": "prod",
		},
	})

	if opts.QueueURL != "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue" {
		t.Errorf("unexpected queue URL: %s", opts.QueueURL)
	}
	if opts.MessageGroupID != "group-1" {
		t.Errorf("unexpected message group ID: %s", opts.MessageGroupID)
	}
	if opts.DelaySeconds != 10 {
		t.Errorf("unexpected delay seconds: %d", opts.DelaySeconds)
	}
	if opts.Region != "us-east-1" {
		t.Errorf("unexpected region: %s", opts.Region)
	}
	if opts.Attributes["env"] != "prod" {
		t.Error("expected attribute env=prod")
	}
}

func TestCategorizeSNSSQSError(t *testing.T) {
	tests := []struct {
		errMsg   string
		expected DeliveryErrorType
	}{
		{"request timeout", ErrorTypeTimeout},
		{"AccessDenied: not allowed", ErrorTypeAuth},
		{"NonExistentQueue", ErrorTypeClientError},
		{"Throttling: too many requests", ErrorTypeRateLimit},
		{"some other error", ErrorTypeServer},
	}
	for _, tt := range tests {
		got := categorizeSNSSQSError(fmt.Errorf("%s", tt.errMsg))
		if got != tt.expected {
			t.Errorf("categorizeSNSSQSError(%q) = %s, want %s", tt.errMsg, got, tt.expected)
		}
	}
}

// --- Protocol Adapters Tests ---

func TestProtocolObserver(t *testing.T) {
	observer := NewProtocolObserver()

	// Record some metrics
	observer.Record(ProtocolHTTP, 100, true)
	observer.Record(ProtocolHTTP, 200, true)
	observer.Record(ProtocolHTTP, 50, false)

	metrics := observer.GetMetrics(ProtocolHTTP)
	if metrics.DeliveryCount != 3 {
		t.Errorf("expected delivery count 3, got %d", metrics.DeliveryCount)
	}
	if metrics.SuccessCount != 2 {
		t.Errorf("expected success count 2, got %d", metrics.SuccessCount)
	}
	if metrics.ErrorCount != 1 {
		t.Errorf("expected error count 1, got %d", metrics.ErrorCount)
	}
	if metrics.TotalLatencyMs != 350 {
		t.Errorf("expected total latency 350ms, got %d", metrics.TotalLatencyMs)
	}

	avgLatency := metrics.AverageLatencyMs()
	if avgLatency < 116 || avgLatency > 117 {
		t.Errorf("expected avg latency ~116.67, got %f", avgLatency)
	}

	errorRate := metrics.ErrorRate()
	if errorRate < 0.33 || errorRate > 0.34 {
		t.Errorf("expected error rate ~0.33, got %f", errorRate)
	}
}

func TestProtocolObserverUnknownProtocol(t *testing.T) {
	observer := NewProtocolObserver()
	metrics := observer.GetMetrics(Protocol("unknown"))
	if metrics.DeliveryCount != 0 {
		t.Errorf("expected 0 deliveries for unknown protocol, got %d", metrics.DeliveryCount)
	}
}

func TestProtocolObserverAllMetrics(t *testing.T) {
	observer := NewProtocolObserver()
	observer.Record(ProtocolHTTP, 100, true)
	observer.Record(ProtocolKafka, 50, false)

	all := observer.AllMetrics()
	if len(all) != 2 {
		t.Errorf("expected 2 protocols, got %d", len(all))
	}
	if all[ProtocolHTTP].DeliveryCount != 1 {
		t.Error("expected 1 HTTP delivery")
	}
	if all[ProtocolKafka].ErrorCount != 1 {
		t.Error("expected 1 Kafka error")
	}
}

func TestProtocolMetricsZero(t *testing.T) {
	m := &ProtocolMetrics{}
	if m.AverageLatencyMs() != 0 {
		t.Error("expected 0 average latency for empty metrics")
	}
	if m.ErrorRate() != 0 {
		t.Error("expected 0 error rate for empty metrics")
	}
}

func TestNewProtocolAdapter(t *testing.T) {
	observer := NewProtocolObserver()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	deliverer := NewHTTPDeliverer()
	adapter := NewProtocolAdapter(deliverer, observer)

	if adapter.Name() != ProtocolHTTP {
		t.Errorf("expected protocol HTTP, got %s", adapter.Name())
	}

	config := &DeliveryConfig{
		Target:   server.URL,
		Protocol: ProtocolHTTP,
		Timeout:  5,
	}
	request := &DeliveryRequest{
		ID:          "d-1",
		WebhookID:   "w-1",
		Payload:     []byte(`{"test":true}`),
		ContentType: "application/json",
	}

	result, err := adapter.Deliver(context.Background(), config, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Response.Success {
		t.Errorf("expected success, got error: %s", result.Response.Error)
	}
	if result.Protocol != ProtocolHTTP {
		t.Errorf("expected protocol HTTP in result, got %s", result.Protocol)
	}
	if result.DurationMs < 0 {
		t.Error("expected non-negative duration")
	}

	// Check observer was updated
	metrics := observer.GetMetrics(ProtocolHTTP)
	if metrics.DeliveryCount != 1 {
		t.Errorf("expected 1 delivery in observer, got %d", metrics.DeliveryCount)
	}
	if metrics.SuccessCount != 1 {
		t.Errorf("expected 1 success in observer, got %d", metrics.SuccessCount)
	}
}

func TestNewProtocolAdapterValidate(t *testing.T) {
	adapter := NewProtocolAdapter(NewHTTPDeliverer(), nil)

	err := adapter.ValidateConfig(&DeliveryConfig{Target: "http://example.com"})
	if err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}

	err = adapter.ValidateConfig(&DeliveryConfig{Target: ""})
	if err == nil {
		t.Error("expected validation error for empty target")
	}
}

func TestNewProtocolAdapterClose(t *testing.T) {
	adapter := NewProtocolAdapter(NewHTTPDeliverer(), nil)
	if err := adapter.Close(); err != nil {
		t.Errorf("unexpected close error: %v", err)
	}
}

func TestNewDelivererForProtocol(t *testing.T) {
	tests := []struct {
		protocol Protocol
		wantErr  bool
	}{
		{ProtocolHTTP, false},
		{ProtocolHTTPS, false},
		{ProtocolGRPC, false},
		{ProtocolGRPCS, false},
		{ProtocolWebSocket, false},
		{ProtocolMQTT, false},
		{ProtocolGraphQL, false},
		{ProtocolSMTP, false},
		{ProtocolKafka, false},
		{ProtocolSNS, false},
		{ProtocolSQS, false},
		{Protocol("unknown"), true},
	}

	for _, tt := range tests {
		t.Run(string(tt.protocol), func(t *testing.T) {
			d, err := NewDelivererForProtocol(tt.protocol)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDelivererForProtocol(%s) error = %v, wantErr %v", tt.protocol, err, tt.wantErr)
			}
			if !tt.wantErr && d == nil {
				t.Error("expected non-nil deliverer")
			}
		})
	}
}

func TestUnifiedDeliverer(t *testing.T) {
	observer := NewProtocolObserver()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	registry := NewRegistry()
	registry.Register(ProtocolHTTP, NewHTTPDeliverer())

	ud := NewUnifiedDeliverer(registry, observer)

	udc := &UnifiedDeliveryConfig{
		EndpointID:      "ep-1",
		PrimaryProtocol: ProtocolHTTP,
		Configs: map[Protocol]*DeliveryConfig{
			ProtocolHTTP: {
				Target:   server.URL,
				Protocol: ProtocolHTTP,
				Timeout:  5,
			},
		},
	}

	request := &DeliveryRequest{
		ID:          "d-1",
		WebhookID:   "w-1",
		Payload:     []byte(`{}`),
		ContentType: "application/json",
	}

	result, err := ud.Deliver(context.Background(), udc, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Response.Success {
		t.Errorf("expected success: %s", result.Response.Error)
	}
}

func TestUnifiedDelivererMissingConfig(t *testing.T) {
	registry := NewRegistry()
	registry.Register(ProtocolHTTP, NewHTTPDeliverer())
	ud := NewUnifiedDeliverer(registry, nil)

	udc := &UnifiedDeliveryConfig{
		EndpointID:      "ep-1",
		PrimaryProtocol: ProtocolKafka,
		Configs:         map[Protocol]*DeliveryConfig{},
	}

	_, err := ud.Deliver(context.Background(), udc, &DeliveryRequest{})
	if err == nil {
		t.Error("expected error for missing config")
	}
}

func TestUnifiedDelivererFallback(t *testing.T) {
	observer := NewProtocolObserver()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	registry := NewRegistry()
	registry.Register(ProtocolHTTP, NewHTTPDeliverer())

	ud := NewUnifiedDeliverer(registry, observer)

	// Primary will fail (bad target), fallback will succeed
	udc := &UnifiedDeliveryConfig{
		EndpointID:       "ep-1",
		PrimaryProtocol:  ProtocolHTTP,
		FallbackProtocol: ProtocolHTTP,
		Configs: map[Protocol]*DeliveryConfig{
			ProtocolHTTP: {
				Target:   server.URL,
				Protocol: ProtocolHTTP,
				Timeout:  5,
			},
		},
	}

	request := &DeliveryRequest{
		ID:          "d-1",
		WebhookID:   "w-1",
		Payload:     []byte(`{}`),
		ContentType: "application/json",
	}

	result, err := ud.Deliver(context.Background(), udc, request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Response.Success {
		t.Errorf("expected success: %s", result.Response.Error)
	}
}

func TestIsValidProtocolNewProtocols(t *testing.T) {
	tests := []struct {
		protocol Protocol
		valid    bool
	}{
		{ProtocolKafka, true},
		{ProtocolSNS, true},
		{ProtocolSQS, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.protocol), func(t *testing.T) {
			if IsValidProtocol(tt.protocol) != tt.valid {
				t.Errorf("IsValidProtocol(%s) = %v, want %v", tt.protocol, !tt.valid, tt.valid)
			}
		})
	}
}

func TestDefaultRegistryNewProtocols(t *testing.T) {
	registry := DefaultRegistry()

	for _, p := range []Protocol{ProtocolKafka, ProtocolSNS, ProtocolSQS} {
		d, err := registry.Get(p)
		if err != nil {
			t.Errorf("expected deliverer for protocol %s: %v", p, err)
		}
		if d == nil {
			t.Errorf("expected non-nil deliverer for protocol %s", p)
		}
	}
}
