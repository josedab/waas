package mtls

import "time"

// Certificate represents a managed TLS certificate
type Certificate struct {
	ID            string     `json:"id" db:"id"`
	TenantID      string     `json:"tenant_id" db:"tenant_id"`
	EndpointID    string     `json:"endpoint_id,omitempty" db:"endpoint_id"`
	Domain        string     `json:"domain" db:"domain"`
	Issuer        string     `json:"issuer" db:"issuer"`
	SerialNumber  string     `json:"serial_number" db:"serial_number"`
	NotBefore     time.Time  `json:"not_before" db:"not_before"`
	NotAfter      time.Time  `json:"not_after" db:"not_after"`
	Fingerprint   string     `json:"fingerprint" db:"fingerprint"`
	Status        CertStatus `json:"status" db:"status"`
	AutoRenew     bool       `json:"auto_renew" db:"auto_renew"`
	LastRenewedAt *time.Time `json:"last_renewed_at,omitempty" db:"last_renewed_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// CertStatus represents certificate status
type CertStatus string

const (
	CertStatusPending  CertStatus = "pending"
	CertStatusActive   CertStatus = "active"
	CertStatusExpiring CertStatus = "expiring"
	CertStatusExpired  CertStatus = "expired"
	CertStatusRevoked  CertStatus = "revoked"
)

// TLSPolicy defines per-endpoint TLS configuration
type TLSPolicy struct {
	ID               string    `json:"id" db:"id"`
	TenantID         string    `json:"tenant_id" db:"tenant_id"`
	EndpointID       string    `json:"endpoint_id" db:"endpoint_id"`
	RequireMTLS      bool      `json:"require_mtls" db:"require_mtls"`
	MinTLSVersion    string    `json:"min_tls_version" db:"min_tls_version"`
	AllowedCiphers   []string  `json:"allowed_ciphers"`
	CiphersJSON      string    `json:"-" db:"allowed_ciphers"`
	VerifyServerCert bool      `json:"verify_server_cert" db:"verify_server_cert"`
	CertificateID    string    `json:"certificate_id,omitempty" db:"certificate_id"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// CertificateRequest represents a request to issue a certificate
type CertificateRequest struct {
	Domain     string `json:"domain" binding:"required"`
	EndpointID string `json:"endpoint_id,omitempty"`
	AutoRenew  bool   `json:"auto_renew"`
}

// TLSPolicyRequest represents a request to create/update a TLS policy
type TLSPolicyRequest struct {
	EndpointID       string   `json:"endpoint_id" binding:"required"`
	RequireMTLS      bool     `json:"require_mtls"`
	MinTLSVersion    string   `json:"min_tls_version" binding:"omitempty,oneof=1.2 1.3"`
	AllowedCiphers   []string `json:"allowed_ciphers,omitempty"`
	VerifyServerCert bool     `json:"verify_server_cert"`
	CertificateID    string   `json:"certificate_id,omitempty"`
}

// CertificateInventory provides an overview of certificates for a tenant
type CertificateInventory struct {
	TotalCerts    int           `json:"total_certificates"`
	ActiveCerts   int           `json:"active_certificates"`
	ExpiringCerts int           `json:"expiring_certificates"`
	ExpiredCerts  int           `json:"expired_certificates"`
	Certificates  []Certificate `json:"certificates"`
}
