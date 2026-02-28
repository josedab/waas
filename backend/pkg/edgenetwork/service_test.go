package edgenetwork

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

func TestRegisterNode(t *testing.T) {
	svc := NewService(nil, nil)
	node, err := svc.RegisterNode(context.Background(), &CreateNodeRequest{
		Name:     "us-east-1a",
		Region:   RegionUSEast1,
		Endpoint: "https://edge-us-east.example.com",
		Capacity: 500,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.ID == "" {
		t.Error("expected node ID")
	}
	if node.Status != NodeStatusHealthy {
		t.Errorf("expected healthy status, got %s", node.Status)
	}
}

func TestRegisterNode_InvalidRegion(t *testing.T) {
	svc := NewService(nil, nil)
	_, err := svc.RegisterNode(context.Background(), &CreateNodeRequest{
		Name:     "invalid",
		Region:   "invalid-region",
		Endpoint: "https://example.com",
	})
	if err == nil {
		t.Error("expected error for invalid region")
	}
}

func TestResolveRoute(t *testing.T) {
	svc := NewService(nil, nil)
	_, err := svc.RegisterNode(context.Background(), &CreateNodeRequest{
		Name: "node1", Region: RegionUSEast1, Endpoint: "https://e1.example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	route, err := svc.ResolveRoute(context.Background(), "tenant-1", "https://target.example.com", RegionUSEast1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if route.Region != RegionUSEast1 {
		t.Errorf("expected us-east-1, got %s", route.Region)
	}
}

func TestResolveRoute_NoNodes(t *testing.T) {
	svc := NewService(nil, nil)
	_, err := svc.ResolveRoute(context.Background(), "tenant-1", "https://target.example.com", RegionUSEast1)
	if err == nil {
		t.Error("expected error when no nodes available")
	}
}

func TestGetTopology(t *testing.T) {
	svc := NewService(nil, nil)
	svc.RegisterNode(context.Background(), &CreateNodeRequest{
		Name: "node1", Region: RegionUSEast1, Endpoint: "https://e1.example.com",
	})
	svc.RegisterNode(context.Background(), &CreateNodeRequest{
		Name: "node2", Region: RegionEUWest1, Endpoint: "https://e2.example.com",
	})

	topo, err := svc.GetTopology(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if topo.TotalNodes != 2 {
		t.Errorf("expected 2 nodes, got %d", topo.TotalNodes)
	}
}

func TestCreateRoutingRule(t *testing.T) {
	svc := NewService(nil, nil)
	rule, err := svc.CreateRoutingRule(context.Background(), "tenant-1", &CreateRoutingRuleRequest{
		Name:          "prefer-us",
		Strategy:      StrategyGeo,
		TargetRegions: []Region{RegionUSEast1, RegionUSWest2},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.ID == "" {
		t.Error("expected rule ID")
	}
}

func TestHaversineDistance(t *testing.T) {
	// NYC to London ≈ 5570 km
	dist := haversineDistance(40.7128, -74.0060, 51.5074, -0.1278)
	if dist < 5500 || dist > 5700 {
		t.Errorf("expected ~5570 km, got %.0f km", dist)
	}
}
