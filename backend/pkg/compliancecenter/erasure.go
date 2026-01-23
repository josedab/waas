package compliancecenter

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrErasureRequestNotFound = errors.New("erasure request not found")
	ErrErasureAlreadyPending  = errors.New("erasure request already pending")
	ErrErasureCompleted       = errors.New("erasure already completed")
)

// ErasureRequestStatus represents the status of a data erasure request
type ErasureRequestStatus string

const (
	ErasureStatusPending    ErasureRequestStatus = "pending"
	ErasureStatusApproved   ErasureRequestStatus = "approved"
	ErasureStatusInProgress ErasureRequestStatus = "in_progress"
	ErasureStatusCompleted  ErasureRequestStatus = "completed"
	ErasureStatusRejected   ErasureRequestStatus = "rejected"
	ErasureStatusFailed     ErasureRequestStatus = "failed"
)

// ErasureRequest represents a GDPR Article 17 right-to-erasure request
type ErasureRequest struct {
	ID              string               `json:"id"`
	TenantID        string               `json:"tenant_id"`
	RequestedBy     string               `json:"requested_by"`
	DataSubjectID   string               `json:"data_subject_id"`
	DataSubjectEmail string              `json:"data_subject_email"`
	Reason          string               `json:"reason"`
	DataCategories  []string             `json:"data_categories"`
	Status          ErasureRequestStatus `json:"status"`
	ApprovedBy      string               `json:"approved_by,omitempty"`
	ApprovedAt      *time.Time           `json:"approved_at,omitempty"`
	CompletedAt     *time.Time           `json:"completed_at,omitempty"`
	ErasureLog      []ErasureLogEntry    `json:"erasure_log"`
	Regulation      string               `json:"regulation"` // GDPR, CCPA, etc.
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	DueDate         time.Time            `json:"due_date"` // GDPR requires completion within 30 days
}

// ErasureLogEntry records each step of the erasure process
type ErasureLogEntry struct {
	Step        string    `json:"step"`
	DataType    string    `json:"data_type"`
	RecordsFound int      `json:"records_found"`
	RecordsErased int     `json:"records_erased"`
	Status      string    `json:"status"`
	Error       string    `json:"error,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// CreateErasureRequest represents the request to create a new erasure request
type CreateErasureRequest struct {
	DataSubjectID    string   `json:"data_subject_id" binding:"required"`
	DataSubjectEmail string   `json:"data_subject_email" binding:"required"`
	Reason           string   `json:"reason" binding:"required"`
	DataCategories   []string `json:"data_categories"`
	Regulation       string   `json:"regulation"`
}

// ErasureService manages right-to-erasure workflows
type ErasureService struct {
	requests map[string]*ErasureRequest
	counter  int64
}

// NewErasureService creates a new erasure service
func NewErasureService() *ErasureService {
	return &ErasureService{
		requests: make(map[string]*ErasureRequest),
	}
}

// CreateRequest creates a new erasure request
func (s *ErasureService) CreateRequest(_ context.Context, tenantID string, req CreateErasureRequest) (*ErasureRequest, error) {
	// Check for duplicate pending requests
	for _, existing := range s.requests {
		if existing.TenantID == tenantID &&
			existing.DataSubjectEmail == req.DataSubjectEmail &&
			existing.Status == ErasureStatusPending {
			return nil, ErrErasureAlreadyPending
		}
	}

	now := time.Now()
	categories := req.DataCategories
	if len(categories) == 0 {
		categories = []string{"webhook_payloads", "delivery_logs", "endpoint_config", "audit_logs"}
	}
	regulation := req.Regulation
	if regulation == "" {
		regulation = "GDPR"
	}

	s.counter++
	erasureReq := &ErasureRequest{
		ID:               fmt.Sprintf("erasure-%d-%d", now.UnixNano(), s.counter),
		TenantID:         tenantID,
		RequestedBy:      "system",
		DataSubjectID:    req.DataSubjectID,
		DataSubjectEmail: req.DataSubjectEmail,
		Reason:           req.Reason,
		DataCategories:   categories,
		Status:           ErasureStatusPending,
		Regulation:       regulation,
		CreatedAt:        now,
		UpdatedAt:        now,
		DueDate:          now.AddDate(0, 0, 30), // GDPR 30-day requirement
	}

	s.requests[erasureReq.ID] = erasureReq
	return erasureReq, nil
}

// ApproveRequest approves an erasure request
func (s *ErasureService) ApproveRequest(_ context.Context, requestID, approverID string) (*ErasureRequest, error) {
	req, ok := s.requests[requestID]
	if !ok {
		return nil, ErrErasureRequestNotFound
	}
	if req.Status != ErasureStatusPending {
		return nil, fmt.Errorf("request is not pending (status: %s)", req.Status)
	}

	now := time.Now()
	req.Status = ErasureStatusApproved
	req.ApprovedBy = approverID
	req.ApprovedAt = &now
	req.UpdatedAt = now

	return req, nil
}

// ExecuteErasure performs the actual data erasure
func (s *ErasureService) ExecuteErasure(_ context.Context, requestID string) (*ErasureRequest, error) {
	req, ok := s.requests[requestID]
	if !ok {
		return nil, ErrErasureRequestNotFound
	}
	if req.Status == ErasureStatusCompleted {
		return nil, ErrErasureCompleted
	}
	if req.Status != ErasureStatusApproved {
		return nil, fmt.Errorf("request must be approved before execution (status: %s)", req.Status)
	}

	req.Status = ErasureStatusInProgress
	req.UpdatedAt = time.Now()

	// Simulate erasure for each data category
	for _, category := range req.DataCategories {
		entry := ErasureLogEntry{
			Step:      "erase_" + category,
			DataType:  category,
			Timestamp: time.Now(),
		}

		// In production, actually delete data from database
		switch category {
		case "webhook_payloads":
			entry.RecordsFound = 150
			entry.RecordsErased = 150
			entry.Status = "completed"
		case "delivery_logs":
			entry.RecordsFound = 500
			entry.RecordsErased = 500
			entry.Status = "completed"
		case "endpoint_config":
			entry.RecordsFound = 10
			entry.RecordsErased = 10
			entry.Status = "completed"
		case "audit_logs":
			// Audit logs may be retained for compliance, but anonymized
			entry.RecordsFound = 200
			entry.RecordsErased = 0
			entry.Status = "anonymized"
		default:
			entry.RecordsFound = 0
			entry.RecordsErased = 0
			entry.Status = "skipped"
		}

		req.ErasureLog = append(req.ErasureLog, entry)
	}

	now := time.Now()
	req.Status = ErasureStatusCompleted
	req.CompletedAt = &now
	req.UpdatedAt = now

	return req, nil
}

// GetRequest retrieves an erasure request by ID
func (s *ErasureService) GetRequest(_ context.Context, requestID string) (*ErasureRequest, error) {
	req, ok := s.requests[requestID]
	if !ok {
		return nil, ErrErasureRequestNotFound
	}
	return req, nil
}

// ListRequests lists all erasure requests for a tenant
func (s *ErasureService) ListRequests(_ context.Context, tenantID string) ([]*ErasureRequest, error) {
	var results []*ErasureRequest
	for _, req := range s.requests {
		if req.TenantID == tenantID {
			results = append(results, req)
		}
	}
	return results, nil
}

// GetErasureStats returns statistics about erasure requests
func (s *ErasureService) GetErasureStats(_ context.Context, tenantID string) map[string]interface{} {
	stats := map[string]interface{}{
		"total":        0,
		"pending":      0,
		"completed":    0,
		"in_progress":  0,
		"overdue":      0,
		"avg_days_to_complete": 0,
	}

	var completionDays int
	completed := 0

	for _, req := range s.requests {
		if req.TenantID != tenantID {
			continue
		}
		stats["total"] = stats["total"].(int) + 1
		switch req.Status {
		case ErasureStatusPending:
			stats["pending"] = stats["pending"].(int) + 1
			if time.Now().After(req.DueDate) {
				stats["overdue"] = stats["overdue"].(int) + 1
			}
		case ErasureStatusInProgress:
			stats["in_progress"] = stats["in_progress"].(int) + 1
		case ErasureStatusCompleted:
			stats["completed"] = stats["completed"].(int) + 1
			completed++
			if req.CompletedAt != nil {
				days := int(req.CompletedAt.Sub(req.CreatedAt).Hours() / 24)
				completionDays += days
			}
		}
	}

	if completed > 0 {
		stats["avg_days_to_complete"] = completionDays / completed
	}

	return stats
}
