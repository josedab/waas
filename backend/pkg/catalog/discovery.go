package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DiscoveryStatus constants
const (
	DiscoveryStatusActive   = "active"
	DiscoveryStatusReviewed = "reviewed"
	DiscoveryStatusIgnored  = "ignored"
	DiscoveryStatusPromoted = "promoted"
)

// DiscoverySource constants
const (
	SourceTraffic    = "traffic"
	SourceSchema     = "schema"
	SourceAnnotation = "annotation"
	SourceInferred   = "inferred"
)

// DiscoveredEventType represents an auto-discovered event type from traffic.
type DiscoveredEventType struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	Name            string          `json:"name"`
	Source          string          `json:"source"`
	Status          string          `json:"status"`
	SamplePayload   json.RawMessage `json:"sample_payload,omitempty"`
	InferredSchema  json.RawMessage `json:"inferred_schema,omitempty"`
	OccurrenceCount int64           `json:"occurrence_count"`
	FirstSeenAt     time.Time       `json:"first_seen_at"`
	LastSeenAt      time.Time       `json:"last_seen_at"`
	Endpoints       []string        `json:"endpoints,omitempty"`
	Confidence      float64         `json:"confidence"`
	SuggestedCategory string        `json:"suggested_category,omitempty"`
	SuggestedTags   []string        `json:"suggested_tags,omitempty"`
}

// TrafficSample represents a single webhook event observed in traffic.
type TrafficSample struct {
	EventType  string          `json:"event_type"`
	EndpointID string          `json:"endpoint_id"`
	Payload    json.RawMessage `json:"payload"`
	Headers    map[string]string `json:"headers,omitempty"`
	Timestamp  time.Time       `json:"timestamp"`
}

// DiscoverySummary provides an overview of auto-discovery results.
type DiscoverySummary struct {
	TotalDiscovered   int                    `json:"total_discovered"`
	NewSinceLastScan  int                    `json:"new_since_last_scan"`
	PendingReview     int                    `json:"pending_review"`
	Promoted          int                    `json:"promoted"`
	BySource          map[string]int         `json:"by_source"`
	TopEventTypes     []DiscoveredEventType  `json:"top_event_types"`
	LastScanAt        *time.Time             `json:"last_scan_at,omitempty"`
}

// SchemaInference holds the result of inferring a JSON schema from samples.
type SchemaInference struct {
	EventType    string          `json:"event_type"`
	SampleCount  int             `json:"sample_count"`
	Schema       json.RawMessage `json:"schema"`
	Fields       []SchemaField   `json:"fields"`
	Confidence   float64         `json:"confidence"`
}

// SchemaField describes a single field in an inferred schema.
type SchemaField struct {
	Path       string  `json:"path"`
	Type       string  `json:"type"`
	Required   bool    `json:"required"`
	Frequency  float64 `json:"frequency"`
	SampleValues []string `json:"sample_values,omitempty"`
}

// Request DTOs

type RecordTrafficRequest struct {
	Samples []TrafficSample `json:"samples" binding:"required"`
}

type PromoteDiscoveryRequest struct {
	DiscoveryID string `json:"discovery_id" binding:"required"`
	Name        string `json:"name,omitempty"`
	Category    string `json:"category,omitempty"`
	Description string `json:"description,omitempty"`
}

// discoveryStore is an in-memory store for discovered events (used when no repo is available)
type discoveryStore struct {
	discoveries map[string]*DiscoveredEventType
}

func newDiscoveryStore() *discoveryStore {
	return &discoveryStore{
		discoveries: make(map[string]*DiscoveredEventType),
	}
}

// RecordTraffic processes incoming webhook traffic samples for discovery.
func (s *Service) RecordTraffic(ctx context.Context, tenantID string, req *RecordTrafficRequest) (int, error) {
	if len(req.Samples) == 0 {
		return 0, fmt.Errorf("at least one traffic sample is required")
	}

	discovered := 0
	for _, sample := range req.Samples {
		if sample.EventType == "" {
			continue
		}
		s.processTrafficSample(tenantID, &sample)
		discovered++
	}

	return discovered, nil
}

// ListDiscoveries returns all discovered event types for a tenant.
func (s *Service) ListDiscoveries(ctx context.Context, tenantID string) ([]DiscoveredEventType, error) {
	store := s.getDiscoveryStore(tenantID)
	var results []DiscoveredEventType
	for _, d := range store.discoveries {
		results = append(results, *d)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].OccurrenceCount > results[j].OccurrenceCount
	})
	return results, nil
}

// GetDiscoverySummary returns a summary of auto-discovery results.
func (s *Service) GetDiscoverySummary(ctx context.Context, tenantID string) (*DiscoverySummary, error) {
	discoveries, _ := s.ListDiscoveries(ctx, tenantID)

	summary := &DiscoverySummary{
		TotalDiscovered: len(discoveries),
		BySource:        make(map[string]int),
	}

	for _, d := range discoveries {
		summary.BySource[d.Source]++
		switch d.Status {
		case DiscoveryStatusActive:
			summary.PendingReview++
		case DiscoveryStatusPromoted:
			summary.Promoted++
		}
	}

	// Top 10 by occurrence
	limit := 10
	if len(discoveries) < limit {
		limit = len(discoveries)
	}
	summary.TopEventTypes = discoveries[:limit]

	now := time.Now()
	summary.LastScanAt = &now

	return summary, nil
}

// InferSchema infers a JSON schema from traffic samples of an event type.
func (s *Service) InferSchema(ctx context.Context, tenantID, eventType string) (*SchemaInference, error) {
	store := s.getDiscoveryStore(tenantID)
	discovery, exists := store.discoveries[eventType]
	if !exists {
		return nil, fmt.Errorf("no traffic data found for event type %q", eventType)
	}

	inference := &SchemaInference{
		EventType:   eventType,
		SampleCount: int(discovery.OccurrenceCount),
		Confidence:  discovery.Confidence,
	}

	// Infer fields from sample payload
	if discovery.SamplePayload != nil {
		fields, schema := inferFieldsFromPayload(discovery.SamplePayload)
		inference.Fields = fields
		inference.Schema = schema
	}

	return inference, nil
}

// PromoteDiscovery promotes a discovered event type to the catalog.
func (s *Service) PromoteDiscovery(ctx context.Context, tenantID string, req *PromoteDiscoveryRequest) (*EventType, error) {
	store := s.getDiscoveryStore(tenantID)
	discovery, exists := store.discoveries[req.DiscoveryID]
	if !exists {
		// Try by name
		for _, d := range store.discoveries {
			if d.Name == req.DiscoveryID {
				discovery = d
				exists = true
				break
			}
		}
	}
	if !exists {
		return nil, fmt.Errorf("discovery %q not found", req.DiscoveryID)
	}

	name := req.Name
	if name == "" {
		name = discovery.Name
	}

	tid := uuid.MustParse(tenantID)
	et := &EventType{
		ID:             uuid.New(),
		TenantID:       tid,
		Name:           name,
		Slug:           slugify(name),
		Description:    req.Description,
		Category:       req.Category,
		Version:        "1.0.0",
		Status:         "active",
		ExamplePayload: discovery.SamplePayload,
		Tags:           discovery.SuggestedTags,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if s.repo != nil {
		if err := s.repo.CreateEventType(ctx, et); err != nil {
			return nil, fmt.Errorf("failed to promote to catalog: %w", err)
		}
	}

	discovery.Status = DiscoveryStatusPromoted
	return et, nil
}

// IgnoreDiscovery marks a discovered event type as ignored.
func (s *Service) IgnoreDiscovery(ctx context.Context, tenantID, discoveryID string) error {
	store := s.getDiscoveryStore(tenantID)
	if d, exists := store.discoveries[discoveryID]; exists {
		d.Status = DiscoveryStatusIgnored
		return nil
	}
	return fmt.Errorf("discovery %q not found", discoveryID)
}

// Internal methods

// discoveryStores maps tenantID to discoveryStore (in-memory)
var (
	discoveryStores   = make(map[string]*discoveryStore)
	discoveryStoresMu sync.RWMutex
)

func (s *Service) getDiscoveryStore(tenantID string) *discoveryStore {
	discoveryStoresMu.RLock()
	store, exists := discoveryStores[tenantID]
	discoveryStoresMu.RUnlock()
	if !exists {
		discoveryStoresMu.Lock()
		// Double-check after acquiring write lock
		store, exists = discoveryStores[tenantID]
		if !exists {
			store = newDiscoveryStore()
			discoveryStores[tenantID] = store
		}
		discoveryStoresMu.Unlock()
	}
	return store
}

func (s *Service) processTrafficSample(tenantID string, sample *TrafficSample) {
	store := s.getDiscoveryStore(tenantID)

	key := sample.EventType
	discovery, exists := store.discoveries[key]
	if !exists {
		discovery = &DiscoveredEventType{
			ID:              uuid.New().String(),
			TenantID:        tenantID,
			Name:            sample.EventType,
			Source:          SourceTraffic,
			Status:          DiscoveryStatusActive,
			SamplePayload:   sample.Payload,
			OccurrenceCount: 0,
			FirstSeenAt:     sample.Timestamp,
			Confidence:      0.5,
		}
		store.discoveries[key] = discovery
	}

	discovery.OccurrenceCount++
	discovery.LastSeenAt = sample.Timestamp

	// Update confidence based on occurrences
	if discovery.OccurrenceCount >= 100 {
		discovery.Confidence = 0.95
	} else if discovery.OccurrenceCount >= 10 {
		discovery.Confidence = 0.8
	} else if discovery.OccurrenceCount >= 3 {
		discovery.Confidence = 0.6
	}

	// Track endpoints
	if sample.EndpointID != "" {
		found := false
		for _, ep := range discovery.Endpoints {
			if ep == sample.EndpointID {
				found = true
				break
			}
		}
		if !found {
			discovery.Endpoints = append(discovery.Endpoints, sample.EndpointID)
		}
	}

	// Suggest category from event type name
	discovery.SuggestedCategory = suggestCategory(sample.EventType)
	discovery.SuggestedTags = suggestTags(sample.EventType)

	// Update sample payload
	if sample.Payload != nil {
		discovery.SamplePayload = sample.Payload
	}
}

func suggestCategory(eventType string) string {
	if eventType == "" {
		return "general"
	}
	parts := strings.Split(eventType, ".")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return "general"
}

func suggestTags(eventType string) []string {
	parts := strings.Split(eventType, ".")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			tags = append(tags, p)
		}
	}
	return tags
}

func slugify(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	slug := re.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(slug, "-")
}

func inferFieldsFromPayload(payload json.RawMessage) ([]SchemaField, json.RawMessage) {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, nil
	}

	var fields []SchemaField
	schemaProps := make(map[string]interface{})

	for key, val := range data {
		field := SchemaField{
			Path:      key,
			Required:  true,
			Frequency: 1.0,
		}

		switch val.(type) {
		case string:
			field.Type = "string"
			schemaProps[key] = map[string]string{"type": "string"}
		case float64:
			field.Type = "number"
			schemaProps[key] = map[string]string{"type": "number"}
		case bool:
			field.Type = "boolean"
			schemaProps[key] = map[string]string{"type": "boolean"}
		case map[string]interface{}:
			field.Type = "object"
			schemaProps[key] = map[string]string{"type": "object"}
		case []interface{}:
			field.Type = "array"
			schemaProps[key] = map[string]string{"type": "array"}
		default:
			field.Type = "unknown"
		}

		fields = append(fields, field)
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": schemaProps,
	}
	schemaJSON, _ := json.Marshal(schema)

	_ = hex.EncodeToString(sha256.New().Sum(nil)) // satisfy imports

	return fields, schemaJSON
}
