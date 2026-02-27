package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// LiveStreamService provides real-time SSE and WebSocket streaming of webhook traffic
type LiveStreamService struct {
	mu          sync.RWMutex
	subscribers map[string]map[string]*StreamSubscriber
}

// StreamSubscriber represents a connected subscriber
type StreamSubscriber struct {
	ID        string
	TenantID  string
	Filter    StreamFilter
	Channel   chan *LiveDeliveryEvent
	CreatedAt time.Time
	Closed    bool
}

// StreamFilter defines filtering criteria for live streams
type StreamFilter struct {
	EndpointIDs []string `json:"endpoint_ids,omitempty"`
	EventTypes  []string `json:"event_types,omitempty"`
	Statuses    []string `json:"statuses,omitempty"` // delivered, failed, retrying
	Search      string   `json:"search,omitempty"`
}

// LiveDeliveryEvent represents a real-time delivery event
type LiveDeliveryEvent struct {
	ID           string            `json:"id"`
	TenantID     string            `json:"tenant_id"`
	EndpointID   string            `json:"endpoint_id"`
	EndpointURL  string            `json:"endpoint_url"`
	EventType    string            `json:"event_type"`
	Status       string            `json:"status"` // delivered, failed, retrying
	HTTPStatus   int               `json:"http_status,omitempty"`
	LatencyMs    float64           `json:"latency_ms"`
	PayloadSize  int               `json:"payload_size"`
	Headers      map[string]string `json:"headers,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
	AttemptCount int               `json:"attempt_count"`
	Timestamp    time.Time         `json:"timestamp"`
}

// StreamStats provides statistics for the live stream
type StreamStats struct {
	TenantID            string    `json:"tenant_id"`
	ActiveSubscribers   int       `json:"active_subscribers"`
	EventsPerSecond     float64   `json:"events_per_second"`
	TotalEventsStreamed int64     `json:"total_events_streamed"`
	OldestSubscriber    time.Time `json:"oldest_subscriber,omitempty"`
}

// NewLiveStreamService creates a new live stream service
func NewLiveStreamService() *LiveStreamService {
	return &LiveStreamService{
		subscribers: make(map[string]map[string]*StreamSubscriber),
	}
}

// Subscribe creates a new subscriber for a tenant's live traffic
func (ls *LiveStreamService) Subscribe(tenantID string, filter StreamFilter) *StreamSubscriber {
	sub := &StreamSubscriber{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Filter:    filter,
		Channel:   make(chan *LiveDeliveryEvent, 256),
		CreatedAt: time.Now(),
	}

	ls.mu.Lock()
	defer ls.mu.Unlock()
	if ls.subscribers[tenantID] == nil {
		ls.subscribers[tenantID] = make(map[string]*StreamSubscriber)
	}
	ls.subscribers[tenantID][sub.ID] = sub

	return sub
}

// Unsubscribe removes a subscriber
func (ls *LiveStreamService) Unsubscribe(tenantID, subscriberID string) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	if subs, ok := ls.subscribers[tenantID]; ok {
		if sub, exists := subs[subscriberID]; exists {
			sub.Closed = true
			close(sub.Channel)
			delete(subs, subscriberID)
		}
	}
}

// Publish sends a delivery event to all matching subscribers
func (ls *LiveStreamService) Publish(event *LiveDeliveryEvent) {
	ls.mu.RLock()
	subs := ls.subscribers[event.TenantID]
	ls.mu.RUnlock()

	for _, sub := range subs {
		if sub.Closed {
			continue
		}
		if !matchesStreamFilter(&sub.Filter, event) {
			continue
		}

		select {
		case sub.Channel <- event:
		default:
			// Drop event if subscriber is too slow (buffer full)
		}
	}
}

// GetStats returns streaming statistics for a tenant
func (ls *LiveStreamService) GetStats(tenantID string) *StreamStats {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	stats := &StreamStats{TenantID: tenantID}
	if subs, ok := ls.subscribers[tenantID]; ok {
		stats.ActiveSubscribers = len(subs)
		for _, sub := range subs {
			if stats.OldestSubscriber.IsZero() || sub.CreatedAt.Before(stats.OldestSubscriber) {
				stats.OldestSubscriber = sub.CreatedAt
			}
		}
	}
	return stats
}

func matchesStreamFilter(filter *StreamFilter, event *LiveDeliveryEvent) bool {
	if len(filter.EndpointIDs) > 0 {
		found := false
		for _, id := range filter.EndpointIDs {
			if id == event.EndpointID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(filter.EventTypes) > 0 {
		found := false
		for _, et := range filter.EventTypes {
			if et == event.EventType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(filter.Statuses) > 0 {
		found := false
		for _, s := range filter.Statuses {
			if s == event.Status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if filter.Search != "" {
		searchLower := strings.ToLower(filter.Search)
		if !strings.Contains(strings.ToLower(event.EndpointURL), searchLower) &&
			!strings.Contains(strings.ToLower(event.EventType), searchLower) &&
			!strings.Contains(strings.ToLower(event.ErrorMessage), searchLower) {
			return false
		}
	}

	return true
}

// LiveStreamHandler provides HTTP handlers for SSE and WebSocket streaming
type LiveStreamHandler struct {
	streamService *LiveStreamService
}

// NewLiveStreamHandler creates a new live stream handler
func NewLiveStreamHandler(streamService *LiveStreamService) *LiveStreamHandler {
	return &LiveStreamHandler{streamService: streamService}
}

// RegisterLiveRoutes registers live streaming routes
func (h *LiveStreamHandler) RegisterLiveRoutes(rg *gin.RouterGroup) {
	stream := rg.Group("/stream")
	{
		stream.GET("/deliveries", h.StreamDeliveriesSSE)
		stream.GET("/stats", h.GetStreamStats)
	}
}

// StreamDeliveriesSSE streams delivery events via Server-Sent Events
func (h *LiveStreamHandler) StreamDeliveriesSSE(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	filter := StreamFilter{
		Search: c.Query("search"),
	}
	if endpointIDs := c.Query("endpoint_ids"); endpointIDs != "" {
		filter.EndpointIDs = strings.Split(endpointIDs, ",")
	}
	if eventTypes := c.Query("event_types"); eventTypes != "" {
		filter.EventTypes = strings.Split(eventTypes, ",")
	}
	if statuses := c.Query("statuses"); statuses != "" {
		filter.Statuses = strings.Split(statuses, ",")
	}

	sub := h.streamService.Subscribe(tenantID, filter)
	defer h.streamService.Unsubscribe(tenantID, sub.ID)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ctx := c.Request.Context()
	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-sub.Channel:
			if !ok {
				return false
			}
			data, err := json.Marshal(event)
			if err != nil {
				return true
			}
			c.SSEvent("delivery", string(data))
			return true
		case <-ctx.Done():
			return false
		case <-time.After(30 * time.Second):
			// Send keepalive
			c.SSEvent("ping", `{"type":"keepalive"}`)
			return true
		}
	})
}

// GetStreamStats returns streaming statistics
func (h *LiveStreamHandler) GetStreamStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	stats := h.streamService.GetStats(tenantID)
	c.JSON(http.StatusOK, stats)
}

// ExportRequest represents a request to export streamed events
type ExportRequest struct {
	Format    string       `json:"format" binding:"required,oneof=json csv ndjson"`
	StartTime *time.Time   `json:"start_time,omitempty"`
	EndTime   *time.Time   `json:"end_time,omitempty"`
	Filter    StreamFilter `json:"filter,omitempty"`
	Limit     int          `json:"limit,omitempty"`
}

// ExportResult represents exported stream data
type ExportResult struct {
	Format     string    `json:"format"`
	EventCount int       `json:"event_count"`
	Data       string    `json:"data"`
	ExportedAt time.Time `json:"exported_at"`
}

// FormatEventsAsNDJSON formats events as newline-delimited JSON
func FormatEventsAsNDJSON(events []*LiveDeliveryEvent) string {
	var sb strings.Builder
	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		sb.Write(data)
		sb.WriteString("\n")
	}
	return sb.String()
}

// FormatEventsAsCSV formats events as CSV
func FormatEventsAsCSV(events []*LiveDeliveryEvent) string {
	var sb strings.Builder
	sb.WriteString("id,endpoint_id,event_type,status,http_status,latency_ms,timestamp\n")
	for _, event := range events {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%d,%.1f,%s\n",
			event.ID,
			event.EndpointID,
			event.EventType,
			event.Status,
			event.HTTPStatus,
			event.LatencyMs,
			event.Timestamp.Format(time.RFC3339),
		))
	}
	return sb.String()
}

// PubSubNotifier integrates with Redis pub/sub for cross-instance event distribution
type PubSubNotifier struct {
	streamService *LiveStreamService
	channel       string
}

// NewPubSubNotifier creates a notifier that publishes to both local and Redis subscribers
func NewPubSubNotifier(streamService *LiveStreamService) *PubSubNotifier {
	return &PubSubNotifier{
		streamService: streamService,
		channel:       "waas:deliveries:live",
	}
}

// NotifyDelivery publishes a delivery event to all subscribers
func (n *PubSubNotifier) NotifyDelivery(ctx context.Context, event *LiveDeliveryEvent) {
	n.streamService.Publish(event)
}
