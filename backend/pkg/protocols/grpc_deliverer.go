package protocols

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

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

func JSONToGRPCPayload(data []byte) (map[string]interface{}, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON payload: %w", err)
	}
	return payload, nil
}
