package metaevents

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

// Emitter is responsible for publishing and delivering meta-events
type Emitter struct {
	repo       Repository
	httpClient *http.Client
	queue      chan *MetaEvent
	workers    int
	wg         sync.WaitGroup
	stopCh     chan struct{}
	logger     *utils.Logger
}

// NewEmitter creates a new meta-event emitter
func NewEmitter(repo Repository, workers int) *Emitter {
	if workers <= 0 {
		workers = 5
	}

	return &Emitter{
		repo: repo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		queue:   make(chan *MetaEvent, 1000),
		workers: workers,
		stopCh:  make(chan struct{}),
		logger:  utils.NewLogger("metaevents"),
	}
}

// Start begins processing meta-events
func (e *Emitter) Start() {
	for i := 0; i < e.workers; i++ {
		e.wg.Add(1)
		go e.worker()
	}
	e.logger.Info("started workers", map[string]interface{}{"count": e.workers})
}

// Stop gracefully stops the emitter
func (e *Emitter) Stop() {
	close(e.stopCh)
	e.wg.Wait()
	e.logger.Info("emitter stopped", nil)
}

// Emit publishes a meta-event for delivery
func (e *Emitter) Emit(ctx context.Context, tenantID string, eventType EventType, source, sourceID string, data map[string]interface{}) error {
	event := &MetaEvent{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		Type:       eventType,
		Source:     source,
		SourceID:   sourceID,
		Data:       data,
		OccurredAt: time.Now(),
		CreatedAt:  time.Now(),
	}

	// Store the event
	if err := e.repo.CreateEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to store meta-event: %w", err)
	}

	// Queue for async delivery
	select {
	case e.queue <- event:
	default:
		e.logger.Warn("queue full, processing synchronously", map[string]interface{}{"event_id": event.ID})
		e.processEvent(ctx, event)
	}

	return nil
}

// EmitDeliveryEvent is a convenience method for delivery lifecycle events
func (e *Emitter) EmitDeliveryEvent(ctx context.Context, tenantID string, eventType EventType, data *DeliveryData) error {
	return e.Emit(ctx, tenantID, eventType, "delivery", data.WebhookID, map[string]interface{}{
		"webhook_id":    data.WebhookID,
		"endpoint_id":   data.EndpointID,
		"endpoint_url":  data.EndpointURL,
		"attempt":       data.Attempt,
		"status_code":   data.StatusCode,
		"latency_ms":    data.LatencyMs,
		"error_message": data.ErrorMessage,
		"error_type":    data.ErrorType,
	})
}

// EmitEndpointEvent is a convenience method for endpoint events
func (e *Emitter) EmitEndpointEvent(ctx context.Context, tenantID string, eventType EventType, data *EndpointData) error {
	return e.Emit(ctx, tenantID, eventType, "endpoint", data.EndpointID, map[string]interface{}{
		"endpoint_id":  data.EndpointID,
		"endpoint_url": data.EndpointURL,
		"name":         data.Name,
		"old_status":   data.OldStatus,
		"new_status":   data.NewStatus,
	})
}

// EmitThresholdEvent is a convenience method for threshold events
func (e *Emitter) EmitThresholdEvent(ctx context.Context, tenantID string, eventType EventType, data *ThresholdData) error {
	return e.Emit(ctx, tenantID, eventType, "threshold", data.EndpointID, map[string]interface{}{
		"metric":        data.Metric,
		"value":         data.Value,
		"threshold":     data.Threshold,
		"endpoint_id":   data.EndpointID,
		"window_period": data.WindowPeriod,
	})
}

// EmitAnomalyEvent is a convenience method for anomaly events
func (e *Emitter) EmitAnomalyEvent(ctx context.Context, tenantID string, data *AnomalyData) error {
	return e.Emit(ctx, tenantID, EventAnomalyDetected, "anomaly", data.EndpointID, map[string]interface{}{
		"metric":      data.Metric,
		"expected":    data.Expected,
		"actual":      data.Actual,
		"deviation":   data.Deviation,
		"endpoint_id": data.EndpointID,
		"detected_at": data.DetectedAt,
	})
}

func (e *Emitter) worker() {
	defer e.wg.Done()

	for {
		select {
		case <-e.stopCh:
			return
		case event := <-e.queue:
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			e.processEvent(ctx, event)
			cancel()
		}
	}
}

func (e *Emitter) processEvent(ctx context.Context, event *MetaEvent) {
	// Get all active subscriptions for this tenant and event type
	subs, err := e.repo.GetSubscriptionsByEventType(ctx, event.TenantID, event.Type)
	if err != nil {
		e.logger.Error("failed to get subscriptions", map[string]interface{}{"error": err.Error()})
		return
	}

	for _, sub := range subs {
		// Apply filters
		if !e.matchesFilter(event, sub.Filters) {
			continue
		}

		// Deliver to subscription
		e.deliver(ctx, event, &sub)
	}
}

func (e *Emitter) matchesFilter(event *MetaEvent, filter *EventFilter) bool {
	if filter == nil {
		return true
	}

	// Check endpoint ID filter
	if len(filter.EndpointIDs) > 0 {
		sourceID := event.SourceID
		if endpointID, ok := event.Data["endpoint_id"].(string); ok {
			sourceID = endpointID
		}
		found := false
		for _, id := range filter.EndpointIDs {
			if id == sourceID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check source filter
	if len(filter.Sources) > 0 {
		found := false
		for _, src := range filter.Sources {
			if src == event.Source {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (e *Emitter) deliver(ctx context.Context, event *MetaEvent, sub *Subscription) {
	payload := DeliveryPayload{
		ID:         event.ID,
		Type:       event.Type,
		Source:     event.Source,
		SourceID:   event.SourceID,
		TenantID:   event.TenantID,
		Data:       event.Data,
		OccurredAt: event.OccurredAt,
		Timestamp:  time.Now(),
	}

	body, _ := json.Marshal(payload)

	// Create delivery record
	delivery := &Delivery{
		ID:             uuid.New().String(),
		SubscriptionID: sub.ID,
		EventID:        event.ID,
		TenantID:       event.TenantID,
		Status:         "pending",
		Attempt:        1,
		CreatedAt:      time.Now(),
	}

	retryPolicy := sub.RetryPolicy
	if retryPolicy == nil {
		retryPolicy = DefaultRetryPolicy()
	}

	// Attempt delivery with retries
	for attempt := 1; attempt <= retryPolicy.MaxRetries+1; attempt++ {
		delivery.Attempt = attempt

		req, err := http.NewRequestWithContext(ctx, "POST", sub.URL, bytes.NewReader(body))
		if err != nil {
			delivery.Status = "failed"
			delivery.Error = err.Error()
			e.repo.CreateDelivery(ctx, delivery)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Meta-Event-ID", event.ID)
		req.Header.Set("X-Meta-Event-Type", string(event.Type))
		req.Header.Set("X-Meta-Event-Timestamp", event.OccurredAt.Format(time.RFC3339))

		// Add HMAC signature
		if sub.Secret != "" {
			signature := computeHMAC(body, sub.Secret)
			req.Header.Set("X-Meta-Event-Signature", signature)
		}

		// Add custom headers
		for k, v := range sub.Headers {
			req.Header.Set(k, v)
		}

		start := time.Now()
		resp, err := e.httpClient.Do(req)
		delivery.LatencyMs = int(time.Since(start).Milliseconds())

		if err != nil {
			delivery.Status = "retrying"
			delivery.Error = err.Error()
		} else {
			delivery.StatusCode = resp.StatusCode
			resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				delivery.Status = "delivered"
				e.repo.CreateDelivery(ctx, delivery)
				return
			}

			delivery.Status = "retrying"
			delivery.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}

		// Calculate next retry
		if attempt < retryPolicy.MaxRetries+1 {
			interval := float64(retryPolicy.InitialInterval)
			for i := 1; i < attempt; i++ {
				interval *= retryPolicy.Multiplier
			}
			if interval > float64(retryPolicy.MaxInterval) {
				interval = float64(retryPolicy.MaxInterval)
			}
			nextRetry := time.Now().Add(time.Duration(interval))
			delivery.NextRetry = &nextRetry

			e.repo.CreateDelivery(ctx, delivery)
			time.Sleep(time.Duration(interval))
		}
	}

	delivery.Status = "exhausted"
	e.repo.CreateDelivery(ctx, delivery)
}

func computeHMAC(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}
