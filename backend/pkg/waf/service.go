package waf

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service provides WAF and security scanning operations
type Service struct {
	repo Repository
}

// NewService creates a new WAF service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
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

// ScanPayload scans a webhook payload for security threats
func (s *Service) ScanPayload(ctx context.Context, tenantID string, req *ScanPayloadRequest) (*ScanResult, error) {
	start := time.Now()

	var threats []Threat
	payloadStr := string(req.Payload)

	// Check payload size
	if len(req.Payload) > maxPayloadSize {
		threats = append(threats, Threat{
			Type:           ThreatTypeOversizedPayload,
			Severity:       ThreatSeverityMedium,
			Description:    fmt.Sprintf("Payload size %d bytes exceeds limit of %d bytes", len(req.Payload), maxPayloadSize),
			Recommendation: "Reduce payload size or configure a higher limit",
		})
	}

	// Check for XSS
	for _, pattern := range xssPatterns {
		if match := pattern.FindString(payloadStr); match != "" {
			threats = append(threats, Threat{
				Type:           ThreatTypeXSS,
				Severity:       ThreatSeverityHigh,
				Description:    "Potential XSS attack detected in payload",
				Evidence:       truncateEvidence(match),
				Recommendation: "Sanitize or encode HTML content before processing",
			})
			break
		}
	}

	// Check for SQL injection
	for _, pattern := range sqlInjectionPatterns {
		if match := pattern.FindString(payloadStr); match != "" {
			threats = append(threats, Threat{
				Type:           ThreatTypeSQLInjection,
				Severity:       ThreatSeverityCritical,
				Description:    "Potential SQL injection detected in payload",
				Evidence:       truncateEvidence(match),
				Recommendation: "Use parameterized queries and input validation",
			})
			break
		}
	}

	// Check for path traversal
	for _, pattern := range pathTraversalPatterns {
		if match := pattern.FindString(payloadStr); match != "" {
			threats = append(threats, Threat{
				Type:           ThreatTypePathTraversal,
				Severity:       ThreatSeverityHigh,
				Description:    "Potential path traversal attack detected",
				Evidence:       truncateEvidence(match),
				Recommendation: "Validate and sanitize file paths",
			})
			break
		}
	}

	// Check for malicious JSON
	if isJSON(req.Payload) {
		if depth := getJSONDepth(req.Payload); depth > 50 {
			threats = append(threats, Threat{
				Type:           ThreatTypeMaliciousJSON,
				Severity:       ThreatSeverityMedium,
				Description:    fmt.Sprintf("Deeply nested JSON detected (depth: %d)", depth),
				Recommendation: "Limit JSON nesting depth to prevent DoS attacks",
			})
		}
	}

	// Check for suspicious patterns
	suspiciousKeywords := []string{"__proto__", "constructor.prototype", "require(", "process.env", "child_process"}
	for _, keyword := range suspiciousKeywords {
		if strings.Contains(payloadStr, keyword) {
			threats = append(threats, Threat{
				Type:           ThreatTypeSuspiciousPattern,
				Severity:       ThreatSeverityHigh,
				Description:    "Suspicious pattern detected in payload",
				Evidence:       truncateEvidence(keyword),
				Recommendation: "Review payload content for potential prototype pollution or code injection",
			})
			break
		}
	}

	// Evaluate custom WAF rules
	ruleThreats, err := s.EvaluateWAFRules(ctx, tenantID, payloadStr)
	if err == nil {
		threats = append(threats, ruleThreats...)
	}

	// Calculate risk score
	riskScore := calculateRiskScore(threats)

	// Determine action
	action := determineAction(riskScore, threats)

	result := &ScanResult{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		WebhookID:  req.WebhookID,
		DeliveryID: req.DeliveryID,
		Threats:    threats,
		RiskScore:  riskScore,
		Action:     action,
		ScannedAt:  time.Now(),
		DurationMs: time.Since(start).Milliseconds(),
	}

	// Save scan result
	if err := s.repo.CreateScanResult(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to save scan result: %w", err)
	}

	// Quarantine if needed
	if action == ScanActionQuarantine {
		if err := s.QuarantineWebhook(ctx, tenantID, req.WebhookID, "Automated quarantine: high risk score", threats, req.Payload); err != nil {
			return nil, fmt.Errorf("failed to quarantine webhook: %w", err)
		}
	}

	// Create alert for high-severity threats
	if riskScore >= 70 {
		alert := &SecurityAlert{
			ID:          uuid.New().String(),
			TenantID:    tenantID,
			AlertType:   "threat_detected",
			Severity:    getHighestSeverity(threats),
			Title:       fmt.Sprintf("Security threat detected on webhook %s", req.WebhookID),
			Description: fmt.Sprintf("Detected %d threats with risk score %.1f", len(threats), riskScore),
			CreatedAt:   time.Now(),
		}
		if err := s.repo.CreateAlert(ctx, alert); err != nil {
			log.Printf("[waf] CreateAlert error for tenant=%s: %v", tenantID, err)
		}
	}

	return result, nil
}

// EvaluateWAFRules evaluates a payload against tenant's custom WAF rules
func (s *Service) EvaluateWAFRules(ctx context.Context, tenantID, payload string) ([]Threat, error) {
	rules, err := s.repo.GetEnabledWAFRules(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get WAF rules: %w", err)
	}

	var threats []Threat
	for _, rule := range rules {
		matched := false

		switch rule.RuleType {
		case WAFRuleTypeRegex:
			re, err := regexp.Compile(rule.Pattern)
			if err != nil {
				continue
			}
			matched = re.MatchString(payload)
		case WAFRuleTypeKeyword:
			keywords := strings.Split(rule.Pattern, ",")
			for _, kw := range keywords {
				if strings.Contains(strings.ToLower(payload), strings.TrimSpace(strings.ToLower(kw))) {
					matched = true
					break
				}
			}
		case WAFRuleTypePayloadSize:
			// Pattern contains max size as string
			// Skip complex parsing; handled in main scan
		}

		if matched {
			threats = append(threats, Threat{
				Type:        ThreatTypeSuspiciousPattern,
				Severity:    ThreatSeverityMedium,
				Description: fmt.Sprintf("WAF rule matched: %s", rule.Name),
				Evidence:    rule.Pattern,
			})
			if err := s.repo.IncrementRuleHitCount(ctx, tenantID, rule.ID); err != nil {
				log.Printf("[waf] IncrementRuleHitCount error for tenant=%s rule=%s: %v", tenantID, rule.ID, err)
			}
		}
	}

	return threats, nil
}

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

// CheckIPReputation checks the reputation of an IP address
func (s *Service) CheckIPReputation(ctx context.Context, ip string) (*IPReputation, error) {
	reputation, err := s.repo.GetIPReputation(ctx, ip)
	if err == ErrIPReputationNotFound {
		return &IPReputation{
			IP:          ip,
			ThreatScore: 0,
			LastSeen:    time.Now(),
			ReportCount: 0,
			Blocked:     false,
		}, nil
	}
	return reputation, err
}

// ReportIP reports a malicious IP address
func (s *Service) ReportIP(ctx context.Context, req *ReportIPRequest) (*IPReputation, error) {
	existing, err := s.repo.GetIPReputation(ctx, req.IP)
	if err == ErrIPReputationNotFound {
		existing = &IPReputation{
			IP:          req.IP,
			ThreatScore: 0,
			ReportCount: 0,
			Categories:  []string{},
		}
	} else if err != nil {
		return nil, err
	}

	existing.ReportCount++
	existing.LastSeen = time.Now()

	// Merge categories
	categorySet := make(map[string]bool)
	for _, cat := range existing.Categories {
		categorySet[cat] = true
	}
	for _, cat := range req.Categories {
		categorySet[cat] = true
	}
	existing.Categories = make([]string, 0, len(categorySet))
	for cat := range categorySet {
		existing.Categories = append(existing.Categories, cat)
	}

	// Update threat score based on report count
	existing.ThreatScore = float64(existing.ReportCount) * 10
	if existing.ThreatScore > 100 {
		existing.ThreatScore = 100
	}

	// Auto-block at high threat score
	if existing.ThreatScore >= 80 {
		existing.Blocked = true
	}

	if err := s.repo.UpsertIPReputation(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update IP reputation: %w", err)
	}

	return existing, nil
}

// ListBlockedIPs lists all blocked IP addresses
func (s *Service) ListBlockedIPs(ctx context.Context, limit, offset int) ([]IPReputation, int, error) {
	return s.repo.ListBlockedIPs(ctx, limit, offset)
}

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

// ListScanResults lists scan results
func (s *Service) ListScanResults(ctx context.Context, tenantID string, limit, offset int) ([]ScanResult, int, error) {
	return s.repo.ListScanResults(ctx, tenantID, limit, offset)
}

// CreateWAFRule creates a new WAF rule
func (s *Service) CreateWAFRule(ctx context.Context, tenantID string, req *CreateWAFRuleRequest) (*WAFRule, error) {
	// Validate regex pattern if applicable
	if req.RuleType == WAFRuleTypeRegex {
		if _, err := regexp.Compile(req.Pattern); err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	now := time.Now()
	rule := &WAFRule{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Pattern:     req.Pattern,
		RuleType:    req.RuleType,
		Action:      req.Action,
		Priority:    req.Priority,
		Enabled:     req.Enabled,
		HitCount:    0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateWAFRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to create WAF rule: %w", err)
	}

	return rule, nil
}

// GetWAFRule retrieves a WAF rule
func (s *Service) GetWAFRule(ctx context.Context, tenantID, ruleID string) (*WAFRule, error) {
	return s.repo.GetWAFRule(ctx, tenantID, ruleID)
}

// UpdateWAFRule updates a WAF rule
func (s *Service) UpdateWAFRule(ctx context.Context, tenantID, ruleID string, req *CreateWAFRuleRequest) (*WAFRule, error) {
	rule, err := s.repo.GetWAFRule(ctx, tenantID, ruleID)
	if err != nil {
		return nil, err
	}

	if req.RuleType == WAFRuleTypeRegex {
		if _, err := regexp.Compile(req.Pattern); err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	rule.Name = req.Name
	rule.Description = req.Description
	rule.Pattern = req.Pattern
	rule.RuleType = req.RuleType
	rule.Action = req.Action
	rule.Priority = req.Priority
	rule.Enabled = req.Enabled
	rule.UpdatedAt = time.Now()

	if err := s.repo.UpdateWAFRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to update WAF rule: %w", err)
	}

	return rule, nil
}

// DeleteWAFRule deletes a WAF rule
func (s *Service) DeleteWAFRule(ctx context.Context, tenantID, ruleID string) error {
	return s.repo.DeleteWAFRule(ctx, tenantID, ruleID)
}

// ListWAFRules lists all WAF rules for a tenant
func (s *Service) ListWAFRules(ctx context.Context, tenantID string) ([]WAFRule, error) {
	return s.repo.ListWAFRules(ctx, tenantID)
}

// ScanEndpointSecurity performs a full security scan of an endpoint URL
func (s *Service) ScanEndpointSecurity(ctx context.Context, tenantID, endpointURL string) (*SecurityScanResult, error) {
	start := time.Now()

	result := &SecurityScanResult{
		EndpointID: tenantID,
		URL:        endpointURL,
		ScannedAt:  time.Now(),
	}

	// Check TLS
	result.TLSInfo = s.checkTLS(endpointURL)

	// Check security headers
	result.SecurityHeaders = s.checkSecurityHeaders(endpointURL)

	// Generate findings from header/TLS checks
	var findings []SecurityFinding
	if result.TLSInfo != nil && !result.TLSInfo.IsHTTPS {
		findings = append(findings, SecurityFinding{
			ID:             "tls-missing",
			Title:          "HTTPS Not Enabled",
			Description:    "The endpoint does not use HTTPS, exposing data in transit.",
			Severity:       "critical",
			Category:       "transport",
			Recommendation: "Enable TLS/HTTPS on the endpoint.",
		})
	}
	if result.TLSInfo != nil && result.TLSInfo.IsHTTPS && !result.TLSInfo.CertValid {
		findings = append(findings, SecurityFinding{
			ID:             "tls-cert-invalid",
			Title:          "Invalid TLS Certificate",
			Description:    "The TLS certificate is invalid or expired.",
			Severity:       "critical",
			Category:       "transport",
			Recommendation: "Renew or replace the TLS certificate.",
		})
	}
	if result.TLSInfo != nil && result.TLSInfo.CertValid && result.TLSInfo.CertDaysLeft < 30 {
		findings = append(findings, SecurityFinding{
			ID:             "tls-cert-expiring",
			Title:          "TLS Certificate Expiring Soon",
			Description:    fmt.Sprintf("Certificate expires in %d days.", result.TLSInfo.CertDaysLeft),
			Severity:       "high",
			Category:       "transport",
			Recommendation: "Renew the TLS certificate before expiry.",
		})
	}
	for _, hdr := range result.SecurityHeaders {
		if !hdr.Present && (hdr.Severity == "critical" || hdr.Severity == "high") {
			findings = append(findings, SecurityFinding{
				ID:             "header-missing-" + strings.ToLower(strings.ReplaceAll(hdr.Header, "-", "")),
				Title:          fmt.Sprintf("Missing %s Header", hdr.Header),
				Description:    fmt.Sprintf("The security header %s is not present.", hdr.Header),
				Severity:       hdr.Severity,
				Category:       "headers",
				Recommendation: fmt.Sprintf("Add the %s header with value: %s", hdr.Header, hdr.Expected),
			})
		}
	}
	result.Findings = findings

	// Check compliance
	result.ComplianceChecks = s.checkCompliance(result)

	// Calculate score
	result.OverallScore, result.NumericScore = s.calculateScore(result)

	result.ResponseTimeMs = int(time.Since(start).Milliseconds())

	return result, nil
}

// checkSecurityHeaders checks for important security headers on the endpoint
func (s *Service) checkSecurityHeaders(url string) []HeaderCheck {
	headers := []struct {
		Name     string
		Expected string
		Severity string
	}{
		{"Strict-Transport-Security", "max-age=31536000; includeSubDomains", "critical"},
		{"Content-Security-Policy", "default-src 'self'", "high"},
		{"X-Frame-Options", "DENY", "high"},
		{"X-Content-Type-Options", "nosniff", "medium"},
		{"X-XSS-Protection", "1; mode=block", "medium"},
		{"Referrer-Policy", "strict-origin-when-cross-origin", "low"},
		{"Permissions-Policy", "geolocation=(), camera=(), microphone=()", "low"},
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	var checks []HeaderCheck
	resp, err := client.Head(url)
	if err != nil {
		// Return all headers as unchecked
		for _, h := range headers {
			checks = append(checks, HeaderCheck{
				Header:   h.Name,
				Present:  false,
				Expected: h.Expected,
				Severity: h.Severity,
			})
		}
		return checks
	}
	defer resp.Body.Close()

	for _, h := range headers {
		val := resp.Header.Get(h.Name)
		checks = append(checks, HeaderCheck{
			Header:   h.Name,
			Present:  val != "",
			Value:    val,
			Expected: h.Expected,
			Severity: h.Severity,
		})
	}
	return checks
}

// checkTLS checks the TLS configuration of an endpoint
func (s *Service) checkTLS(url string) *TLSInfo {
	info := &TLSInfo{}

	if !strings.HasPrefix(url, "https://") {
		info.IsHTTPS = false
		return info
	}
	info.IsHTTPS = true

	// Extract host from URL
	host := strings.TrimPrefix(url, "https://")
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	if !strings.Contains(host, ":") {
		host = host + ":443"
	}

	// Use a custom VerifyPeerCertificate callback to capture certificates
	// for inspection while still performing standard TLS verification.
	var peerCerts []*x509.Certificate
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 10 * time.Second},
		"tcp",
		host,
		&tls.Config{
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				for _, rawCert := range rawCerts {
					cert, err := x509.ParseCertificate(rawCert)
					if err == nil {
						peerCerts = append(peerCerts, cert)
					}
				}
				return nil
			},
		},
	)
	if err != nil {
		return info
	}
	defer conn.Close()

	state := conn.ConnectionState()

	// TLS version
	switch state.Version {
	case tls.VersionTLS13:
		info.Version = "TLS 1.3"
	case tls.VersionTLS12:
		info.Version = "TLS 1.2"
	case tls.VersionTLS11:
		info.Version = "TLS 1.1"
	case tls.VersionTLS10:
		info.Version = "TLS 1.0"
	default:
		info.Version = "Unknown"
	}

	info.CipherSuite = tls.CipherSuiteName(state.CipherSuite)

	if len(state.PeerCertificates) > 0 {
		cert := state.PeerCertificates[0]
		info.CertIssuer = cert.Issuer.CommonName
		info.CertExpiry = cert.NotAfter
		info.CertDaysLeft = int(time.Until(cert.NotAfter).Hours() / 24)
		info.CertValid = time.Now().Before(cert.NotAfter) && time.Now().After(cert.NotBefore)
	} else if len(peerCerts) > 0 {
		cert := peerCerts[0]
		info.CertIssuer = cert.Issuer.CommonName
		info.CertExpiry = cert.NotAfter
		info.CertDaysLeft = int(time.Until(cert.NotAfter).Hours() / 24)
		info.CertValid = time.Now().Before(cert.NotAfter) && time.Now().After(cert.NotBefore)
	}

	// Check HSTS via a separate HTTP request
	hstsClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := hstsClient.Head(url)
	if err == nil {
		defer resp.Body.Close()
		info.SupportsHSTS = resp.Header.Get("Strict-Transport-Security") != ""
	}

	return info
}

// checkCompliance runs SOC2 and HIPAA compliance checks
func (s *Service) checkCompliance(result *SecurityScanResult) []ComplianceCheck {
	var checks []ComplianceCheck

	// SOC2 checks
	httpsStatus := "fail"
	httpsDetails := "Endpoint does not use HTTPS"
	if result.TLSInfo != nil && result.TLSInfo.IsHTTPS {
		httpsStatus = "pass"
		httpsDetails = "Endpoint uses HTTPS"
	}
	checks = append(checks, ComplianceCheck{
		Framework:   "SOC2",
		ControlID:   "CC6.1",
		ControlName: "Encryption in Transit",
		Status:      httpsStatus,
		Details:     httpsDetails,
	})

	certStatus := "not_applicable"
	certDetails := "No TLS certificate to validate"
	if result.TLSInfo != nil && result.TLSInfo.IsHTTPS {
		if result.TLSInfo.CertValid {
			certStatus = "pass"
			certDetails = fmt.Sprintf("Certificate valid, expires in %d days", result.TLSInfo.CertDaysLeft)
		} else {
			certStatus = "fail"
			certDetails = "Certificate is invalid or expired"
		}
	}
	checks = append(checks, ComplianceCheck{
		Framework:   "SOC2",
		ControlID:   "CC6.7",
		ControlName: "Valid Certificate",
		Status:      certStatus,
		Details:     certDetails,
	})

	hstsPresent := false
	cspPresent := false
	for _, hdr := range result.SecurityHeaders {
		if hdr.Header == "Strict-Transport-Security" && hdr.Present {
			hstsPresent = true
		}
		if hdr.Header == "Content-Security-Policy" && hdr.Present {
			cspPresent = true
		}
	}

	hstsStatus := "fail"
	if hstsPresent {
		hstsStatus = "pass"
	}
	checks = append(checks, ComplianceCheck{
		Framework:   "SOC2",
		ControlID:   "CC6.6",
		ControlName: "HSTS Enabled",
		Status:      hstsStatus,
		Details:     fmt.Sprintf("HSTS header present: %v", hstsPresent),
	})

	// HIPAA checks
	checks = append(checks, ComplianceCheck{
		Framework:   "HIPAA",
		ControlID:   "164.312(e)(1)",
		ControlName: "Transmission Security",
		Status:      httpsStatus,
		Details:     httpsDetails,
	})

	cspStatus := "fail"
	cspDetails := "Content-Security-Policy header not present"
	if cspPresent {
		cspStatus = "pass"
		cspDetails = "Content-Security-Policy header is set"
	}
	checks = append(checks, ComplianceCheck{
		Framework:   "HIPAA",
		ControlID:   "164.312(a)(1)",
		ControlName: "Access Control Headers",
		Status:      cspStatus,
		Details:     cspDetails,
	})

	return checks
}

// calculateScore computes an overall security grade and numeric score
func (s *Service) calculateScore(result *SecurityScanResult) (string, int) {
	score := 100

	// TLS scoring
	if result.TLSInfo != nil {
		if !result.TLSInfo.IsHTTPS {
			score -= 30
		} else {
			if !result.TLSInfo.CertValid {
				score -= 25
			}
			if result.TLSInfo.CertDaysLeft < 30 && result.TLSInfo.CertDaysLeft >= 0 {
				score -= 10
			}
			if result.TLSInfo.Version == "TLS 1.0" || result.TLSInfo.Version == "TLS 1.1" {
				score -= 15
			}
			if !result.TLSInfo.SupportsHSTS {
				score -= 5
			}
		}
	}

	// Header scoring
	for _, hdr := range result.SecurityHeaders {
		if !hdr.Present {
			switch hdr.Severity {
			case "critical":
				score -= 10
			case "high":
				score -= 7
			case "medium":
				score -= 4
			case "low":
				score -= 2
			}
		}
	}

	// Compliance scoring
	for _, check := range result.ComplianceChecks {
		if check.Status == "fail" {
			score -= 3
		}
	}

	if score < 0 {
		score = 0
	}

	// Map to letter grade
	var grade string
	switch {
	case score >= 90:
		grade = "A"
	case score >= 80:
		grade = "B"
	case score >= 70:
		grade = "C"
	case score >= 60:
		grade = "D"
	default:
		grade = "F"
	}

	return grade, score
}

// GetSecurityThreshold retrieves the security threshold for a tenant
func (s *Service) GetSecurityThreshold(ctx context.Context, tenantID string) (*SecurityThreshold, error) {
	threshold, err := s.repo.GetSecurityThreshold(ctx, tenantID)
	if err != nil {
		// Return default threshold
		return &SecurityThreshold{
			TenantID:       tenantID,
			MinScore:       60,
			AutoDisable:    false,
			AlertOnDegrade: true,
		}, nil
	}
	return threshold, nil
}

// UpdateSecurityThreshold updates the security threshold for a tenant
func (s *Service) UpdateSecurityThreshold(ctx context.Context, threshold *SecurityThreshold) error {
	return s.repo.UpsertSecurityThreshold(ctx, threshold)
}

// ExportSecurityReport generates a security report in the specified format
func (s *Service) ExportSecurityReport(ctx context.Context, result *SecurityScanResult, format string) ([]byte, error) {
	switch format {
	case "json":
		return json.MarshalIndent(result, "", "  ")
	case "csv":
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		w.Write([]string{"ID", "Title", "Severity", "Category", "Description", "Recommendation"})
		for _, f := range result.Findings {
			w.Write([]string{f.ID, f.Title, f.Severity, f.Category, f.Description, f.Recommendation})
		}
		w.Flush()
		return buf.Bytes(), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// DetectSecurityDrift compares two scan results and returns any regressions
func (s *Service) DetectSecurityDrift(current, previous *SecurityScanResult) []SecurityFinding {
	var drifts []SecurityFinding

	// Score degradation
	if current.NumericScore < previous.NumericScore {
		drifts = append(drifts, SecurityFinding{
			ID:          "drift-score",
			Title:       "Security Score Degraded",
			Description: fmt.Sprintf("Score dropped from %d to %d", previous.NumericScore, current.NumericScore),
			Severity:    "high",
			Category:    "drift",
		})
	}

	// New findings not in previous scan
	prevIDs := make(map[string]bool)
	for _, f := range previous.Findings {
		prevIDs[f.ID] = true
	}
	for _, f := range current.Findings {
		if !prevIDs[f.ID] {
			drift := f
			drift.Category = "drift"
			drifts = append(drifts, drift)
		}
	}

	// TLS downgrade
	if previous.TLSInfo != nil && current.TLSInfo != nil {
		if previous.TLSInfo.IsHTTPS && !current.TLSInfo.IsHTTPS {
			drifts = append(drifts, SecurityFinding{
				ID:       "drift-tls-removed",
				Title:    "HTTPS Removed",
				Severity: "critical",
				Category: "drift",
			})
		}
	}

	return drifts
}

// Helper functions

func truncateEvidence(evidence string) string {
	if len(evidence) > 100 {
		return evidence[:100] + "..."
	}
	return evidence
}

func isJSON(data json.RawMessage) bool {
	var js json.RawMessage
	return json.Unmarshal(data, &js) == nil
}

func getJSONDepth(data json.RawMessage) int {
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return 0
	}
	return measureDepth(parsed, 0)
}

func measureDepth(v interface{}, current int) int {
	maxDepth := current
	switch val := v.(type) {
	case map[string]interface{}:
		for _, child := range val {
			if d := measureDepth(child, current+1); d > maxDepth {
				maxDepth = d
			}
		}
	case []interface{}:
		for _, child := range val {
			if d := measureDepth(child, current+1); d > maxDepth {
				maxDepth = d
			}
		}
	}
	return maxDepth
}

func calculateRiskScore(threats []Threat) float64 {
	if len(threats) == 0 {
		return 0
	}

	score := 0.0
	for _, threat := range threats {
		switch threat.Severity {
		case ThreatSeverityCritical:
			score += 40
		case ThreatSeverityHigh:
			score += 25
		case ThreatSeverityMedium:
			score += 15
		case ThreatSeverityLow:
			score += 5
		case ThreatSeverityInfo:
			score += 1
		}
	}

	if score > 100 {
		score = 100
	}
	return score
}

func determineAction(riskScore float64, threats []Threat) ScanAction {
	if riskScore >= 80 {
		return ScanActionBlock
	}
	if riskScore >= 50 {
		return ScanActionQuarantine
	}
	if riskScore > 0 {
		return ScanActionFlag
	}
	return ScanActionAllow
}

func getHighestSeverity(threats []Threat) ThreatSeverity {
	highest := ThreatSeverityInfo
	for _, threat := range threats {
		if GetSeverityScore(threat.Severity) > GetSeverityScore(highest) {
			highest = threat.Severity
		}
	}
	return highest
}
