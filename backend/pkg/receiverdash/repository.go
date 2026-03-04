package receiverdash

import (
	"crypto/subtle"
	"fmt"
	"time"
)

// Repository defines the data access interface for receiver dashboard.
type Repository interface {
	// Token management
	CreateToken(token *ReceiverToken) error
	GetToken(tokenID string) (*ReceiverToken, error)
	GetTokenByValue(token string) (*ReceiverToken, error)
	ListTokens(tenantID string) ([]*ReceiverToken, error)
	RevokeToken(tokenID string) error

	// Delivery history
	GetDeliveryHistory(endpointIDs []string, req *DeliveryHistoryRequest) ([]*DeliveryRecord, int, error)
	GetDeliveryPayload(deliveryID string) (*PayloadInspection, error)

	// Retry status
	GetRetryStatus(endpointIDs []string, activeOnly bool) ([]*RetryStatus, error)
	GetRetryStatusByDelivery(deliveryID string) (*RetryStatus, error)

	// Health metrics
	GetEndpointHealth(endpointID string, period string) (*EndpointHealth, error)
	GetHealthSummary(endpointIDs []string, period string) (*HealthSummary, error)
}

// MemoryRepository provides an in-memory implementation for development/testing.
type MemoryRepository struct {
	tokens     map[string]*ReceiverToken
	deliveries map[string]*DeliveryRecord
	retries    map[string]*RetryStatus
	payloads   map[string]*PayloadInspection
}

// NewMemoryRepository creates a new in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		tokens:     make(map[string]*ReceiverToken),
		deliveries: make(map[string]*DeliveryRecord),
		retries:    make(map[string]*RetryStatus),
		payloads:   make(map[string]*PayloadInspection),
	}
}

func (r *MemoryRepository) CreateToken(token *ReceiverToken) error {
	r.tokens[token.ID] = token
	return nil
}

func (r *MemoryRepository) GetToken(tokenID string) (*ReceiverToken, error) {
	if t, ok := r.tokens[tokenID]; ok {
		return t, nil
	}
	return nil, fmt.Errorf("token not found: %s", tokenID)
}

func (r *MemoryRepository) GetTokenByValue(token string) (*ReceiverToken, error) {
	for _, t := range r.tokens {
		if subtle.ConstantTimeCompare([]byte(t.Token), []byte(token)) == 1 && t.RevokedAt == nil {
			return t, nil
		}
	}
	return nil, fmt.Errorf("token not found or revoked")
}

func (r *MemoryRepository) ListTokens(tenantID string) ([]*ReceiverToken, error) {
	var result []*ReceiverToken
	for _, t := range r.tokens {
		if t.TenantID == tenantID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (r *MemoryRepository) RevokeToken(tokenID string) error {
	if t, ok := r.tokens[tokenID]; ok {
		now := time.Now()
		t.RevokedAt = &now
		return nil
	}
	return fmt.Errorf("token not found: %s", tokenID)
}

func (r *MemoryRepository) GetDeliveryHistory(endpointIDs []string, req *DeliveryHistoryRequest) ([]*DeliveryRecord, int, error) {
	epSet := make(map[string]bool)
	for _, id := range endpointIDs {
		epSet[id] = true
	}

	var results []*DeliveryRecord
	for _, d := range r.deliveries {
		if !epSet[d.EndpointID] {
			continue
		}
		if req.EndpointID != "" && d.EndpointID != req.EndpointID {
			continue
		}
		if req.EventType != "" && d.EventType != req.EventType {
			continue
		}
		if req.Status == "success" && !d.Success {
			continue
		}
		if req.Status == "failed" && d.Success {
			continue
		}
		results = append(results, d)
	}

	total := len(results)
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := req.Offset
	if offset > len(results) {
		offset = len(results)
	}
	end := offset + limit
	if end > len(results) {
		end = len(results)
	}
	return results[offset:end], total, nil
}

func (r *MemoryRepository) GetDeliveryPayload(deliveryID string) (*PayloadInspection, error) {
	if p, ok := r.payloads[deliveryID]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("payload not found: %s", deliveryID)
}

func (r *MemoryRepository) GetRetryStatus(endpointIDs []string, activeOnly bool) ([]*RetryStatus, error) {
	epSet := make(map[string]bool)
	for _, id := range endpointIDs {
		epSet[id] = true
	}

	var results []*RetryStatus
	for _, rs := range r.retries {
		if !epSet[rs.EndpointID] {
			continue
		}
		if activeOnly && (rs.CurrentState == "succeeded" || rs.CurrentState == "exhausted") {
			continue
		}
		results = append(results, rs)
	}
	return results, nil
}

func (r *MemoryRepository) GetRetryStatusByDelivery(deliveryID string) (*RetryStatus, error) {
	if rs, ok := r.retries[deliveryID]; ok {
		return rs, nil
	}
	return nil, fmt.Errorf("retry status not found: %s", deliveryID)
}

func (r *MemoryRepository) GetEndpointHealth(endpointID string, period string) (*EndpointHealth, error) {
	return &EndpointHealth{
		EndpointID:       endpointID,
		HealthScore:      95.0,
		SuccessRate:      98.5,
		AvgLatencyMs:     120.0,
		P95LatencyMs:     350.0,
		TotalDeliveries:  1000,
		FailedDeliveries: 15,
		ActiveRetries:    3,
		Period:           period,
	}, nil
}

func (r *MemoryRepository) GetHealthSummary(endpointIDs []string, period string) (*HealthSummary, error) {
	var endpoints []EndpointHealth
	for _, id := range endpointIDs {
		h, _ := r.GetEndpointHealth(id, period)
		endpoints = append(endpoints, *h)
	}
	return &HealthSummary{
		Endpoints:       endpoints,
		OverallScore:    95.0,
		OverallSuccess:  98.5,
		TotalDeliveries: len(endpoints) * 1000,
		TotalFailed:     len(endpoints) * 15,
		ActiveRetries:   len(endpoints) * 3,
		Period:          period,
	}, nil
}
