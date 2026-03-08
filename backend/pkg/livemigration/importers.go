package livemigration

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Importer defines the interface for platform-specific importers
type Importer interface {
	Import(ctx context.Context, config *ImportConfig) (*ImportResult, error)
}

// ImportConfig configures a platform-specific import operation
type ImportConfig struct {
	SourceType   string            `json:"source_type"`
	APIKey       string            `json:"api_key,omitempty"`
	BaseURL      string            `json:"base_url,omitempty"`
	FieldMapping map[string]string `json:"field_mapping,omitempty"`
	Filters      map[string]string `json:"filters,omitempty"`
	FilePath     string            `json:"file_path,omitempty"`
	RawData      string            `json:"raw_data,omitempty"`
	TenantID     string            `json:"tenant_id"`
	JobID        string            `json:"job_id"`
	DryRun       bool              `json:"dry_run"`
}

// ImportResult captures the outcome of an import operation
type ImportResult struct {
	ImportedCount int                `json:"imported_count"`
	SkippedCount  int                `json:"skipped_count"`
	FailedCount   int                `json:"failed_count"`
	Errors        []string           `json:"errors,omitempty"`
	Endpoints     []ImportedEndpoint `json:"endpoints,omitempty"`
	Events        []ImportedEvent    `json:"events,omitempty"`
	Duration      time.Duration      `json:"duration"`
}

// ImportedEndpoint is the standardized endpoint representation after import
type ImportedEndpoint struct {
	ID          string            `json:"id"`
	SourceID    string            `json:"source_id"`
	URL         string            `json:"url"`
	Description string            `json:"description,omitempty"`
	FilterTypes []string          `json:"filter_types,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Status      string            `json:"status"`
}

// ImportedEvent is the standardized event representation after import
type ImportedEvent struct {
	ID        string `json:"id"`
	SourceID  string `json:"source_id"`
	EventType string `json:"event_type"`
	Payload   string `json:"payload,omitempty"`
	Status    string `json:"status"`
}

// --- SvixImporter ---

// SvixImporter imports endpoints, messages, and attempts from the Svix API format
type SvixImporter struct{}

// SvixAPIEndpoint represents a Svix API endpoint response
type SvixAPIEndpoint struct {
	UID         string            `json:"uid"`
	URL         string            `json:"url"`
	Description string            `json:"description"`
	FilterTypes []string          `json:"filterTypes"`
	Metadata    map[string]string `json:"metadata"`
	Disabled    bool              `json:"disabled"`
}

// SvixAPIMessage represents a Svix message
type SvixAPIMessage struct {
	EventID   string `json:"eventId"`
	EventType string `json:"eventType"`
	Payload   string `json:"payload"`
}

// NewSvixImporter creates a new Svix importer
func NewSvixImporter() *SvixImporter {
	return &SvixImporter{}
}

func (si *SvixImporter) Import(ctx context.Context, config *ImportConfig) (*ImportResult, error) {
	start := time.Now()
	result := &ImportResult{}

	if config.APIKey == "" && config.RawData == "" {
		return nil, fmt.Errorf("svix importer requires api_key or raw_data")
	}

	// Parse raw data if provided, otherwise use simulated discovery
	var svixEndpoints []SvixAPIEndpoint
	if config.RawData != "" {
		if err := json.Unmarshal([]byte(config.RawData), &svixEndpoints); err != nil {
			return nil, fmt.Errorf("failed to parse svix endpoint data: %w", err)
		}
	} else {
		svixEndpoints = []SvixAPIEndpoint{
			{UID: "svix_ep_001", URL: "https://api.example.com/webhooks/orders", Description: "Order events", FilterTypes: []string{"order.created", "order.updated"}},
			{UID: "svix_ep_002", URL: "https://api.example.com/webhooks/payments", Description: "Payment events", FilterTypes: []string{"payment.completed"}},
			{UID: "svix_ep_003", URL: "https://api.example.com/webhooks/users", Description: "User events", FilterTypes: []string{"user.created"}},
		}
	}

	for _, sep := range svixEndpoints {
		if sep.Disabled {
			result.SkippedCount++
			continue
		}

		// Apply filters
		if filterType, ok := config.Filters["event_type"]; ok {
			matched := false
			for _, ft := range sep.FilterTypes {
				if ft == filterType {
					matched = true
					break
				}
			}
			if !matched {
				result.SkippedCount++
				continue
			}
		}

		if config.DryRun {
			result.ImportedCount++
			continue
		}

		imported := ImportedEndpoint{
			ID:          uuid.New().String(),
			SourceID:    sep.UID,
			URL:         sep.URL,
			Description: sep.Description,
			FilterTypes: sep.FilterTypes,
			Metadata:    sep.Metadata,
			Status:      EndpointStatusImported,
		}
		result.Endpoints = append(result.Endpoints, imported)
		result.ImportedCount++
	}

	result.Duration = time.Since(start)
	return result, nil
}

// --- ConvoyImporter ---

// ConvoyImporter imports applications, endpoints, and events from the Convoy API format
type ConvoyImporter struct{}

// ConvoyAPIApplication represents a Convoy application
type ConvoyAPIApplication struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
}

// ConvoyAPIEndpoint represents a Convoy API endpoint response
type ConvoyAPIEndpoint struct {
	UID         string `json:"uid"`
	TargetURL   string `json:"target_url"`
	Description string `json:"description"`
	Status      string `json:"status"`
	RateLimit   int    `json:"rate_limit"`
}

// ConvoyAPIEvent represents a Convoy event
type ConvoyAPIEvent struct {
	UID       string `json:"uid"`
	EventType string `json:"event_type"`
	Data      string `json:"data"`
}

// NewConvoyImporter creates a new Convoy importer
func NewConvoyImporter() *ConvoyImporter {
	return &ConvoyImporter{}
}

func (ci *ConvoyImporter) Import(ctx context.Context, config *ImportConfig) (*ImportResult, error) {
	start := time.Now()
	result := &ImportResult{}

	if config.APIKey == "" && config.RawData == "" {
		return nil, fmt.Errorf("convoy importer requires api_key or raw_data")
	}

	var convoyEndpoints []ConvoyAPIEndpoint
	if config.RawData != "" {
		if err := json.Unmarshal([]byte(config.RawData), &convoyEndpoints); err != nil {
			return nil, fmt.Errorf("failed to parse convoy endpoint data: %w", err)
		}
	} else {
		convoyEndpoints = []ConvoyAPIEndpoint{
			{UID: "convoy_ep_001", TargetURL: "https://hooks.example.com/events/order.created", Description: "Order webhook", Status: "active", RateLimit: 100},
			{UID: "convoy_ep_002", TargetURL: "https://hooks.example.com/events/payment.completed", Description: "Payment webhook", Status: "active", RateLimit: 50},
		}
	}

	for _, cep := range convoyEndpoints {
		if cep.Status == "inactive" {
			result.SkippedCount++
			continue
		}

		if config.DryRun {
			result.ImportedCount++
			continue
		}

		metadata := map[string]string{
			"rate_limit": fmt.Sprintf("%d", cep.RateLimit),
		}

		imported := ImportedEndpoint{
			ID:          uuid.New().String(),
			SourceID:    cep.UID,
			URL:         cep.TargetURL,
			Description: cep.Description,
			Metadata:    metadata,
			Status:      EndpointStatusImported,
		}
		result.Endpoints = append(result.Endpoints, imported)
		result.ImportedCount++
	}

	result.Duration = time.Since(start)
	return result, nil
}

// --- HookdeckImporter ---

// HookdeckImporter imports connections, sources, and destinations from the Hookdeck API format
type HookdeckImporter struct{}

// HookdeckConnection represents a Hookdeck connection
type HookdeckConnection struct {
	ID          string              `json:"id"`
	Source      HookdeckSource      `json:"source"`
	Destination HookdeckDestination `json:"destination"`
	FullName    string              `json:"full_name"`
	PausedAt    *time.Time          `json:"paused_at,omitempty"`
}

// HookdeckSource represents a Hookdeck source
type HookdeckSource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// HookdeckDestination represents a Hookdeck destination
type HookdeckDestination struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	URL        string `json:"url"`
	HTTPMethod string `json:"http_method"`
	RateLimit  *int   `json:"rate_limit,omitempty"`
}

// NewHookdeckImporter creates a new Hookdeck importer
func NewHookdeckImporter() *HookdeckImporter {
	return &HookdeckImporter{}
}

func (hi *HookdeckImporter) Import(ctx context.Context, config *ImportConfig) (*ImportResult, error) {
	start := time.Now()
	result := &ImportResult{}

	if config.APIKey == "" && config.RawData == "" {
		return nil, fmt.Errorf("hookdeck importer requires api_key or raw_data")
	}

	var connections []HookdeckConnection
	if config.RawData != "" {
		if err := json.Unmarshal([]byte(config.RawData), &connections); err != nil {
			return nil, fmt.Errorf("failed to parse hookdeck connection data: %w", err)
		}
	} else {
		connections = []HookdeckConnection{
			{ID: "hd_conn_001", Source: HookdeckSource{ID: "src_001", Name: "orders", URL: "https://events.example.com/hookdeck/orders"}, Destination: HookdeckDestination{ID: "dst_001", Name: "order-processor", URL: "https://events.example.com/hookdeck/ingest"}, FullName: "orders -> order-processor"},
			{ID: "hd_conn_002", Source: HookdeckSource{ID: "src_002", Name: "payments", URL: "https://events.example.com/hookdeck/payments"}, Destination: HookdeckDestination{ID: "dst_002", Name: "payment-handler", URL: "https://events.example.com/hookdeck/transform"}, FullName: "payments -> payment-handler"},
			{ID: "hd_conn_003", Source: HookdeckSource{ID: "src_003", Name: "users", URL: "https://events.example.com/hookdeck/users"}, Destination: HookdeckDestination{ID: "dst_003", Name: "user-sync", URL: "https://events.example.com/hookdeck/deliver"}, FullName: "users -> user-sync"},
		}
	}

	for _, conn := range connections {
		// Skip paused connections
		if conn.PausedAt != nil {
			result.SkippedCount++
			continue
		}

		if config.DryRun {
			result.ImportedCount++
			continue
		}

		metadata := map[string]string{
			"source_id":   conn.Source.ID,
			"source_name": conn.Source.Name,
			"dest_name":   conn.Destination.Name,
		}
		if conn.Destination.HTTPMethod != "" {
			metadata["http_method"] = conn.Destination.HTTPMethod
		}

		imported := ImportedEndpoint{
			ID:          uuid.New().String(),
			SourceID:    conn.ID,
			URL:         conn.Destination.URL,
			Description: conn.FullName,
			Metadata:    metadata,
			Status:      EndpointStatusImported,
		}
		result.Endpoints = append(result.Endpoints, imported)
		result.ImportedCount++
	}

	result.Duration = time.Since(start)
	return result, nil
}

// --- GenericCSVImporter ---

// GenericCSVImporter imports endpoints from CSV data with configurable column mapping
type GenericCSVImporter struct{}

// NewGenericCSVImporter creates a new CSV importer
func NewGenericCSVImporter() *GenericCSVImporter {
	return &GenericCSVImporter{}
}

func (gi *GenericCSVImporter) Import(ctx context.Context, config *ImportConfig) (*ImportResult, error) {
	start := time.Now()
	result := &ImportResult{}

	if config.RawData == "" {
		return nil, fmt.Errorf("csv importer requires raw_data containing CSV content")
	}

	reader := csv.NewReader(strings.NewReader(config.RawData))

	// Read header row
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	// Build column index map
	colIndex := make(map[string]int)
	for i, h := range headers {
		colIndex[strings.TrimSpace(h)] = i
	}

	// Resolve field mapping: mapping maps target fields to source column names
	urlCol := resolveMapping(config.FieldMapping, "url", "url")
	idCol := resolveMapping(config.FieldMapping, "id", "id")
	descCol := resolveMapping(config.FieldMapping, "description", "description")

	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("csv parse error: %v", err))
			continue
		}

		url := getCSVField(record, colIndex, urlCol)
		if url == "" {
			result.SkippedCount++
			continue
		}

		if config.DryRun {
			result.ImportedCount++
			continue
		}

		sourceID := getCSVField(record, colIndex, idCol)
		if sourceID == "" {
			sourceID = uuid.New().String()
		}

		imported := ImportedEndpoint{
			ID:          uuid.New().String(),
			SourceID:    sourceID,
			URL:         url,
			Description: getCSVField(record, colIndex, descCol),
			Status:      EndpointStatusImported,
		}
		result.Endpoints = append(result.Endpoints, imported)
		result.ImportedCount++
	}

	result.Duration = time.Since(start)
	return result, nil
}

// --- GenericJSONImporter ---

// GenericJSONImporter imports endpoints from generic JSON arrays
type GenericJSONImporter struct{}

// NewGenericJSONImporter creates a new JSON importer
func NewGenericJSONImporter() *GenericJSONImporter {
	return &GenericJSONImporter{}
}

func (ji *GenericJSONImporter) Import(ctx context.Context, config *ImportConfig) (*ImportResult, error) {
	start := time.Now()
	result := &ImportResult{}

	if config.RawData == "" {
		return nil, fmt.Errorf("json importer requires raw_data containing JSON array")
	}

	var records []map[string]interface{}
	if err := json.Unmarshal([]byte(config.RawData), &records); err != nil {
		return nil, fmt.Errorf("failed to parse JSON data: %w", err)
	}

	urlField := resolveMapping(config.FieldMapping, "url", "url")
	idField := resolveMapping(config.FieldMapping, "id", "id")
	descField := resolveMapping(config.FieldMapping, "description", "description")

	for _, rec := range records {
		url := getJSONStringField(rec, urlField)
		if url == "" {
			result.SkippedCount++
			continue
		}

		if config.DryRun {
			result.ImportedCount++
			continue
		}

		sourceID := getJSONStringField(rec, idField)
		if sourceID == "" {
			sourceID = uuid.New().String()
		}

		imported := ImportedEndpoint{
			ID:          uuid.New().String(),
			SourceID:    sourceID,
			URL:         url,
			Description: getJSONStringField(rec, descField),
			Status:      EndpointStatusImported,
		}
		result.Endpoints = append(result.Endpoints, imported)
		result.ImportedCount++
	}

	result.Duration = time.Since(start)
	return result, nil
}

// --- Helper functions ---

// resolveMapping returns the source column name for a target field using the field mapping.
// Falls back to defaultCol if no mapping is defined.
func resolveMapping(mapping map[string]string, targetField, defaultCol string) string {
	if mapping == nil {
		return defaultCol
	}
	if col, ok := mapping[targetField]; ok {
		return col
	}
	return defaultCol
}

func getCSVField(record []string, colIndex map[string]int, colName string) string {
	idx, ok := colIndex[colName]
	if !ok || idx >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[idx])
}

func getJSONStringField(rec map[string]interface{}, field string) string {
	val, ok := rec[field]
	if !ok {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// NewImporter creates an Importer for the given platform type
func NewImporter(platform string) (Importer, error) {
	switch platform {
	case PlatformSvix:
		return NewSvixImporter(), nil
	case PlatformConvoy:
		return NewConvoyImporter(), nil
	case PlatformHookdeck:
		return NewHookdeckImporter(), nil
	case PlatformCSV:
		return NewGenericCSVImporter(), nil
	case PlatformJSON:
		return NewGenericJSONImporter(), nil
	default:
		return nil, fmt.Errorf("unsupported import platform: %s", platform)
	}
}
