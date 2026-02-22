package timetravel

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDebugSession(t *testing.T) {
	svc := NewService(nil)

	session, err := svc.CreateDebugSession(context.Background(), "tenant-1", &CreateDebugSessionRequest{
		Name:     "debug-order-flow",
		EventIDs: []string{"evt-1", "evt-2"},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, session.ID)
	assert.Equal(t, "debug-order-flow", session.Name)
	assert.Equal(t, "active", session.Status)
	assert.Len(t, session.EventIDs, 2)
}

func TestCreateDebugSessionValidation(t *testing.T) {
	svc := NewService(nil)

	_, err := svc.CreateDebugSession(context.Background(), "t1", &CreateDebugSessionRequest{})
	assert.Error(t, err)

	_, err = svc.CreateDebugSession(context.Background(), "t1", &CreateDebugSessionRequest{Name: "test"})
	assert.Error(t, err)
}

func TestReplayWithModification(t *testing.T) {
	svc := NewService(nil)

	step, err := svc.ReplayWithModification(context.Background(), "tenant-1", &ReplayWithModificationRequest{
		EventID:         "evt-1",
		ModifiedPayload: json.RawMessage(`{"amount": 200}`),
		DryRun:          true,
	})
	require.NoError(t, err)
	assert.Equal(t, "dry_run_replay", step.Action)
	assert.Equal(t, "evt-1", step.EventID)
}

func TestReplayWithModificationValidation(t *testing.T) {
	svc := NewService(nil)

	_, err := svc.ReplayWithModification(context.Background(), "t1", &ReplayWithModificationRequest{})
	assert.Error(t, err)
}

func TestAddBreakpoint(t *testing.T) {
	svc := NewService(nil)

	bp, err := svc.AddBreakpoint(context.Background(), "session-1", &AddBreakpointRequest{
		Type:      "status_code",
		Condition: "status >= 400",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, bp.ID)
	assert.True(t, bp.Enabled)
}
