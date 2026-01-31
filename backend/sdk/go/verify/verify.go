// Package verify provides standalone, zero-dependency webhook signature verification.
// It supports HMAC-SHA256 signatures in the standard WaaS format.
package verify

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

var (
	ErrMissingSignature = errors.New("waas: missing signature header")
	ErrInvalidSignature = errors.New("waas: invalid signature")
	ErrTimestampExpired = errors.New("waas: timestamp outside tolerance")
	ErrMalformedHeader  = errors.New("waas: malformed signature header")
)

const (
	DefaultTimestampTolerance = 5 * time.Minute
	SignatureHeaderName       = "X-WaaS-Signature"
	TimestampHeaderName       = "X-WaaS-Timestamp"
)

// Verifier verifies webhook signatures using HMAC-SHA256.
type Verifier struct {
	secrets            []string
	timestampTolerance time.Duration
}

// Option configures a Verifier.
type Option func(*Verifier)

// WithTimestampTolerance sets the maximum allowed age of a webhook timestamp.
func WithTimestampTolerance(d time.Duration) Option {
	return func(v *Verifier) {
		v.timestampTolerance = d
	}
}

// WithSecrets adds additional signing secrets for key rotation support.
func WithSecrets(secrets ...string) Option {
	return func(v *Verifier) {
		v.secrets = append(v.secrets, secrets...)
	}
}

// New creates a Verifier with the given primary secret and options.
func New(secret string, opts ...Option) *Verifier {
	v := &Verifier{
		secrets:            []string{secret},
		timestampTolerance: DefaultTimestampTolerance,
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// Verify checks the webhook signature against the payload.
// signatureHeader is the value of X-WaaS-Signature (format: "v1=<hex>").
// timestamp is the value of X-WaaS-Timestamp (Unix seconds).
func (v *Verifier) Verify(payload []byte, signatureHeader, timestamp string) error {
	if signatureHeader == "" {
		return ErrMissingSignature
	}

	if err := v.verifyTimestamp(timestamp); err != nil {
		return err
	}

	sig, err := parseSignature(signatureHeader)
	if err != nil {
		return err
	}

	signedPayload := fmt.Sprintf("%s.%s", timestamp, string(payload))

	for _, secret := range v.secrets {
		expected := computeHMAC([]byte(secret), []byte(signedPayload))
		if hmac.Equal(sig, expected) {
			return nil
		}
	}

	return ErrInvalidSignature
}

// Sign generates a signature for the given payload. Used for testing.
func (v *Verifier) Sign(payload []byte, timestamp string) string {
	signedPayload := fmt.Sprintf("%s.%s", timestamp, string(payload))
	mac := computeHMAC([]byte(v.secrets[0]), []byte(signedPayload))
	return "v1=" + hex.EncodeToString(mac)
}

func (v *Verifier) verifyTimestamp(timestamp string) error {
	if timestamp == "" || v.timestampTolerance == 0 {
		return nil
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrMalformedHeader, err)
	}

	diff := time.Since(time.Unix(ts, 0))
	if math.Abs(diff.Seconds()) > v.timestampTolerance.Seconds() {
		return ErrTimestampExpired
	}

	return nil
}

func parseSignature(header string) ([]byte, error) {
	parts := strings.SplitN(header, "=", 2)
	if len(parts) != 2 {
		return nil, ErrMalformedHeader
	}
	decoded, err := hex.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformedHeader, err)
	}
	return decoded, nil
}

func computeHMAC(secret, payload []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	return mac.Sum(nil)
}
