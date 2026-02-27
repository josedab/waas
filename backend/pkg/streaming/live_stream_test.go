package streaming

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLiveStreamService_SubscribeUnsubscribe(t *testing.T) {
	svc := NewLiveStreamService()

	sub := svc.Subscribe("tenant-1", StreamFilter{})
	assert.NotEmpty(t, sub.ID)
	assert.Equal(t, "tenant-1", sub.TenantID)

	stats := svc.GetStats("tenant-1")
	assert.Equal(t, 1, stats.ActiveSubscribers)

	svc.Unsubscribe("tenant-1", sub.ID)
	stats = svc.GetStats("tenant-1")
	assert.Equal(t, 0, stats.ActiveSubscribers)
}

func TestLiveStreamService_Publish(t *testing.T) {
	svc := NewLiveStreamService()

	sub := svc.Subscribe("tenant-1", StreamFilter{})
	defer svc.Unsubscribe("tenant-1", sub.ID)

	event := &LiveDeliveryEvent{
		ID:         "event-1",
		TenantID:   "tenant-1",
		EndpointID: "ep-1",
		EventType:  "user.created",
		Status:     "delivered",
		HTTPStatus: 200,
		LatencyMs:  45.5,
		Timestamp:  time.Now(),
	}

	svc.Publish(event)

	select {
	case received := <-sub.Channel:
		assert.Equal(t, "event-1", received.ID)
		assert.Equal(t, "user.created", received.EventType)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestLiveStreamService_FilteredPublish(t *testing.T) {
	svc := NewLiveStreamService()

	// Subscribe with endpoint filter
	sub := svc.Subscribe("tenant-1", StreamFilter{EndpointIDs: []string{"ep-1"}})
	defer svc.Unsubscribe("tenant-1", sub.ID)

	// Publish event for different endpoint - should not match
	svc.Publish(&LiveDeliveryEvent{
		ID:         "event-1",
		TenantID:   "tenant-1",
		EndpointID: "ep-2",
		Timestamp:  time.Now(),
	})

	// Publish event for matching endpoint
	svc.Publish(&LiveDeliveryEvent{
		ID:         "event-2",
		TenantID:   "tenant-1",
		EndpointID: "ep-1",
		Timestamp:  time.Now(),
	})

	select {
	case received := <-sub.Channel:
		assert.Equal(t, "event-2", received.ID, "should only receive matching event")
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for filtered event")
	}
}

func TestMatchesStreamFilter(t *testing.T) {
	event := &LiveDeliveryEvent{
		EndpointID:   "ep-1",
		EndpointURL:  "https://api.example.com/webhooks",
		EventType:    "order.created",
		Status:       "failed",
		ErrorMessage: "connection refused",
	}

	tests := []struct {
		name   string
		filter StreamFilter
		want   bool
	}{
		{"empty filter matches all", StreamFilter{}, true},
		{"endpoint match", StreamFilter{EndpointIDs: []string{"ep-1"}}, true},
		{"endpoint mismatch", StreamFilter{EndpointIDs: []string{"ep-2"}}, false},
		{"event type match", StreamFilter{EventTypes: []string{"order.created"}}, true},
		{"status match", StreamFilter{Statuses: []string{"failed"}}, true},
		{"status mismatch", StreamFilter{Statuses: []string{"delivered"}}, false},
		{"search in URL", StreamFilter{Search: "example.com"}, true},
		{"search in error", StreamFilter{Search: "connection"}, true},
		{"search mismatch", StreamFilter{Search: "timeout"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, matchesStreamFilter(&tt.filter, event))
		})
	}
}

func TestFormatEventsAsCSV(t *testing.T) {
	events := []*LiveDeliveryEvent{
		{ID: "1", EndpointID: "ep-1", EventType: "user.created", Status: "delivered", HTTPStatus: 200, LatencyMs: 45.5, Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	csv := FormatEventsAsCSV(events)
	assert.Contains(t, csv, "id,endpoint_id,event_type,status")
	assert.Contains(t, csv, "1,ep-1,user.created,delivered,200,45.5")
}

func TestFormatEventsAsNDJSON(t *testing.T) {
	events := []*LiveDeliveryEvent{
		{ID: "1", EndpointID: "ep-1"},
		{ID: "2", EndpointID: "ep-2"},
	}
	ndjson := FormatEventsAsNDJSON(events)
	lines := splitNonEmpty(ndjson)
	assert.Len(t, lines, 2)

	var e1 LiveDeliveryEvent
	assert.NoError(t, json.Unmarshal([]byte(lines[0]), &e1))
	assert.Equal(t, "1", e1.ID)
}

func splitNonEmpty(s string) []string {
	var result []string
	for _, line := range splitLines(s) {
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
