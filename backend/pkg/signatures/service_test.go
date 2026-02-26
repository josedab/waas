package signatures

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"
)

// --- In-memory repository for testing ---

type memoryRepository struct {
	schemes   map[string]*SignatureScheme
	keys      map[string]*SigningKey
	rotations map[string]*KeyRotation
	stats     map[string]*SchemeStats
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		schemes:   make(map[string]*SignatureScheme),
		keys:      make(map[string]*SigningKey),
		rotations: make(map[string]*KeyRotation),
		stats:     make(map[string]*SchemeStats),
	}
}

func (r *memoryRepository) SaveScheme(_ context.Context, s *SignatureScheme) error {
	r.schemes[s.ID] = s
	return nil
}
func (r *memoryRepository) GetScheme(_ context.Context, tenantID, schemeID string) (*SignatureScheme, error) {
	s, ok := r.schemes[schemeID]
	if !ok || s.TenantID != tenantID {
		return nil, fmt.Errorf("signature scheme not found")
	}
	return s, nil
}
func (r *memoryRepository) ListSchemes(_ context.Context, tenantID string) ([]SignatureScheme, error) {
	var out []SignatureScheme
	for _, s := range r.schemes {
		if s.TenantID == tenantID {
			out = append(out, *s)
		}
	}
	return out, nil
}
func (r *memoryRepository) DeleteScheme(_ context.Context, tenantID, schemeID string) error {
	delete(r.schemes, schemeID)
	return nil
}
func (r *memoryRepository) SaveKey(_ context.Context, k *SigningKey) error {
	r.keys[k.ID] = k
	return nil
}
func (r *memoryRepository) GetKey(_ context.Context, keyID string) (*SigningKey, error) {
	k, ok := r.keys[keyID]
	if !ok {
		return nil, fmt.Errorf("signing key not found")
	}
	return k, nil
}
func (r *memoryRepository) GetPrimaryKey(_ context.Context, schemeID string) (*SigningKey, error) {
	for _, k := range r.keys {
		if k.SchemeID == schemeID && k.Status == KeyPrimary {
			return k, nil
		}
	}
	// Fallback to active
	for _, k := range r.keys {
		if k.SchemeID == schemeID && k.Status == KeyActive {
			return k, nil
		}
	}
	return nil, fmt.Errorf("no active signing key found")
}
func (r *memoryRepository) ListKeys(_ context.Context, schemeID string) ([]SigningKey, error) {
	var out []SigningKey
	for _, k := range r.keys {
		if k.SchemeID == schemeID {
			out = append(out, *k)
		}
	}
	return out, nil
}
func (r *memoryRepository) UpdateKeyStatus(_ context.Context, keyID string, status KeyStatus) error {
	if k, ok := r.keys[keyID]; ok {
		k.Status = status
	}
	return nil
}
func (r *memoryRepository) UpdateKeyUsage(_ context.Context, keyID string) error {
	if k, ok := r.keys[keyID]; ok {
		now := time.Now()
		k.LastUsedAt = &now
		k.UsageCount++
	}
	return nil
}
func (r *memoryRepository) SaveRotation(_ context.Context, rot *KeyRotation) error {
	r.rotations[rot.ID] = rot
	return nil
}
func (r *memoryRepository) GetRotation(_ context.Context, rotID string) (*KeyRotation, error) {
	rot, ok := r.rotations[rotID]
	if !ok {
		return nil, fmt.Errorf("rotation not found")
	}
	return rot, nil
}
func (r *memoryRepository) ListRotations(_ context.Context, schemeID string) ([]KeyRotation, error) {
	var out []KeyRotation
	for _, rot := range r.rotations {
		if rot.SchemeID == schemeID {
			out = append(out, *rot)
		}
	}
	return out, nil
}
func (r *memoryRepository) GetPendingRotations(_ context.Context) ([]KeyRotation, error) {
	var out []KeyRotation
	for _, rot := range r.rotations {
		if rot.Status == RotationScheduledStatus || rot.Status == RotationInProgress {
			out = append(out, *rot)
		}
	}
	return out, nil
}
func (r *memoryRepository) GetSchemeStats(_ context.Context, schemeID string) (*SchemeStats, error) {
	s, ok := r.stats[schemeID]
	if !ok {
		return &SchemeStats{SchemeID: schemeID}, nil
	}
	return s, nil
}
func (r *memoryRepository) IncrementSignCount(_ context.Context, schemeID string) error {
	s := r.stats[schemeID]
	if s == nil {
		s = &SchemeStats{SchemeID: schemeID}
		r.stats[schemeID] = s
	}
	s.TotalSigned++
	return nil
}
func (r *memoryRepository) IncrementVerifyCount(_ context.Context, schemeID string, success bool) error {
	s := r.stats[schemeID]
	if s == nil {
		s = &SchemeStats{SchemeID: schemeID}
		r.stats[schemeID] = s
	}
	if success {
		s.TotalVerified++
	} else {
		s.TotalFailed++
	}
	return nil
}

// --- Tests ---

func newTestService() (*Service, *memoryRepository) {
	repo := newMemoryRepository()
	svc := NewService(repo, DefaultServiceConfig())
	return svc, repo
}

func TestCreateScheme(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "test-scheme",
		Type: TypeGitHub,
	})
	if err != nil {
		t.Fatalf("CreateScheme: %v", err)
	}
	if scheme.ID == "" {
		t.Fatal("expected non-empty scheme ID")
	}
	if scheme.Algorithm != AlgorithmHMACSHA256 {
		t.Fatalf("expected default algorithm hmac-sha256, got %s", scheme.Algorithm)
	}
	if scheme.Status != SchemeActive {
		t.Fatalf("expected status active, got %s", scheme.Status)
	}
}

func TestCreateScheme_MaxSchemesLimit(t *testing.T) {
	svc, _ := newTestService()
	svc.config.MaxSchemesPerTenant = 2
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		_, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
			Name: fmt.Sprintf("scheme-%d", i),
			Type: TypeCustomHMAC,
		})
		if err != nil {
			t.Fatalf("CreateScheme %d: %v", i, err)
		}
	}

	_, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "overflow",
		Type: TypeCustomHMAC,
	})
	if err == nil {
		t.Fatal("expected error when exceeding max schemes")
	}
}

func TestKeyGeneration(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "key-test",
		Type: TypeCustomHMAC,
	})
	if err != nil {
		t.Fatalf("CreateScheme: %v", err)
	}

	// CreateScheme generates an initial key
	keys, err := repo.ListKeys(ctx, scheme.ID)
	if err != nil {
		t.Fatalf("ListKeys: %v", err)
	}
	if len(keys) == 0 {
		t.Fatal("expected at least one key after scheme creation")
	}

	key := keys[0]
	if key.Status != KeyPrimary {
		t.Fatalf("expected primary key, got %s", key.Status)
	}
	if key.SecretKey == "" {
		t.Fatal("expected non-empty secret key")
	}
	if key.Fingerprint == "" {
		t.Fatal("expected non-empty fingerprint")
	}
}

func TestSignAndVerify_HMACSHA256(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name:      "hmac-test",
		Type:      TypeCustomHMAC,
		Algorithm: AlgorithmHMACSHA256,
	})
	if err != nil {
		t.Fatalf("CreateScheme: %v", err)
	}

	payload := []byte(`{"event":"test","data":"hello"}`)
	now := time.Now()

	signResult, err := svc.Sign(ctx, "tenant-1", &SignatureRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Timestamp: &now,
	})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if signResult.Signature == "" {
		t.Fatal("expected non-empty signature")
	}
	if signResult.Algorithm != AlgorithmHMACSHA256 {
		t.Fatalf("expected algorithm hmac-sha256, got %s", signResult.Algorithm)
	}

	// Verify the same signature
	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Signature: signResult.Signature,
		Timestamp: &now,
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !verifyResult.Valid {
		t.Fatalf("expected valid signature, got error: %s", verifyResult.Error)
	}
}

func TestSignAndVerify_HMACSHA512(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name:      "sha512-test",
		Type:      TypeCustomHMAC,
		Algorithm: AlgorithmHMACSHA512,
	})
	if err != nil {
		t.Fatalf("CreateScheme: %v", err)
	}

	payload := []byte(`{"event":"test512"}`)
	now := time.Now()

	signResult, err := svc.Sign(ctx, "tenant-1", &SignatureRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Timestamp: &now,
	})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Signature: signResult.Signature,
		Timestamp: &now,
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !verifyResult.Valid {
		t.Fatalf("expected valid sha512 signature, got error: %s", verifyResult.Error)
	}
}

func TestVerify_InvalidSignature(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "invalid-sig-test",
		Type: TypeCustomHMAC,
	})
	if err != nil {
		t.Fatalf("CreateScheme: %v", err)
	}

	payload := []byte(`{"event":"test"}`)
	now := time.Now()

	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Signature: "definitely-wrong-signature",
		Timestamp: &now,
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if verifyResult.Valid {
		t.Fatal("expected invalid signature")
	}
	if verifyResult.ErrorCode != "INVALID_SIGNATURE" {
		t.Fatalf("expected INVALID_SIGNATURE error code, got %s", verifyResult.ErrorCode)
	}
}

func TestVerify_TimestampExpired(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "ts-test",
		Type: TypeCustomHMAC,
		Config: &SignatureConfig{
			SignatureHeader:       "X-Sig",
			SignatureFormat:       "hex",
			IncludeTimestamp:      true,
			SignedPayloadTemplate: "{timestamp}.{body}",
			TimestampToleranceSec: 60,
		},
	})
	if err != nil {
		t.Fatalf("CreateScheme: %v", err)
	}

	payload := []byte(`{"event":"test"}`)
	oldTime := time.Now().Add(-2 * time.Hour)

	signResult, err := svc.Sign(ctx, "tenant-1", &SignatureRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Timestamp: &oldTime,
	})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Signature: signResult.Signature,
		Timestamp: &oldTime,
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if verifyResult.Valid {
		t.Fatal("expected timestamp expired rejection")
	}
	if verifyResult.ErrorCode != "TIMESTAMP_EXPIRED" {
		t.Fatalf("expected TIMESTAMP_EXPIRED, got %s", verifyResult.ErrorCode)
	}
}

func TestKeyRotation(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "rotation-test",
		Type: TypeCustomHMAC,
	})
	if err != nil {
		t.Fatalf("CreateScheme: %v", err)
	}

	// Sign with original key
	payload := []byte(`{"event":"before-rotation"}`)
	now := time.Now()
	signBefore, err := svc.Sign(ctx, "tenant-1", &SignatureRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Timestamp: &now,
	})
	if err != nil {
		t.Fatalf("Sign before rotation: %v", err)
	}

	// Rotate key (immediate)
	rotation, err := svc.RotateKey(ctx, "tenant-1", scheme.ID, &RotateKeyRequest{
		Immediate: true,
		Reason:    "test rotation",
	})
	if err != nil {
		t.Fatalf("RotateKey: %v", err)
	}
	if rotation.Status != RotationCompleted {
		t.Fatalf("expected completed rotation, got %s", rotation.Status)
	}

	// Verify old signature still works (old key should be in rotating state)
	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Signature: signBefore.Signature,
		Timestamp: &now,
	})
	if err != nil {
		t.Fatalf("Verify after rotation: %v", err)
	}
	if !verifyResult.Valid {
		t.Fatalf("expected old signature to still verify during overlap, error: %s", verifyResult.Error)
	}

	// Sign with new key
	payload2 := []byte(`{"event":"after-rotation"}`)
	now2 := time.Now()
	signAfter, err := svc.Sign(ctx, "tenant-1", &SignatureRequest{
		SchemeID:  scheme.ID,
		Payload:   payload2,
		Timestamp: &now2,
	})
	if err != nil {
		t.Fatalf("Sign after rotation: %v", err)
	}

	verifyResult2, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   payload2,
		Signature: signAfter.Signature,
		Timestamp: &now2,
	})
	if err != nil {
		t.Fatalf("Verify new signature: %v", err)
	}
	if !verifyResult2.Valid {
		t.Fatalf("expected new signature to verify, error: %s", verifyResult2.Error)
	}
}

func TestComputeSignature_UnsupportedAlgorithm(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.computeSignature("unsupported-algo", "secret", []byte("payload"))
	if err == nil {
		t.Fatal("expected error for unsupported algorithm")
	}
	if !strings.Contains(err.Error(), "unsupported algorithm") {
		t.Fatalf("expected 'unsupported algorithm' error, got: %v", err)
	}
}

func TestComputeSignature_HMACSHA256(t *testing.T) {
	svc, _ := newTestService()
	secret := "test-secret"
	payload := []byte("test-payload")

	sig, err := svc.computeSignature(AlgorithmHMACSHA256, secret, payload)
	if err != nil {
		t.Fatalf("computeSignature: %v", err)
	}

	// Verify against standard library
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	if sig != expected {
		t.Fatalf("signature mismatch: got %s, want %s", sig, expected)
	}
}

func TestFormatSignature_Base64(t *testing.T) {
	svc, _ := newTestService()

	scheme := &SignatureScheme{
		Type: TypeCustomHMAC,
		Config: SignatureConfig{
			SignatureFormat: "base64",
			SignaturePrefix: "v1=",
		},
	}

	// hex-encoded input
	hexSig := hex.EncodeToString([]byte("test-signature-data"))
	result := svc.formatSignature(scheme, hexSig, time.Now())

	if !strings.HasPrefix(result, "v1=") {
		t.Fatalf("expected v1= prefix, got: %s", result)
	}

	// Decode the base64 portion
	b64Part := strings.TrimPrefix(result, "v1=")
	decoded, err := base64.StdEncoding.DecodeString(b64Part)
	if err != nil {
		t.Fatalf("failed to decode base64: %v", err)
	}
	if string(decoded) != "test-signature-data" {
		t.Fatalf("unexpected decoded value: %s", string(decoded))
	}
}

func TestFormatSignature_Stripe(t *testing.T) {
	svc, _ := newTestService()

	scheme := &SignatureScheme{
		Type: TypeStripe,
		Config: SignatureConfig{
			SignatureFormat: "hex",
			SignaturePrefix: "v1=",
		},
	}

	ts := time.Unix(1234567890, 0)
	result := svc.formatSignature(scheme, "abcdef1234", ts)

	if !strings.HasPrefix(result, "t=1234567890,") {
		t.Fatalf("expected Stripe-style format, got: %s", result)
	}
}

func TestParseSignature_Stripe(t *testing.T) {
	svc, _ := newTestService()

	scheme := &SignatureScheme{
		Type: TypeStripe,
		Config: SignatureConfig{
			SignaturePrefix: "v1=",
		},
	}

	parsed := svc.parseSignature(scheme, "t=1234567890,v1=abcdef1234")
	if parsed != "abcdef1234" {
		t.Fatalf("expected abcdef1234, got: %s", parsed)
	}
}

func TestGetSupportedSchemes(t *testing.T) {
	schemes := GetSupportedSchemes()
	if len(schemes) == 0 {
		t.Fatal("expected at least one supported scheme")
	}

	typeSet := make(map[SignatureType]bool)
	for _, s := range schemes {
		typeSet[s.Type] = true
	}

	for _, expected := range []SignatureType{TypeStandardWebhooks, TypeStripe, TypeGitHub, TypeSlack, TypeCustomHMAC} {
		if !typeSet[expected] {
			t.Fatalf("expected supported type %s", expected)
		}
	}
}

func TestGetDefaultConfig(t *testing.T) {
	tests := []struct {
		sigType        SignatureType
		expectedHeader string
	}{
		{TypeStandardWebhooks, "Webhook-Signature"},
		{TypeStripe, "Stripe-Signature"},
		{TypeGitHub, "X-Hub-Signature-256"},
		{TypeSlack, "X-Slack-Signature"},
	}

	for _, tc := range tests {
		cfg := GetDefaultConfig(tc.sigType)
		if cfg.SignatureHeader != tc.expectedHeader {
			t.Fatalf("type %s: expected header %s, got %s", tc.sigType, tc.expectedHeader, cfg.SignatureHeader)
		}
	}
}

func TestUpdateScheme(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "original",
		Type: TypeCustomHMAC,
	})
	if err != nil {
		t.Fatalf("CreateScheme: %v", err)
	}

	newName := "updated"
	updated, err := svc.UpdateScheme(ctx, "tenant-1", scheme.ID, &UpdateSchemeRequest{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("UpdateScheme: %v", err)
	}
	if updated.Name != "updated" {
		t.Fatalf("expected name 'updated', got '%s'", updated.Name)
	}
}

func TestDeleteScheme(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "to-delete",
		Type: TypeCustomHMAC,
	})
	if err != nil {
		t.Fatalf("CreateScheme: %v", err)
	}

	err = svc.DeleteScheme(ctx, "tenant-1", scheme.ID)
	if err != nil {
		t.Fatalf("DeleteScheme: %v", err)
	}

	_, err = svc.GetScheme(ctx, "tenant-1", scheme.ID)
	if err == nil {
		t.Fatal("expected error after deleting scheme")
	}
}

func TestBuildSignedPayload(t *testing.T) {
	svc, _ := newTestService()

	scheme := &SignatureScheme{
		Config: SignatureConfig{
			SignedPayloadTemplate: "{id}.{timestamp}.{body}",
		},
	}

	ts := time.Unix(1234567890, 0)
	result := svc.buildSignedPayload(scheme, []byte("hello"), ts, "msg-123")

	expected := "msg-123.1234567890.hello"
	if string(result) != expected {
		t.Fatalf("expected %q, got %q", expected, string(result))
	}
}

func TestBuildSignedPayload_NoTemplate(t *testing.T) {
	svc, _ := newTestService()

	scheme := &SignatureScheme{
		Config: SignatureConfig{},
	}

	payload := []byte("raw-payload")
	result := svc.buildSignedPayload(scheme, payload, time.Now(), "")

	if string(result) != "raw-payload" {
		t.Fatalf("expected raw payload, got %q", string(result))
	}
}

// --- Benchmarks ---

func BenchmarkSign_HMACSHA256(b *testing.B) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-bench", &CreateSchemeRequest{
		Name:      "bench-scheme",
		Type:      TypeCustomHMAC,
		Algorithm: AlgorithmHMACSHA256,
	})
	if err != nil {
		b.Fatalf("CreateScheme: %v", err)
	}

	payload := []byte(`{"event":"benchmark","data":{"id":12345,"name":"test"}}`)
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.Sign(ctx, "tenant-bench", &SignatureRequest{
			SchemeID:  scheme.ID,
			Payload:   payload,
			Timestamp: &now,
		})
		if err != nil {
			b.Fatalf("Sign: %v", err)
		}
	}
}

func BenchmarkVerify_HMACSHA256(b *testing.B) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-bench", &CreateSchemeRequest{
		Name:      "bench-verify",
		Type:      TypeCustomHMAC,
		Algorithm: AlgorithmHMACSHA256,
	})
	if err != nil {
		b.Fatalf("CreateScheme: %v", err)
	}

	payload := []byte(`{"event":"benchmark","data":{"id":12345,"name":"test"}}`)
	now := time.Now()

	signResult, err := svc.Sign(ctx, "tenant-bench", &SignatureRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Timestamp: &now,
	})
	if err != nil {
		b.Fatalf("Sign: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := svc.Verify(ctx, "tenant-bench", &VerifyRequest{
			SchemeID:  scheme.ID,
			Payload:   payload,
			Signature: signResult.Signature,
			Timestamp: &now,
		})
		if err != nil {
			b.Fatalf("Verify: %v", err)
		}
		if !result.Valid {
			b.Fatalf("expected valid signature")
		}
	}
}

func BenchmarkComputeSignature_HMACSHA256(b *testing.B) {
	svc, _ := newTestService()
	secret := "bench-secret-key-value"
	payload := []byte(`{"event":"benchmark","data":{"id":12345,"name":"test"}}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.computeSignature(AlgorithmHMACSHA256, secret, payload)
		if err != nil {
			b.Fatalf("computeSignature: %v", err)
		}
	}
}

func BenchmarkComputeSignature_HMACSHA512(b *testing.B) {
	svc, _ := newTestService()
	secret := "bench-secret-key-value"
	payload := []byte(`{"event":"benchmark","data":{"id":12345,"name":"test"}}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.computeSignature(AlgorithmHMACSHA512, secret, payload)
		if err != nil {
			b.Fatalf("computeSignature: %v", err)
		}
	}
}
