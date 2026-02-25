package waf

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

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
			s.logger.Error("CreateAlert error", map[string]interface{}{"tenant_id": tenantID, "error": err.Error()})
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
				s.logger.Error("IncrementRuleHitCount error", map[string]interface{}{"tenant_id": tenantID, "rule_id": rule.ID, "error": err.Error()})
			}
		}
	}

	return threats, nil
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

// ListScanResults lists scan results
func (s *Service) ListScanResults(ctx context.Context, tenantID string, limit, offset int) ([]ScanResult, int, error) {
	return s.repo.ListScanResults(ctx, tenantID, limit, offset)
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
