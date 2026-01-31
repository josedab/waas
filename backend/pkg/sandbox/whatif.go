package sandbox

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TimeRangeReplayRequest supports time-range based replay selection
type TimeRangeReplayRequest struct {
	SandboxID    string                 `json:"sandbox_id" binding:"required"`
	StartTime    time.Time              `json:"start_time" binding:"required"`
	EndTime      time.Time              `json:"end_time" binding:"required"`
	EndpointID   string                 `json:"endpoint_id,omitempty"`
	Status       string                 `json:"status,omitempty"` // filter: delivered, failed, retrying
	EventTypes   []string               `json:"event_types,omitempty"`
	MaxEvents    int                    `json:"max_events,omitempty"`
	WhatIf       *WhatIfConfig          `json:"what_if,omitempty"`
	ModifyPayload map[string]interface{} `json:"modify_payload,omitempty"`
}

// WhatIfConfig enables hypothetical scenario testing
type WhatIfConfig struct {
	// OverrideURL sends replays to a different endpoint
	OverrideURL string `json:"override_url,omitempty"`
	// InjectHeaders adds headers to all replayed requests
	InjectHeaders map[string]string `json:"inject_headers,omitempty"`
	// InjectLatencyMs adds artificial latency
	InjectLatencyMs int `json:"inject_latency_ms,omitempty"`
	// SimulateFailureRate causes random failures at the given rate (0-1)
	SimulateFailureRate float64 `json:"simulate_failure_rate,omitempty"`
	// TransformPayload applies JSONPath-like transforms
	TransformPayload map[string]interface{} `json:"transform_payload,omitempty"`
}

// SharedSession allows team members to share sandbox sessions
type SharedSession struct {
	ID          string    `json:"id" db:"id"`
	SandboxID   string    `json:"sandbox_id" db:"sandbox_id"`
	TenantID    string    `json:"tenant_id" db:"tenant_id"`
	CreatedBy   string    `json:"created_by" db:"created_by"`
	ShareToken  string    `json:"share_token" db:"share_token"`
	Permissions string    `json:"permissions" db:"permissions"` // read, replay, admin
	ExpiresAt   time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// ShareSessionRequest is the DTO for sharing a sandbox session
type ShareSessionRequest struct {
	SandboxID   string `json:"sandbox_id" binding:"required"`
	Permissions string `json:"permissions" binding:"required,oneof=read replay admin"`
	TTLHours    int    `json:"ttl_hours" binding:"required,min=1,max=168"`
}

// DiffResult compares original and replayed request/response pairs
type DiffResult struct {
	EventID          string       `json:"event_id"`
	OriginalRequest  DiffPayload  `json:"original_request"`
	ReplayedRequest  DiffPayload  `json:"replayed_request"`
	OriginalResponse DiffPayload  `json:"original_response"`
	ReplayedResponse DiffPayload  `json:"replayed_response"`
	FieldDiffs       []FieldDiff  `json:"field_diffs"`
	LatencyDelta     int64        `json:"latency_delta_ms"`
	StatusChanged    bool         `json:"status_changed"`
	IsDifferent      bool         `json:"is_different"`
}

// DiffPayload holds the data being compared
type DiffPayload struct {
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body"`
	StatusCode int               `json:"status_code,omitempty"`
	LatencyMs  int64             `json:"latency_ms,omitempty"`
}

// FieldDiff describes a difference in a specific field
type FieldDiff struct {
	Path     string `json:"path"`
	Original string `json:"original"`
	Replayed string `json:"replayed"`
	Type     string `json:"type"` // added, removed, changed
}

// ReplaySummary provides a summary of a time-range replay
type ReplaySummary struct {
	SandboxID      string       `json:"sandbox_id"`
	TimeRange      string       `json:"time_range"`
	TotalEvents    int          `json:"total_events"`
	Replayed       int          `json:"replayed"`
	Failed         int          `json:"failed"`
	Skipped        int          `json:"skipped"`
	StatusMatches  int          `json:"status_matches"`
	StatusChanges  int          `json:"status_changes"`
	AvgLatencyMs   float64      `json:"avg_latency_ms"`
	Diffs          []DiffResult `json:"diffs,omitempty"`
	DurationMs     int64        `json:"duration_ms"`
	GeneratedAt    time.Time    `json:"generated_at"`
}

// CreateSharedSession creates a share token for a sandbox
func CreateSharedSession(tenantID, sandboxID, createdBy, permissions string, ttlHours int) *SharedSession {
	token := fmt.Sprintf("share_%s", uuid.New().String()[:12])
	return &SharedSession{
		ID:          uuid.New().String(),
		SandboxID:   sandboxID,
		TenantID:    tenantID,
		CreatedBy:   createdBy,
		ShareToken:  token,
		Permissions: permissions,
		ExpiresAt:   time.Now().Add(time.Duration(ttlHours) * time.Hour),
		CreatedAt:   time.Now(),
	}
}

// ComputeDiff compares original and replayed sessions
func ComputeDiff(original, replayed *ReplaySession) *DiffResult {
	diff := &DiffResult{
		EventID: original.SourceEventID,
		OriginalResponse: DiffPayload{
			Body:       original.ResponseBody,
			StatusCode: original.ResponseStatus,
			LatencyMs:  original.ResponseLatencyMs,
		},
		ReplayedResponse: DiffPayload{
			Body:       replayed.ResponseBody,
			StatusCode: replayed.ResponseStatus,
			LatencyMs:  replayed.ResponseLatencyMs,
		},
		OriginalRequest: DiffPayload{
			Body: original.OriginalPayload,
		},
		ReplayedRequest: DiffPayload{
			Body: replayed.MaskedPayload,
		},
		LatencyDelta:  replayed.ResponseLatencyMs - original.ResponseLatencyMs,
		StatusChanged: original.ResponseStatus != replayed.ResponseStatus,
	}

	// Compute field-level diffs on response bodies
	diff.FieldDiffs = computeJSONDiffs(original.ResponseBody, replayed.ResponseBody)
	diff.IsDifferent = diff.StatusChanged || len(diff.FieldDiffs) > 0

	return diff
}

// computeJSONDiffs finds differences between two JSON strings
func computeJSONDiffs(originalJSON, replayedJSON string) []FieldDiff {
	var original, replayed map[string]interface{}

	if err := json.Unmarshal([]byte(originalJSON), &original); err != nil {
		return nil
	}
	if err := json.Unmarshal([]byte(replayedJSON), &replayed); err != nil {
		return nil
	}

	var diffs []FieldDiff
	flatOriginal := flattenJSON("", original)
	flatReplayed := flattenJSON("", replayed)

	// Check for changed and removed fields
	for path, origVal := range flatOriginal {
		if repVal, exists := flatReplayed[path]; !exists {
			diffs = append(diffs, FieldDiff{Path: path, Original: origVal, Type: "removed"})
		} else if origVal != repVal {
			diffs = append(diffs, FieldDiff{Path: path, Original: origVal, Replayed: repVal, Type: "changed"})
		}
	}

	// Check for added fields
	for path, repVal := range flatReplayed {
		if _, exists := flatOriginal[path]; !exists {
			diffs = append(diffs, FieldDiff{Path: path, Replayed: repVal, Type: "added"})
		}
	}

	sort.Slice(diffs, func(i, j int) bool { return diffs[i].Path < diffs[j].Path })
	return diffs
}

// flattenJSON flattens a nested map into dot-notation paths
func flattenJSON(prefix string, data map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for key, value := range data {
		fullPath := key
		if prefix != "" {
			fullPath = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			for k, val := range flattenJSON(fullPath, v) {
				result[k] = val
			}
		default:
			result[fullPath] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

// FormatDiffReport generates a human-readable diff report
func FormatDiffReport(diffs []DiffResult) string {
	var b strings.Builder
	b.WriteString("═══ Replay Diff Report ═══\n\n")

	changed := 0
	unchanged := 0
	for _, d := range diffs {
		if d.IsDifferent {
			changed++
		} else {
			unchanged++
		}
	}

	b.WriteString(fmt.Sprintf("Total: %d events | Changed: %d | Unchanged: %d\n\n", len(diffs), changed, unchanged))

	for _, d := range diffs {
		if !d.IsDifferent {
			continue
		}
		b.WriteString(fmt.Sprintf("─── Event: %s ───\n", d.EventID))
		if d.StatusChanged {
			b.WriteString(fmt.Sprintf("  Status: %d → %d\n", d.OriginalResponse.StatusCode, d.ReplayedResponse.StatusCode))
		}
		if d.LatencyDelta != 0 {
			sign := "+"
			if d.LatencyDelta < 0 {
				sign = ""
			}
			b.WriteString(fmt.Sprintf("  Latency: %s%dms\n", sign, d.LatencyDelta))
		}
		for _, fd := range d.FieldDiffs {
			switch fd.Type {
			case "added":
				b.WriteString(fmt.Sprintf("  + %s: %s\n", fd.Path, fd.Replayed))
			case "removed":
				b.WriteString(fmt.Sprintf("  - %s: %s\n", fd.Path, fd.Original))
			case "changed":
				b.WriteString(fmt.Sprintf("  ~ %s: %s → %s\n", fd.Path, fd.Original, fd.Replayed))
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

// PayloadChecksum generates a stable checksum for payload deduplication
func PayloadChecksum(payload string) string {
	h := sha256.Sum256([]byte(payload))
	return fmt.Sprintf("%x", h[:8])
}
