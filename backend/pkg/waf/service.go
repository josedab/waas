package waf

import (
	"context"
	"fmt"
	"regexp"

	"github.com/josedab/waas/pkg/utils"
)

// Service provides WAF and security scanning operations
type Service struct {
	repo   Repository
	logger *utils.Logger
}

// NewService creates a new WAF service
func NewService(repo Repository) *Service {
	return &Service{repo: repo, logger: utils.NewLogger("waf")}
}

// Common XSS patterns
var xssPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)<script[\s>]`),
	regexp.MustCompile(`(?i)javascript\s*:`),
	regexp.MustCompile(`(?i)on\w+\s*=`),
	regexp.MustCompile(`(?i)<iframe[\s>]`),
	regexp.MustCompile(`(?i)<object[\s>]`),
	regexp.MustCompile(`(?i)eval\s*\(`),
	regexp.MustCompile(`(?i)document\.cookie`),
}

// Common SQL injection patterns
var sqlInjectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(\b(union|select|insert|update|delete|drop|alter|create|exec)\b.*\b(from|into|table|database|where)\b)`),
	regexp.MustCompile(`(?i)(\bor\b\s+\d+\s*=\s*\d+)`),
	regexp.MustCompile(`(?i)(--\s|;\s*drop\s|;\s*delete\s)`),
	regexp.MustCompile(`(?i)('\s*(or|and)\s+')`),
	regexp.MustCompile(`(?i)(\/\*.*\*\/)`),
}

// Common path traversal patterns
var pathTraversalPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\.\./`),
	regexp.MustCompile(`\.\.\\`),
	regexp.MustCompile(`(?i)%2e%2e[/\\]`),
	regexp.MustCompile(`(?i)/etc/passwd`),
	regexp.MustCompile(`(?i)/proc/self`),
	regexp.MustCompile(`(?i)\\windows\\`),
}

const maxPayloadSize = 10 * 1024 * 1024 // 10MB

// GetSecurityDashboard returns aggregated security metrics
func (s *Service) GetSecurityDashboard(ctx context.Context, tenantID string) (*SecurityDashboard, error) {
	totalScans, err := s.repo.GetTotalScans(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get total scans: %w", err)
	}

	threatsDetected, err := s.repo.GetThreatsDetected(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get threats detected: %w", err)
	}

	threatsBlocked, err := s.repo.GetThreatsBlocked(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get threats blocked: %w", err)
	}

	quarantineCount, err := s.repo.GetQuarantineCount(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get quarantine count: %w", err)
	}

	topThreats, err := s.repo.GetTopThreats(ctx, tenantID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get top threats: %w", err)
	}

	alerts, _, err := s.repo.ListAlerts(ctx, tenantID, 10, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent alerts: %w", err)
	}

	riskTrend, err := s.repo.GetRiskTrend(ctx, tenantID, 30)
	if err != nil {
		return nil, fmt.Errorf("failed to get risk trend: %w", err)
	}

	return &SecurityDashboard{
		TotalScans:      totalScans,
		ThreatsDetected: threatsDetected,
		ThreatsBlocked:  threatsBlocked,
		QuarantineCount: quarantineCount,
		TopThreats:      topThreats,
		RecentAlerts:    alerts,
		RiskTrend:       riskTrend,
	}, nil
}

// GetSecurityAlerts lists security alerts
func (s *Service) GetSecurityAlerts(ctx context.Context, tenantID string, limit, offset int) ([]SecurityAlert, int, error) {
	return s.repo.ListAlerts(ctx, tenantID, limit, offset)
}

// AcknowledgeAlert acknowledges a security alert
func (s *Service) AcknowledgeAlert(ctx context.Context, tenantID, alertID string) (*SecurityAlert, error) {
	alert, err := s.repo.GetAlert(ctx, tenantID, alertID)
	if err != nil {
		return nil, err
	}

	if alert.Acknowledged {
		return nil, ErrAlreadyAcknowledged
	}

	alert.Acknowledged = true
	if err := s.repo.UpdateAlert(ctx, alert); err != nil {
		return nil, fmt.Errorf("failed to acknowledge alert: %w", err)
	}

	return alert, nil
}
