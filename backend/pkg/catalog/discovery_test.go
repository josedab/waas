package catalog

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordTraffic(t *testing.T) {
	svc := NewService(nil)

	count, err := svc.RecordTraffic(context.Background(), "tenant-1", &RecordTrafficRequest{
		Samples: []TrafficSample{
			{EventType: "order.created", EndpointID: "ep-1", Payload: json.RawMessage(`{"id": "123"}`), Timestamp: time.Now()},
			{EventType: "order.created", EndpointID: "ep-2", Payload: json.RawMessage(`{"id": "456"}`), Timestamp: time.Now()},
			{EventType: "payment.completed", EndpointID: "ep-1", Payload: json.RawMessage(`{"amount": 99.99}`), Timestamp: time.Now()},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	discoveries, err := svc.ListDiscoveries(context.Background(), "tenant-1")
	require.NoError(t, err)
	assert.Len(t, discoveries, 2)
}

func TestRecordTrafficEmpty(t *testing.T) {
	svc := NewService(nil)
	_, err := svc.RecordTraffic(context.Background(), "t1", &RecordTrafficRequest{})
	assert.Error(t, err)
}

func TestDiscoverySummary(t *testing.T) {
	svc := NewService(nil)

	// Record some traffic with unique tenant
	_, _ = svc.RecordTraffic(context.Background(), "tenant-summary-1", &RecordTrafficRequest{
		Samples: []TrafficSample{
			{EventType: "order.created", Payload: json.RawMessage(`{}`), Timestamp: time.Now()},
			{EventType: "user.signed_up", Payload: json.RawMessage(`{}`), Timestamp: time.Now()},
		},
	})

	summary, err := svc.GetDiscoverySummary(context.Background(), "tenant-summary-1")
	require.NoError(t, err)
	assert.Equal(t, 2, summary.TotalDiscovered)
	assert.Equal(t, 2, summary.PendingReview)
	assert.NotNil(t, summary.LastScanAt)
}

func TestInferSchema(t *testing.T) {
	svc := NewService(nil)

	_, _ = svc.RecordTraffic(context.Background(), "tenant-1", &RecordTrafficRequest{
		Samples: []TrafficSample{
			{EventType: "order.created", Payload: json.RawMessage(`{"id": "123", "amount": 99.99, "active": true}`), Timestamp: time.Now()},
		},
	})

	inference, err := svc.InferSchema(context.Background(), "tenant-1", "order.created")
	require.NoError(t, err)
	assert.Equal(t, "order.created", inference.EventType)
	assert.NotEmpty(t, inference.Fields)
	assert.NotNil(t, inference.Schema)
}

func TestInferSchemaNotFound(t *testing.T) {
	svc := NewService(nil)
	_, err := svc.InferSchema(context.Background(), "t1", "nonexistent")
	assert.Error(t, err)
}

func TestSuggestCategory(t *testing.T) {
	assert.Equal(t, "order", suggestCategory("order.created"))
	assert.Equal(t, "payment", suggestCategory("payment.completed.refund"))
	assert.Equal(t, "general", suggestCategory(""))
}

func TestSuggestTags(t *testing.T) {
	tags := suggestTags("order.created")
	assert.Contains(t, tags, "order")
	assert.Contains(t, tags, "created")
}

func TestSlugify(t *testing.T) {
	assert.Equal(t, "order-created", slugify("order.created"))
	assert.Equal(t, "my-event-type", slugify("My Event Type"))
}
