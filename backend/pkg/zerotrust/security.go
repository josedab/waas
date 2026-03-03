// Package zerotrust provides mTLS, certificate pinning, and request signing
package zerotrust

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"hash"
	"net/http"

	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/josedab/waas/pkg/utils"
)

var (
	ErrInvalidCertificate     = errors.New("invalid certificate")
	ErrCertificateExpired     = errors.New("certificate expired")
	ErrSignatureVerifyFailed  = errors.New("signature verification failed")
	ErrKeyNotFound            = errors.New("signing key not found")
	ErrUnsupportedAlgorithm   = errors.New("unsupported algorithm")
	ErrCertificatePinMismatch = errors.New("certificate pin mismatch")
)

// Certificate represents a parsed X.509 certificate
type Certificate struct {
	ID              string    `json:"id"`
	EndpointID      string    `json:"endpoint_id"`
	TenantID        string    `json:"tenant_id"`
	Type            string    `json:"type"`
	PEM             string    `json:"pem"`
	PrivateKeyPEM   string    `json:"private_key_pem,omitempty"`
	CommonName      string    `json:"common_name"`
	Issuer          string    `json:"issuer"`
	SerialNumber    string    `json:"serial_number"`
	FingerprintSHA256 string  `json:"fingerprint_sha256"`
	NotBefore       time.Time `json:"not_before"`
	NotAfter        time.Time `json:"not_after"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// SigningKey represents a key for request signing
type SigningKey struct {
	ID                  string     `json:"id"`
	TenantID            string     `json:"tenant_id"`
	Name                string     `json:"name"`
	Description         string     `json:"description,omitempty"`
	KeyType             string     `json:"key_type"`
	PublicKey           string     `json:"public_key,omitempty"`
	PrivateKeyEncrypted string     `json:"-"`
	KeyID               string     `json:"key_id"`
	Status              string     `json:"status"`
	Version             int        `json:"version"`
	ExpiresAt           *time.Time `json:"expires_at,omitempty"`
	RotatedFrom         *string    `json:"rotated_from,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// SecurityProfile represents endpoint security configuration
type SecurityProfile struct {
	ID                   string   `json:"id"`
	EndpointID           string   `json:"endpoint_id"`
	TenantID             string   `json:"tenant_id"`
	MTLSEnabled          bool     `json:"mtls_enabled"`
	ClientCertificateID  *string  `json:"client_certificate_id,omitempty"`
	VerifyServerCert     bool     `json:"verify_server_cert"`
	PinningEnabled       bool     `json:"pinning_enabled"`
	PinningMode          *string  `json:"pinning_mode,omitempty"`
	PinnedCertificateIDs []string `json:"pinned_certificate_ids,omitempty"`
	SigningEnabled       bool     `json:"signing_enabled"`
	SigningKeyID         *string  `json:"signing_key_id,omitempty"`
	SigningAlgorithm     *string  `json:"signing_algorithm,omitempty"`
	SignatureHeader      string   `json:"signature_header"`
	TimestampHeader      string   `json:"timestamp_header"`
	SignatureFormat      string   `json:"signature_format"`
	IncludeBody          bool     `json:"include_body"`
	IncludeHeaders       []string `json:"include_headers,omitempty"`
	RequireHTTPS         bool     `json:"require_https"`
	MinTLSVersion        string   `json:"min_tls_version"`
	AllowedCipherSuites  []string `json:"allowed_cipher_suites,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// NewDefaultSecurityProfile returns a SecurityProfile with secure defaults.
// VerifyServerCert defaults to true; callers must explicitly disable it.
func NewDefaultSecurityProfile() SecurityProfile {
	return SecurityProfile{
		VerifyServerCert: true,
		RequireHTTPS:     true,
		MinTLSVersion:    "1.2",
		SignatureHeader:  "X-Webhook-Signature",
		TimestampHeader:  "X-Webhook-Timestamp",
		SignatureFormat:  "hex",
	}
}

// CertificateManager handles certificate operations
type CertificateManager struct {
	logger *utils.Logger
}

// NewCertificateManager creates a new certificate manager
func NewCertificateManager() *CertificateManager {
	return &CertificateManager{logger: utils.NewLogger("zerotrust")}
}

// ParseCertificate parses a PEM-encoded certificate
func (m *CertificateManager) ParseCertificate(pemData string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, ErrInvalidCertificate
	}

	return x509.ParseCertificate(block.Bytes)
}

// GetFingerprint returns SHA256 fingerprint of a certificate
func (m *CertificateManager) GetFingerprint(cert *x509.Certificate) string {
	hash := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(hash[:])
}

// GetSPKIFingerprint returns SHA256 fingerprint of Subject Public Key Info
func (m *CertificateManager) GetSPKIFingerprint(cert *x509.Certificate) string {
	hash := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
	return hex.EncodeToString(hash[:])
}

// ValidateCertificate checks if a certificate is valid
func (m *CertificateManager) ValidateCertificate(cert *x509.Certificate) error {
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("certificate not yet valid")
	}
	if now.After(cert.NotAfter) {
		return ErrCertificateExpired
	}
	return nil
}

// BuildTLSConfig creates a TLS configuration for an endpoint
func (m *CertificateManager) BuildTLSConfig(profile *SecurityProfile, clientCert *Certificate, pinnedCerts []*Certificate) (*tls.Config, error) {
	if !profile.VerifyServerCert {
		m.logger.Warn("VerifyServerCert=false requested but insecure TLS is not permitted — verification remains enabled", map[string]interface{}{"endpoint_id": profile.EndpointID})
	}
	config := &tls.Config{
		InsecureSkipVerify: false,
	}

	// Set minimum TLS version (only TLS 1.2+ allowed)
	switch profile.MinTLSVersion {
	case "1.3":
		config.MinVersion = tls.VersionTLS13
	default:
		config.MinVersion = tls.VersionTLS12
	}

	// Configure client certificate for mTLS
	if profile.MTLSEnabled && clientCert != nil {
		cert, err := tls.X509KeyPair([]byte(clientCert.PEM), []byte(clientCert.PrivateKeyPEM))
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		config.Certificates = []tls.Certificate{cert}
	}

	// Configure certificate pinning
	if profile.PinningEnabled && len(pinnedCerts) > 0 {
		config.VerifyPeerCertificate = m.buildPinVerifier(profile.PinningMode, pinnedCerts)
	}

	return config, nil
}

func (m *CertificateManager) buildPinVerifier(mode *string, pinnedCerts []*Certificate) func([][]byte, [][]*x509.Certificate) error {
	pins := make(map[string]bool)
	for _, pc := range pinnedCerts {
		pins[pc.FingerprintSHA256] = true
	}

	return func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		for _, rawCert := range rawCerts {
			cert, err := x509.ParseCertificate(rawCert)
			if err != nil {
				continue
			}

			var fingerprint string
			if mode != nil && *mode == "spki" {
				fingerprint = m.GetSPKIFingerprint(cert)
			} else {
				fingerprint = m.GetFingerprint(cert)
			}

			if pins[fingerprint] {
				return nil
			}
		}
		return ErrCertificatePinMismatch
	}
}

// RequestSigner handles webhook request signing
type RequestSigner struct {
	keys map[string]*SigningKey
}

// NewRequestSigner creates a new request signer
func NewRequestSigner() *RequestSigner {
	return &RequestSigner{
		keys: make(map[string]*SigningKey),
	}
}

// RegisterKey registers a signing key
func (s *RequestSigner) RegisterKey(key *SigningKey) {
	s.keys[key.KeyID] = key
}

// SignRequest signs an HTTP request
func (s *RequestSigner) SignRequest(req *http.Request, profile *SecurityProfile, body []byte, keySecret []byte) (string, error) {
	if profile.SigningKeyID == nil {
		return "", ErrKeyNotFound
	}

	// Build signed payload
	payload := s.buildSignedPayload(req, profile, body)

	// Generate signature
	algorithm := "hmac_sha256"
	if profile.SigningAlgorithm != nil {
		algorithm = *profile.SigningAlgorithm
	}

	signature, err := s.computeSignature(algorithm, payload, keySecret)
	if err != nil {
		return "", err
	}

	// Format signature
	if profile.SignatureFormat == "hex" {
		return hex.EncodeToString(signature), nil
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func (s *RequestSigner) buildSignedPayload(req *http.Request, profile *SecurityProfile, body []byte) []byte {
	var parts []string

	// Add timestamp if present
	if ts := req.Header.Get(profile.TimestampHeader); ts != "" {
		parts = append(parts, ts)
	}

	// Add specified headers
	if len(profile.IncludeHeaders) > 0 {
		sort.Strings(profile.IncludeHeaders)
		for _, h := range profile.IncludeHeaders {
			if v := req.Header.Get(h); v != "" {
				parts = append(parts, fmt.Sprintf("%s:%s", strings.ToLower(h), v))
			}
		}
	}

	// Add body if configured
	if profile.IncludeBody && len(body) > 0 {
		parts = append(parts, string(body))
	}

	return []byte(strings.Join(parts, "."))
}

func (s *RequestSigner) computeSignature(algorithm string, payload, secret []byte) ([]byte, error) {
	switch algorithm {
	case "hmac_sha256":
		mac := hmac.New(sha256.New, secret)
		mac.Write(payload)
		return mac.Sum(nil), nil

	case "hmac_sha512":
		mac := hmac.New(sha512.New, secret)
		mac.Write(payload)
		return mac.Sum(nil), nil

	default:
		return nil, ErrUnsupportedAlgorithm
	}
}

// VerifySignature verifies a request signature
func (s *RequestSigner) VerifySignature(signature string, payload, secret []byte, algorithm, format string) error {
	var sigBytes []byte
	var err error

	if format == "hex" {
		sigBytes, err = hex.DecodeString(signature)
	} else {
		sigBytes, err = base64.StdEncoding.DecodeString(signature)
	}
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	expected, err := s.computeSignature(algorithm, payload, secret)
	if err != nil {
		return err
	}

	if !hmac.Equal(sigBytes, expected) {
		return ErrSignatureVerifyFailed
	}

	return nil
}

// AsymmetricSigner handles RSA/Ed25519 signing
type AsymmetricSigner struct{}

// NewAsymmetricSigner creates a new asymmetric signer
func NewAsymmetricSigner() *AsymmetricSigner {
	return &AsymmetricSigner{}
}

// Sign creates a signature using the private key
func (s *AsymmetricSigner) Sign(algorithm string, payload []byte, privateKeyPEM string) ([]byte, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, errors.New("failed to parse private key PEM")
	}

	switch algorithm {
	case "rsa_sha256":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse RSA key: %w", err)
			}
		}

		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("key is not RSA")
		}

		hashed := sha256.Sum256(payload)
		return rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, hashed[:])

	case "ed25519":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Ed25519 key: %w", err)
		}

		edKey, ok := key.(ed25519.PrivateKey)
		if !ok {
			return nil, errors.New("key is not Ed25519")
		}

		return ed25519.Sign(edKey, payload), nil

	default:
		return nil, ErrUnsupportedAlgorithm
	}
}

// Verify verifies a signature using the public key
func (s *AsymmetricSigner) Verify(algorithm string, payload, signature []byte, publicKeyPEM string) error {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return errors.New("failed to parse public key PEM")
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	switch algorithm {
	case "rsa_sha256":
		rsaKey, ok := pubKey.(*rsa.PublicKey)
		if !ok {
			return errors.New("key is not RSA")
		}

		hashed := sha256.Sum256(payload)
		return rsa.VerifyPKCS1v15(rsaKey, crypto.SHA256, hashed[:], signature)

	case "ecdsa_sha256":
		ecdsaKey, ok := pubKey.(*ecdsa.PublicKey)
		if !ok {
			return errors.New("key is not ECDSA")
		}

		hashed := sha256.Sum256(payload)
		if !ecdsa.VerifyASN1(ecdsaKey, hashed[:], signature) {
			return ErrSignatureVerifyFailed
		}
		return nil

	case "ed25519":
		edKey, ok := pubKey.(ed25519.PublicKey)
		if !ok {
			return errors.New("key is not Ed25519")
		}

		if !ed25519.Verify(edKey, payload, signature) {
			return ErrSignatureVerifyFailed
		}
		return nil

	default:
		return ErrUnsupportedAlgorithm
	}
}

// WebhookSecurityMiddleware provides security middleware for webhook delivery
type WebhookSecurityMiddleware struct {
	certManager *CertificateManager
	signer      *RequestSigner
}

// NewWebhookSecurityMiddleware creates security middleware
func NewWebhookSecurityMiddleware() *WebhookSecurityMiddleware {
	return &WebhookSecurityMiddleware{
		certManager: NewCertificateManager(),
		signer:      NewRequestSigner(),
	}
}

// PrepareRequest adds security headers and configures TLS
func (m *WebhookSecurityMiddleware) PrepareRequest(req *http.Request, profile *SecurityProfile, body []byte, signingSecret []byte) error {
	// Add timestamp
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	req.Header.Set(profile.TimestampHeader, timestamp)

	// Sign request if enabled
	if profile.SigningEnabled && len(signingSecret) > 0 {
		signature, err := m.signer.SignRequest(req, profile, body, signingSecret)
		if err != nil {
			return err
		}
		req.Header.Set(profile.SignatureHeader, signature)
	}

	return nil
}

// HashPayload creates a content hash for integrity verification
func HashPayload(body []byte, algorithm string) string {
	var h hash.Hash
	switch algorithm {
	case "sha512":
		h = sha512.New()
	default:
		h = sha256.New()
	}
	h.Write(body)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
