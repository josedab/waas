package waf

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// QuarantineWebhook quarantines a webhook delivery
func (s *Service) QuarantineWebhook(ctx context.Context, tenantID, webhookID, reason string, threats []Threat, payload json.RawMessage) error {
	quarantine := &QuarantinedWebhook{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		WebhookID:       webhookID,
		Reason:          reason,
		Threats:         threats,
		OriginalPayload: payload,
		QuarantinedAt:   time.Now(),
	}

	return s.repo.CreateQuarantine(ctx, quarantine)
}

// ReviewQuarantine reviews a quarantined webhook (approve or reject)
func (s *Service) ReviewQuarantine(ctx context.Context, tenantID, quarantineID, reviewedBy string, req *ReviewQuarantineRequest) (*QuarantinedWebhook, error) {
	quarantine, err := s.repo.GetQuarantine(ctx, tenantID, quarantineID)
	if err != nil {
		return nil, err
	}

	if quarantine.ReviewedAt != nil {
		return nil, ErrAlreadyReviewed
	}

	now := time.Now()
	quarantine.ReviewedAt = &now
	quarantine.ReviewedBy = reviewedBy
	quarantine.Decision = req.Decision

	if err := s.repo.UpdateQuarantine(ctx, quarantine); err != nil {
		return nil, fmt.Errorf("failed to update quarantine: %w", err)
	}

	return quarantine, nil
}

// ListQuarantined lists quarantined webhooks
func (s *Service) ListQuarantined(ctx context.Context, tenantID string, limit, offset int) ([]QuarantinedWebhook, int, error) {
	return s.repo.ListQuarantined(ctx, tenantID, limit, offset)
}
