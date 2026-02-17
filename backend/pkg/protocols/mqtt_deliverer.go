package protocols

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

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
