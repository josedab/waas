package progressive

import (
	"context"
	"testing"
)

func TestNewService(t *testing.T) {
	svc := NewService(nil, nil)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestCreateRollout(t *testing.T) {
	svc := NewService(nil, nil)
	rollout, err := svc.CreateRollout(context.Background(), "tenant-1", &CreateRolloutRequest{
		Name:       "Canary v2 endpoint",
		EndpointID: "ep-123",
		Strategy:   StrategyCanary,
		TargetConfig: RolloutConfig{
			URL: "https://target.example.com/webhook",
		},
		BaselineConfig: RolloutConfig{
			URL: "https://baseline.example.com/webhook",
		},
		TrafficSplit: TrafficSplit{BaselinePercent: 90, TargetPercent: 10},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rollout.Status != StatusPending {
		t.Errorf("expected pending, got %s", rollout.Status)
	}
	if rollout.TrafficSplit.TargetPercent != 10 {
		t.Errorf("expected target 10%%, got %d%%", rollout.TrafficSplit.TargetPercent)
	}
}

func TestStartRollout(t *testing.T) {
	svc := NewService(nil, nil)
	rollout, _ := svc.CreateRollout(context.Background(), "tenant-1", &CreateRolloutRequest{
		Name:       "Start test",
		EndpointID: "ep-123",
		Strategy:   StrategyCanary,
		TargetConfig: RolloutConfig{
			URL: "https://target.example.com/webhook",
		},
		BaselineConfig: RolloutConfig{
			URL: "https://baseline.example.com/webhook",
		},
	})

	started, err := svc.StartRollout(context.Background(), "tenant-1", rollout.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if started.Status != StatusActive {
		t.Errorf("expected active, got %s", started.Status)
	}
}

func TestUpdateTraffic(t *testing.T) {
	svc := NewService(nil, nil)
	rollout, _ := svc.CreateRollout(context.Background(), "tenant-1", &CreateRolloutRequest{
		Name:       "Traffic test",
		EndpointID: "ep-123",
		Strategy:   StrategyPercentage,
		TargetConfig: RolloutConfig{
			URL: "https://target.example.com/webhook",
		},
		BaselineConfig: RolloutConfig{
			URL: "https://baseline.example.com/webhook",
		},
	})
	svc.StartRollout(context.Background(), "tenant-1", rollout.ID)

	updated, err := svc.UpdateTraffic(context.Background(), "tenant-1", rollout.ID, &UpdateTrafficRequest{
		BaselinePercent: 50,
		TargetPercent:   50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.TrafficSplit.TargetPercent != 50 {
		t.Errorf("expected target 50%%, got %d%%", updated.TrafficSplit.TargetPercent)
	}
}

func TestUpdateTraffic_InvalidSplit(t *testing.T) {
	svc := NewService(nil, nil)
	rollout, _ := svc.CreateRollout(context.Background(), "tenant-1", &CreateRolloutRequest{
		Name:       "Invalid split test",
		EndpointID: "ep-123",
		Strategy:   StrategyCanary,
		TargetConfig: RolloutConfig{
			URL: "https://target.example.com/webhook",
		},
		BaselineConfig: RolloutConfig{
			URL: "https://baseline.example.com/webhook",
		},
	})
	svc.StartRollout(context.Background(), "tenant-1", rollout.ID)

	_, err := svc.UpdateTraffic(context.Background(), "tenant-1", rollout.ID, &UpdateTrafficRequest{
		BaselinePercent: 60,
		TargetPercent:   60,
	})
	if err == nil {
		t.Error("expected error for invalid traffic split")
	}
}

func TestCompleteRollout(t *testing.T) {
	svc := NewService(nil, nil)
	rollout, _ := svc.CreateRollout(context.Background(), "tenant-1", &CreateRolloutRequest{
		Name:       "Complete test",
		EndpointID: "ep-123",
		Strategy:   StrategyBlueGreen,
		TargetConfig: RolloutConfig{
			URL: "https://target.example.com/webhook",
		},
		BaselineConfig: RolloutConfig{
			URL: "https://baseline.example.com/webhook",
		},
	})
	svc.StartRollout(context.Background(), "tenant-1", rollout.ID)

	completed, err := svc.CompleteRollout(context.Background(), "tenant-1", rollout.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if completed.Status != StatusCompleted {
		t.Errorf("expected completed, got %s", completed.Status)
	}
	if completed.TrafficSplit.TargetPercent != 100 {
		t.Errorf("expected target 100%%, got %d%%", completed.TrafficSplit.TargetPercent)
	}
}

func TestRollbackRollout(t *testing.T) {
	svc := NewService(nil, nil)
	rollout, _ := svc.CreateRollout(context.Background(), "tenant-1", &CreateRolloutRequest{
		Name:       "Rollback test",
		EndpointID: "ep-123",
		Strategy:   StrategyCanary,
		TargetConfig: RolloutConfig{
			URL: "https://target.example.com/webhook",
		},
		BaselineConfig: RolloutConfig{
			URL: "https://baseline.example.com/webhook",
		},
	})
	svc.StartRollout(context.Background(), "tenant-1", rollout.ID)

	rolledBack, err := svc.RollbackRollout(context.Background(), "tenant-1", rollout.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rolledBack.Status != StatusRolledBack {
		t.Errorf("expected rolled_back, got %s", rolledBack.Status)
	}
	if rolledBack.TrafficSplit.BaselinePercent != 100 {
		t.Errorf("expected baseline 100%%, got %d%%", rolledBack.TrafficSplit.BaselinePercent)
	}
}

func TestEvaluateRollout(t *testing.T) {
	svc := NewService(nil, nil)
	rollout, _ := svc.CreateRollout(context.Background(), "tenant-1", &CreateRolloutRequest{
		Name:       "Evaluate test",
		EndpointID: "ep-123",
		Strategy:   StrategyCanary,
		TargetConfig: RolloutConfig{
			URL: "https://target.example.com/webhook",
		},
		BaselineConfig: RolloutConfig{
			URL: "https://baseline.example.com/webhook",
		},
		SuccessCriteria: SuccessCriteria{
			MinSuccessRate: 0.95,
			MaxLatencyMs:   500,
			MinSampleSize:  100,
		},
	})

	// Without enough samples, evaluation should report insufficient data
	result, err := svc.EvaluateRollout(context.Background(), "tenant-1", rollout.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected evaluation to not pass with zero samples")
	}

	// Simulate metrics that meet criteria
	r, _ := svc.repo.Get(context.Background(), "tenant-1", rollout.ID)
	r.Metrics.TargetMetrics = VariantMetrics{
		Requests:     200,
		Successes:    196,
		Failures:     4,
		AvgLatencyMs: 120,
		P99LatencyMs: 350,
		SuccessRate:  0.98,
	}
	svc.repo.Update(context.Background(), r)

	result, err = svc.EvaluateRollout(context.Background(), "tenant-1", rollout.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected evaluation to pass, got message: %s", result.Message)
	}
}
