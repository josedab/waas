package replay

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// DiffVisualization represents a side-by-side diff of original vs replayed delivery
type DiffVisualization struct {
	OriginalDeliveryID string        `json:"original_delivery_id"`
	ReplayedDeliveryID string        `json:"replayed_delivery_id"`
	PayloadDiff        *PayloadDiff  `json:"payload_diff,omitempty"`
	HeaderDiff         *HeaderDiff   `json:"header_diff,omitempty"`
	ResponseDiff       *ResponseDiff `json:"response_diff,omitempty"`
	TimingDiff         *TimingDiff   `json:"timing_diff"`
	Summary            DiffSummary   `json:"summary"`
}

// PayloadDiff shows differences in request payloads
type PayloadDiff struct {
	Original  json.RawMessage `json:"original"`
	Replayed  json.RawMessage `json:"replayed"`
	Changes   []FieldChange   `json:"changes,omitempty"`
	Identical bool            `json:"identical"`
}

// HeaderDiff shows differences in HTTP headers
type HeaderDiff struct {
	Original  map[string]string `json:"original"`
	Replayed  map[string]string `json:"replayed"`
	Added     []string          `json:"added,omitempty"`
	Removed   []string          `json:"removed,omitempty"`
	Modified  []FieldChange     `json:"modified,omitempty"`
	Identical bool              `json:"identical"`
}

// ResponseDiff shows differences in delivery responses
type ResponseDiff struct {
	OriginalStatus int    `json:"original_status"`
	ReplayedStatus int    `json:"replayed_status"`
	OriginalBody   string `json:"original_body,omitempty"`
	ReplayedBody   string `json:"replayed_body,omitempty"`
	StatusChanged  bool   `json:"status_changed"`
	BodyChanged    bool   `json:"body_changed"`
}

// TimingDiff shows differences in delivery timing
type TimingDiff struct {
	OriginalLatencyMs float64   `json:"original_latency_ms"`
	ReplayedLatencyMs float64   `json:"replayed_latency_ms"`
	DeltaMs           float64   `json:"delta_ms"`
	OriginalTime      time.Time `json:"original_time"`
	ReplayedTime      time.Time `json:"replayed_time"`
}

// FieldChange represents a single field difference
type FieldChange struct {
	Path     string `json:"path"`
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
	Type     string `json:"type"` // added, removed, modified
}

// DiffSummary provides a quick overview of differences
type DiffSummary struct {
	TotalChanges    int  `json:"total_changes"`
	PayloadChanged  bool `json:"payload_changed"`
	HeadersChanged  bool `json:"headers_changed"`
	ResponseChanged bool `json:"response_changed"`
	TimingChanged   bool `json:"timing_changed"`
}

// ComputePayloadDiff compares two JSON payloads and produces a diff
func ComputePayloadDiff(original, replayed json.RawMessage) *PayloadDiff {
	diff := &PayloadDiff{
		Original: original,
		Replayed: replayed,
	}

	if string(original) == string(replayed) {
		diff.Identical = true
		return diff
	}

	var origMap, replayMap map[string]interface{}
	if err := json.Unmarshal(original, &origMap); err != nil {
		diff.Changes = append(diff.Changes, FieldChange{
			Path: "$", Type: "modified", OldValue: string(original), NewValue: string(replayed),
		})
		return diff
	}
	if err := json.Unmarshal(replayed, &replayMap); err != nil {
		diff.Changes = append(diff.Changes, FieldChange{
			Path: "$", Type: "modified", OldValue: string(original), NewValue: string(replayed),
		})
		return diff
	}

	diff.Changes = compareObjects(origMap, replayMap, "$")
	diff.Identical = len(diff.Changes) == 0
	return diff
}

// ComputeHeaderDiff compares two sets of headers
func ComputeHeaderDiff(original, replayed map[string]string) *HeaderDiff {
	diff := &HeaderDiff{
		Original: original,
		Replayed: replayed,
	}

	for key, origVal := range original {
		repVal, exists := replayed[key]
		if !exists {
			diff.Removed = append(diff.Removed, key)
		} else if origVal != repVal {
			diff.Modified = append(diff.Modified, FieldChange{
				Path: key, OldValue: origVal, NewValue: repVal, Type: "modified",
			})
		}
	}

	for key := range replayed {
		if _, exists := original[key]; !exists {
			diff.Added = append(diff.Added, key)
		}
	}

	diff.Identical = len(diff.Added) == 0 && len(diff.Removed) == 0 && len(diff.Modified) == 0
	return diff
}

func compareObjects(orig, replay map[string]interface{}, prefix string) []FieldChange {
	var changes []FieldChange

	for key, origVal := range orig {
		path := prefix + "." + key
		repVal, exists := replay[key]
		if !exists {
			changes = append(changes, FieldChange{
				Path: path, OldValue: fmt.Sprintf("%v", origVal), Type: "removed",
			})
			continue
		}

		origStr := fmt.Sprintf("%v", origVal)
		repStr := fmt.Sprintf("%v", repVal)

		// Recurse into nested objects
		origMap, origIsMap := origVal.(map[string]interface{})
		repMap, repIsMap := repVal.(map[string]interface{})
		if origIsMap && repIsMap {
			changes = append(changes, compareObjects(origMap, repMap, path)...)
			continue
		}

		if origStr != repStr {
			changes = append(changes, FieldChange{
				Path: path, OldValue: origStr, NewValue: repStr, Type: "modified",
			})
		}
	}

	for key, repVal := range replay {
		if _, exists := orig[key]; !exists {
			path := prefix + "." + key
			changes = append(changes, FieldChange{
				Path: path, NewValue: fmt.Sprintf("%v", repVal), Type: "added",
			})
		}
	}

	return changes
}

// ConditionalReplayFilter defines filter conditions for conditional replay
type ConditionalReplayFilter struct {
	EventTypes    []string               `json:"event_types,omitempty"`
	StatusCodes   []int                  `json:"status_codes,omitempty"`
	EndpointIDs   []string               `json:"endpoint_ids,omitempty"`
	PayloadMatch  map[string]interface{} `json:"payload_match,omitempty"`
	HeaderMatch   map[string]string      `json:"header_match,omitempty"`
	ErrorContains string                 `json:"error_contains,omitempty"`
	MinLatencyMs  float64                `json:"min_latency_ms,omitempty"`
	MaxLatencyMs  float64                `json:"max_latency_ms,omitempty"`
	TimeRange     *TimeRange             `json:"time_range,omitempty"`
	Breakpoints   []ReplayBreakpoint     `json:"breakpoints,omitempty"`
}

// Note: TimeRange is defined in models.go

// ReplayBreakpoint defines a conditional breakpoint during replay
type ReplayBreakpoint struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Condition  string                 `json:"condition"` // CEL expression or simple match
	Action     string                 `json:"action"`    // pause, log, skip, modify
	ModifySpec map[string]interface{} `json:"modify_spec,omitempty"`
}

// ConditionalReplayRequest extends BulkReplayRequest with conditional filters
type ConditionalReplayRequest struct {
	BulkReplayRequest
	Filter      ConditionalReplayFilter `json:"filter"`
	Breakpoints []ReplayBreakpoint      `json:"breakpoints,omitempty"`
	StepMode    bool                    `json:"step_mode"`
}

// ConditionalReplayResult extends BulkReplayResult with filter/breakpoint info
type ConditionalReplayResult struct {
	BulkReplayResult
	FilterMatched  int                 `json:"filter_matched"`
	FilterSkipped  int                 `json:"filter_skipped"`
	BreakpointHits []BreakpointHit     `json:"breakpoint_hits,omitempty"`
	Diffs          []DiffVisualization `json:"diffs,omitempty"`
}

// BreakpointHit records when a breakpoint was triggered
type BreakpointHit struct {
	BreakpointID string    `json:"breakpoint_id"`
	DeliveryID   string    `json:"delivery_id"`
	Action       string    `json:"action"`
	HitAt        time.Time `json:"hit_at"`
}

// MatchesFilter checks if a delivery archive matches the conditional filter
func (f *ConditionalReplayFilter) MatchesFilter(archive *DeliveryArchive) bool {
	if len(f.EndpointIDs) > 0 && !containsString(f.EndpointIDs, archive.EndpointID) {
		return false
	}

	if len(f.StatusCodes) > 0 && !containsInt(f.StatusCodes, archive.LastHTTPStatus) {
		return false
	}

	if f.ErrorContains != "" && !strings.Contains(archive.LastError, f.ErrorContains) {
		return false
	}

	if f.TimeRange != nil {
		if archive.CreatedAt.Before(f.TimeRange.Start) || archive.CreatedAt.After(f.TimeRange.End) {
			return false
		}
	}

	// Payload matching
	if len(f.PayloadMatch) > 0 && archive.Payload != nil {
		var payloadMap map[string]interface{}
		if err := json.Unmarshal(archive.Payload, &payloadMap); err == nil {
			for key, expected := range f.PayloadMatch {
				actual, exists := payloadMap[key]
				if !exists || fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected) {
					return false
				}
			}
		}
	}

	return true
}

// Note: DeliveryArchive and TimeRange are defined in models.go

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func containsInt(slice []int, n int) bool {
	for _, v := range slice {
		if v == n {
			return true
		}
	}
	return false
}
