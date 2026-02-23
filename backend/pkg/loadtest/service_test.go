package loadtest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateConfig(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	// Missing URL
	_, err := svc.CreateTestRun("t1", &TestConfig{RPS: 10, Duration: "1m"})
	assert.Error(t, err)

	// Missing RPS
	_, err = svc.CreateTestRun("t1", &TestConfig{EndpointURL: "http://example.com", Duration: "1m"})
	assert.Error(t, err)

	// RPS too high
	_, err = svc.CreateTestRun("t1", &TestConfig{EndpointURL: "http://example.com", RPS: 100000, Duration: "1m"})
	assert.Error(t, err)

	// Invalid duration
	_, err = svc.CreateTestRun("t1", &TestConfig{EndpointURL: "http://example.com", RPS: 10, Duration: "invalid"})
	assert.Error(t, err)
}

func TestComputeRPS(t *testing.T) {
	svc := NewService(NewMemoryRepository(), nil)

	t.Run("constant", func(t *testing.T) {
		cfg := &TestConfig{RPS: 100, Pattern: PatternConstant}
		assert.Equal(t, 100, svc.computeRPS(cfg, 30, 60))
	})

	t.Run("ramp-up", func(t *testing.T) {
		cfg := &TestConfig{RPS: 100, Pattern: PatternRampUp, RampUpStart: 10}
		rps := svc.computeRPS(cfg, 30, 60) // halfway
		assert.Greater(t, rps, 10)
		assert.LessOrEqual(t, rps, 100)
	})

	t.Run("sine-wave", func(t *testing.T) {
		cfg := &TestConfig{RPS: 100, Pattern: PatternSineWave}
		rps := svc.computeRPS(cfg, 15, 60) // quarter
		assert.Greater(t, rps, 0)
	})
}

func TestPercentile(t *testing.T) {
	data := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	assert.Equal(t, 50.0, percentile(data, 50))
	assert.Equal(t, 100.0, percentile(data, 99))
	assert.Equal(t, 10.0, percentile(data, 10))
}

func TestDefaultScenarios(t *testing.T) {
	scenarios := DefaultScenarios()
	assert.Len(t, scenarios, 5)
	assert.Equal(t, "smoke", scenarios[0].ID)
}

func TestGenerateRecommendations(t *testing.T) {
	svc := NewService(NewMemoryRepository(), nil)

	report := &TestReport{
		TotalRequests: 1000,
		FailureCount:  200,
		ErrorRate:     20.0,
		LatencyP99:    6000,
	}

	recs := svc.generateRecommendations(report)
	assert.NotEmpty(t, recs)
}

func TestDefaultServiceConfig(t *testing.T) {
	config := DefaultServiceConfig()
	assert.Equal(t, 10000, config.MaxRPS)
	assert.Equal(t, 1*time.Hour, config.MaxDuration)
	assert.Equal(t, 5, config.MaxConcurrentTests)
}

func TestGetScenarios(t *testing.T) {
	svc := NewService(NewMemoryRepository(), nil)
	scenarios := svc.GetScenarios()
	require.Len(t, scenarios, 5)
}
