package billing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateFlatRate_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		basePrice int64
		usage     int64
		want      int64
	}{
		{"zero usage", 2900, 0, 2900},
		{"normal usage", 2900, 5000, 2900},
		{"huge usage", 2900, 10_000_000, 2900},
		{"zero price", 0, 1000, 0},
		{"negative usage", 2900, -1, 2900},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &PricingPlan{BasePrice: tt.basePrice}
			assert.Equal(t, tt.want, CalculateFlatRate(plan, tt.usage))
		})
	}
}

func TestCalculatePerEvent_TableDriven(t *testing.T) {
	plan := &PricingPlan{
		BasePrice:      2900,
		IncludedEvents: 10000,
		OveragePrice:   100, // per 1000 events
	}

	tests := []struct {
		name  string
		usage int64
		want  int64
	}{
		{"zero usage", 0, 2900},
		{"within included", 5000, 2900},
		{"at included limit", 10000, 2900},
		{"1 event over", 10001, 3000},
		{"999 over", 10999, 3000},
		{"exactly 1000 over", 11000, 3000},
		{"1001 over", 11001, 3100},
		{"5000 over", 15000, 3400},
		{"large overage", 110000, 12900},
		{"negative usage", -100, 2900},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CalculatePerEvent(plan, tt.usage))
		})
	}
}

func TestCalculatePerEvent_ZeroUsageAllowance(t *testing.T) {
	plan := &PricingPlan{
		BasePrice:      0,
		IncludedEvents: 0,
		OveragePrice:   50,
	}
	// Any usage is overage
	assert.Equal(t, int64(50), CalculatePerEvent(plan, 1))
	assert.Equal(t, int64(50), CalculatePerEvent(plan, 999))
	assert.Equal(t, int64(50), CalculatePerEvent(plan, 1000))
}

func TestCalculateTiered_TableDriven(t *testing.T) {
	tiers := []PricingTier{
		{UpTo: 1000, PricePerUnit: 10},
		{UpTo: 10000, PricePerUnit: 5},
		{UpTo: -1, PricePerUnit: 2}, // Unlimited
	}

	tests := []struct {
		name  string
		usage int64
		want  int64
	}{
		{"zero usage", 0, 0},
		{"1 event", 1, 10},
		{"within first tier", 500, 5000},
		{"at first tier boundary", 1000, 10000},
		{"first event in second tier", 1001, 10000 + 5},
		{"within second tier", 5000, 10000 + 20000},
		{"at second tier boundary", 10000, 10000 + 45000},
		{"into unlimited tier", 15000, 10000 + 45000 + 10000},
		{"large usage", 100000, 10000 + 45000 + 180000},
		{"negative usage", -1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CalculateTiered(tiers, tt.usage))
		})
	}
}

func TestCalculateTiered_SingleTier(t *testing.T) {
	tiers := []PricingTier{
		{UpTo: -1, PricePerUnit: 3},
	}
	assert.Equal(t, int64(3000), CalculateTiered(tiers, 1000))
}

func TestCalculateTiered_NilAndEmpty(t *testing.T) {
	assert.Equal(t, int64(0), CalculateTiered(nil, 1000))
	assert.Equal(t, int64(0), CalculateTiered([]PricingTier{}, 500))
}

func TestCalculateOverage_TableDriven(t *testing.T) {
	plan := &PricingPlan{IncludedEvents: 10000}

	tests := []struct {
		name  string
		usage int64
		want  int64
	}{
		{"zero usage", 0, 0},
		{"within included", 5000, 0},
		{"at exact boundary", 10000, 0},
		{"1 over", 10001, 1},
		{"large overage", 100000, 90000},
		{"negative usage", -100, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CalculateOverage(plan, tt.usage))
		})
	}
}

func TestCalculateOverage_ZeroIncluded(t *testing.T) {
	plan := &PricingPlan{IncludedEvents: 0}
	assert.Equal(t, int64(100), CalculateOverage(plan, 100))
	assert.Equal(t, int64(0), CalculateOverage(plan, 0))
}
