package protocolgw

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCloudEventsAdapter_Receive(t *testing.T) {
	adapter := &CloudEventsAdapter{}
	now := time.Now()

	cePayload, _ := json.Marshal(CloudEvent{
		SpecVersion:     "1.0",
		Type:            "com.example.user.created",
		Source:          "/my-service",
		ID:              "evt-123",
		Time:            &now,
		DataContentType: "application/json",
		Data:            json.RawMessage(`{"user_id": "456"}`),
	})

	msg := &GatewayMessage{
		ID:       "msg-1",
		TenantID: "tenant-1",
		Payload:  cePayload,
	}

	result, err := adapter.Receive(context.Background(), msg)
	assert.NoError(t, err)
	assert.Equal(t, "cloudevents", result.SourceProtocol)
	assert.Equal(t, "com.example.user.created", result.EventType)
	assert.Equal(t, "evt-123", result.ID)
}

func TestCloudEventsAdapter_Receive_InvalidEvent(t *testing.T) {
	adapter := &CloudEventsAdapter{}

	// Missing required fields
	cePayload, _ := json.Marshal(map[string]string{"type": "test"})
	msg := &GatewayMessage{Payload: cePayload}

	_, err := adapter.Receive(context.Background(), msg)
	assert.Error(t, err)
}

func TestToCloudEvent(t *testing.T) {
	msg := &GatewayMessage{
		ID:        "msg-123",
		TenantID:  "tenant-1",
		EventType: "order.created",
		Payload:   json.RawMessage(`{"order_id": "789"}`),
		Timestamp: time.Now(),
	}

	ce := ToCloudEvent(msg)
	assert.Equal(t, "1.0", ce.SpecVersion)
	assert.Equal(t, "order.created", ce.Type)
	assert.Equal(t, "msg-123", ce.ID)
	assert.Contains(t, ce.Source, "tenant-1")
	assert.Equal(t, "application/json", ce.DataContentType)
}

func TestFromCloudEvent(t *testing.T) {
	now := time.Now()
	ce := &CloudEvent{
		SpecVersion: "1.0",
		Type:        "user.updated",
		Source:      "/api",
		ID:          "ce-1",
		Time:        &now,
		Data:        json.RawMessage(`{"name": "test"}`),
	}

	msg := FromCloudEvent(ce, "tenant-1")
	assert.Equal(t, "ce-1", msg.ID)
	assert.Equal(t, "user.updated", msg.EventType)
	assert.Equal(t, "cloudevents", msg.SourceProtocol)
	assert.Equal(t, "tenant-1", msg.TenantID)
}

func TestValidateCloudEvent(t *testing.T) {
	tests := []struct {
		name    string
		event   CloudEvent
		wantErr bool
	}{
		{
			name:    "valid event",
			event:   CloudEvent{SpecVersion: "1.0", Type: "test", Source: "/src", ID: "1"},
			wantErr: false,
		},
		{
			name:    "missing specversion",
			event:   CloudEvent{Type: "test", Source: "/src", ID: "1"},
			wantErr: true,
		},
		{
			name:    "missing type",
			event:   CloudEvent{SpecVersion: "1.0", Source: "/src", ID: "1"},
			wantErr: true,
		},
		{
			name:    "unsupported version",
			event:   CloudEvent{SpecVersion: "0.3", Type: "test", Source: "/src", ID: "1"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCloudEvent(&tt.event)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGRPCGatewayAdapter_Receive(t *testing.T) {
	adapter := NewGRPCGatewayAdapter(0)

	msg := &GatewayMessage{
		ID:       "msg-1",
		TenantID: "tenant-1",
		Payload:  json.RawMessage(`{"method": "CreateUser"}`),
	}

	result, err := adapter.Receive(context.Background(), msg)
	assert.NoError(t, err)
	assert.Equal(t, ProtocolGRPC, result.SourceProtocol)
	assert.Equal(t, "application/grpc+json", result.ContentType)
}

func TestGRPCGatewayAdapter_MessageSizeLimit(t *testing.T) {
	adapter := NewGRPCGatewayAdapter(10) // 10 byte limit

	msg := &GatewayMessage{
		Payload: json.RawMessage(`{"very_large_payload": true}`),
	}

	_, err := adapter.Receive(context.Background(), msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds max")
}

func TestMQTTBridgeAdapter_Receive(t *testing.T) {
	adapter := NewMQTTBridgeAdapter()

	msg := &GatewayMessage{
		ID:       "msg-1",
		TenantID: "tenant-1",
		Payload:  json.RawMessage(`{"temperature": 22.5}`),
		Metadata: map[string]interface{}{
			"mqtt_topic": "sensors/temp/living-room",
		},
	}

	result, err := adapter.Receive(context.Background(), msg)
	assert.NoError(t, err)
	assert.Equal(t, ProtocolMQTT, result.SourceProtocol)
	assert.Equal(t, "sensors.temp.living-room", result.EventType)
}

func TestTopicToEventType(t *testing.T) {
	assert.Equal(t, "sensors.temp.room", topicToEventType("sensors/temp/room"))
	assert.Equal(t, "mqtt.message", topicToEventType(""))
}

func TestEventTypeToTopic(t *testing.T) {
	assert.Equal(t, "waas/webhooks/user/created", eventTypeToTopic("user.created"))
	assert.Equal(t, "waas/webhooks/default", eventTypeToTopic(""))
}
