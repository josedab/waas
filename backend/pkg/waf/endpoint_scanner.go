package waf

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

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
