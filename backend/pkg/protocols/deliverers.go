package protocols

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// HTTPDeliverer implements HTTP/HTTPS webhook delivery
type HTTPDeliverer struct {
	client *http.Client
}

// NewHTTPDeliverer creates a new HTTP deliverer
func NewHTTPDeliverer() *HTTPDeliverer {
	return &HTTPDeliverer{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// Protocol returns the protocol
func (d *HTTPDeliverer) Protocol() Protocol {
	return ProtocolHTTP
}

// Validate validates the delivery config
func (d *HTTPDeliverer) Validate(config *DeliveryConfig) error {
	if config.Target == "" {
		return fmt.Errorf("target URL is required")
	}

	u, err := url.Parse(config.Target)
	if err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme: %s (expected http or https)", u.Scheme)
	}

	return nil
}

// Deliver performs the HTTP delivery
func (d *HTTPDeliverer) Deliver(ctx context.Context, config *DeliveryConfig, request *DeliveryRequest) (*DeliveryResponse, error) {
	start := time.Now()
	response := &DeliveryResponse{}

	// Get HTTP options
	opts := parseHTTPOptions(config.Options)

	// Build request
	method := opts.Method
	if method == "" {
		method = http.MethodPost
	}

	targetURL := config.Target
	if opts.Path != "" {
		if !strings.HasSuffix(targetURL, "/") && !strings.HasPrefix(opts.Path, "/") {
			targetURL += "/"
		}
		targetURL += opts.Path
	}

	// Add query params
	if len(opts.QueryParams) > 0 {
		u, err := url.Parse(targetURL)
		if err == nil {
			q := u.Query()
			for k, v := range opts.QueryParams {
				q.Set(k, v)
			}
			u.RawQuery = q.Encode()
			targetURL = u.String()
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL, bytes.NewReader(request.Payload))
	if err != nil {
		response.Duration = time.Since(start)
		response.Error = err.Error()
		response.ErrorType = ErrorTypeConnection
		return response, nil
	}

	// Set headers
	req.Header.Set("Content-Type", request.ContentType)
	if request.ContentType == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set delivery-specific headers
	for k, v := range request.Headers {
		req.Header.Set(k, v)
	}

	// Set config headers
	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	// Add standard webhook headers
	req.Header.Set("X-Webhook-ID", request.WebhookID)
	req.Header.Set("X-Delivery-ID", request.ID)
	req.Header.Set("X-Delivery-Attempt", fmt.Sprintf("%d", request.AttemptNumber))

	// Apply authentication
	if config.Auth != nil {
		applyHTTPAuth(req, config.Auth)
	}

	// Create client with custom TLS if needed
	client := d.client
	if config.TLS != nil && config.TLS.Enabled {
		transport := &http.Transport{
			TLSClientConfig: buildTLSConfig(config.TLS),
		}
		client = &http.Client{
			Timeout:   time.Duration(config.Timeout) * time.Second,
			Transport: transport,
		}
	}

	if config.Timeout > 0 {
		client.Timeout = time.Duration(config.Timeout) * time.Second
	}

	// Configure redirects
	if !opts.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else if opts.MaxRedirects > 0 {
		maxRedirects := opts.MaxRedirects
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			return nil
		}
	}

	// Perform request
	resp, err := client.Do(req)
	response.Duration = time.Since(start)

	if err != nil {
		response.Error = err.Error()
		response.ErrorType = categorizeHTTPError(err)
		return response, nil
	}
	defer resp.Body.Close()

	response.StatusCode = resp.StatusCode
	response.Headers = make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			response.Headers[k] = v[0]
		}
	}

	// Read body (limited to 1MB)
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	response.Body = body

	// Check for retry-after header
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		if seconds := parseRetryAfter(retryAfter); seconds > 0 {
			d := time.Duration(seconds) * time.Second
			response.RetryAfter = &d
		}
	}

	// Determine success
	expectedStatuses := opts.ExpectedStatuses
	if len(expectedStatuses) == 0 {
		expectedStatuses = []int{200, 201, 202, 204}
	}

	response.Success = false
	for _, status := range expectedStatuses {
		if resp.StatusCode == status {
			response.Success = true
			break
		}
	}

	if !response.Success {
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			response.ErrorType = ErrorTypeClientError
		} else if resp.StatusCode >= 500 {
			response.ErrorType = ErrorTypeServer
		}
		if resp.StatusCode == 429 {
			response.ErrorType = ErrorTypeRateLimit
		}
		response.Error = fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
	}

	return response, nil
}

// Close closes the deliverer
func (d *HTTPDeliverer) Close() error {
	d.client.CloseIdleConnections()
	return nil
}

func parseHTTPOptions(opts map[string]interface{}) HTTPOptions {
	options := HTTPOptions{
		Method:          "POST",
		FollowRedirects: true,
		MaxRedirects:    10,
	}

	if opts == nil {
		return options
	}

	if method, ok := opts["method"].(string); ok {
		options.Method = method
	}
	if path, ok := opts["path"].(string); ok {
		options.Path = path
	}
	if qp, ok := opts["query_params"].(map[string]interface{}); ok {
		options.QueryParams = make(map[string]string)
		for k, v := range qp {
			options.QueryParams[k] = fmt.Sprintf("%v", v)
		}
	}
	if fr, ok := opts["follow_redirects"].(bool); ok {
		options.FollowRedirects = fr
	}
	if mr, ok := opts["max_redirects"].(float64); ok {
		options.MaxRedirects = int(mr)
	}
	if es, ok := opts["expected_statuses"].([]interface{}); ok {
		options.ExpectedStatuses = make([]int, 0, len(es))
		for _, s := range es {
			if status, ok := s.(float64); ok {
				options.ExpectedStatuses = append(options.ExpectedStatuses, int(status))
			}
		}
	}

	return options
}

func applyHTTPAuth(req *http.Request, auth *AuthConfig) {
	if auth == nil {
		return
	}

	switch auth.Type {
	case AuthBasic:
		username := auth.Credentials["username"]
		password := auth.Credentials["password"]
		req.SetBasicAuth(username, password)
	case AuthBearer:
		token := auth.Credentials["token"]
		req.Header.Set("Authorization", "Bearer "+token)
	case AuthAPIKey:
		header := auth.Credentials["header"]
		if header == "" {
			header = "X-API-Key"
		}
		key := auth.Credentials["key"]
		req.Header.Set(header, key)
	}
}

func buildTLSConfig(config *TLSConfig) *tls.Config {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.InsecureSkipVerify,
	}

	if config.ServerName != "" {
		tlsConfig.ServerName = config.ServerName
	}

	return tlsConfig
}

func categorizeHTTPError(err error) DeliveryErrorType {
	errStr := err.Error()

	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline") {
		return ErrorTypeTimeout
	}
	if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "no such host") {
		return ErrorTypeConnection
	}
	if strings.Contains(errStr, "certificate") || strings.Contains(errStr, "tls") {
		return ErrorTypeTLS
	}

	return ErrorTypeConnection
}

func parseRetryAfter(value string) int {
	// Try parsing as seconds
	var seconds int
	if _, err := fmt.Sscanf(value, "%d", &seconds); err == nil {
		return seconds
	}

	// Try parsing as HTTP date
	if t, err := http.ParseTime(value); err == nil {
		return int(time.Until(t).Seconds())
	}

	return 0
}

// GRPCDeliverer implements gRPC webhook delivery
type GRPCDeliverer struct {
	mu          sync.RWMutex
	connections map[string]*grpcConnection
}

type grpcConnection struct {
	conn     *grpc.ClientConn
	target   string
	lastUsed time.Time
	mu       sync.Mutex
}

// NewGRPCDeliverer creates a new gRPC deliverer
func NewGRPCDeliverer() *GRPCDeliverer {
	return &GRPCDeliverer{
		connections: make(map[string]*grpcConnection),
	}
}

// Protocol returns the protocol
func (d *GRPCDeliverer) Protocol() Protocol {
	return ProtocolGRPC
}

// Validate validates the delivery config
func (d *GRPCDeliverer) Validate(config *DeliveryConfig) error {
	if config.Target == "" {
		return fmt.Errorf("target is required")
	}

	opts := parseGRPCOptions(config.Options)
	if opts.Service == "" {
		return fmt.Errorf("grpc service name is required")
	}
	if opts.Method == "" {
		return fmt.Errorf("grpc method name is required")
	}

	return nil
}

// Deliver performs the gRPC delivery
func (d *GRPCDeliverer) Deliver(ctx context.Context, config *DeliveryConfig, request *DeliveryRequest) (*DeliveryResponse, error) {
	start := time.Now()
	response := &DeliveryResponse{
		ProtocolInfo: make(map[string]any),
	}

	opts := parseGRPCOptions(config.Options)

	// Get or create connection
	conn, err := d.getConnection(ctx, config, opts)
	if err != nil {
		response.Duration = time.Since(start)
		response.Error = err.Error()
		response.ErrorType = ErrorTypeConnection
		return response, nil
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	// Build method name: /package.Service/Method
	fullMethod := fmt.Sprintf("/%s/%s", opts.Service, opts.Method)

	// Set timeout
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Add metadata
	if len(opts.Metadata) > 0 {
		md := metadata.New(opts.Metadata)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	// Create call options
	var callOpts []grpc.CallOption
	if opts.MaxMessageSize > 0 {
		callOpts = append(callOpts, grpc.MaxCallRecvMsgSize(opts.MaxMessageSize))
		callOpts = append(callOpts, grpc.MaxCallSendMsgSize(opts.MaxMessageSize))
	}
	if opts.WaitForReady {
		callOpts = append(callOpts, grpc.WaitForReady(true))
	}

	// Invoke the method with raw bytes (JSON payload will be sent as-is)
	// For full proto support, we would need proto descriptors
	var responseBytes []byte
	err = conn.conn.Invoke(ctx, fullMethod, request.Payload, &responseBytes, callOpts...)

	conn.lastUsed = time.Now()

	if err != nil {
		response.Duration = time.Since(start)
		response.Error = err.Error()
		response.ErrorType = categorizeGRPCError(err)
		return response, nil
	}

	response.Duration = time.Since(start)
	response.Success = true
	response.Body = responseBytes
	response.ProtocolInfo["service"] = opts.Service
	response.ProtocolInfo["method"] = opts.Method
	response.ProtocolInfo["target"] = config.Target

	return response, nil
}

func (d *GRPCDeliverer) getConnection(ctx context.Context, config *DeliveryConfig, opts GRPCOptions) (*grpcConnection, error) {
	d.mu.RLock()
	conn, exists := d.connections[config.Target]
	d.mu.RUnlock()

	if exists && conn.conn.GetState() != connectivity.Shutdown && time.Since(conn.lastUsed) < 5*time.Minute {
		return conn, nil
	}

	// Create new connection
	var dialOpts []grpc.DialOption

	// Configure TLS
	if config.TLS != nil && config.TLS.Enabled {
		tlsConfig := buildTLSConfig(config.TLS)
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Connection timeout
	connTimeout := 10 * time.Second
	if opts.ConnectionTimeout > 0 {
		connTimeout = time.Duration(opts.ConnectionTimeout) * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	// Create the connection
	grpcConn, err := grpc.DialContext(ctx, config.Target, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("grpc dial failed: %w", err)
	}

	newConn := &grpcConnection{
		conn:     grpcConn,
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

// Close closes the deliverer
func (d *GRPCDeliverer) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, conn := range d.connections {
		conn.conn.Close()
	}
	d.connections = make(map[string]*grpcConnection)
	return nil
}

func categorizeGRPCError(err error) DeliveryErrorType {
	if err == nil {
		return ""
	}

	st, ok := status.FromError(err)
	if !ok {
		return ErrorTypeServer
	}

	switch st.Code() {
	case codes.DeadlineExceeded:
		return ErrorTypeTimeout
	case codes.Unavailable, codes.Aborted:
		return ErrorTypeConnection
	case codes.Unauthenticated, codes.PermissionDenied:
		return ErrorTypeAuth
	case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.FailedPrecondition:
		return ErrorTypeClientError
	case codes.ResourceExhausted:
		return ErrorTypeRateLimit
	default:
		return ErrorTypeServer
	}
}

func parseGRPCOptions(opts map[string]interface{}) GRPCOptions {
	options := GRPCOptions{}

	if opts == nil {
		return options
	}

	if service, ok := opts["service"].(string); ok {
		options.Service = service
	}
	if method, ok := opts["method"].(string); ok {
		options.Method = method
	}
	if protoFile, ok := opts["proto_file"].(string); ok {
		options.ProtoFile = protoFile
	}
	if metadata, ok := opts["metadata"].(map[string]interface{}); ok {
		options.Metadata = make(map[string]string)
		for k, v := range metadata {
			options.Metadata[k] = fmt.Sprintf("%v", v)
		}
	}
	if maxSize, ok := opts["max_message_size"].(float64); ok {
		options.MaxMessageSize = int(maxSize)
	}
	if waitForReady, ok := opts["wait_for_ready"].(bool); ok {
		options.WaitForReady = waitForReady
	}

	return options
}

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

// MQTTDeliverer implements MQTT webhook delivery
type MQTTDeliverer struct {
	mu      sync.RWMutex
	clients map[string]*mqttClientConn
}

type mqttClientConn struct {
	client   mqtt.Client
	broker   string
	clientID string
	lastUsed time.Time
	mu       sync.Mutex
}

// NewMQTTDeliverer creates a new MQTT deliverer
func NewMQTTDeliverer() *MQTTDeliverer {
	return &MQTTDeliverer{
		clients: make(map[string]*mqttClientConn),
	}
}

// Protocol returns the protocol
func (d *MQTTDeliverer) Protocol() Protocol {
	return ProtocolMQTT
}

// Validate validates the delivery config
func (d *MQTTDeliverer) Validate(config *DeliveryConfig) error {
	if config.Target == "" {
		return fmt.Errorf("target broker is required")
	}

	opts := parseMQTTOptions(config.Options)
	if opts.Topic == "" {
		return fmt.Errorf("mqtt topic is required")
	}

	return nil
}

// Deliver performs the MQTT delivery
func (d *MQTTDeliverer) Deliver(ctx context.Context, config *DeliveryConfig, request *DeliveryRequest) (*DeliveryResponse, error) {
	start := time.Now()
	response := &DeliveryResponse{
		ProtocolInfo: make(map[string]any),
	}

	opts := parseMQTTOptions(config.Options)

	// Get or create client
	client, err := d.getClient(ctx, config, opts)
	if err != nil {
		response.Duration = time.Since(start)
		response.Error = err.Error()
		response.ErrorType = ErrorTypeConnection
		return response, nil
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	// Set timeout
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Publish the message
	token := client.client.Publish(opts.Topic, byte(opts.QoS), opts.Retain, request.Payload)

	// Wait for publish to complete
	if !token.WaitTimeout(timeout) {
		response.Duration = time.Since(start)
		response.Error = "publish timeout"
		response.ErrorType = ErrorTypeTimeout
		return response, nil
	}

	if err := token.Error(); err != nil {
		response.Duration = time.Since(start)
		response.Error = err.Error()
		response.ErrorType = categorizeMQTTError(err)
		return response, nil
	}

	client.lastUsed = time.Now()

	response.Duration = time.Since(start)
	response.Success = true
	response.ProtocolInfo["broker"] = config.Target
	response.ProtocolInfo["topic"] = opts.Topic
	response.ProtocolInfo["qos"] = opts.QoS
	response.ProtocolInfo["retain"] = opts.Retain

	return response, nil
}

func (d *MQTTDeliverer) getClient(ctx context.Context, config *DeliveryConfig, opts MQTTOptions) (*mqttClientConn, error) {
	key := config.Target

	d.mu.RLock()
	client, exists := d.clients[key]
	d.mu.RUnlock()

	if exists && client.client.IsConnected() && time.Since(client.lastUsed) < 5*time.Minute {
		return client, nil
	}

	// Create new client
	clientID := opts.ClientID
	if clientID == "" {
		clientID = fmt.Sprintf("waas-%s-%d", config.Target, time.Now().UnixNano())
	}

	mqttOpts := mqtt.NewClientOptions()
	mqttOpts.AddBroker(config.Target)
	mqttOpts.SetClientID(clientID)
	mqttOpts.SetCleanSession(opts.CleanStart)
	mqttOpts.SetAutoReconnect(true)
	mqttOpts.SetConnectTimeout(10 * time.Second)
	mqttOpts.SetWriteTimeout(30 * time.Second)

	// Set credentials if provided
	if opts.Username != "" {
		mqttOpts.SetUsername(opts.Username)
	}
	if opts.Password != "" {
		mqttOpts.SetPassword(opts.Password)
	}

	// Configure TLS if needed
	if config.TLS != nil && config.TLS.Enabled {
		mqttOpts.SetTLSConfig(buildTLSConfig(config.TLS))
	}

	// Apply auth
	if config.Auth != nil {
		if config.Auth.Type == AuthBasic {
			mqttOpts.SetUsername(config.Auth.Credentials["username"])
			mqttOpts.SetPassword(config.Auth.Credentials["password"])
		}
	}

	mqttClient := mqtt.NewClient(mqttOpts)

	// Connect
	timeout := 10 * time.Second
	token := mqttClient.Connect()
	if !token.WaitTimeout(timeout) {
		return nil, fmt.Errorf("mqtt connection timeout")
	}
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("mqtt connection failed: %w", err)
	}

	newClient := &mqttClientConn{
		client:   mqttClient,
		broker:   config.Target,
		clientID: clientID,
		lastUsed: time.Now(),
	}

	d.mu.Lock()
	// Disconnect old client if exists
	if oldClient, ok := d.clients[key]; ok {
		oldClient.client.Disconnect(1000)
	}
	d.clients[key] = newClient
	d.mu.Unlock()

	return newClient, nil
}

// Close closes the deliverer
func (d *MQTTDeliverer) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, client := range d.clients {
		client.client.Disconnect(1000)
	}
	d.clients = make(map[string]*mqttClientConn)
	return nil
}

func categorizeMQTTError(err error) DeliveryErrorType {
	if err == nil {
		return ""
	}

	errStr := err.Error()

	if strings.Contains(errStr, "timeout") {
		return ErrorTypeTimeout
	}
	if strings.Contains(errStr, "connection") || strings.Contains(errStr, "network") {
		return ErrorTypeConnection
	}
	if strings.Contains(errStr, "not authorized") || strings.Contains(errStr, "not authorised") {
		return ErrorTypeClientError
	}

	return ErrorTypeServer
}

func parseMQTTOptions(opts map[string]interface{}) MQTTOptions {
	options := MQTTOptions{
		QoS: 1,
	}

	if opts == nil {
		return options
	}

	if topic, ok := opts["topic"].(string); ok {
		options.Topic = topic
	}
	if qos, ok := opts["qos"].(float64); ok {
		options.QoS = int(qos)
	}
	if retain, ok := opts["retain"].(bool); ok {
		options.Retain = retain
	}
	if clientID, ok := opts["client_id"].(string); ok {
		options.ClientID = clientID
	}
	if cleanStart, ok := opts["clean_start"].(bool); ok {
		options.CleanStart = cleanStart
	}
	if username, ok := opts["username"].(string); ok {
		options.Username = username
	}
	if password, ok := opts["password"].(string); ok {
		options.Password = password
	}

	return options
}

// JSONToGRPCPayload converts a JSON payload to a map for gRPC
func JSONToGRPCPayload(data []byte) (map[string]interface{}, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON payload: %w", err)
	}
	return payload, nil
}
