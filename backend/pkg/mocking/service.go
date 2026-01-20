package mocking

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Service manages mock webhook operations
type Service struct {
	repo       Repository
	generator  *Generator
	httpClient *http.Client
	scheduler  *Scheduler
}

// NewService creates a new mocking service
func NewService(repo Repository) *Service {
	return &Service{
		repo:      repo,
		generator: NewGenerator(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		scheduler: NewScheduler(),
	}
}

// CreateMockEndpoint creates a new mock endpoint
func (s *Service) CreateMockEndpoint(ctx context.Context, tenantID string, req *CreateMockEndpointRequest) (*MockEndpoint, error) {
	endpoint := &MockEndpoint{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		URL:         req.URL,
		EventType:   req.EventType,
		Template:    req.Template,
		Schedule:    req.Schedule,
		Settings:    req.Settings,
		Metadata:    req.Metadata,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.CreateMockEndpoint(ctx, endpoint); err != nil {
		return nil, err
	}

	// Register with scheduler if has schedule
	if endpoint.Schedule != nil && endpoint.IsActive {
		s.scheduler.Register(endpoint)
	}

	return endpoint, nil
}

// GetMockEndpoint retrieves a mock endpoint by ID
func (s *Service) GetMockEndpoint(ctx context.Context, tenantID, endpointID string) (*MockEndpoint, error) {
	return s.repo.GetMockEndpoint(ctx, tenantID, endpointID)
}

// ListMockEndpoints lists mock endpoints for a tenant
func (s *Service) ListMockEndpoints(ctx context.Context, tenantID string, limit, offset int) ([]MockEndpoint, int, error) {
	return s.repo.ListMockEndpoints(ctx, tenantID, limit, offset)
}

// UpdateMockEndpoint updates a mock endpoint
func (s *Service) UpdateMockEndpoint(ctx context.Context, tenantID, endpointID string, req *UpdateMockEndpointRequest) (*MockEndpoint, error) {
	endpoint, err := s.repo.GetMockEndpoint(ctx, tenantID, endpointID)
	if err != nil {
		return nil, err
	}
	if endpoint == nil {
		return nil, fmt.Errorf("mock endpoint not found")
	}

	if req.Name != nil {
		endpoint.Name = *req.Name
	}
	if req.Description != nil {
		endpoint.Description = *req.Description
	}
	if req.URL != nil {
		endpoint.URL = *req.URL
	}
	if req.EventType != nil {
		endpoint.EventType = *req.EventType
	}
	if req.Template != nil {
		endpoint.Template = req.Template
	}
	if req.Schedule != nil {
		endpoint.Schedule = req.Schedule
	}
	if req.Settings != nil {
		endpoint.Settings = *req.Settings
	}
	if req.IsActive != nil {
		endpoint.IsActive = *req.IsActive
	}
	if req.Metadata != nil {
		endpoint.Metadata = req.Metadata
	}

	endpoint.UpdatedAt = time.Now()

	if err := s.repo.UpdateMockEndpoint(ctx, endpoint); err != nil {
		return nil, err
	}

	// Update scheduler
	if endpoint.IsActive && endpoint.Schedule != nil {
		s.scheduler.Register(endpoint)
	} else {
		s.scheduler.Unregister(endpoint.ID)
	}

	return endpoint, nil
}

// DeleteMockEndpoint deletes a mock endpoint
func (s *Service) DeleteMockEndpoint(ctx context.Context, tenantID, endpointID string) error {
	s.scheduler.Unregister(endpointID)
	return s.repo.DeleteMockEndpoint(ctx, tenantID, endpointID)
}

// TriggerMock triggers mock webhook deliveries
func (s *Service) TriggerMock(ctx context.Context, tenantID, endpointID string, req *TriggerMockRequest) ([]MockDelivery, error) {
	endpoint, err := s.repo.GetMockEndpoint(ctx, tenantID, endpointID)
	if err != nil {
		return nil, err
	}
	if endpoint == nil {
		return nil, fmt.Errorf("mock endpoint not found")
	}

	count := req.Count
	if count <= 0 {
		count = 1
	}
	if count > 100 {
		count = 100 // Limit to prevent abuse
	}

	var deliveries []MockDelivery

	for i := 0; i < count; i++ {
		// Generate payload
		var payload map[string]interface{}
		if req.Payload != nil {
			payload = req.Payload
		} else {
			payload, err = s.generator.GeneratePayload(endpoint.Template)
			if err != nil {
				return nil, fmt.Errorf("failed to generate payload: %w", err)
			}
		}

		// Add standard fields
		payload["event_type"] = endpoint.EventType
		payload["timestamp"] = time.Now().Format(time.RFC3339)
		payload["id"] = uuid.New().String()

		delivery := s.sendMock(ctx, endpoint, payload, req.DelayMs)
		deliveries = append(deliveries, delivery)

		// Delay between multiple
		if i < count-1 && req.Interval != "" {
			if d, err := time.ParseDuration(req.Interval); err == nil {
				time.Sleep(d)
			}
		}
	}

	return deliveries, nil
}

// sendMock sends a mock webhook
func (s *Service) sendMock(ctx context.Context, endpoint *MockEndpoint, payload map[string]interface{}, delayMs int) MockDelivery {
	delivery := MockDelivery{
		ID:         uuid.New().String(),
		EndpointID: endpoint.ID,
		TenantID:   endpoint.TenantID,
		Payload:    payload,
		Headers:    make(map[string]string),
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	// Apply delay
	if delayMs > 0 {
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	} else if endpoint.Settings.DelayMs > 0 {
		time.Sleep(time.Duration(endpoint.Settings.DelayMs) * time.Millisecond)
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint.URL, bytes.NewReader(body))
	if err != nil {
		delivery.Status = "failed"
		delivery.Error = err.Error()
		s.repo.CreateMockDelivery(ctx, &delivery)
		return delivery
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Mock-Event-ID", delivery.ID)
	req.Header.Set("X-Mock-Event-Type", endpoint.EventType)
	req.Header.Set("X-Mock-Timestamp", time.Now().Format(time.RFC3339))

	for k, v := range endpoint.Settings.Headers {
		req.Header.Set(k, v)
		delivery.Headers[k] = v
	}

	// Add signature if configured
	if endpoint.Settings.Signature && endpoint.Settings.SignatureKey != "" {
		sig := computeSignature(body, endpoint.Settings.SignatureKey)
		req.Header.Set("X-Mock-Signature", sig)
		delivery.Headers["X-Mock-Signature"] = sig
	}

	now := time.Now()
	delivery.SentAt = &now

	start := time.Now()
	resp, err := s.httpClient.Do(req)
	delivery.LatencyMs = int(time.Since(start).Milliseconds())

	if err != nil {
		delivery.Status = "failed"
		delivery.Error = err.Error()
	} else {
		delivery.StatusCode = resp.StatusCode
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			delivery.Status = "delivered"
		} else {
			delivery.Status = "failed"
			delivery.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
	}

	s.repo.CreateMockDelivery(ctx, &delivery)
	return delivery
}

// GeneratePreview generates a preview of mock data
func (s *Service) GeneratePreview(template *PayloadTemplate, count int) ([]map[string]interface{}, error) {
	if count <= 0 {
		count = 1
	}
	if count > 10 {
		count = 10
	}

	var previews []map[string]interface{}
	for i := 0; i < count; i++ {
		payload, err := s.generator.GeneratePayload(template)
		if err != nil {
			return nil, err
		}
		previews = append(previews, payload)
	}

	return previews, nil
}

// ListDeliveries lists mock deliveries
func (s *Service) ListDeliveries(ctx context.Context, tenantID, endpointID string, limit, offset int) ([]MockDelivery, int, error) {
	return s.repo.ListMockDeliveries(ctx, tenantID, endpointID, limit, offset)
}

// Template operations

// CreateTemplate creates a mock template
func (s *Service) CreateTemplate(ctx context.Context, tenantID string, req *CreateTemplateRequest) (*MockTemplate, error) {
	template := &MockTemplate{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		EventType:   req.EventType,
		Category:    req.Category,
		Template:    req.Template,
		Examples:    req.Examples,
		IsPublic:    req.IsPublic,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.CreateTemplate(ctx, template); err != nil {
		return nil, err
	}

	return template, nil
}

// GetTemplate retrieves a template by ID
func (s *Service) GetTemplate(ctx context.Context, tenantID, templateID string) (*MockTemplate, error) {
	return s.repo.GetTemplate(ctx, tenantID, templateID)
}

// ListTemplates lists templates
func (s *Service) ListTemplates(ctx context.Context, tenantID string, includePublic bool, limit, offset int) ([]MockTemplate, int, error) {
	return s.repo.ListTemplates(ctx, tenantID, includePublic, limit, offset)
}

// DeleteTemplate deletes a template
func (s *Service) DeleteTemplate(ctx context.Context, tenantID, templateID string) error {
	return s.repo.DeleteTemplate(ctx, tenantID, templateID)
}

// GetFakerTypes returns available faker types
func (s *Service) GetFakerTypes() []FakerType {
	return GetAvailableFakerTypes()
}

func computeSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// Scheduler manages scheduled mock deliveries
type Scheduler struct {
	endpoints map[string]*MockEndpoint
	mu        sync.RWMutex
	stopCh    chan struct{}
}

// NewScheduler creates a new scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		endpoints: make(map[string]*MockEndpoint),
		stopCh:    make(chan struct{}),
	}
}

// Register registers an endpoint for scheduling
func (s *Scheduler) Register(endpoint *MockEndpoint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.endpoints[endpoint.ID] = endpoint
}

// Unregister removes an endpoint from scheduling
func (s *Scheduler) Unregister(endpointID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.endpoints, endpointID)
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context, service *Service) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.processSchedules(ctx, service)
		}
	}
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	close(s.stopCh)
}

func (s *Scheduler) processSchedules(ctx context.Context, service *Service) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, endpoint := range s.endpoints {
		if endpoint.Schedule == nil || !endpoint.IsActive {
			continue
		}

		// Check if should run based on schedule
		if s.shouldRun(endpoint) {
			go func(ep *MockEndpoint) {
				_, err := service.TriggerMock(ctx, ep.TenantID, ep.ID, &TriggerMockRequest{Count: 1})
				if err != nil {
					log.Printf("[mocking] scheduled trigger failed for %s: %v", ep.ID, err)
				}
			}(endpoint)
		}
	}
}

func (s *Scheduler) shouldRun(endpoint *MockEndpoint) bool {
	schedule := endpoint.Schedule
	if schedule == nil {
		return false
	}

	// Check max runs
	if schedule.MaxRuns > 0 && schedule.RunCount >= schedule.MaxRuns {
		return false
	}

	// Check time bounds
	now := time.Now()
	if schedule.StartAt != nil && now.Before(*schedule.StartAt) {
		return false
	}
	if schedule.EndAt != nil && now.After(*schedule.EndAt) {
		return false
	}

	// Type-specific logic would go here (cron parsing, interval checking)
	return false // Simplified - would need proper schedule evaluation
}
