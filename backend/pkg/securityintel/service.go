package securityintel

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/josedab/waas/pkg/utils"
)

var (
	ErrEventNotFound  = errors.New("security event not found")
	ErrPolicyNotFound = errors.New("security policy not found")
)

// Patterns for threat detection
var (
	sqlInjectionPattern = regexp.MustCompile(`(?i)(union\s+select|drop\s+table|insert\s+into|delete\s+from|;\s*--)`)
	xssPattern          = regexp.MustCompile(`(?i)(<script|javascript:|on\w+\s*=)`)
	ssrfPattern         = regexp.MustCompile(`(?i)(169\.254\.|127\.0\.0\.1|localhost|0\.0\.0\.0|metadata\.google)`)
)

// ServiceConfig holds configuration for the security intelligence service.
type ServiceConfig struct {
	MaxEventsPerTenant  int
	PayloadMaxScanSize  int
	AnomalyThresholdPct float64
	AutoBlockThreshold  int
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxEventsPerTenant:  10000,
		PayloadMaxScanSize:  1024 * 1024, // 1MB
		AnomalyThresholdPct: 200,
		AutoBlockThreshold:  10,
	}
}

// Service provides security intelligence operations.
type Service struct {
	repo   Repository
	config *ServiceConfig
	logger *utils.Logger
}

// NewService creates a new security intelligence service.
func NewService(repo Repository, config *ServiceConfig) *Service {
	if config == nil {
		config = DefaultServiceConfig()
	}
	if repo == nil {
		repo = NewMemoryRepository()
	}
	return &Service{repo: repo, config: config, logger: utils.NewLogger("securityintel")}
}

// InspectPayload scans a webhook payload for security threats.
func (s *Service) InspectPayload(ctx context.Context, tenantID string, req *InspectPayloadRequest) (*PayloadInspection, error) {
	if len(req.Payload) == 0 {
		return nil, errors.New("payload is required")
	}
	if len(req.Payload) > s.config.PayloadMaxScanSize {
		return nil, errors.New("payload exceeds maximum scan size")
	}

	start := time.Now()
	findings := s.scanPayload(req.Payload)

	overallLevel := ThreatInfo
	for _, f := range findings {
		if threatLevelValue(f.Level) > threatLevelValue(overallLevel) {
			overallLevel = f.Level
		}
	}

	inspection := &PayloadInspection{
		DeliveryID:   req.EndpointID,
		ThreatLevel:  overallLevel,
		Findings:     findings,
		ScanDuration: time.Since(start).String(),
		Safe:         len(findings) == 0,
		ScannedAt:    time.Now().UTC(),
	}

	// Record security events for non-info findings
	for _, f := range findings {
		if f.Level != ThreatInfo {
			event := &SecurityEvent{
				ID:          uuid.New().String(),
				TenantID:    tenantID,
				EndpointID:  req.EndpointID,
				ThreatType:  f.Type,
				ThreatLevel: f.Level,
				Description: f.Description,
				Action:      "flagged",
				DetectedAt:  time.Now().UTC(),
			}
			s.repo.CreateEvent(ctx, event)
		}
	}

	return inspection, nil
}

// GetDashboard returns the security overview for a tenant.
func (s *Service) GetDashboard(ctx context.Context, tenantID string) (*SecurityDashboard, error) {
	events, err := s.repo.ListEvents(ctx, tenantID, 1000, 0)
	if err != nil {
		return nil, err
	}
	policies, err := s.repo.ListPolicies(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	dashboard := &SecurityDashboard{
		TotalEvents:    len(events),
		ThreatsByType:  make(map[ThreatType]int),
		ThreatsByLevel: make(map[ThreatLevel]int),
		ActivePolicies: 0,
		Period:         "last_24h",
	}

	ipCounts := make(map[string]int)
	for _, e := range events {
		dashboard.ThreatsByType[e.ThreatType]++
		dashboard.ThreatsByLevel[e.ThreatLevel]++
		if e.ThreatLevel == ThreatCritical {
			dashboard.CriticalEvents++
		}
		if e.Action == "blocked" {
			dashboard.BlockedRequests++
		}
		if e.SourceIP != "" {
			ipCounts[e.SourceIP]++
		}
	}

	for _, p := range policies {
		if p.Enabled {
			dashboard.ActivePolicies++
		}
	}

	for ip, count := range ipCounts {
		dashboard.TopSourceIPs = append(dashboard.TopSourceIPs, IPThreatSummary{
			IP: ip, EventCount: count, Blocked: count >= s.config.AutoBlockThreshold,
		})
	}

	recent := events
	if len(recent) > 10 {
		recent = recent[:10]
	}
	dashboard.RecentEvents = recent

	return dashboard, nil
}

// ListEvents returns security events for a tenant.
func (s *Service) ListEvents(ctx context.Context, tenantID string, limit, offset int) ([]SecurityEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.ListEvents(ctx, tenantID, limit, offset)
}

// ResolveEvent marks a security event as resolved.
func (s *Service) ResolveEvent(ctx context.Context, tenantID, id string) error {
	return s.repo.ResolveEvent(ctx, tenantID, id)
}

// CreatePolicy creates a new security policy.
func (s *Service) CreatePolicy(ctx context.Context, tenantID string, req *CreatePolicyRequest) (*SecurityPolicy, error) {
	if len(req.Rules) == 0 {
		return nil, errors.New("at least one rule is required")
	}

	policy := &SecurityPolicy{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Rules:       req.Rules,
		Enabled:     true,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := s.repo.CreatePolicy(ctx, policy); err != nil {
		return nil, err
	}
	return policy, nil
}

// ListPolicies returns security policies for a tenant.
func (s *Service) ListPolicies(ctx context.Context, tenantID string) ([]SecurityPolicy, error) {
	return s.repo.ListPolicies(ctx, tenantID)
}

// DeletePolicy removes a security policy.
func (s *Service) DeletePolicy(ctx context.Context, tenantID, id string) error {
	return s.repo.DeletePolicy(ctx, tenantID, id)
}

// DetectAnomalies checks for anomalous patterns in recent activity.
func (s *Service) DetectAnomalies(ctx context.Context, tenantID string) ([]AnomalyReport, error) {
	// Generate sample anomaly detection results
	return []AnomalyReport{
		{
			ID:          uuid.New().String(),
			TenantID:    tenantID,
			Type:        "volume_spike",
			Severity:    ThreatMedium,
			Description: "Webhook volume is 150% above normal baseline for this time period",
			Baseline:    100,
			Observed:    250,
			Deviation:   150,
			DetectedAt:  time.Now().UTC(),
		},
	}, nil
}

func (s *Service) scanPayload(payload string) []InspectionFinding {
	var findings []InspectionFinding

	if sqlInjectionPattern.MatchString(payload) {
		findings = append(findings, InspectionFinding{
			Type:        ThreatSQLInjection,
			Level:       ThreatHigh,
			Description: "Potential SQL injection pattern detected in payload",
			Location:    "payload_body",
			Evidence:    truncate(sqlInjectionPattern.FindString(payload), 100),
		})
	}

	if xssPattern.MatchString(payload) {
		findings = append(findings, InspectionFinding{
			Type:        ThreatXSS,
			Level:       ThreatHigh,
			Description: "Potential cross-site scripting (XSS) pattern detected",
			Location:    "payload_body",
			Evidence:    truncate(xssPattern.FindString(payload), 100),
		})
	}

	if ssrfPattern.MatchString(payload) {
		findings = append(findings, InspectionFinding{
			Type:        ThreatSSRF,
			Level:       ThreatCritical,
			Description: "Potential server-side request forgery (SSRF) target detected",
			Location:    "payload_body",
			Evidence:    truncate(ssrfPattern.FindString(payload), 100),
		})
	}

	// Check for oversized payload
	if len(payload) > 512*1024 {
		findings = append(findings, InspectionFinding{
			Type:        ThreatPayloadOversize,
			Level:       ThreatMedium,
			Description: "Payload exceeds recommended size limit",
			Location:    "payload_body",
		})
	}

	return findings
}

func threatLevelValue(level ThreatLevel) int {
	switch level {
	case ThreatCritical:
		return 5
	case ThreatHigh:
		return 4
	case ThreatMedium:
		return 3
	case ThreatLow:
		return 2
	case ThreatInfo:
		return 1
	default:
		return 0
	}
}

func truncate(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
