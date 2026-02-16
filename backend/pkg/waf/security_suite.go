package waf

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SecurityPosture represents the unified security state
type SecurityPosture struct {
	TenantID          string                 `json:"tenant_id"`
	WAFEnabled        bool                   `json:"waf_enabled"`
	MTLSEnabled       bool                   `json:"mtls_enabled"`
	ZeroTrustEnabled  bool                   `json:"zero_trust_enabled"`
	OverallScore      float64                `json:"overall_score"` // 0-100
	ThreatSummary     *SecurityThreatSummary `json:"threat_summary"`
	CertificateStatus *CertificateSummary    `json:"certificate_status"`
	IPAllowlistCount  int                    `json:"ip_allowlist_count"`
	ActiveRules       int                    `json:"active_rules"`
	LastAuditAt       *time.Time             `json:"last_audit_at,omitempty"`
	GeneratedAt       time.Time              `json:"generated_at"`
}

// SecurityThreatSummary provides overview of threat activity
type SecurityThreatSummary struct {
	TotalScans       int64         `json:"total_scans"`
	ThreatsDetected  int64         `json:"threats_detected"`
	ThreatsBlocked   int64         `json:"threats_blocked"`
	TopThreats       []ThreatCount `json:"top_threats"`
	RiskScore        float64       `json:"risk_score"`     // 0-10
	Last24HoursTrend string        `json:"last_24h_trend"` // increasing, decreasing, stable
}

// ThreatCount represents a threat type count
type ThreatCount struct {
	Type  ThreatType `json:"type"`
	Count int64      `json:"count"`
}

// CertificateSummary provides overview of certificate status
type CertificateSummary struct {
	TotalCerts    int `json:"total_certificates"`
	ActiveCerts   int `json:"active_certificates"`
	ExpiringCerts int `json:"expiring_certificates"`
	ExpiredCerts  int `json:"expired_certificates"`
	RevokedCerts  int `json:"revoked_certificates"`
}

// IPAllowlistEntry represents an IP address or CIDR in the allowlist
type IPAllowlistEntry struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	CIDR        string     `json:"cidr"` // IP or CIDR notation
	Description string     `json:"description,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CreatedBy   string     `json:"created_by,omitempty"`
}

// ZeroTrustVerification represents a zero-trust verification result
type ZeroTrustVerification struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	EndpointID     string    `json:"endpoint_id"`
	DeliveryID     string    `json:"delivery_id"`
	IPVerified     bool      `json:"ip_verified"`
	MTLSVerified   bool      `json:"mtls_verified"`
	SignatureValid bool      `json:"signature_valid"`
	PayloadScanned bool      `json:"payload_scanned"`
	ThreatFree     bool      `json:"threat_free"`
	OverallPassed  bool      `json:"overall_passed"`
	FailureReason  string    `json:"failure_reason,omitempty"`
	VerifiedAt     time.Time `json:"verified_at"`
	DurationMs     float64   `json:"duration_ms"`
}

// SecurityAuditLog represents a security audit event
type SecurityAuditLog struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Details   string    `json:"details"`
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	Result    string    `json:"result"` // allowed, blocked, flagged
	Timestamp time.Time `json:"timestamp"`
}

// CreateIPAllowlistRequest represents a request to add an IP to the allowlist
type CreateIPAllowlistRequest struct {
	CIDR        string     `json:"cidr" binding:"required"`
	Description string     `json:"description,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// SecuritySuiteRepository defines storage for the unified security suite
type SecuritySuiteRepository interface {
	// IP Allowlist
	AddIPAllowlist(ctx context.Context, entry *IPAllowlistEntry) error
	RemoveIPAllowlist(ctx context.Context, tenantID, entryID string) error
	ListIPAllowlist(ctx context.Context, tenantID string) ([]IPAllowlistEntry, error)
	CheckIPAllowed(ctx context.Context, tenantID, ip string) (bool, error)

	// Zero-Trust Verification
	SaveVerification(ctx context.Context, verification *ZeroTrustVerification) error
	ListVerifications(ctx context.Context, tenantID string, limit int) ([]ZeroTrustVerification, error)
	GetVerificationStats(ctx context.Context, tenantID string) (total int64, passRate float64, err error)

	// Audit Logs
	SaveAuditLog(ctx context.Context, log *SecurityAuditLog) error
	ListAuditLogs(ctx context.Context, tenantID string, limit int) ([]SecurityAuditLog, error)

	// Security Posture
	GetThreatSummary(ctx context.Context, tenantID string) (*SecurityThreatSummary, error)
	GetCertificateSummary(ctx context.Context, tenantID string) (*CertificateSummary, error)
}

// SecuritySuite provides unified security operations
type SecuritySuite struct {
	repo    SecuritySuiteRepository
	wafSvc  *Service
	ipCache map[string]map[string]bool // tenantID -> CIDR set
	mu      sync.RWMutex
}

// NewSecuritySuite creates a new unified security suite
func NewSecuritySuite(repo SecuritySuiteRepository, wafSvc *Service) *SecuritySuite {
	return &SecuritySuite{
		repo:    repo,
		wafSvc:  wafSvc,
		ipCache: make(map[string]map[string]bool),
	}
}

// AddIPAllowlist adds an IP/CIDR to the allowlist
func (s *SecuritySuite) AddIPAllowlist(ctx context.Context, tenantID string, req *CreateIPAllowlistRequest) (*IPAllowlistEntry, error) {
	if err := validateCIDR(req.CIDR); err != nil {
		return nil, err
	}

	entry := &IPAllowlistEntry{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		CIDR:        req.CIDR,
		Description: req.Description,
		ExpiresAt:   req.ExpiresAt,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.AddIPAllowlist(ctx, entry); err != nil {
		return nil, fmt.Errorf("failed to add IP allowlist entry: %w", err)
	}

	// Update cache
	s.mu.Lock()
	if s.ipCache[tenantID] == nil {
		s.ipCache[tenantID] = make(map[string]bool)
	}
	s.ipCache[tenantID][req.CIDR] = true
	s.mu.Unlock()

	return entry, nil
}

// RemoveIPAllowlist removes an IP from the allowlist
func (s *SecuritySuite) RemoveIPAllowlist(ctx context.Context, tenantID, entryID string) error {
	return s.repo.RemoveIPAllowlist(ctx, tenantID, entryID)
}

// ListIPAllowlist lists all IP allowlist entries
func (s *SecuritySuite) ListIPAllowlist(ctx context.Context, tenantID string) ([]IPAllowlistEntry, error) {
	return s.repo.ListIPAllowlist(ctx, tenantID)
}

// VerifyDelivery performs zero-trust verification for a delivery
func (s *SecuritySuite) VerifyDelivery(ctx context.Context, tenantID, endpointID, deliveryID, sourceIP string, payload []byte) (*ZeroTrustVerification, error) {
	start := time.Now()

	verification := &ZeroTrustVerification{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		EndpointID: endpointID,
		DeliveryID: deliveryID,
		VerifiedAt: start,
	}

	// Step 1: IP verification
	ipAllowed, ipErr := s.repo.CheckIPAllowed(ctx, tenantID, sourceIP)
	if ipErr != nil {
		log.Printf("[waf] CheckIPAllowed error for tenant=%s ip=%s: %v", tenantID, sourceIP, ipErr)
		return nil, fmt.Errorf("IP allowlist check failed: %w", ipErr)
	}
	verification.IPVerified = ipAllowed

	// Step 2: Payload scanning via WAF
	if s.wafSvc != nil && payload != nil {
		scanReq := &ScanPayloadRequest{Payload: payload, WebhookID: deliveryID}
		scanResult, err := s.wafSvc.ScanPayload(ctx, tenantID, scanReq)
		if err == nil {
			verification.PayloadScanned = true
			verification.ThreatFree = scanResult.Action != ScanActionBlock && scanResult.Action != ScanActionQuarantine
		}
	} else {
		verification.PayloadScanned = true
		verification.ThreatFree = true
	}

	// Determine overall pass
	verification.OverallPassed = verification.IPVerified && verification.ThreatFree
	if !verification.OverallPassed {
		reasons := []string{}
		if !verification.IPVerified {
			reasons = append(reasons, "IP not in allowlist")
		}
		if !verification.ThreatFree {
			reasons = append(reasons, "payload threat detected")
		}
		verification.FailureReason = strings.Join(reasons, "; ")
	}

	verification.DurationMs = float64(time.Since(start).Microseconds()) / 1000.0

	if err := s.repo.SaveVerification(ctx, verification); err != nil {
		log.Printf("[waf] SaveVerification error for tenant=%s delivery=%s: %v", tenantID, deliveryID, err)
	}

	// Audit log
	result := "allowed"
	if !verification.OverallPassed {
		result = "blocked"
	}
	if err := s.repo.SaveAuditLog(ctx, &SecurityAuditLog{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Action:    "zero_trust_verify",
		Resource:  deliveryID,
		Details:   fmt.Sprintf("IP: %s, Passed: %v", sourceIP, verification.OverallPassed),
		IPAddress: sourceIP,
		Result:    result,
		Timestamp: time.Now(),
	}); err != nil {
		log.Printf("[waf] SaveAuditLog error for tenant=%s delivery=%s: %v", tenantID, deliveryID, err)
	}

	return verification, nil
}

// GetSecurityPosture returns the unified security posture
func (s *SecuritySuite) GetSecurityPosture(ctx context.Context, tenantID string) (*SecurityPosture, error) {
	threatSummary, _ := s.repo.GetThreatSummary(ctx, tenantID)
	if threatSummary == nil {
		threatSummary = &SecurityThreatSummary{}
	}

	certSummary, _ := s.repo.GetCertificateSummary(ctx, tenantID)
	if certSummary == nil {
		certSummary = &CertificateSummary{}
	}

	ipList, _ := s.repo.ListIPAllowlist(ctx, tenantID)

	_, passRate, _ := s.repo.GetVerificationStats(ctx, tenantID)

	// Calculate overall security score
	score := calculateSecurityScore(threatSummary, certSummary, len(ipList), passRate)

	return &SecurityPosture{
		TenantID:          tenantID,
		WAFEnabled:        true,
		MTLSEnabled:       certSummary.ActiveCerts > 0,
		ZeroTrustEnabled:  len(ipList) > 0,
		OverallScore:      score,
		ThreatSummary:     threatSummary,
		CertificateStatus: certSummary,
		IPAllowlistCount:  len(ipList),
		GeneratedAt:       time.Now(),
	}, nil
}

// ListAuditLogs lists security audit logs
func (s *SecuritySuite) ListAuditLogs(ctx context.Context, tenantID string, limit int) ([]SecurityAuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListAuditLogs(ctx, tenantID, limit)
}

func validateCIDR(cidr string) error {
	if strings.Contains(cidr, "/") {
		_, _, err := net.ParseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("invalid CIDR notation: %w", err)
		}
	} else {
		if net.ParseIP(cidr) == nil {
			return fmt.Errorf("invalid IP address: %s", cidr)
		}
	}
	return nil
}

func calculateSecurityScore(threats *SecurityThreatSummary, certs *CertificateSummary, ipCount int, passRate float64) float64 {
	score := 50.0 // Base score

	// Threat risk reduces score
	if threats.RiskScore > 0 {
		score -= threats.RiskScore * 5
	}

	// Active certs improve score
	if certs.ActiveCerts > 0 {
		score += 15
	}

	// Expiring/expired certs reduce score
	score -= float64(certs.ExpiringCerts) * 3
	score -= float64(certs.ExpiredCerts) * 5

	// IP allowlist improves score
	if ipCount > 0 {
		score += 10
	}

	// High pass rate improves score
	score += passRate * 25

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}
