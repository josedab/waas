package deliveryreceipt

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service provides delivery receipt business logic.
type Service struct {
	repo Repository
}

// NewService creates a new delivery receipt service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateReceipt records a new delivery receipt when endpoint responds with 202.
func (s *Service) CreateReceipt(ctx context.Context, tenantID string, req *CreateReceiptRequest) (*DeliveryReceipt, error) {
	if req.ConfirmWindowSec <= 0 {
		req.ConfirmWindowSec = 300 // default 5 minutes
	}
	if req.ConfirmWindowSec > 86400 {
		return nil, fmt.Errorf("confirm_window_sec cannot exceed 86400")
	}

	status := StatusAccepted
	if req.HTTPStatus == 202 && req.ReceiptURL != "" {
		status = StatusPending
	} else if req.HTTPStatus >= 200 && req.HTTPStatus < 300 {
		status = StatusConfirmed
	} else {
		status = StatusFailed
	}

	now := time.Now()
	receipt := &DeliveryReceipt{
		ID:               uuid.New().String(),
		TenantID:         tenantID,
		DeliveryID:       req.DeliveryID,
		WebhookID:        req.WebhookID,
		EndpointID:       req.EndpointID,
		ReceiptURL:       req.ReceiptURL,
		Status:           status,
		HTTPStatus:       req.HTTPStatus,
		ConfirmWindowSec: req.ConfirmWindowSec,
		ExpiresAt:        now.Add(time.Duration(req.ConfirmWindowSec) * time.Second),
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if status == StatusConfirmed {
		receipt.ConfirmedAt = &now
		receipt.ProcessingStatus = ProcessingSuccess
	}

	if s.repo != nil {
		if err := s.repo.CreateReceipt(ctx, receipt); err != nil {
			return nil, fmt.Errorf("failed to create receipt: %w", err)
		}
	}

	return receipt, nil
}

// ConfirmReceipt processes a receipt confirmation from the receiver.
func (s *Service) ConfirmReceipt(ctx context.Context, tenantID, receiptID string, req *ConfirmReceiptRequest) (*DeliveryReceipt, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}

	receipt, err := s.repo.GetReceipt(ctx, tenantID, receiptID)
	if err != nil {
		return nil, err
	}

	if receipt.Status == StatusConfirmed {
		return nil, fmt.Errorf("receipt already confirmed")
	}
	if receipt.Status == StatusExpired {
		return nil, fmt.Errorf("receipt has expired")
	}

	if err := validateProcessingStatus(req.ProcessingStatus); err != nil {
		return nil, err
	}

	now := time.Now()
	receipt.Status = StatusConfirmed
	receipt.ProcessingStatus = req.ProcessingStatus
	receipt.ProcessingDetails = req.ProcessingDetails
	receipt.ConfirmedAt = &now
	receipt.UpdatedAt = now

	if req.ProcessingStatus == ProcessingFailed {
		receipt.Status = StatusFailed
	}

	if err := s.repo.UpdateReceipt(ctx, receipt); err != nil {
		return nil, fmt.Errorf("failed to confirm receipt: %w", err)
	}

	return receipt, nil
}

// GetReceipt retrieves a delivery receipt.
func (s *Service) GetReceipt(ctx context.Context, tenantID, receiptID string) (*DeliveryReceipt, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	return s.repo.GetReceipt(ctx, tenantID, receiptID)
}

// ListReceipts lists delivery receipts for a tenant.
func (s *Service) ListReceipts(ctx context.Context, tenantID string, limit, offset int) ([]DeliveryReceipt, error) {
	if s.repo == nil {
		return []DeliveryReceipt{}, nil
	}
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListReceipts(ctx, tenantID, limit, offset)
}

// GetStats returns receipt statistics for a tenant.
func (s *Service) GetStats(ctx context.Context, tenantID string) (*ReceiptStats, error) {
	if s.repo == nil {
		return &ReceiptStats{}, nil
	}
	return s.repo.GetStats(ctx, tenantID)
}

func validateProcessingStatus(status string) error {
	switch status {
	case ProcessingSuccess, ProcessingPartial, ProcessingFailed, ProcessingRetry:
		return nil
	}
	return fmt.Errorf("invalid processing_status %q: must be success, partial, failed, or retry_requested", status)
}
