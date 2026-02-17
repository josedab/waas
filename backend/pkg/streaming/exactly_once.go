package streaming

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ExactlyOnceConfig configures exactly-once delivery semantics
type ExactlyOnceConfig struct {
	Enabled            bool          `json:"enabled"`
	DeduplicationTTL   time.Duration `json:"deduplication_ttl"`
	OutboxPollInterval time.Duration `json:"outbox_poll_interval"`
	MaxOutboxBatchSize int           `json:"max_outbox_batch_size"`
}

// DefaultExactlyOnceConfig returns sensible defaults
func DefaultExactlyOnceConfig() *ExactlyOnceConfig {
	return &ExactlyOnceConfig{
		Enabled:            true,
		DeduplicationTTL:   24 * time.Hour,
		OutboxPollInterval: 500 * time.Millisecond,
		MaxOutboxBatchSize: 100,
	}
}

// IdempotencyStore tracks processed event IDs for deduplication
type IdempotencyStore struct {
	processed map[string]time.Time
	ttl       time.Duration
	mu        sync.RWMutex
}

// NewIdempotencyStore creates a new in-memory idempotency store
func NewIdempotencyStore(ctx context.Context, ttl time.Duration) *IdempotencyStore {
	store := &IdempotencyStore{
		processed: make(map[string]time.Time),
		ttl:       ttl,
	}
	go store.cleanup(ctx)
	return store
}

// IsDuplicate checks if an event has already been processed
func (s *IdempotencyStore) IsDuplicate(eventID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.processed[eventID]
	return exists
}

// MarkProcessed records an event as processed
func (s *IdempotencyStore) MarkProcessed(eventID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.processed[eventID] = time.Now()
}

// cleanup periodically removes expired entries
func (s *IdempotencyStore) cleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			cutoff := time.Now().Add(-s.ttl)
			for id, ts := range s.processed {
				if ts.Before(cutoff) {
					delete(s.processed, id)
				}
			}
			s.mu.Unlock()
		}
	}
}

// Size returns the number of tracked events
func (s *IdempotencyStore) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.processed)
}

// OutboxEntry represents a transactional outbox record
type OutboxEntry struct {
	ID            string            `json:"id"`
	BridgeID      string            `json:"bridge_id"`
	TenantID      string            `json:"tenant_id"`
	EventID       string            `json:"event_id"`
	Payload       json.RawMessage   `json:"payload"`
	Headers       map[string]string `json:"headers,omitempty"`
	Destination   string            `json:"destination"`
	Status        OutboxStatus      `json:"status"`
	Attempts      int               `json:"attempts"`
	MaxAttempts   int               `json:"max_attempts"`
	LastAttemptAt *time.Time        `json:"last_attempt_at,omitempty"`
	Error         string            `json:"error,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	ProcessedAt   *time.Time        `json:"processed_at,omitempty"`
}

// OutboxStatus represents the status of an outbox entry
type OutboxStatus string

const (
	OutboxStatusPending    OutboxStatus = "pending"
	OutboxStatusProcessing OutboxStatus = "processing"
	OutboxStatusCompleted  OutboxStatus = "completed"
	OutboxStatusFailed     OutboxStatus = "failed"
)

// TransactionalOutbox implements the outbox pattern for reliable event delivery
type TransactionalOutbox struct {
	entries    []*OutboxEntry
	mu         sync.Mutex
	config     *ExactlyOnceConfig
	processor  OutboxProcessor
	cancelFunc context.CancelFunc
}

// OutboxProcessor processes outbox entries
type OutboxProcessor interface {
	Process(ctx context.Context, entry *OutboxEntry) error
}

// NewTransactionalOutbox creates a new transactional outbox
func NewTransactionalOutbox(config *ExactlyOnceConfig, processor OutboxProcessor) *TransactionalOutbox {
	if config == nil {
		config = DefaultExactlyOnceConfig()
	}
	return &TransactionalOutbox{
		entries:   make([]*OutboxEntry, 0),
		config:    config,
		processor: processor,
	}
}

// Enqueue adds an entry to the outbox
func (o *TransactionalOutbox) Enqueue(entry *OutboxEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Status == "" {
		entry.Status = OutboxStatusPending
	}
	if entry.MaxAttempts == 0 {
		entry.MaxAttempts = 3
	}
	entry.CreatedAt = time.Now()

	o.mu.Lock()
	o.entries = append(o.entries, entry)
	o.mu.Unlock()
	return nil
}

// Start begins the outbox polling loop
func (o *TransactionalOutbox) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	o.cancelFunc = cancel

	go func() {
		ticker := time.NewTicker(o.config.OutboxPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				o.processBatch(ctx)
			}
		}
	}()
}

// Stop halts the outbox processor
func (o *TransactionalOutbox) Stop() {
	if o.cancelFunc != nil {
		o.cancelFunc()
	}
}

func (o *TransactionalOutbox) processBatch(ctx context.Context) {
	o.mu.Lock()
	batch := make([]*OutboxEntry, 0, o.config.MaxOutboxBatchSize)
	for _, entry := range o.entries {
		if entry.Status == OutboxStatusPending && len(batch) < o.config.MaxOutboxBatchSize {
			entry.Status = OutboxStatusProcessing
			batch = append(batch, entry)
		}
	}
	o.mu.Unlock()

	for _, entry := range batch {
		now := time.Now()
		entry.LastAttemptAt = &now
		entry.Attempts++

		if err := o.processor.Process(ctx, entry); err != nil {
			if entry.Attempts >= entry.MaxAttempts {
				entry.Status = OutboxStatusFailed
				entry.Error = err.Error()
			} else {
				entry.Status = OutboxStatusPending
				entry.Error = err.Error()
			}
		} else {
			entry.Status = OutboxStatusCompleted
			entry.ProcessedAt = &now
		}
	}

	// Purge completed entries older than TTL
	o.mu.Lock()
	active := make([]*OutboxEntry, 0, len(o.entries))
	cutoff := time.Now().Add(-o.config.DeduplicationTTL)
	for _, entry := range o.entries {
		if entry.Status == OutboxStatusCompleted && entry.ProcessedAt != nil && entry.ProcessedAt.Before(cutoff) {
			continue
		}
		active = append(active, entry)
	}
	o.entries = active
	o.mu.Unlock()
}

// PendingCount returns the number of pending outbox entries
func (o *TransactionalOutbox) PendingCount() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	count := 0
	for _, e := range o.entries {
		if e.Status == OutboxStatusPending {
			count++
		}
	}
	return count
}

// ExactlyOnceBridge wraps a DeliveryBridge with exactly-once semantics
type ExactlyOnceBridge struct {
	*DeliveryBridge
	idempotencyStore *IdempotencyStore
	outbox           *TransactionalOutbox
	config           *ExactlyOnceConfig
}

// NewExactlyOnceBridge creates a bridge with exactly-once delivery guarantees
func NewExactlyOnceBridge(
	bridgeID, tenantID string,
	source EventSource,
	publisher WebhookPublisher,
	bridgeConfig *DeliveryBridgeConfig,
	eoConfig *ExactlyOnceConfig,
) *ExactlyOnceBridge {
	if eoConfig == nil {
		eoConfig = DefaultExactlyOnceConfig()
	}

	bridge := NewDeliveryBridge(bridgeID, tenantID, source, publisher, bridgeConfig)
	store := NewIdempotencyStore(context.Background(), eoConfig.DeduplicationTTL)

	eob := &ExactlyOnceBridge{
		DeliveryBridge:   bridge,
		idempotencyStore: store,
		config:           eoConfig,
	}

	processor := &bridgeOutboxProcessor{bridge: eob}
	eob.outbox = NewTransactionalOutbox(eoConfig, processor)

	return eob
}

// Start begins the exactly-once bridge
func (eob *ExactlyOnceBridge) Start(ctx context.Context) error {
	eob.outbox.Start(ctx)
	// Override the source handler with dedup-aware version
	ctx, cancel := context.WithCancel(ctx)
	eob.mu.Lock()
	eob.cancelFunc = cancel
	eob.mu.Unlock()

	return eob.source.Subscribe(ctx, eob.handleEventWithDedup)
}

// Stop halts the exactly-once bridge
func (eob *ExactlyOnceBridge) Stop() error {
	eob.outbox.Stop()
	return eob.DeliveryBridge.Stop()
}

func (eob *ExactlyOnceBridge) handleEventWithDedup(ctx context.Context, event *StreamEvent) error {
	// Generate idempotency key from event content
	idempotencyKey := eob.computeIdempotencyKey(event)

	if eob.idempotencyStore.IsDuplicate(idempotencyKey) {
		eob.metrics.mu.Lock()
		eob.metrics.EventsFiltered++
		eob.metrics.mu.Unlock()
		return nil // Already processed
	}

	// Enqueue to outbox for reliable delivery
	entry := &OutboxEntry{
		BridgeID:    eob.bridgeID,
		TenantID:    eob.tenantID,
		EventID:     event.ID,
		Payload:     event.Value,
		Headers:     event.Headers,
		Destination: "webhook",
	}

	if err := eob.outbox.Enqueue(entry); err != nil {
		return fmt.Errorf("failed to enqueue to outbox: %w", err)
	}

	eob.idempotencyStore.MarkProcessed(idempotencyKey)
	return nil
}

func (eob *ExactlyOnceBridge) computeIdempotencyKey(event *StreamEvent) string {
	if event.ID != "" {
		return event.ID
	}
	h := sha256.New()
	h.Write([]byte(event.BridgeID))
	h.Write(event.Value)
	h.Write([]byte(fmt.Sprintf("%d", event.Offset)))
	return hex.EncodeToString(h.Sum(nil))
}

// bridgeOutboxProcessor processes outbox entries by delivering to webhook endpoints
type bridgeOutboxProcessor struct {
	bridge *ExactlyOnceBridge
}

func (p *bridgeOutboxProcessor) Process(ctx context.Context, entry *OutboxEntry) error {
	headers := entry.Headers
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["X-Idempotency-Key"] = entry.EventID
	headers["X-Outbox-ID"] = entry.ID

	for _, endpointID := range p.bridge.DeliveryBridge.config.TargetEndpointIDs {
		if _, err := p.bridge.publisher.Publish(ctx, entry.TenantID, endpointID, entry.Payload, headers); err != nil {
			return err
		}
	}

	p.bridge.metrics.mu.Lock()
	p.bridge.metrics.EventsDelivered++
	p.bridge.metrics.mu.Unlock()
	return nil
}

// ExactlyOnceMetrics returns metrics including dedup stats
func (eob *ExactlyOnceBridge) ExactlyOnceMetrics() map[string]interface{} {
	base := eob.GetMetrics()
	return map[string]interface{}{
		"events_received":      base.EventsReceived,
		"events_delivered":     base.EventsDelivered,
		"events_filtered":      base.EventsFiltered,
		"events_failed":        base.EventsFailed,
		"avg_latency_ms":       base.AvgLatencyMs,
		"dedup_store_size":     eob.idempotencyStore.Size(),
		"outbox_pending":       eob.outbox.PendingCount(),
		"exactly_once_enabled": eob.config.Enabled,
	}
}
