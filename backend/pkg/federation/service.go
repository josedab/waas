package federation

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/httputil"
)

// Service provides federation operations
type Service struct {
	repo       Repository
	httpClient *http.Client
	selfMember *FederationMember
}

// NewService creates a new federation service
func NewService(repo Repository, selfMember *FederationMember) *Service {
	return &Service{
		repo:       repo,
		httpClient: httputil.NewSSRFSafeClient(30 * time.Second),
		selfMember: selfMember,
	}
}

// RegisterMember registers a new federation member
func (s *Service) RegisterMember(ctx context.Context, tenantID string, req *RegisterMemberRequest) (*FederationMember, error) {
	// Verify the domain if provided
	if req.Domain != "" {
		if err := s.verifyDomain(ctx, req.Domain); err != nil {
			return nil, fmt.Errorf("domain verification failed: %w", err)
		}
	}

	now := time.Now()
	member := &FederationMember{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		OrganizationID: req.OrganizationID,
		Name:           req.Name,
		Domain:         req.Domain,
		Status:         MemberPending,
		PublicKey:      req.PublicKey,
		Endpoints:      req.Endpoints,
		Capabilities:   req.Capabilities,
		TrustLevel:     TrustNone,
		Metadata:       req.Metadata,
		JoinedAt:       now,
		LastSeenAt:     now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.SaveMember(ctx, member); err != nil {
		return nil, fmt.Errorf("save member: %w", err)
	}

	// Store keys if provided
	if req.PublicKey != "" {
		keys := &CryptoKeys{
			MemberID:  member.ID,
			PublicKey: req.PublicKey,
			Algorithm: "ed25519",
			KeyID:     s.generateKeyID(req.PublicKey),
			CreatedAt: now,
		}
		s.repo.SaveKeys(ctx, keys)
	}

	return member, nil
}

// GetMember retrieves a member
func (s *Service) GetMember(ctx context.Context, memberID string) (*FederationMember, error) {
	return s.repo.GetMember(ctx, memberID)
}

// ListMembers lists members
func (s *Service) ListMembers(ctx context.Context, tenantID string, status *MemberStatus) ([]FederationMember, error) {
	return s.repo.ListMembers(ctx, tenantID, status)
}

// ActivateMember activates a member
func (s *Service) ActivateMember(ctx context.Context, memberID string) (*FederationMember, error) {
	member, err := s.repo.GetMember(ctx, memberID)
	if err != nil {
		return nil, err
	}

	member.Status = MemberActive
	member.UpdatedAt = time.Now()

	if err := s.repo.SaveMember(ctx, member); err != nil {
		return nil, fmt.Errorf("update member: %w", err)
	}

	return member, nil
}

// SuspendMember suspends a member
func (s *Service) SuspendMember(ctx context.Context, memberID string) (*FederationMember, error) {
	member, err := s.repo.GetMember(ctx, memberID)
	if err != nil {
		return nil, err
	}

	member.Status = MemberSuspended
	member.UpdatedAt = time.Now()

	if err := s.repo.SaveMember(ctx, member); err != nil {
		return nil, fmt.Errorf("update member: %w", err)
	}

	return member, nil
}

// RequestTrust initiates a trust request
func (s *Service) RequestTrust(ctx context.Context, tenantID string, req *CreateTrustRequest) (*TrustRequest, error) {
	trustReq := &TrustRequest{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		RequesterID:    req.RequesterID,
		TargetMemberID: req.TargetMemberID,
		RequestedLevel: req.RequestedLevel,
		Permissions:    req.Permissions,
		Message:        req.Message,
		Status:         TrustReqPending,
		CreatedAt:      time.Now(),
	}

	if req.ExpiresInDays > 0 {
		expiry := time.Now().AddDate(0, 0, req.ExpiresInDays)
		trustReq.ExpiresAt = &expiry
	}

	if err := s.repo.SaveTrustRequest(ctx, trustReq); err != nil {
		return nil, fmt.Errorf("save trust request: %w", err)
	}

	// Check if auto-accept is enabled
	policy, _ := s.repo.GetPolicy(ctx, tenantID)
	if policy != nil && policy.AutoAcceptTrust {
		return s.ApproveTrust(ctx, trustReq.ID, "Auto-approved by policy")
	}

	return trustReq, nil
}

// ApproveTrust approves a trust request
func (s *Service) ApproveTrust(ctx context.Context, reqID, response string) (*TrustRequest, error) {
	trustReq, err := s.repo.GetTrustRequest(ctx, reqID)
	if err != nil {
		return nil, err
	}

	if trustReq.Status != TrustReqPending {
		return nil, fmt.Errorf("trust request is not pending")
	}

	// Create trust relationship
	trust := &TrustRelationship{
		ID:             uuid.New().String(),
		TenantID:       trustReq.TenantID,
		SourceMemberID: trustReq.RequesterID,
		TargetMemberID: trustReq.TargetMemberID,
		Status:         TrustStatusActive,
		TrustLevel:     trustReq.RequestedLevel,
		Permissions:    trustReq.Permissions,
		ExpiresAt:      trustReq.ExpiresAt,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.repo.SaveTrustRelationship(ctx, trust); err != nil {
		return nil, fmt.Errorf("save trust relationship: %w", err)
	}

	// Update request status
	if err := s.repo.UpdateTrustRequestStatus(ctx, reqID, TrustReqApproved, response); err != nil {
		return nil, fmt.Errorf("update trust request: %w", err)
	}

	trustReq.Status = TrustReqApproved
	trustReq.Response = response
	now := time.Now()
	trustReq.RespondedAt = &now

	return trustReq, nil
}

// RejectTrust rejects a trust request
func (s *Service) RejectTrust(ctx context.Context, reqID, response string) error {
	trustReq, err := s.repo.GetTrustRequest(ctx, reqID)
	if err != nil {
		return err
	}

	if trustReq.Status != TrustReqPending {
		return fmt.Errorf("trust request is not pending")
	}

	return s.repo.UpdateTrustRequestStatus(ctx, reqID, TrustReqRejected, response)
}

// GetTrustRequests lists trust requests
func (s *Service) GetTrustRequests(ctx context.Context, tenantID string, status *TrustReqStatus) ([]TrustRequest, error) {
	return s.repo.ListTrustRequests(ctx, tenantID, status)
}

// GetTrustRelationships lists trust relationships
func (s *Service) GetTrustRelationships(ctx context.Context, tenantID, memberID string) ([]TrustRelationship, error) {
	return s.repo.ListTrustRelationships(ctx, tenantID, memberID)
}

// CreateCatalog creates an event catalog
func (s *Service) CreateCatalog(ctx context.Context, tenantID string, req *CreateCatalogRequest) (*EventCatalog, error) {
	catalog := &EventCatalog{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		MemberID:    req.MemberID,
		Name:        req.Name,
		Description: req.Description,
		EventTypes:  req.EventTypes,
		Version:     req.Version,
		Public:      req.Public,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if catalog.Version == "" {
		catalog.Version = "1.0.0"
	}

	if err := s.repo.SaveCatalog(ctx, catalog); err != nil {
		return nil, fmt.Errorf("save catalog: %w", err)
	}

	return catalog, nil
}

// GetCatalog retrieves a catalog
func (s *Service) GetCatalog(ctx context.Context, catalogID string) (*EventCatalog, error) {
	return s.repo.GetCatalog(ctx, catalogID)
}

// ListCatalogs lists catalogs
func (s *Service) ListCatalogs(ctx context.Context, tenantID string, public bool) ([]EventCatalog, error) {
	return s.repo.ListCatalogs(ctx, tenantID, public)
}

// DiscoverCatalogs discovers public catalogs
func (s *Service) DiscoverCatalogs(ctx context.Context) ([]EventCatalog, error) {
	return s.repo.ListPublicCatalogs(ctx)
}

// Subscribe creates a subscription
func (s *Service) Subscribe(ctx context.Context, tenantID string, req *CreateSubscriptionRequest) (*FederatedSubscription, error) {
	// Verify trust exists
	trust, err := s.repo.GetTrustBetween(ctx, req.SourceMemberID, req.TargetMemberID)
	if err != nil {
		return nil, fmt.Errorf("no trust relationship exists")
	}

	// Check permissions
	if !s.hasPermission(trust.Permissions, PermissionSubscribe, req.EventTypes) {
		return nil, fmt.Errorf("subscription not permitted")
	}

	sub := &FederatedSubscription{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		SourceMemberID: req.SourceMemberID,
		TargetMemberID: req.TargetMemberID,
		CatalogID:      req.CatalogID,
		EventTypes:     req.EventTypes,
		Filter:         req.Filter,
		Status:         SubStatusActive,
		DeliveryConfig: req.DeliveryConfig,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Set defaults
	if sub.DeliveryConfig.Timeout == 0 {
		sub.DeliveryConfig.Timeout = 30
	}
	if sub.DeliveryConfig.RetryPolicy.MaxRetries == 0 {
		sub.DeliveryConfig.RetryPolicy = RetryPolicy{
			MaxRetries:    5,
			InitialDelay:  5,
			MaxDelay:      300,
			BackoffFactor: 2.0,
		}
	}

	if err := s.repo.SaveSubscription(ctx, sub); err != nil {
		return nil, fmt.Errorf("save subscription: %w", err)
	}

	return sub, nil
}

// GetSubscription retrieves a subscription
func (s *Service) GetSubscription(ctx context.Context, subID string) (*FederatedSubscription, error) {
	return s.repo.GetSubscription(ctx, subID)
}

// ListSubscriptions lists subscriptions
func (s *Service) ListSubscriptions(ctx context.Context, tenantID string, status *SubStatus) ([]FederatedSubscription, error) {
	return s.repo.ListSubscriptions(ctx, tenantID, status)
}

// PauseSubscription pauses a subscription
func (s *Service) PauseSubscription(ctx context.Context, subID string) error {
	sub, err := s.repo.GetSubscription(ctx, subID)
	if err != nil {
		return err
	}

	sub.Status = SubStatusPaused
	sub.UpdatedAt = time.Now()

	return s.repo.SaveSubscription(ctx, sub)
}

// ResumeSubscription resumes a subscription
func (s *Service) ResumeSubscription(ctx context.Context, subID string) error {
	sub, err := s.repo.GetSubscription(ctx, subID)
	if err != nil {
		return err
	}

	sub.Status = SubStatusActive
	sub.UpdatedAt = time.Now()

	return s.repo.SaveSubscription(ctx, sub)
}

// PublishEvent publishes an event to federation
func (s *Service) PublishEvent(ctx context.Context, event *FederationEvent) error {
	// Find subscriptions for this event type
	subs, err := s.repo.ListSubscriptionsByMember(ctx, event.SourceMemberID)
	if err != nil {
		return fmt.Errorf("list subscriptions: %w", err)
	}

	for _, sub := range subs {
		if !s.matchesSubscription(&sub, event) {
			continue
		}

		delivery := &FederatedDelivery{
			ID:             uuid.New().String(),
			TenantID:       sub.TenantID,
			SubscriptionID: sub.ID,
			SourceMemberID: event.SourceMemberID,
			TargetMemberID: sub.TargetMemberID,
			EventType:      event.EventType,
			EventID:        event.EventID,
			Payload:        event.Payload,
			Status:         DeliveryPending,
			Attempts:       0,
			CreatedAt:      time.Now(),
		}

		if err := s.repo.SaveDelivery(ctx, delivery); err != nil {
			continue
		}

		// Deliver asynchronously
		go func(d *FederatedDelivery, sub *FederatedSubscription) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			s.deliverEvent(ctx, d, sub)
		}(delivery, &sub)
	}

	return nil
}

// deliverEvent delivers an event to the subscriber
func (s *Service) deliverEvent(ctx context.Context, delivery *FederatedDelivery, sub *FederatedSubscription) {
	startTime := time.Now()

	// Build request
	payload, _ := json.Marshal(map[string]any{
		"event_id":   delivery.EventID,
		"event_type": delivery.EventType,
		"source":     delivery.SourceMemberID,
		"payload":    delivery.Payload,
		"timestamp":  delivery.CreatedAt.Format(time.RFC3339),
	})

	req, err := http.NewRequestWithContext(ctx, sub.DeliveryConfig.Method, sub.DeliveryConfig.Endpoint, bytes.NewReader(payload))
	if err != nil {
		s.repo.UpdateDeliveryStatus(ctx, delivery.ID, DeliveryFailed, err.Error(), 0)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range sub.DeliveryConfig.Headers {
		req.Header.Set(k, v)
	}

	// Sign request if key is configured
	if sub.DeliveryConfig.SignatureKey != "" {
		signature := s.signPayload(payload, sub.DeliveryConfig.SignatureKey)
		req.Header.Set("X-Federation-Signature", signature)
	}

	// Execute request
	delivery.Status = DeliveryInFlight
	s.repo.SaveDelivery(ctx, delivery)

	resp, err := s.httpClient.Do(req)
	latency := time.Since(startTime).Milliseconds()
	delivery.Latency = latency

	if err != nil {
		s.handleDeliveryFailure(ctx, delivery, sub, err.Error(), 0)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	delivery.ResponseCode = resp.StatusCode
	delivery.ResponseBody = string(body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		now := time.Now()
		delivery.Status = DeliverySucceeded
		delivery.DeliveredAt = &now
		s.repo.SaveDelivery(ctx, delivery)
	} else {
		s.handleDeliveryFailure(ctx, delivery, sub, fmt.Sprintf("HTTP %d", resp.StatusCode), resp.StatusCode)
	}
}

// handleDeliveryFailure handles delivery failure and schedules retry
func (s *Service) handleDeliveryFailure(ctx context.Context, delivery *FederatedDelivery, sub *FederatedSubscription, errMsg string, respCode int) {
	delivery.Error = errMsg
	delivery.ResponseCode = respCode
	delivery.Attempts++

	if delivery.Attempts >= sub.DeliveryConfig.RetryPolicy.MaxRetries {
		delivery.Status = DeliveryFailed
	} else {
		delivery.Status = DeliveryRetrying
		delay := s.calculateRetryDelay(delivery.Attempts, &sub.DeliveryConfig.RetryPolicy)
		nextRetry := time.Now().Add(delay)
		delivery.NextRetryAt = &nextRetry
	}

	now := time.Now()
	delivery.LastAttemptAt = &now
	s.repo.SaveDelivery(ctx, delivery)
}

// calculateRetryDelay calculates retry delay with exponential backoff
func (s *Service) calculateRetryDelay(attempt int, policy *RetryPolicy) time.Duration {
	delay := float64(policy.InitialDelay)
	for i := 1; i < attempt; i++ {
		delay *= policy.BackoffFactor
		if delay > float64(policy.MaxDelay) {
			delay = float64(policy.MaxDelay)
			break
		}
	}
	return time.Duration(delay) * time.Second
}

// ProcessPendingDeliveries processes pending deliveries
func (s *Service) ProcessPendingDeliveries(ctx context.Context, limit int) error {
	deliveries, err := s.repo.ListPendingDeliveries(ctx, limit)
	if err != nil {
		return err
	}

	for _, delivery := range deliveries {
		sub, err := s.repo.GetSubscription(ctx, delivery.SubscriptionID)
		if err != nil {
			continue
		}

		go s.deliverEvent(ctx, &delivery, sub)
	}

	return nil
}

// GetPolicy retrieves federation policy
func (s *Service) GetPolicy(ctx context.Context, tenantID string) (*FederationPolicy, error) {
	return s.repo.GetPolicy(ctx, tenantID)
}

// UpdatePolicy updates federation policy
func (s *Service) UpdatePolicy(ctx context.Context, tenantID string, req *UpdatePolicyRequest) (*FederationPolicy, error) {
	policy, err := s.repo.GetPolicy(ctx, tenantID)
	if err != nil {
		// Create new policy
		policy = &FederationPolicy{
			ID:                 uuid.New().String(),
			TenantID:           tenantID,
			Enabled:            true,
			MinTrustLevel:      TrustBasic,
			MaxSubscriptions:   100,
			RateLimitPerMember: 1000,
			CreatedAt:          time.Now(),
		}
	}

	if req.Enabled != nil {
		policy.Enabled = *req.Enabled
	}
	if req.AutoAcceptTrust != nil {
		policy.AutoAcceptTrust = *req.AutoAcceptTrust
	}
	if req.MinTrustLevel != "" {
		policy.MinTrustLevel = req.MinTrustLevel
	}
	if req.AllowedDomains != nil {
		policy.AllowedDomains = req.AllowedDomains
	}
	if req.BlockedDomains != nil {
		policy.BlockedDomains = req.BlockedDomains
	}
	if req.RequireEncryption != nil {
		policy.RequireEncryption = *req.RequireEncryption
	}
	if req.AllowRelay != nil {
		policy.AllowRelay = *req.AllowRelay
	}
	if req.MaxSubscriptions > 0 {
		policy.MaxSubscriptions = req.MaxSubscriptions
	}
	if req.RateLimitPerMember > 0 {
		policy.RateLimitPerMember = req.RateLimitPerMember
	}
	policy.UpdatedAt = time.Now()

	if err := s.repo.SavePolicy(ctx, policy); err != nil {
		return nil, fmt.Errorf("save policy: %w", err)
	}

	return policy, nil
}

// GetMetrics retrieves federation metrics
func (s *Service) GetMetrics(ctx context.Context, tenantID string) (*FederationMetrics, error) {
	return s.repo.GetMetrics(ctx, tenantID)
}

// HealthCheck performs health check on a member
func (s *Service) HealthCheck(ctx context.Context, memberID string) (*HealthCheck, error) {
	member, err := s.repo.GetMember(ctx, memberID)
	if err != nil {
		return nil, err
	}

	health := &HealthCheck{
		MemberID:  memberID,
		CheckedAt: time.Now(),
	}

	// Find health endpoint
	var healthEndpoint string
	for _, ep := range member.Endpoints {
		healthEndpoint = ep.URL + "/health"
		break
	}

	if healthEndpoint == "" {
		health.Status = "unknown"
		return health, nil
	}

	// Make health check request
	start := time.Now()
	resp, err := s.httpClient.Get(healthEndpoint)
	health.Latency = time.Since(start).Milliseconds()

	if err != nil {
		health.Status = "unhealthy"
		health.Details = map[string]any{"error": err.Error()}
		return health, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		health.Status = "healthy"
	} else if resp.StatusCode >= 500 {
		health.Status = "unhealthy"
	} else {
		health.Status = "degraded"
	}

	// Update last seen
	member.LastSeenAt = time.Now()
	s.repo.SaveMember(ctx, member)

	return health, nil
}

// GenerateKeyPair generates a new key pair for federation
func (s *Service) GenerateKeyPair() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return &KeyPair{
		PublicKey:  base64.StdEncoding.EncodeToString(pub),
		PrivateKey: base64.StdEncoding.EncodeToString(priv),
		Algorithm:  "ed25519",
		KeyID:      s.generateKeyID(base64.StdEncoding.EncodeToString(pub)),
	}, nil
}

// Helper functions

func (s *Service) verifyDomain(ctx context.Context, domain string) error {
	// Would verify domain ownership via DNS TXT record or well-known endpoint
	return nil
}

func (s *Service) generateKeyID(publicKey string) string {
	hash := sha256.Sum256([]byte(publicKey))
	return hex.EncodeToString(hash[:8])
}

func (s *Service) hasPermission(perms []Permission, permType PermissionType, eventTypes []string) bool {
	for _, p := range perms {
		if p.Type == permType {
			if len(p.EventTypes) == 0 {
				return true // All event types allowed
			}
			for _, et := range eventTypes {
				for _, allowed := range p.EventTypes {
					if et == allowed || allowed == "*" {
						return true
					}
				}
			}
		}
	}
	return false
}

func (s *Service) matchesSubscription(sub *FederatedSubscription, event *FederationEvent) bool {
	// Check event type
	for _, et := range sub.EventTypes {
		if et == event.EventType || et == "*" || strings.HasSuffix(et, ".*") {
			prefix := strings.TrimSuffix(et, ".*")
			if strings.HasPrefix(event.EventType, prefix) {
				return s.matchesFilter(sub.Filter, event.Payload)
			}
		}
	}
	return false
}

func (s *Service) matchesFilter(filter *EventFilter, payload map[string]any) bool {
	if filter == nil || len(filter.Conditions) == 0 {
		return true
	}

	for _, cond := range filter.Conditions {
		val := s.getNestedValue(payload, cond.Field)
		if !s.evaluateCondition(val, cond.Operator, cond.Value) {
			return false
		}
	}
	return true
}

func (s *Service) getNestedValue(data map[string]any, path string) any {
	parts := strings.Split(path, ".")
	current := any(data)

	for _, part := range parts {
		if m, ok := current.(map[string]any); ok {
			current = m[part]
		} else {
			return nil
		}
	}
	return current
}

func (s *Service) evaluateCondition(value any, operator string, expected any) bool {
	switch operator {
	case "eq":
		return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", expected)
	case "ne":
		return fmt.Sprintf("%v", value) != fmt.Sprintf("%v", expected)
	case "contains":
		if s, ok := value.(string); ok {
			if e, ok := expected.(string); ok {
				return strings.Contains(s, e)
			}
		}
	}
	return false
}

func (s *Service) signPayload(payload []byte, key string) string {
	hash := sha256.Sum256(payload)
	return fmt.Sprintf("sha256=%x", hash)
}

// Request types

// RegisterMemberRequest for registering a member
type RegisterMemberRequest struct {
	OrganizationID string         `json:"organization_id" binding:"required"`
	Name           string         `json:"name" binding:"required"`
	Domain         string         `json:"domain"`
	PublicKey      string         `json:"public_key"`
	Endpoints      []FedEndpoint  `json:"endpoints"`
	Capabilities   []Capability   `json:"capabilities"`
	Metadata       map[string]any `json:"metadata"`
}

// CreateTrustRequest for creating a trust request
type CreateTrustRequest struct {
	RequesterID    string       `json:"requester_id" binding:"required"`
	TargetMemberID string       `json:"target_member_id" binding:"required"`
	RequestedLevel TrustLevel   `json:"requested_level"`
	Permissions    []Permission `json:"permissions"`
	Message        string       `json:"message"`
	ExpiresInDays  int          `json:"expires_in_days"`
}

// CreateCatalogRequest for creating a catalog
type CreateCatalogRequest struct {
	MemberID    string      `json:"member_id" binding:"required"`
	Name        string      `json:"name" binding:"required"`
	Description string      `json:"description"`
	EventTypes  []EventType `json:"event_types" binding:"required"`
	Version     string      `json:"version"`
	Public      bool        `json:"public"`
}

// CreateSubscriptionRequest for creating a subscription
type CreateSubscriptionRequest struct {
	SourceMemberID string         `json:"source_member_id" binding:"required"`
	TargetMemberID string         `json:"target_member_id" binding:"required"`
	CatalogID      string         `json:"catalog_id" binding:"required"`
	EventTypes     []string       `json:"event_types" binding:"required"`
	Filter         *EventFilter   `json:"filter"`
	DeliveryConfig DeliveryConfig `json:"delivery_config" binding:"required"`
}

// UpdatePolicyRequest for updating policy
type UpdatePolicyRequest struct {
	Enabled            *bool      `json:"enabled"`
	AutoAcceptTrust    *bool      `json:"auto_accept_trust"`
	MinTrustLevel      TrustLevel `json:"min_trust_level"`
	AllowedDomains     []string   `json:"allowed_domains"`
	BlockedDomains     []string   `json:"blocked_domains"`
	RequireEncryption  *bool      `json:"require_encryption"`
	AllowRelay         *bool      `json:"allow_relay"`
	MaxSubscriptions   int        `json:"max_subscriptions"`
	RateLimitPerMember int        `json:"rate_limit_per_member"`
}

// KeyPair generated key pair
type KeyPair struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
	Algorithm  string `json:"algorithm"`
	KeyID      string `json:"key_id"`
}
