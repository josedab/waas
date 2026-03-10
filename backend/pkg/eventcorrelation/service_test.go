package eventcorrelation

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRule_Valid(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	rule, err := svc.CreateRule(context.Background(), "tenant-1", &CreateRuleRequest{
		Name:           "Order → Payment",
		TriggerEvent:   "order.created",
		FollowEvent:    "payment.completed",
		TimeWindowSec:  300,
		MatchFields:    []string{"order_id"},
		CompositeEvent: "order.paid",
	})

	require.NoError(t, err)
	assert.Equal(t, "order.created", rule.TriggerEvent)
	assert.Equal(t, "payment.completed", rule.FollowEvent)
	assert.Equal(t, 300, rule.TimeWindowSec)
	assert.True(t, rule.IsEnabled)
}

func TestCreateRule_SameEvents(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	_, err := svc.CreateRule(context.Background(), "tenant-1", &CreateRuleRequest{
		Name:           "Bad",
		TriggerEvent:   "order.created",
		FollowEvent:    "order.created",
		CompositeEvent: "bad",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "different")
}

func TestCreateRule_ExcessiveWindow(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	_, err := svc.CreateRule(context.Background(), "tenant-1", &CreateRuleRequest{
		Name:           "Bad",
		TriggerEvent:   "a",
		FollowEvent:    "b",
		TimeWindowSec:  100000,
		CompositeEvent: "c",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "86400")
}

func TestCreateRule_DefaultWindow(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	rule, err := svc.CreateRule(context.Background(), "tenant-1", &CreateRuleRequest{
		Name:           "Test",
		TriggerEvent:   "a",
		FollowEvent:    "b",
		CompositeEvent: "c",
	})

	require.NoError(t, err)
	assert.Equal(t, 300, rule.TimeWindowSec)
}

func TestComputeMatchKey_Deterministic(t *testing.T) {
	t.Parallel()

	payload := json.RawMessage(`{"order_id":"123","customer":"alice"}`)
	fields := []string{"order_id"}

	k1 := computeMatchKey("rule-1", payload, fields)
	k2 := computeMatchKey("rule-1", payload, fields)
	assert.Equal(t, k1, k2)

	k3 := computeMatchKey("rule-1", payload, []string{"customer"})
	assert.NotEqual(t, k1, k3)
}

func TestComputeMatchKey_DifferentRules(t *testing.T) {
	t.Parallel()

	payload := json.RawMessage(`{"order_id":"123"}`)
	fields := []string{"order_id"}

	k1 := computeMatchKey("rule-1", payload, fields)
	k2 := computeMatchKey("rule-2", payload, fields)
	assert.NotEqual(t, k1, k2)
}

func TestBuildCorrelationGraph(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	ctx := context.Background()

	svc.CreateRule(ctx, "t1", &CreateRuleRequest{
		Name: "Order→Payment", TriggerEvent: "order.created", FollowEvent: "payment.completed", CompositeEvent: "order.paid",
	})
	svc.CreateRule(ctx, "t1", &CreateRuleRequest{
		Name: "Payment→Ship", TriggerEvent: "payment.completed", FollowEvent: "shipment.created", CompositeEvent: "order.shipped",
	})

	graph, err := svc.BuildCorrelationGraph(ctx, "t1")
	require.NoError(t, err)
	assert.Len(t, graph.Nodes, 3) // order.created, payment.completed, shipment.created
	assert.Len(t, graph.Edges, 2)
}

func TestBuildCausalChain(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	ctx := context.Background()

	chain, err := svc.BuildCausalChain(ctx, "t1", "event-1")
	require.NoError(t, err)
	assert.NotEmpty(t, chain.ID)
	assert.Equal(t, "event-1", chain.RootEvent)
	assert.Len(t, chain.Events, 1) // only root
}

func TestCrossTenantJoin(t *testing.T) {
	svc := NewService(NewMemoryRepository())
	ctx := context.Background()

	result, err := svc.CrossTenantJoin(ctx, &CrossTenantJoinRequest{
		SourceTenantID: "t1",
		TargetTenantID: "t2",
		JoinField:      "order_id",
		SourceEvent:    "order.created",
		TargetEvent:    "fulfillment.started",
	})
	require.NoError(t, err)
	assert.Equal(t, "order_id", result.JoinField)

	// Same tenant should error
	_, err = svc.CrossTenantJoin(ctx, &CrossTenantJoinRequest{
		SourceTenantID: "t1",
		TargetTenantID: "t1",
		JoinField:      "order_id",
		SourceEvent:    "a",
		TargetEvent:    "b",
	})
	assert.Error(t, err)
}
