package protocolgw

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ProtocolAdapter translates and delivers messages to a specific protocol
type ProtocolAdapter interface {
	Protocol() string
	Deliver(ctx context.Context, msg *ProtocolMessage) (*TranslationResult, error)
	Close() error
}

// WebSocketAdapter delivers messages via WebSocket connections
type WebSocketAdapter struct {
	config WebSocketConfig
	conn   *websocket.Conn
	mu     sync.Mutex
}

// NewWebSocketAdapter creates a new WebSocket adapter
func NewWebSocketAdapter(config WebSocketConfig) *WebSocketAdapter {
	return &WebSocketAdapter{config: config}
}

func (a *WebSocketAdapter) Protocol() string { return ProtocolWebSocket }

func (a *WebSocketAdapter) Deliver(ctx context.Context, msg *ProtocolMessage) (*TranslationResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	start := time.Now()

	if a.conn == nil {
		dialer := websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
		}
		headers := http.Header{}
		for k, v := range a.config.Headers {
			headers.Set(k, v)
		}
		conn, _, err := dialer.DialContext(ctx, a.config.URL, headers)
		if err != nil {
			return &TranslationResult{
				Success: false,
				Error:   fmt.Sprintf("websocket dial failed: %v", err),
			}, nil
		}
		a.conn = conn
	}

	if err := a.conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
		a.conn = nil
		return &TranslationResult{
			Success: false,
			Error:   fmt.Sprintf("websocket write failed: %v", err),
		}, nil
	}

	return &TranslationResult{
		Success:   true,
		LatencyMs: time.Since(start).Milliseconds(),
	}, nil
}

func (a *WebSocketAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.conn != nil {
		return a.conn.Close()
	}
	return nil
}

// SNSAdapter delivers messages to AWS SNS topics
type SNSAdapter struct {
	config SNSConfig
	client *http.Client
}

// NewSNSAdapter creates a new SNS adapter
func NewSNSAdapter(config SNSConfig) *SNSAdapter {
	return &SNSAdapter{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *SNSAdapter) Protocol() string { return ProtocolSNS }

func (a *SNSAdapter) Deliver(ctx context.Context, msg *ProtocolMessage) (*TranslationResult, error) {
	start := time.Now()

	snsURL := fmt.Sprintf("https://sns.%s.amazonaws.com/", a.config.Region)
	body := fmt.Sprintf("Action=Publish&TopicArn=%s&Message=%s", a.config.TopicARN, msg.Payload)

	req, err := http.NewRequestWithContext(ctx, "POST", snsURL, bytes.NewBufferString(body))
	if err != nil {
		return &TranslationResult{Success: false, Error: err.Error()}, nil
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.client.Do(req)
	if err != nil {
		return &TranslationResult{Success: false, Error: fmt.Sprintf("sns publish failed: %v", err)}, nil
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	return &TranslationResult{
		Success:   resp.StatusCode >= 200 && resp.StatusCode < 300,
		LatencyMs: time.Since(start).Milliseconds(),
		Error:     statusToError(resp.StatusCode),
	}, nil
}

func (a *SNSAdapter) Close() error { return nil }

// EventBridgeAdapter delivers messages to AWS EventBridge
type EventBridgeAdapter struct {
	config EventBridgeConfig
	client *http.Client
}

// NewEventBridgeAdapter creates a new EventBridge adapter
func NewEventBridgeAdapter(config EventBridgeConfig) *EventBridgeAdapter {
	return &EventBridgeAdapter{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *EventBridgeAdapter) Protocol() string { return ProtocolEventBridge }

func (a *EventBridgeAdapter) Deliver(ctx context.Context, msg *ProtocolMessage) (*TranslationResult, error) {
	start := time.Now()

	entry := map[string]interface{}{
		"Entries": []map[string]interface{}{
			{
				"Source":       a.config.Source,
				"DetailType":   a.config.DetailType,
				"Detail":       msg.Payload,
				"EventBusName": a.config.EventBusName,
			},
		},
	}
	body, _ := json.Marshal(entry)

	ebURL := fmt.Sprintf("https://events.%s.amazonaws.com/", a.config.Region)
	req, err := http.NewRequestWithContext(ctx, "POST", ebURL, bytes.NewReader(body))
	if err != nil {
		return &TranslationResult{Success: false, Error: err.Error()}, nil
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSEvents.PutEvents")

	resp, err := a.client.Do(req)
	if err != nil {
		return &TranslationResult{Success: false, Error: fmt.Sprintf("eventbridge put failed: %v", err)}, nil
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	return &TranslationResult{
		Success:   resp.StatusCode >= 200 && resp.StatusCode < 300,
		LatencyMs: time.Since(start).Milliseconds(),
		Error:     statusToError(resp.StatusCode),
	}, nil
}

func (a *EventBridgeAdapter) Close() error { return nil }

// PubSubAdapter delivers messages to GCP Pub/Sub
type PubSubAdapter struct {
	config PubSubConfig
	client *http.Client
}

// NewPubSubAdapter creates a new Pub/Sub adapter
func NewPubSubAdapter(config PubSubConfig) *PubSubAdapter {
	return &PubSubAdapter{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *PubSubAdapter) Protocol() string { return ProtocolPubSub }

func (a *PubSubAdapter) Deliver(ctx context.Context, msg *ProtocolMessage) (*TranslationResult, error) {
	start := time.Now()

	pubsubURL := fmt.Sprintf("https://pubsub.googleapis.com/v1/projects/%s/topics/%s:publish",
		a.config.ProjectID, a.config.TopicID)

	body, _ := json.Marshal(map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"data":       msg.Payload,
				"attributes": a.config.Attributes,
			},
		},
	})

	req, err := http.NewRequestWithContext(ctx, "POST", pubsubURL, bytes.NewReader(body))
	if err != nil {
		return &TranslationResult{Success: false, Error: err.Error()}, nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return &TranslationResult{Success: false, Error: fmt.Sprintf("pubsub publish failed: %v", err)}, nil
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	return &TranslationResult{
		Success:   resp.StatusCode >= 200 && resp.StatusCode < 300,
		LatencyMs: time.Since(start).Milliseconds(),
		Error:     statusToError(resp.StatusCode),
	}, nil
}

func (a *PubSubAdapter) Close() error { return nil }

func statusToError(code int) string {
	if code >= 200 && code < 300 {
		return ""
	}
	return fmt.Sprintf("HTTP %d", code)
}

// GetAdapter returns the appropriate adapter for a protocol
func GetAdapter(protocol string, config map[string]interface{}) (ProtocolAdapter, error) {
	configJSON, _ := json.Marshal(config)

	switch protocol {
	case ProtocolWebSocket:
		var cfg WebSocketConfig
		json.Unmarshal(configJSON, &cfg)
		return NewWebSocketAdapter(cfg), nil
	case ProtocolSNS:
		var cfg SNSConfig
		json.Unmarshal(configJSON, &cfg)
		return NewSNSAdapter(cfg), nil
	case ProtocolEventBridge:
		var cfg EventBridgeConfig
		json.Unmarshal(configJSON, &cfg)
		return NewEventBridgeAdapter(cfg), nil
	case ProtocolPubSub:
		var cfg PubSubConfig
		json.Unmarshal(configJSON, &cfg)
		return NewPubSubAdapter(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}
