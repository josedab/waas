package replay

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestComputePayloadDiff_Identical(t *testing.T) {
	payload := json.RawMessage(`{"id": "123", "name": "test"}`)
	diff := ComputePayloadDiff(payload, payload)
	assert.True(t, diff.Identical)
	assert.Empty(t, diff.Changes)
}

func TestComputePayloadDiff_Modified(t *testing.T) {
	original := json.RawMessage(`{"id": "123", "name": "old"}`)
	replayed := json.RawMessage(`{"id": "123", "name": "new"}`)
	diff := ComputePayloadDiff(original, replayed)
	assert.False(t, diff.Identical)
	assert.NotEmpty(t, diff.Changes)

	found := false
	for _, c := range diff.Changes {
		if c.Path == "$.name" && c.Type == "modified" {
			found = true
			assert.Equal(t, "old", c.OldValue)
			assert.Equal(t, "new", c.NewValue)
		}
	}
	assert.True(t, found, "should find name change")
}

func TestComputePayloadDiff_AddedRemoved(t *testing.T) {
	original := json.RawMessage(`{"a": 1, "b": 2}`)
	replayed := json.RawMessage(`{"b": 2, "c": 3}`)
	diff := ComputePayloadDiff(original, replayed)
	assert.False(t, diff.Identical)

	var added, removed int
	for _, c := range diff.Changes {
		if c.Type == "added" {
			added++
		}
		if c.Type == "removed" {
			removed++
		}
	}
	assert.Equal(t, 1, added)
	assert.Equal(t, 1, removed)
}

func TestComputeHeaderDiff(t *testing.T) {
	original := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer old-token",
		"X-Old-Header":  "value",
	}
	replayed := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer new-token",
		"X-New-Header":  "value",
	}

	diff := ComputeHeaderDiff(original, replayed)
	assert.False(t, diff.Identical)
	assert.Contains(t, diff.Added, "X-New-Header")
	assert.Contains(t, diff.Removed, "X-Old-Header")
	assert.Len(t, diff.Modified, 1)
	assert.Equal(t, "Authorization", diff.Modified[0].Path)
}

func TestConditionalReplayFilter_MatchesFilter(t *testing.T) {
	now := time.Now()

	archive := &DeliveryArchive{
		ID:             "del-1",
		EndpointID:     "ep-1",
		LastHTTPStatus: 500,
		LastError:      "connection timeout",
		CreatedAt:      now,
		Payload:        json.RawMessage(`{"user_id": "123"}`),
	}

	tests := []struct {
		name   string
		filter ConditionalReplayFilter
		want   bool
	}{
		{
			name:   "empty filter matches all",
			filter: ConditionalReplayFilter{},
			want:   true,
		},
		{
			name:   "endpoint match",
			filter: ConditionalReplayFilter{EndpointIDs: []string{"ep-1"}},
			want:   true,
		},
		{
			name:   "endpoint mismatch",
			filter: ConditionalReplayFilter{EndpointIDs: []string{"ep-2"}},
			want:   false,
		},
		{
			name:   "status code match",
			filter: ConditionalReplayFilter{StatusCodes: []int{500, 502}},
			want:   true,
		},
		{
			name:   "error contains match",
			filter: ConditionalReplayFilter{ErrorContains: "timeout"},
			want:   true,
		},
		{
			name:   "error contains mismatch",
			filter: ConditionalReplayFilter{ErrorContains: "auth failure"},
			want:   false,
		},
		{
			name: "payload match",
			filter: ConditionalReplayFilter{
				PayloadMatch: map[string]interface{}{"user_id": "123"},
			},
			want: true,
		},
		{
			name: "payload mismatch",
			filter: ConditionalReplayFilter{
				PayloadMatch: map[string]interface{}{"user_id": "456"},
			},
			want: false,
		},
		{
			name: "time range match",
			filter: ConditionalReplayFilter{
				TimeRange: &TimeRange{Start: now.Add(-1 * time.Hour), End: now.Add(1 * time.Hour)},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.filter.MatchesFilter(archive))
		})
	}
}
