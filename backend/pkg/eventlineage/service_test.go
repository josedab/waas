package eventlineage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordLineage_Valid(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	entry, err := svc.RecordLineage(context.Background(), "tenant-1", &RecordLineageRequest{
		EventID:   "evt-1",
		EventType: "order.created",
		Operation: OpIngest,
		Source:    "api-gateway",
	})

	require.NoError(t, err)
	assert.Equal(t, "evt-1", entry.EventID)
	assert.Equal(t, OpIngest, entry.Operation)
}

func TestRecordLineage_WithParent(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	entry, err := svc.RecordLineage(context.Background(), "tenant-1", &RecordLineageRequest{
		EventID:       "evt-2",
		ParentEventID: "evt-1",
		EventType:     "order.created",
		Operation:     OpTransform,
	})

	require.NoError(t, err)
	assert.Equal(t, "evt-1", entry.ParentEventID)
}

func TestRecordLineage_InvalidOperation(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	_, err := svc.RecordLineage(context.Background(), "tenant-1", &RecordLineageRequest{
		EventID:   "evt-3",
		EventType: "test",
		Operation: "invalid_op",
	})

	assert.Error(t, err)
}

func TestCalculateDepth(t *testing.T) {
	t.Parallel()

	edges := []LineageEdge{
		{FromEventID: "a", ToEventID: "b", Operation: OpTransform},
		{FromEventID: "b", ToEventID: "c", Operation: OpFanOut},
		{FromEventID: "b", ToEventID: "d", Operation: OpFanOut},
		{FromEventID: "c", ToEventID: "e", Operation: OpDeliver},
	}

	depth := calculateDepth(edges, "a")
	assert.Equal(t, 3, depth) // a -> b -> c -> e
}
