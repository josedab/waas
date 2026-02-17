package protocols

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketDeliverer implements WebSocket webhook delivery
type WebSocketDeliverer struct {
	mu          sync.RWMutex
	connections map[string]*wsConnection
}

type wsConnection struct {
	conn     *websocket.Conn
	target   string
	lastUsed time.Time
	mu       sync.Mutex
}

// NewWebSocketDeliverer creates a new WebSocket deliverer
func NewWebSocketDeliverer() *WebSocketDeliverer {
	return &WebSocketDeliverer{
		connections: make(map[string]*wsConnection),
	}
}

// Protocol returns the protocol
func (d *WebSocketDeliverer) Protocol() Protocol {
	return ProtocolWebSocket
}

// Validate validates the delivery config
func (d *WebSocketDeliverer) Validate(config *DeliveryConfig) error {
	if config.Target == "" {
		return fmt.Errorf("target is required")
	}

	u, err := url.Parse(config.Target)
	if err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}

	if u.Scheme != "ws" && u.Scheme != "wss" {
		return fmt.Errorf("invalid URL scheme: %s (expected ws or wss)", u.Scheme)
	}

	return nil
}

// Deliver performs the WebSocket delivery
func (d *WebSocketDeliverer) Deliver(ctx context.Context, config *DeliveryConfig, request *DeliveryRequest) (*DeliveryResponse, error) {
	start := time.Now()
	response := &DeliveryResponse{
		ProtocolInfo: make(map[string]any),
	}

	opts := parseWebSocketOptions(config.Options)

	// Get or create connection
	conn, err := d.getConnection(ctx, config, opts)
	if err != nil {
		response.Duration = time.Since(start)
		response.Error = err.Error()
		response.ErrorType = ErrorTypeConnection
		return response, nil
	}

	// Build target URL with path
	targetURL := config.Target
	if opts.Path != "" {
		if !strings.HasSuffix(targetURL, "/") && !strings.HasPrefix(opts.Path, "/") {
			targetURL += "/"
		}
		targetURL += opts.Path
	}

	// Set write deadline
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	if err := conn.conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		response.Duration = time.Since(start)
		response.Error = fmt.Sprintf("failed to set write deadline: %v", err)
		response.ErrorType = ErrorTypeConnection
		return response, nil
	}

	// Send the message
	messageType := websocket.TextMessage
	if opts.BinaryMode {
		messageType = websocket.BinaryMessage
	}

	if err := conn.conn.WriteMessage(messageType, request.Payload); err != nil {
		response.Duration = time.Since(start)
		response.Error = err.Error()
		response.ErrorType = categorizeWebSocketError(err)
		// Remove bad connection from cache
		d.removeConnection(config.Target)
		return response, nil
	}

	// Optionally wait for response
	if opts.WaitForResponse {
		if err := conn.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			response.Duration = time.Since(start)
			response.Error = fmt.Sprintf("failed to set read deadline: %v", err)
			response.ErrorType = ErrorTypeConnection
			return response, nil
		}

		_, respBody, err := conn.conn.ReadMessage()
		if err != nil {
			response.Duration = time.Since(start)
			response.Error = err.Error()
			response.ErrorType = categorizeWebSocketError(err)
			return response, nil
		}
		response.Body = respBody
	}

	conn.lastUsed = time.Now()

	response.Duration = time.Since(start)
	response.Success = true
	response.ProtocolInfo["target"] = targetURL
	response.ProtocolInfo["message_type"] = messageType

	return response, nil
}

func (d *WebSocketDeliverer) getConnection(ctx context.Context, config *DeliveryConfig, opts WebSocketOptions) (*wsConnection, error) {
	d.mu.RLock()
	conn, exists := d.connections[config.Target]
	d.mu.RUnlock()

	if exists && time.Since(conn.lastUsed) < 5*time.Minute {
		return conn, nil
	}

	// Create new connection
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	// Configure TLS
	if config.TLS != nil && config.TLS.Enabled {
		dialer.TLSClientConfig = buildTLSConfig(config.TLS)
	}

	// Prepare headers
	headers := http.Header{}
	for k, v := range config.Headers {
		headers.Set(k, v)
	}
	for k, v := range opts.Headers {
		headers.Set(k, v)
	}

	// Apply authentication
	if config.Auth != nil {
		switch config.Auth.Type {
		case AuthBearer:
			headers.Set("Authorization", "Bearer "+config.Auth.Credentials["token"])
		case AuthAPIKey:
			header := config.Auth.Credentials["header"]
			if header == "" {
				header = "X-API-Key"
			}
			headers.Set(header, config.Auth.Credentials["key"])
		}
	}

	// Connect
	targetURL := config.Target
	if opts.Path != "" {
		if !strings.HasSuffix(targetURL, "/") && !strings.HasPrefix(opts.Path, "/") {
			targetURL += "/"
		}
		targetURL += opts.Path
	}

	wsConn, _, err := dialer.DialContext(ctx, targetURL, headers)
	if err != nil {
		return nil, fmt.Errorf("websocket dial failed: %w", err)
	}

	newConn := &wsConnection{
		conn:     wsConn,
		target:   config.Target,
		lastUsed: time.Now(),
	}

	d.mu.Lock()
	// Close old connection if exists
	if oldConn, ok := d.connections[config.Target]; ok {
		oldConn.conn.Close()
	}
	d.connections[config.Target] = newConn
	d.mu.Unlock()

	return newConn, nil
}

func (d *WebSocketDeliverer) removeConnection(target string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if conn, ok := d.connections[target]; ok {
		conn.conn.Close()
		delete(d.connections, target)
	}
}

// Close closes the deliverer
func (d *WebSocketDeliverer) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, conn := range d.connections {
		conn.conn.Close()
	}
	d.connections = make(map[string]*wsConnection)
	return nil
}

func categorizeWebSocketError(err error) DeliveryErrorType {
	if err == nil {
		return ""
	}

	errStr := err.Error()

	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline") {
		return ErrorTypeTimeout
	}
	if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "no such host") {
		return ErrorTypeConnection
	}
	if strings.Contains(errStr, "going away") || strings.Contains(errStr, "close") {
		return ErrorTypeServer
	}

	return ErrorTypeConnection
}

func parseWebSocketOptions(opts map[string]interface{}) WebSocketOptions {
	options := WebSocketOptions{}

	if opts == nil {
		return options
	}

	if path, ok := opts["path"].(string); ok {
		options.Path = path
	}
	if subprotocols, ok := opts["subprotocols"].([]interface{}); ok {
		options.Subprotocols = make([]string, 0, len(subprotocols))
		for _, sp := range subprotocols {
			if s, ok := sp.(string); ok {
				options.Subprotocols = append(options.Subprotocols, s)
			}
		}
	}
	if headers, ok := opts["headers"].(map[string]interface{}); ok {
		options.Headers = make(map[string]string)
		for k, v := range headers {
			options.Headers[k] = fmt.Sprintf("%v", v)
		}
	}
	if pingInterval, ok := opts["ping_interval"].(float64); ok {
		options.PingInterval = int(pingInterval)
	}
	if reconnect, ok := opts["reconnect_on_failure"].(bool); ok {
		options.ReconnectOnFailure = reconnect
	}
	if binaryMode, ok := opts["binary_mode"].(bool); ok {
		options.BinaryMode = binaryMode
	}
	if waitForResponse, ok := opts["wait_for_response"].(bool); ok {
		options.WaitForResponse = waitForResponse
	}

	return options
}
