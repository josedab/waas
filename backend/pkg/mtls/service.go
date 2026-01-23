package mtls

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service provides mTLS certificate management functionality
type Service struct {
	repo         Repository
	renewBefore  time.Duration
}

// NewService creates a new mTLS service
func NewService(repo Repository) *Service {
	return &Service{
		repo:        repo,
		renewBefore: 30 * 24 * time.Hour, // 30 days before expiry
	}
}

// IssueCertificate issues a new managed certificate
func (s *Service) IssueCertificate(ctx context.Context, tenantID string, req *CertificateRequest) (*Certificate, error) {
	if req.Domain == "" {
		return nil, fmt.Errorf("domain is required")
	}

	serial := generateSerialNumber()
	fingerprint := generateFingerprint(tenantID, req.Domain)

	now := time.Now()
	cert := &Certificate{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		EndpointID:   req.EndpointID,
		Domain:       req.Domain,
		Issuer:       "WaaS Internal CA",
		SerialNumber: serial,
		NotBefore:    now,
		NotAfter:     now.Add(90 * 24 * time.Hour), // 90-day validity
		Fingerprint:  fingerprint,
		Status:       CertStatusActive,
		AutoRenew:    req.AutoRenew,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreateCertificate(ctx, cert); err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	return cert, nil
}

// GetCertificate retrieves a certificate by ID
func (s *Service) GetCertificate(ctx context.Context, tenantID, certID string) (*Certificate, error) {
	return s.repo.GetCertificate(ctx, tenantID, certID)
}

// ListCertificates lists all certificates for a tenant
func (s *Service) ListCertificates(ctx context.Context, tenantID string) ([]Certificate, error) {
	return s.repo.ListCertificates(ctx, tenantID)
}

// RevokeCertificate revokes a certificate
func (s *Service) RevokeCertificate(ctx context.Context, tenantID, certID string) error {
	cert, err := s.repo.GetCertificate(ctx, tenantID, certID)
	if err != nil {
		return err
	}

	cert.Status = CertStatusRevoked
	cert.UpdatedAt = time.Now()

	return s.repo.UpdateCertificate(ctx, cert)
}

// RenewCertificate renews an existing certificate
func (s *Service) RenewCertificate(ctx context.Context, tenantID, certID string) (*Certificate, error) {
	existing, err := s.repo.GetCertificate(ctx, tenantID, certID)
	if err != nil {
		return nil, err
	}

	// Mark old cert as expired
	existing.Status = CertStatusExpired
	existing.UpdatedAt = time.Now()
	if err := s.repo.UpdateCertificate(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to expire old certificate: %w", err)
	}

	// Issue a new cert for the same domain
	return s.IssueCertificate(ctx, tenantID, &CertificateRequest{
		Domain:     existing.Domain,
		EndpointID: existing.EndpointID,
		AutoRenew:  existing.AutoRenew,
	})
}

// GetInventory returns the certificate inventory for a tenant
func (s *Service) GetInventory(ctx context.Context, tenantID string) (*CertificateInventory, error) {
	certs, err := s.repo.ListCertificates(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	inv := &CertificateInventory{
		TotalCerts:   len(certs),
		Certificates: certs,
	}

	now := time.Now()
	for _, c := range certs {
		switch {
		case c.Status == CertStatusRevoked:
			// skip
		case c.NotAfter.Before(now):
			inv.ExpiredCerts++
		case c.NotAfter.Before(now.Add(s.renewBefore)):
			inv.ExpiringCerts++
		default:
			inv.ActiveCerts++
		}
	}

	return inv, nil
}

// CreateTLSPolicy creates a TLS policy for an endpoint
func (s *Service) CreateTLSPolicy(ctx context.Context, tenantID string, req *TLSPolicyRequest) (*TLSPolicy, error) {
	if req.MinTLSVersion == "" {
		req.MinTLSVersion = "1.2"
	}

	policy := &TLSPolicy{
		ID:               uuid.New().String(),
		TenantID:         tenantID,
		EndpointID:       req.EndpointID,
		RequireMTLS:      req.RequireMTLS,
		MinTLSVersion:    req.MinTLSVersion,
		AllowedCiphers:   req.AllowedCiphers,
		VerifyServerCert: req.VerifyServerCert,
		CertificateID:    req.CertificateID,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := s.repo.CreateTLSPolicy(ctx, policy); err != nil {
		return nil, fmt.Errorf("failed to create TLS policy: %w", err)
	}

	return policy, nil
}

// GetTLSPolicy retrieves a TLS policy for an endpoint
func (s *Service) GetTLSPolicy(ctx context.Context, tenantID, endpointID string) (*TLSPolicy, error) {
	return s.repo.GetTLSPolicy(ctx, tenantID, endpointID)
}

// ListTLSPolicies lists all TLS policies for a tenant
func (s *Service) ListTLSPolicies(ctx context.Context, tenantID string) ([]TLSPolicy, error) {
	return s.repo.ListTLSPolicies(ctx, tenantID)
}

// UpdateTLSPolicy updates a TLS policy
func (s *Service) UpdateTLSPolicy(ctx context.Context, tenantID, policyID string, req *TLSPolicyRequest) (*TLSPolicy, error) {
	policy, err := s.repo.GetTLSPolicy(ctx, tenantID, req.EndpointID)
	if err != nil {
		return nil, err
	}

	policy.RequireMTLS = req.RequireMTLS
	policy.MinTLSVersion = req.MinTLSVersion
	policy.AllowedCiphers = req.AllowedCiphers
	policy.VerifyServerCert = req.VerifyServerCert
	policy.CertificateID = req.CertificateID
	policy.UpdatedAt = time.Now()

	if err := s.repo.UpdateTLSPolicy(ctx, policy); err != nil {
		return nil, fmt.Errorf("failed to update TLS policy: %w", err)
	}

	return policy, nil
}

// DeleteTLSPolicy deletes a TLS policy
func (s *Service) DeleteTLSPolicy(ctx context.Context, tenantID, policyID string) error {
	return s.repo.DeleteTLSPolicy(ctx, tenantID, policyID)
}

// CheckExpiringCerts finds certificates expiring soon and auto-renews if enabled
func (s *Service) CheckExpiringCerts(ctx context.Context) (renewed int, err error) {
	certs, err := s.repo.ListExpiringCertificates(ctx, 30)
	if err != nil {
		return 0, err
	}

	for _, cert := range certs {
		if cert.AutoRenew && cert.Status == CertStatusActive {
			if _, err := s.RenewCertificate(ctx, cert.TenantID, cert.ID); err == nil {
				renewed++
			}
		}
	}

	return renewed, nil
}

func generateSerialNumber() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateFingerprint(tenantID, domain string) string {
	h := sha256.New()
	h.Write([]byte(tenantID + ":" + domain + ":" + time.Now().String()))
	return hex.EncodeToString(h.Sum(nil))[:40]
}
