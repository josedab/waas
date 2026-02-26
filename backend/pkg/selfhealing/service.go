package selfhealing

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

// Service implements the self-healing endpoint discovery logic.
type Service struct {
	repo   Repository
	config *ServiceConfig
	client *http.Client
	logger *utils.Logger
}

// NewService creates a new self-healing service.
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	return &Service{
		repo:   repo,
		config: config,
		client: &http.Client{
			Timeout: config.ValidationTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // don't follow redirects automatically
			},
		},
		logger: utils.NewLogger("selfhealing-service"),
	}
}

// RecordFailure tracks a delivery failure and triggers healing if threshold is met.
func (s *Service) RecordFailure(tenantID, endpointID, currentURL string) (*EndpointDiscovery, error) {
	ft, err := s.repo.GetFailureTracker(endpointID)
	if err != nil {
		return nil, err
	}

	ft.ConsecutiveFailures++
	ft.LastFailureAt = time.Now()

	if err := s.repo.UpsertFailureTracker(ft); err != nil {
		return nil, err
	}

	if ft.ConsecutiveFailures >= s.config.FailureThreshold && !ft.HealingTriggered {
		ft.HealingTriggered = true
		if err := s.repo.UpsertFailureTracker(ft); err != nil {
			s.logger.Error("failed to upsert failure tracker", map[string]interface{}{"error": err.Error(), "endpoint_id": endpointID})
		}
		return s.DiscoverNewURL(tenantID, endpointID, currentURL)
	}

	return nil, nil
}

// RecordSuccess resets the failure tracker on a successful delivery.
func (s *Service) RecordSuccess(endpointID string) error {
	return s.repo.ResetFailureTracker(endpointID)
}

// DiscoverNewURL attempts to find a new URL for a failing endpoint.
func (s *Service) DiscoverNewURL(tenantID, endpointID, currentURL string) (*EndpointDiscovery, error) {
	// Try .well-known first
	if newURL, err := s.checkWellKnown(currentURL); err == nil && newURL != "" {
		return s.createDiscovery(tenantID, endpointID, currentURL, newURL, "well-known")
	}

	// Try HTTP redirect detection
	if newURL, err := s.checkHTTPRedirect(currentURL); err == nil && newURL != "" {
		return s.createDiscovery(tenantID, endpointID, currentURL, newURL, "http-redirect")
	}

	// Try DNS TXT lookup
	if newURL, err := s.checkDNSTXT(currentURL); err == nil && newURL != "" {
		return s.createDiscovery(tenantID, endpointID, currentURL, newURL, "dns-txt")
	}

	return nil, fmt.Errorf("no alternative URL discovered for endpoint %s", endpointID)
}

// ValidateAndApply tests a discovered URL and applies it if valid.
func (s *Service) ValidateAndApply(discoveryID string) (*EndpointDiscovery, error) {
	discovery, err := s.repo.GetDiscovery(discoveryID)
	if err != nil {
		return nil, err
	}

	// Test the new URL
	result := s.testURL(discovery.DiscoveredURL)
	discovery.TestResult = result

	if result.Success {
		discovery.Status = "validated"
		now := time.Now()
		discovery.AppliedAt = &now
		discovery.Status = "applied"

		// Emit migration event
		s.emitMigrationEvent(discovery)

		// Reset failure tracker
		if err := s.repo.ResetFailureTracker(discovery.EndpointID); err != nil {
			s.logger.Error("failed to reset failure tracker", map[string]interface{}{"error": err.Error(), "endpoint_id": discovery.EndpointID})
		}
	} else {
		discovery.Status = "rejected"
	}

	if err := s.repo.UpdateDiscovery(discovery); err != nil {
		return nil, err
	}
	return discovery, nil
}

// GetDiscoveries returns all discoveries for an endpoint.
func (s *Service) GetDiscoveries(endpointID string) ([]*EndpointDiscovery, error) {
	return s.repo.ListDiscoveries(endpointID)
}

// GetMigrationEvents returns recent migration events.
func (s *Service) GetMigrationEvents(tenantID string, limit int) ([]*MigrationEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListMigrationEvents(tenantID, limit)
}

// GetFailureStatus returns the current failure status for an endpoint.
func (s *Service) GetFailureStatus(endpointID string) (*FailureTracker, error) {
	return s.repo.GetFailureTracker(endpointID)
}

// GenerateWellKnownSpec generates a .well-known/waas-webhooks document.
func (s *Service) GenerateWellKnownSpec(endpoints []WellKnownEndpoint) *WellKnownSpec {
	return &WellKnownSpec{
		Version:   "1.0",
		Endpoints: endpoints,
		UpdatedAt: time.Now().Format(time.RFC3339),
	}
}

func (s *Service) createDiscovery(tenantID, endpointID, originalURL, newURL, method string) (*EndpointDiscovery, error) {
	discovery := &EndpointDiscovery{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		EndpointID:    endpointID,
		OriginalURL:   originalURL,
		DiscoveredURL: newURL,
		Method:        method,
		Status:        "pending",
		CreatedAt:     time.Now(),
	}

	if err := s.repo.CreateDiscovery(discovery); err != nil {
		return nil, fmt.Errorf("failed to create discovery: %w", err)
	}
	return discovery, nil
}

func (s *Service) checkWellKnown(currentURL string) (string, error) {
	parsed, err := url.Parse(currentURL)
	if err != nil {
		return "", err
	}

	wellKnownURL := fmt.Sprintf("%s://%s/.well-known/waas-webhooks", parsed.Scheme, parsed.Host)
	resp, err := s.client.Get(wellKnownURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("well-known returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return "", err
	}

	var spec WellKnownSpec
	if err := json.Unmarshal(body, &spec); err != nil {
		return "", err
	}

	for _, ep := range spec.Endpoints {
		if ep.MigratedTo != "" {
			return ep.MigratedTo, nil
		}
	}
	return "", fmt.Errorf("no migration target in well-known")
}

func (s *Service) checkHTTPRedirect(currentURL string) (string, error) {
	resp, err := s.client.Head(currentURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusPermanentRedirect {
		location := resp.Header.Get("Location")
		if location != "" {
			return location, nil
		}
	}
	return "", fmt.Errorf("no redirect detected")
}

func (s *Service) checkDNSTXT(currentURL string) (string, error) {
	parsed, err := url.Parse(currentURL)
	if err != nil {
		return "", err
	}

	// Convention: _waas-webhook.host TXT record
	_ = strings.TrimPrefix(parsed.Hostname(), "www.")
	// DNS lookup would happen here in production; stub for now
	return "", fmt.Errorf("DNS TXT lookup not available in this environment")
}

func (s *Service) testURL(targetURL string) *TestResult {
	start := time.Now()
	resp, err := s.client.Post(targetURL, "application/json", strings.NewReader(s.config.TestPayload))
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return &TestResult{
			Success:   false,
			Error:     err.Error(),
			LatencyMs: latency,
			TestedAt:  time.Now(),
		}
	}
	defer resp.Body.Close()

	return &TestResult{
		StatusCode: resp.StatusCode,
		LatencyMs:  latency,
		Success:    resp.StatusCode >= 200 && resp.StatusCode < 300,
		TestedAt:   time.Now(),
	}
}

func (s *Service) emitMigrationEvent(discovery *EndpointDiscovery) {
	evt := &MigrationEvent{
		ID:         uuid.New().String(),
		TenantID:   discovery.TenantID,
		EndpointID: discovery.EndpointID,
		EventType:  "endpoint.url.changed",
		OldURL:     discovery.OriginalURL,
		NewURL:     discovery.DiscoveredURL,
		Method:     discovery.Method,
		Timestamp:  time.Now(),
	}
	if err := s.repo.AppendMigrationEvent(evt); err != nil {
		s.logger.Error("failed to append migration event", map[string]interface{}{"error": err.Error(), "endpoint_id": discovery.EndpointID})
	}
}
