package piidetection

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service provides PII detection business logic.
type Service struct {
	repo Repository
}

// NewService creates a new PII detection service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreatePolicy creates a new PII detection policy for a tenant.
func (s *Service) CreatePolicy(ctx context.Context, tenantID string, req *CreatePolicyRequest) (*Policy, error) {
	if err := validateSensitivity(req.Sensitivity); err != nil {
		return nil, err
	}
	if err := validateMaskingAction(req.MaskingAction); err != nil {
		return nil, err
	}
	if len(req.Categories) == 0 {
		return nil, fmt.Errorf("at least one PII category is required")
	}
	for _, cat := range req.Categories {
		if err := validateCategory(cat); err != nil {
			return nil, err
		}
	}

	now := time.Now()
	policy := &Policy{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		Name:           req.Name,
		Description:    req.Description,
		Sensitivity:    req.Sensitivity,
		Categories:     req.Categories,
		CustomPatterns: req.CustomPatterns,
		MaskingAction:  req.MaskingAction,
		IsEnabled:      true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if s.repo != nil {
		if err := s.repo.CreatePolicy(ctx, policy); err != nil {
			return nil, fmt.Errorf("failed to create PII policy: %w", err)
		}
	}

	return policy, nil
}

// GetPolicy retrieves a PII detection policy.
func (s *Service) GetPolicy(ctx context.Context, tenantID, policyID string) (*Policy, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	return s.repo.GetPolicy(ctx, tenantID, policyID)
}

// ListPolicies lists all PII detection policies for a tenant.
func (s *Service) ListPolicies(ctx context.Context, tenantID string) ([]Policy, error) {
	if s.repo == nil {
		return []Policy{}, nil
	}
	return s.repo.ListPolicies(ctx, tenantID)
}

// UpdatePolicy updates an existing PII detection policy.
func (s *Service) UpdatePolicy(ctx context.Context, tenantID, policyID string, req *UpdatePolicyRequest) (*Policy, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}

	policy, err := s.repo.GetPolicy(ctx, tenantID, policyID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		policy.Name = *req.Name
	}
	if req.Description != nil {
		policy.Description = *req.Description
	}
	if req.Sensitivity != nil {
		if err := validateSensitivity(*req.Sensitivity); err != nil {
			return nil, err
		}
		policy.Sensitivity = *req.Sensitivity
	}
	if req.Categories != nil {
		for _, cat := range req.Categories {
			if err := validateCategory(cat); err != nil {
				return nil, err
			}
		}
		policy.Categories = req.Categories
	}
	if req.CustomPatterns != nil {
		policy.CustomPatterns = req.CustomPatterns
	}
	if req.MaskingAction != nil {
		if err := validateMaskingAction(*req.MaskingAction); err != nil {
			return nil, err
		}
		policy.MaskingAction = *req.MaskingAction
	}
	if req.IsEnabled != nil {
		policy.IsEnabled = *req.IsEnabled
	}
	policy.UpdatedAt = time.Now()

	if err := s.repo.UpdatePolicy(ctx, policy); err != nil {
		return nil, fmt.Errorf("failed to update PII policy: %w", err)
	}

	return policy, nil
}

// DeletePolicy removes a PII detection policy.
func (s *Service) DeletePolicy(ctx context.Context, tenantID, policyID string) error {
	if s.repo == nil {
		return fmt.Errorf("repository not configured")
	}
	return s.repo.DeletePolicy(ctx, tenantID, policyID)
}

// ScanPayload scans a webhook payload for PII using all enabled policies for
// the tenant, masks detected fields, and stores the result in the compliance vault.
func (s *Service) ScanPayload(ctx context.Context, tenantID string, req *ScanRequest) (*ScanResponse, error) {
	var policies []Policy
	if s.repo != nil {
		var err error
		policies, err = s.repo.GetEnabledPolicies(ctx, tenantID)
		if err != nil {
			return nil, fmt.Errorf("failed to load PII policies: %w", err)
		}
	}

	if len(policies) == 0 {
		return &ScanResponse{
			MaskedPayload: req.Payload,
			Result: &ScanResult{
				ID:            uuid.New().String(),
				TenantID:      tenantID,
				WebhookID:     req.WebhookID,
				EndpointID:    req.EndpointID,
				EventType:     req.EventType,
				MaskingAction: ActionMask,
				OriginalHash:  hashPayload(req.Payload),
				MaskedHash:    hashPayload(req.Payload),
				CreatedAt:     time.Now(),
			},
		}, nil
	}

	start := time.Now()
	maskedPayload := req.Payload
	var allDetections []Detection
	totalScanned := 0
	totalMasked := 0
	policyID := policies[0].ID
	maskingAction := policies[0].MaskingAction

	for _, policy := range policies {
		detector := NewDetector(&policy)
		var detections []Detection
		var scanned, masked int
		maskedPayload, detections, scanned, masked = detector.ScanAndMask(maskedPayload)
		allDetections = append(allDetections, detections...)
		totalScanned += scanned
		totalMasked += masked
	}

	result := &ScanResult{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		PolicyID:       policyID,
		WebhookID:      req.WebhookID,
		EndpointID:     req.EndpointID,
		EventType:      req.EventType,
		Detections:     allDetections,
		FieldsScanned:  totalScanned,
		FieldsMasked:   totalMasked,
		MaskingAction:  maskingAction,
		OriginalHash:   hashPayload(req.Payload),
		MaskedHash:     hashPayload(maskedPayload),
		ScanDurationMs: elapsedMs(start),
		CreatedAt:      time.Now(),
	}

	if s.repo != nil {
		_ = s.repo.StoreScanResult(ctx, result)
	}

	return &ScanResponse{
		MaskedPayload: maskedPayload,
		Result:        result,
	}, nil
}

// GetScanResult retrieves a single scan result.
func (s *Service) GetScanResult(ctx context.Context, tenantID, resultID string) (*ScanResult, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not configured")
	}
	return s.repo.GetScanResult(ctx, tenantID, resultID)
}

// ListScanResults lists scan results for a tenant.
func (s *Service) ListScanResults(ctx context.Context, tenantID string, limit, offset int) ([]ScanResult, error) {
	if s.repo == nil {
		return []ScanResult{}, nil
	}
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListScanResults(ctx, tenantID, limit, offset)
}

// GetDashboardStats returns PII detection metrics for a tenant.
func (s *Service) GetDashboardStats(ctx context.Context, tenantID string) (*DashboardStats, error) {
	if s.repo == nil {
		return &DashboardStats{}, nil
	}
	return s.repo.GetDashboardStats(ctx, tenantID)
}

func validateSensitivity(v string) error {
	switch v {
	case SensitivityLow, SensitivityMedium, SensitivityHigh, SensitivityCritical:
		return nil
	}
	return fmt.Errorf("invalid sensitivity %q: must be low, medium, high, or critical", v)
}

func validateMaskingAction(v string) error {
	switch v {
	case ActionMask, ActionRedact, ActionHash, ActionTokenize:
		return nil
	}
	return fmt.Errorf("invalid masking_action %q: must be mask, redact, hash, or tokenize", v)
}

func validateCategory(v string) error {
	switch v {
	case CategoryEmail, CategoryPhone, CategorySSN, CategoryCreditCard,
		CategoryName, CategoryAddress, CategoryDOB, CategoryIPAddress, CategoryCustom:
		return nil
	}
	return fmt.Errorf("invalid PII category %q", v)
}
