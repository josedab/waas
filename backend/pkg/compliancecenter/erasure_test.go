package compliancecenter

import (
	"context"
	"errors"
	"testing"
)

func TestCreateErasureRequest(t *testing.T) {
	svc := NewErasureService()
	ctx := context.Background()

	req, err := svc.CreateRequest(ctx, "tenant-1", CreateErasureRequest{
		DataSubjectID:    "user-123",
		DataSubjectEmail: "user@example.com",
		Reason:           "GDPR right to be forgotten",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Status != ErasureStatusPending {
		t.Errorf("expected pending status, got %s", req.Status)
	}
	if req.Regulation != "GDPR" {
		t.Errorf("expected GDPR regulation, got %s", req.Regulation)
	}
	if len(req.DataCategories) < 3 {
		t.Error("expected default data categories")
	}
	if req.DueDate.IsZero() {
		t.Error("expected due date to be set")
	}
}

func TestDuplicateErasureRequest(t *testing.T) {
	svc := NewErasureService()
	ctx := context.Background()

	_, _ = svc.CreateRequest(ctx, "tenant-1", CreateErasureRequest{
		DataSubjectID:    "user-123",
		DataSubjectEmail: "user@example.com",
		Reason:           "test",
	})

	_, err := svc.CreateRequest(ctx, "tenant-1", CreateErasureRequest{
		DataSubjectID:    "user-123",
		DataSubjectEmail: "user@example.com",
		Reason:           "test duplicate",
	})

	if !errors.Is(err, ErrErasureAlreadyPending) {
		t.Errorf("expected ErrErasureAlreadyPending, got %v", err)
	}
}

func TestApproveAndExecuteErasure(t *testing.T) {
	svc := NewErasureService()
	ctx := context.Background()

	req, _ := svc.CreateRequest(ctx, "tenant-1", CreateErasureRequest{
		DataSubjectID:    "user-456",
		DataSubjectEmail: "delete@example.com",
		Reason:           "right to erasure",
	})

	// Cannot execute before approval
	_, err := svc.ExecuteErasure(ctx, req.ID)
	if err == nil {
		t.Error("expected error when executing unapproved request")
	}

	// Approve
	approved, err := svc.ApproveRequest(ctx, req.ID, "admin-1")
	if err != nil {
		t.Fatalf("unexpected error approving: %v", err)
	}
	if approved.Status != ErasureStatusApproved {
		t.Errorf("expected approved status, got %s", approved.Status)
	}

	// Execute
	completed, err := svc.ExecuteErasure(ctx, req.ID)
	if err != nil {
		t.Fatalf("unexpected error executing: %v", err)
	}
	if completed.Status != ErasureStatusCompleted {
		t.Errorf("expected completed status, got %s", completed.Status)
	}
	if completed.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
	if len(completed.ErasureLog) == 0 {
		t.Error("expected erasure log entries")
	}

	// Verify log has entries for each category
	for _, entry := range completed.ErasureLog {
		if entry.Status == "" {
			t.Errorf("erasure log entry for %s has empty status", entry.DataType)
		}
	}
}

func TestExecuteCompletedErasure(t *testing.T) {
	svc := NewErasureService()
	ctx := context.Background()

	req, _ := svc.CreateRequest(ctx, "tenant-1", CreateErasureRequest{
		DataSubjectID: "user-789", DataSubjectEmail: "done@example.com", Reason: "test",
	})
	_, _ = svc.ApproveRequest(ctx, req.ID, "admin")
	_, _ = svc.ExecuteErasure(ctx, req.ID)

	_, err := svc.ExecuteErasure(ctx, req.ID)
	if !errors.Is(err, ErrErasureCompleted) {
		t.Errorf("expected ErrErasureCompleted, got %v", err)
	}
}

func TestListErasureRequests(t *testing.T) {
	svc := NewErasureService()
	ctx := context.Background()

	r1, err1 := svc.CreateRequest(ctx, "tenant-1", CreateErasureRequest{
		DataSubjectID: "u1", DataSubjectEmail: "a@test.com", Reason: "test",
	})
	r2, err2 := svc.CreateRequest(ctx, "tenant-1", CreateErasureRequest{
		DataSubjectID: "u2", DataSubjectEmail: "b@test.com", Reason: "test",
	})
	_, _ = svc.CreateRequest(ctx, "tenant-2", CreateErasureRequest{
		DataSubjectID: "u3", DataSubjectEmail: "c@test.com", Reason: "test",
	})

	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected creation errors: %v, %v", err1, err2)
	}
	if r1.ID == r2.ID {
		t.Fatal("expected unique IDs for different requests")
	}

	list, _ := svc.ListRequests(ctx, "tenant-1")
	if len(list) != 2 {
		t.Errorf("expected 2 requests for tenant-1, got %d", len(list))
	}
}

func TestErasureStats(t *testing.T) {
	svc := NewErasureService()
	ctx := context.Background()

	req, _ := svc.CreateRequest(ctx, "tenant-1", CreateErasureRequest{
		DataSubjectID: "u1", DataSubjectEmail: "stats@test.com", Reason: "test",
	})
	_, _ = svc.ApproveRequest(ctx, req.ID, "admin")
	_, _ = svc.ExecuteErasure(ctx, req.ID)

	_, _ = svc.CreateRequest(ctx, "tenant-1", CreateErasureRequest{
		DataSubjectID: "u2", DataSubjectEmail: "stats2@test.com", Reason: "test",
	})

	stats := svc.GetErasureStats(ctx, "tenant-1")
	if stats["total"].(int) != 2 {
		t.Errorf("expected total 2, got %v", stats["total"])
	}
	if stats["completed"].(int) != 1 {
		t.Errorf("expected completed 1, got %v", stats["completed"])
	}
	if stats["pending"].(int) != 1 {
		t.Errorf("expected pending 1, got %v", stats["pending"])
	}
}
