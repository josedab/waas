package experiment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_CreateExperiment_Valid(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	exp, err := svc.CreateExperiment(context.Background(), "tenant-1", &CreateExperimentRequest{
		Name:      "Payload Format Test",
		EventType: "order.created",
		Variants: []Variant{
			{ID: "control", Name: "JSON", TrafficPercent: 50, IsControl: true},
			{ID: "treatment", Name: "Protobuf", TrafficPercent: 50},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, StatusDraft, exp.Status)
	assert.Equal(t, 2, len(exp.Variants))
	assert.Equal(t, 0.95, exp.SuccessCriteria.ConfidenceLevel)
}

func TestService_CreateExperiment_NotEnoughVariants(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	_, err := svc.CreateExperiment(context.Background(), "tenant-1", &CreateExperimentRequest{
		Name:      "Bad",
		EventType: "test",
		Variants:  []Variant{{ID: "only", Name: "Only", TrafficPercent: 100}},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 2 variants")
}

func TestService_CreateExperiment_TrafficNotSum100(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	_, err := svc.CreateExperiment(context.Background(), "tenant-1", &CreateExperimentRequest{
		Name:      "Bad",
		EventType: "test",
		Variants: []Variant{
			{ID: "a", Name: "A", TrafficPercent: 30},
			{ID: "b", Name: "B", TrafficPercent: 30},
		},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sum to 100")
}

func TestAssignVariant_Deterministic(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)

	variants := []Variant{
		{ID: "control", TrafficPercent: 50},
		{ID: "treatment", TrafficPercent: 50},
	}

	v1 := svc.AssignVariant(context.Background(), "exp-1", "wh-123", variants)
	v2 := svc.AssignVariant(context.Background(), "exp-1", "wh-123", variants)
	assert.Equal(t, v1, v2, "same webhook should always get same variant")

	// Different webhooks may get different variants
	v3 := svc.AssignVariant(context.Background(), "exp-1", "wh-456", variants)
	_ = v3 // just verifying it doesn't panic
}

func TestChiSquaredTest_Significant(t *testing.T) {
	t.Parallel()
	variants := []VariantResult{
		{VariantID: "a", SuccessRate: 0.95, SampleSize: 1000},
		{VariantID: "b", SuccessRate: 0.80, SampleSize: 1000},
	}
	assert.True(t, chiSquaredTest(variants, 0.95))
}

func TestChiSquaredTest_NotSignificant(t *testing.T) {
	t.Parallel()
	variants := []VariantResult{
		{VariantID: "a", SuccessRate: 0.90, SampleSize: 10},
		{VariantID: "b", SuccessRate: 0.89, SampleSize: 10},
	}
	assert.False(t, chiSquaredTest(variants, 0.95))
}
