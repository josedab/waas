package eventlineage

import (
	"encoding/json"
	"time"
)

// Operation constants
const (
	OpIngest    = "ingest"
	OpTransform = "transform"
	OpFanOut    = "fan_out"
	OpRoute     = "route"
	OpDeliver   = "deliver"
	OpRetry     = "retry"
	OpFilter    = "filter"
	OpCorrelate = "correlate"
)

// LineageEntry represents a single event in the lineage graph.
type LineageEntry struct {
	ID            string          `json:"id" db:"id"`
	TenantID      string          `json:"tenant_id" db:"tenant_id"`
	EventID       string          `json:"event_id" db:"event_id"`
	ParentEventID string          `json:"parent_event_id,omitempty" db:"parent_event_id"`
	EventType     string          `json:"event_type" db:"event_type"`
	Source        string          `json:"source" db:"source"`
	Operation     string          `json:"operation" db:"operation"`
	Metadata      json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	PayloadHash   string          `json:"payload_hash" db:"payload_hash"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
}

// LineageGraph represents the full event journey graph.
type LineageGraph struct {
	RootEventID string        `json:"root_event_id"`
	Nodes       []LineageNode `json:"nodes"`
	Edges       []LineageEdge `json:"edges"`
	TotalDepth  int           `json:"total_depth"`
}

// LineageNode is a node in the lineage graph visualization.
type LineageNode struct {
	ID        string `json:"id"`
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"`
	Operation string `json:"operation"`
	Source    string `json:"source"`
	Depth     int    `json:"depth"`
}

// LineageEdge connects two nodes in the graph.
type LineageEdge struct {
	FromEventID string `json:"from_event_id"`
	ToEventID   string `json:"to_event_id"`
	Operation   string `json:"operation"`
}

// LineageStats holds aggregated lineage metrics.
type LineageStats struct {
	TotalEvents     int64            `json:"total_events"`
	TotalChains     int64            `json:"total_chains"`
	AvgChainDepth   float64          `json:"avg_chain_depth"`
	OperationCounts map[string]int64 `json:"operation_counts"`
}

// Request DTOs

type RecordLineageRequest struct {
	EventID       string          `json:"event_id" binding:"required"`
	ParentEventID string          `json:"parent_event_id"`
	EventType     string          `json:"event_type" binding:"required"`
	Source        string          `json:"source"`
	Operation     string          `json:"operation" binding:"required"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
	PayloadHash   string          `json:"payload_hash"`
}
