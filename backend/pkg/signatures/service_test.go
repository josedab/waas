package signatures

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// --- Helpers ---

func newTestService() (*Service, *memoryRepository) {
	repo := newMemoryRepository()
	svc := NewService(repo, DefaultServiceConfig())
	return svc, repo
}

func createTestScheme(t *testing.T, svc *Service, tenantID string, sigType SignatureType, algo SignatureAlgorithm) *SignatureScheme {
	t.Helper()
	req := &CreateSchemeRequest{
		Name: fmt.Sprintf("test-%s-%s", sigType, algo),
		Type: sigType,
	}
	if algo != "" {
		req.Algorithm = algo
	}
	scheme, err := svc.CreateScheme(context.Background(), tenantID, req)
	require.NoError(t, err)
	return scheme
}

func signPayload(t *testing.T, svc *Service, tenantID string, schemeID string, payload []byte, ts time.Time) *SignatureResult {
	t.Helper()
	result, err := svc.Sign(context.Background(), tenantID, &SignatureRequest{
		SchemeID:  schemeID,
		Payload:   payload,
		Timestamp: &ts,
	})
	require.NoError(t, err)
	return result
}

// ============================================================
// Service Constructor Tests
// ============================================================

func TestNewService_WithNilConfig(t *testing.T) {
	repo := newMemoryRepository()
	svc := NewService(repo, nil)
	require.NotNil(t, svc)
	assert.Equal(t, 20, svc.config.MaxSchemesPerTenant)
	assert.Equal(t, 10, svc.config.MaxKeysPerScheme)
	assert.Equal(t, 32, svc.config.DefaultKeyLength)
	assert.Equal(t, time.Hour, svc.config.RotationCheckInterval)
}

func TestNewService_WithCustomConfig(t *testing.T) {
	repo := newMemoryRepository()
	cfg := &ServiceConfig{
		MaxSchemesPerTenant:   5,
		MaxKeysPerScheme:      3,
		DefaultKeyLength:      64,
		RotationCheckInterval: 30 * time.Minute,
	}
	svc := NewService(repo, cfg)
	require.NotNil(t, svc)
	assert.Equal(t, 5, svc.config.MaxSchemesPerTenant)
	assert.Equal(t, 3, svc.config.MaxKeysPerScheme)
	assert.Equal(t, 64, svc.config.DefaultKeyLength)
	assert.Equal(t, 30*time.Minute, svc.config.RotationCheckInterval)
}

func TestDefaultServiceConfig(t *testing.T) {
	cfg := DefaultServiceConfig()
	require.NotNil(t, cfg)
	assert.Equal(t, 20, cfg.MaxSchemesPerTenant)
	assert.Equal(t, 10, cfg.MaxKeysPerScheme)
	assert.Equal(t, 32, cfg.DefaultKeyLength)
	assert.Equal(t, time.Hour, cfg.RotationCheckInterval)
}

// ============================================================
// CreateScheme Tests
// ============================================================

func TestCreateScheme(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name:        "test-scheme",
		Description: "A test scheme",
		Type:        TypeGitHub,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, scheme.ID)
	assert.Equal(t, "tenant-1", scheme.TenantID)
	assert.Equal(t, "test-scheme", scheme.Name)
	assert.Equal(t, "A test scheme", scheme.Description)
	assert.Equal(t, TypeGitHub, scheme.Type)
	assert.Equal(t, AlgorithmHMACSHA256, scheme.Algorithm)
	assert.Equal(t, SchemeActive, scheme.Status)
	assert.False(t, scheme.CreatedAt.IsZero())
	assert.False(t, scheme.UpdatedAt.IsZero())
}

func TestCreateScheme_WithExplicitAlgorithm(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name:      "sha512-scheme",
		Type:      TypeCustomHMAC,
		Algorithm: AlgorithmHMACSHA512,
	})

	require.NoError(t, err)
	assert.Equal(t, AlgorithmHMACSHA512, scheme.Algorithm)
}

func TestCreateScheme_WithCustomConfig(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	customCfg := &SignatureConfig{
		SignatureHeader:       "X-Custom-Sig",
		TimestampHeader:       "X-Custom-TS",
		SignatureFormat:       "hex",
		IncludeTimestamp:      true,
		SignedPayloadTemplate: "{timestamp}.{body}",
		TimestampToleranceSec: 120,
	}

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name:   "custom-config",
		Type:   TypeCustomHMAC,
		Config: customCfg,
	})

	require.NoError(t, err)
	assert.Equal(t, "X-Custom-Sig", scheme.Config.SignatureHeader)
	assert.Equal(t, "X-Custom-TS", scheme.Config.TimestampHeader)
	assert.Equal(t, 120, scheme.Config.TimestampToleranceSec)
}

func TestCreateScheme_WithCustomKeyConfig(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	keyCfg := &KeyConfiguration{
		KeyType:        KeyTypeSymmetric,
		RotationPolicy: RotationScheduled,
		MaxKeyAge:      30 * 24 * time.Hour,
		AutoRotate:     true,
		KeyLength:      64,
	}

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name:      "custom-key-config",
		Type:      TypeCustomHMAC,
		KeyConfig: keyCfg,
	})

	require.NoError(t, err)
	assert.Equal(t, RotationScheduled, scheme.KeyConfig.RotationPolicy)
	assert.Equal(t, 64, scheme.KeyConfig.KeyLength)
	assert.True(t, scheme.KeyConfig.AutoRotate)
}

func TestCreateScheme_DefaultConfigs(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "default-cfg",
		Type: TypeStandardWebhooks,
	})

	require.NoError(t, err)
	assert.Equal(t, "Webhook-Signature", scheme.Config.SignatureHeader)
	assert.Equal(t, "base64", scheme.Config.SignatureFormat)
	assert.Equal(t, "{id}.{timestamp}.{body}", scheme.Config.SignedPayloadTemplate)
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
		require.NoError(t, err)
	}

	_, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "overflow",
		Type: TypeCustomHMAC,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum schemes reached")
}

func TestCreateScheme_MaxSchemesPerTenantIsolation(t *testing.T) {
	svc, _ := newTestService()
	svc.config.MaxSchemesPerTenant = 1
	ctx := context.Background()

	_, err := svc.CreateScheme(ctx, "tenant-a", &CreateSchemeRequest{
		Name: "scheme-a",
		Type: TypeCustomHMAC,
	})
	require.NoError(t, err)

	// Different tenant should still be able to create
	_, err = svc.CreateScheme(ctx, "tenant-b", &CreateSchemeRequest{
		Name: "scheme-b",
		Type: TypeCustomHMAC,
	})
	require.NoError(t, err)

	// Same tenant should fail
	_, err = svc.CreateScheme(ctx, "tenant-a", &CreateSchemeRequest{
		Name: "scheme-a2",
		Type: TypeCustomHMAC,
	})
	require.Error(t, err)
}

func TestCreateScheme_GeneratesInitialKey(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "key-gen-test",
		Type: TypeCustomHMAC,
	})
	require.NoError(t, err)

	keys, err := repo.ListKeys(ctx, scheme.ID)
	require.NoError(t, err)
	require.Len(t, keys, 1)

	key := keys[0]
	assert.Equal(t, KeyPrimary, key.Status)
	assert.Equal(t, scheme.ID, key.SchemeID)
	assert.Equal(t, "tenant-1", key.TenantID)
	assert.Equal(t, 1, key.Version)
	assert.NotEmpty(t, key.SecretKey)
	assert.NotEmpty(t, key.Fingerprint)
	assert.Equal(t, AlgorithmHMACSHA256, key.Algorithm)
}

func TestCreateScheme_AllTypes(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	types := []SignatureType{
		TypeStandardWebhooks,
		TypeStripe,
		TypeGitHub,
		TypeSlack,
		TypeCustomHMAC,
	}

	for _, sigType := range types {
		t.Run(string(sigType), func(t *testing.T) {
			scheme, err := svc.CreateScheme(ctx, "tenant-types", &CreateSchemeRequest{
				Name: fmt.Sprintf("test-%s", sigType),
				Type: sigType,
			})
			require.NoError(t, err)
			assert.Equal(t, sigType, scheme.Type)
			assert.Equal(t, SchemeActive, scheme.Status)
		})
	}
}

// ============================================================
// GetScheme & ListSchemes Tests
// ============================================================

func TestGetScheme(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	created, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "get-test",
		Type: TypeCustomHMAC,
	})
	require.NoError(t, err)

	retrieved, err := svc.GetScheme(ctx, "tenant-1", created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.Name, retrieved.Name)
}

func TestGetScheme_NotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.GetScheme(ctx, "tenant-1", "nonexistent-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetScheme_WrongTenant(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "cross-tenant-test",
		Type: TypeCustomHMAC,
	})
	require.NoError(t, err)

	_, err = svc.GetScheme(ctx, "tenant-2", scheme.ID)
	require.Error(t, err)
}

func TestListSchemes(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
			Name: fmt.Sprintf("list-scheme-%d", i),
			Type: TypeCustomHMAC,
		})
		require.NoError(t, err)
	}
	// Different tenant
	_, err := svc.CreateScheme(ctx, "tenant-2", &CreateSchemeRequest{
		Name: "other-tenant",
		Type: TypeCustomHMAC,
	})
	require.NoError(t, err)

	schemes, err := svc.ListSchemes(ctx, "tenant-1")
	require.NoError(t, err)
	assert.Len(t, schemes, 3)

	schemes2, err := svc.ListSchemes(ctx, "tenant-2")
	require.NoError(t, err)
	assert.Len(t, schemes2, 1)
}

func TestListSchemes_EmptyTenant(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	schemes, err := svc.ListSchemes(ctx, "empty-tenant")
	require.NoError(t, err)
	assert.Empty(t, schemes)
}

// ============================================================
// UpdateScheme Tests
// ============================================================

func TestUpdateScheme(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name:        "original",
		Description: "original description",
		Type:        TypeCustomHMAC,
	})
	require.NoError(t, err)

	newName := "updated"
	newDesc := "updated description"
	updated, err := svc.UpdateScheme(ctx, "tenant-1", scheme.ID, &UpdateSchemeRequest{
		Name:        &newName,
		Description: &newDesc,
	})

	require.NoError(t, err)
	assert.Equal(t, "updated", updated.Name)
	assert.Equal(t, "updated description", updated.Description)
	assert.True(t, updated.UpdatedAt.After(scheme.CreatedAt) || updated.UpdatedAt.Equal(scheme.CreatedAt))
}

func TestUpdateScheme_PartialUpdate(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name:        "partial-test",
		Description: "keep me",
		Type:        TypeCustomHMAC,
	})
	require.NoError(t, err)

	newName := "changed-name"
	updated, err := svc.UpdateScheme(ctx, "tenant-1", scheme.ID, &UpdateSchemeRequest{
		Name: &newName,
	})

	require.NoError(t, err)
	assert.Equal(t, "changed-name", updated.Name)
	assert.Equal(t, "keep me", updated.Description)
}

func TestUpdateScheme_Config(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "config-update",
		Type: TypeCustomHMAC,
	})
	require.NoError(t, err)

	newCfg := &SignatureConfig{
		SignatureHeader:       "X-Updated-Sig",
		SignatureFormat:       "base64",
		TimestampToleranceSec: 600,
	}
	updated, err := svc.UpdateScheme(ctx, "tenant-1", scheme.ID, &UpdateSchemeRequest{
		Config: newCfg,
	})

	require.NoError(t, err)
	assert.Equal(t, "X-Updated-Sig", updated.Config.SignatureHeader)
	assert.Equal(t, 600, updated.Config.TimestampToleranceSec)
}

func TestUpdateScheme_NotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	name := "test"
	_, err := svc.UpdateScheme(ctx, "tenant-1", "nonexistent", &UpdateSchemeRequest{
		Name: &name,
	})
	require.Error(t, err)
}

// ============================================================
// DeleteScheme Tests
// ============================================================

func TestDeleteScheme(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "to-delete",
		Type: TypeCustomHMAC,
	})
	require.NoError(t, err)

	err = svc.DeleteScheme(ctx, "tenant-1", scheme.ID)
	require.NoError(t, err)

	_, err = svc.GetScheme(ctx, "tenant-1", scheme.ID)
	require.Error(t, err)
}

func TestDeleteScheme_NonexistentIsIdempotent(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	err := svc.DeleteScheme(ctx, "tenant-1", "nonexistent")
	require.NoError(t, err)
}

// ============================================================
// Key Generation Tests
// ============================================================

func TestKeyGeneration(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "key-test",
		Type: TypeCustomHMAC,
	})
	require.NoError(t, err)

	keys, err := repo.ListKeys(ctx, scheme.ID)
	require.NoError(t, err)
	require.NotEmpty(t, keys)

	key := keys[0]
	assert.Equal(t, KeyPrimary, key.Status)
	assert.NotEmpty(t, key.SecretKey)
	assert.NotEmpty(t, key.Fingerprint)

	// Verify key is valid base64
	_, err = base64.StdEncoding.DecodeString(key.SecretKey)
	assert.NoError(t, err)
}

func TestKeyGeneration_CustomKeyLength(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "custom-key-len",
		Type: TypeCustomHMAC,
		KeyConfig: &KeyConfiguration{
			KeyLength: 64,
		},
	})
	require.NoError(t, err)

	keys, err := repo.ListKeys(ctx, scheme.ID)
	require.NoError(t, err)
	require.NotEmpty(t, keys)

	decoded, err := base64.StdEncoding.DecodeString(keys[0].SecretKey)
	require.NoError(t, err)
	assert.Len(t, decoded, 64)
}

func TestKeyGeneration_ExpiresAt(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	maxAge := 48 * time.Hour
	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "key-expiry",
		Type: TypeCustomHMAC,
		KeyConfig: &KeyConfiguration{
			MaxKeyAge: maxAge,
		},
	})
	require.NoError(t, err)

	keys, err := repo.ListKeys(ctx, scheme.ID)
	require.NoError(t, err)
	require.NotEmpty(t, keys)
	require.NotNil(t, keys[0].ExpiresAt)
	assert.WithinDuration(t, keys[0].CreatedAt.Add(maxAge), *keys[0].ExpiresAt, time.Second)
}

func TestKeyGeneration_VersionIncrement(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")

	// Initial key is version 1
	keys, _ := repo.ListKeys(ctx, scheme.ID)
	require.Len(t, keys, 1)
	assert.Equal(t, 1, keys[0].Version)

	// Rotate to get version 2
	_, err := svc.RotateKey(ctx, "tenant-1", scheme.ID, &RotateKeyRequest{Immediate: true})
	require.NoError(t, err)

	keys, _ = repo.ListKeys(ctx, scheme.ID)
	maxVersion := 0
	for _, k := range keys {
		if k.Version > maxVersion {
			maxVersion = k.Version
		}
	}
	assert.Equal(t, 2, maxVersion)
}

func TestKeyGeneration_UniqueKeys(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")
	keys1, _ := repo.ListKeys(ctx, scheme.ID)
	require.Len(t, keys1, 1)

	_, err := svc.RotateKey(ctx, "tenant-1", scheme.ID, &RotateKeyRequest{Immediate: true})
	require.NoError(t, err)

	keys2, _ := repo.ListKeys(ctx, scheme.ID)
	require.Len(t, keys2, 2)

	// All keys should be unique
	seen := make(map[string]bool)
	for _, k := range keys2 {
		assert.False(t, seen[k.SecretKey], "duplicate secret key found")
		seen[k.SecretKey] = true
	}
}

// ============================================================
// Sign Tests
// ============================================================

func TestSign_HMACSHA256(t *testing.T) {
	svc, _ := newTestService()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, AlgorithmHMACSHA256)
	payload := []byte(`{"event":"test","data":"hello"}`)
	now := time.Now()

	result := signPayload(t, svc, "tenant-1", scheme.ID, payload, now)

	assert.NotEmpty(t, result.Signature)
	assert.Equal(t, AlgorithmHMACSHA256, result.Algorithm)
	assert.NotEmpty(t, result.KeyID)
	assert.Equal(t, 1, result.KeyVersion)
	assert.NotEmpty(t, result.Headers)
}

func TestSign_HMACSHA512(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name:      "sha512-sign",
		Type:      TypeCustomHMAC,
		Algorithm: AlgorithmHMACSHA512,
	})
	require.NoError(t, err)

	result := signPayload(t, svc, "tenant-1", scheme.ID, []byte(`{"event":"test512"}`), time.Now())
	assert.Equal(t, AlgorithmHMACSHA512, result.Algorithm)
	assert.NotEmpty(t, result.Signature)
}

func TestSign_IncludesHeaders(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "headers-test",
		Type: TypeCustomHMAC,
		Config: &SignatureConfig{
			SignatureHeader:       "X-Sig",
			TimestampHeader:       "X-Timestamp",
			IDHeader:              "X-ID",
			SignatureFormat:       "hex",
			IncludeTimestamp:      true,
			IncludeMessageID:      true,
			SignedPayloadTemplate: "{id}.{timestamp}.{body}",
			TimestampToleranceSec: 300,
		},
	})
	require.NoError(t, err)

	now := time.Now()
	result := signPayload(t, svc, "tenant-1", scheme.ID, []byte("test"), now)

	assert.Contains(t, result.Headers, "X-Sig")
	assert.Contains(t, result.Headers, "X-Timestamp")
	assert.Contains(t, result.Headers, "X-ID")
	assert.NotEmpty(t, result.MessageID)
}

func TestSign_WithProvidedMessageID(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")
	now := time.Now()

	result, err := svc.Sign(ctx, "tenant-1", &SignatureRequest{
		SchemeID:  scheme.ID,
		Payload:   []byte("test"),
		MessageID: "my-custom-id",
		Timestamp: &now,
	})
	require.NoError(t, err)
	assert.Equal(t, "my-custom-id", result.MessageID)
}

func TestSign_NonexistentScheme(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.Sign(ctx, "tenant-1", &SignatureRequest{
		SchemeID: "nonexistent",
		Payload:  []byte("test"),
	})
	require.Error(t, err)
}

func TestSign_WrongTenant(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")

	_, err := svc.Sign(ctx, "tenant-2", &SignatureRequest{
		SchemeID: scheme.ID,
		Payload:  []byte("test"),
	})
	require.Error(t, err)
}

func TestSign_EmptyPayload(t *testing.T) {
	svc, _ := newTestService()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, AlgorithmHMACSHA256)
	now := time.Now()

	result := signPayload(t, svc, "tenant-1", scheme.ID, []byte{}, now)
	assert.NotEmpty(t, result.Signature)
}

func TestSign_LargePayload(t *testing.T) {
	svc, _ := newTestService()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, AlgorithmHMACSHA256)
	now := time.Now()

	largePayload := make([]byte, 1024*1024) // 1MB
	for i := range largePayload {
		largePayload[i] = byte(i % 256)
	}

	result := signPayload(t, svc, "tenant-1", scheme.ID, largePayload, now)
	assert.NotEmpty(t, result.Signature)
}

func TestSign_SpecialCharacters(t *testing.T) {
	svc, _ := newTestService()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, AlgorithmHMACSHA256)
	now := time.Now()

	payloads := [][]byte{
		[]byte(`{"emoji":"🎉🔥","unicode":"日本語"}`),
		[]byte(`{"special":"<>&\"'\\\/\n\t\r"}`),
		[]byte(`{"null_byte":"\x00"}`),
		[]byte(`{"nested":{"deep":{"deeper":{"value":true}}}}`),
	}

	for i, payload := range payloads {
		t.Run(fmt.Sprintf("payload-%d", i), func(t *testing.T) {
			result := signPayload(t, svc, "tenant-1", scheme.ID, payload, now)
			assert.NotEmpty(t, result.Signature)
		})
	}
}

func TestSign_UpdatesKeyUsage(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")
	now := time.Now()

	result := signPayload(t, svc, "tenant-1", scheme.ID, []byte("test"), now)

	key, err := repo.GetKey(ctx, result.KeyID)
	require.NoError(t, err)
	assert.NotNil(t, key.LastUsedAt)
	assert.Equal(t, int64(1), key.UsageCount)
}

func TestSign_IncrementsStats(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")
	now := time.Now()

	for i := 0; i < 3; i++ {
		signPayload(t, svc, "tenant-1", scheme.ID, []byte("test"), now)
	}

	stats, err := repo.GetSchemeStats(ctx, scheme.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.TotalSigned)
}

// ============================================================
// Verify Tests
// ============================================================

func TestVerify_HMACSHA256_Valid(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, AlgorithmHMACSHA256)
	payload := []byte(`{"event":"test","data":"hello"}`)
	now := time.Now()

	signResult := signPayload(t, svc, "tenant-1", scheme.ID, payload, now)

	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Signature: signResult.Signature,
		Timestamp: &now,
	})
	require.NoError(t, err)
	assert.True(t, verifyResult.Valid)
	assert.NotEmpty(t, verifyResult.KeyID)
	assert.Equal(t, 1, verifyResult.KeyVersion)
	assert.Empty(t, verifyResult.Error)
	assert.Empty(t, verifyResult.ErrorCode)
}

func TestVerify_HMACSHA512_Valid(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, AlgorithmHMACSHA512)
	payload := []byte(`{"event":"test512"}`)
	now := time.Now()

	signResult := signPayload(t, svc, "tenant-1", scheme.ID, payload, now)

	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Signature: signResult.Signature,
		Timestamp: &now,
	})
	require.NoError(t, err)
	assert.True(t, verifyResult.Valid)
}

func TestVerify_InvalidSignature(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")
	now := time.Now()

	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   []byte(`{"event":"test"}`),
		Signature: "definitely-wrong-signature",
		Timestamp: &now,
	})
	require.NoError(t, err)
	assert.False(t, verifyResult.Valid)
	assert.Equal(t, "INVALID_SIGNATURE", verifyResult.ErrorCode)
}

func TestVerify_TamperedPayload(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, AlgorithmHMACSHA256)
	now := time.Now()

	signResult := signPayload(t, svc, "tenant-1", scheme.ID, []byte(`{"event":"original"}`), now)

	// Verify with tampered payload
	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   []byte(`{"event":"tampered"}`),
		Signature: signResult.Signature,
		Timestamp: &now,
	})
	require.NoError(t, err)
	assert.False(t, verifyResult.Valid)
	assert.Equal(t, "INVALID_SIGNATURE", verifyResult.ErrorCode)
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
	require.NoError(t, err)

	payload := []byte(`{"event":"test"}`)
	oldTime := time.Now().Add(-2 * time.Hour)

	signResult := signPayload(t, svc, "tenant-1", scheme.ID, payload, oldTime)

	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Signature: signResult.Signature,
		Timestamp: &oldTime,
	})
	require.NoError(t, err)
	assert.False(t, verifyResult.Valid)
	assert.Equal(t, "TIMESTAMP_EXPIRED", verifyResult.ErrorCode)
	assert.Contains(t, verifyResult.Error, "Timestamp too old")
}

func TestVerify_TimestampFuture(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "future-ts-test",
		Type: TypeCustomHMAC,
		Config: &SignatureConfig{
			SignatureHeader:       "X-Sig",
			SignatureFormat:       "hex",
			IncludeTimestamp:      true,
			SignedPayloadTemplate: "{timestamp}.{body}",
			TimestampToleranceSec: 60,
		},
	})
	require.NoError(t, err)

	payload := []byte(`{"event":"test"}`)
	futureTime := time.Now().Add(2 * time.Hour)

	signResult := signPayload(t, svc, "tenant-1", scheme.ID, payload, futureTime)

	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Signature: signResult.Signature,
		Timestamp: &futureTime,
	})
	require.NoError(t, err)
	assert.False(t, verifyResult.Valid)
	assert.Equal(t, "TIMESTAMP_FUTURE", verifyResult.ErrorCode)
}

func TestVerify_NoTimestampTolerance(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	// GitHub-style: no timestamp tolerance
	scheme := createTestScheme(t, svc, "tenant-1", TypeGitHub, AlgorithmHMACSHA256)
	payload := []byte(`{"event":"push"}`)
	oldTime := time.Now().Add(-24 * time.Hour)

	signResult := signPayload(t, svc, "tenant-1", scheme.ID, payload, oldTime)

	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Signature: signResult.Signature,
		Timestamp: &oldTime,
	})
	require.NoError(t, err)
	assert.True(t, verifyResult.Valid, "GitHub-style with no tolerance should accept old timestamps")
}

func TestVerify_NonexistentScheme(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	now := time.Now()
	result, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  "nonexistent",
		Payload:   []byte("test"),
		Signature: "sig",
		Timestamp: &now,
	})
	require.NoError(t, err) // Verify returns result, not error
	assert.False(t, result.Valid)
	assert.Equal(t, "SCHEME_NOT_FOUND", result.ErrorCode)
}

func TestVerify_EmptyPayload(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, AlgorithmHMACSHA256)
	now := time.Now()

	signResult := signPayload(t, svc, "tenant-1", scheme.ID, []byte{}, now)

	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   []byte{},
		Signature: signResult.Signature,
		Timestamp: &now,
	})
	require.NoError(t, err)
	assert.True(t, verifyResult.Valid)
}

func TestVerify_TimestampFromHeaders(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme, err := svc.CreateScheme(ctx, "tenant-1", &CreateSchemeRequest{
		Name: "header-ts-test",
		Type: TypeCustomHMAC,
		Config: &SignatureConfig{
			SignatureHeader:       "X-Sig",
			TimestampHeader:       "X-Timestamp",
			SignatureFormat:       "hex",
			IncludeTimestamp:      true,
			SignedPayloadTemplate: "{timestamp}.{body}",
			TimestampToleranceSec: 300,
		},
	})
	require.NoError(t, err)

	now := time.Now()
	payload := []byte(`{"event":"test"}`)
	signResult := signPayload(t, svc, "tenant-1", scheme.ID, payload, now)

	// Pass timestamp via headers instead of directly
	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Signature: signResult.Signature,
		Headers: map[string]string{
			"X-Timestamp": fmt.Sprintf("%d", now.Unix()),
		},
	})
	require.NoError(t, err)
	assert.True(t, verifyResult.Valid)
}

func TestVerify_IncrementsStats_Success(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")
	now := time.Now()

	signResult := signPayload(t, svc, "tenant-1", scheme.ID, []byte("test"), now)

	_, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   []byte("test"),
		Signature: signResult.Signature,
		Timestamp: &now,
	})
	require.NoError(t, err)

	stats, _ := repo.GetSchemeStats(ctx, scheme.ID)
	assert.Equal(t, int64(1), stats.TotalVerified)
	assert.Equal(t, int64(0), stats.TotalFailed)
}

func TestVerify_IncrementsStats_Failure(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")
	now := time.Now()

	_, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   []byte("test"),
		Signature: "bad-sig",
		Timestamp: &now,
	})
	require.NoError(t, err)

	stats, _ := repo.GetSchemeStats(ctx, scheme.ID)
	assert.Equal(t, int64(0), stats.TotalVerified)
	assert.Equal(t, int64(1), stats.TotalFailed)
}

// ============================================================
// Sign+Verify Integration Tests per Scheme Type
// ============================================================

func TestSignAndVerify_StandardWebhooks(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeStandardWebhooks, AlgorithmHMACSHA256)
	payload := []byte(`{"type":"invoice.paid","data":{"id":"inv_123"}}`)
	now := time.Now()

	signResult := signPayload(t, svc, "tenant-1", scheme.ID, payload, now)

	assert.Contains(t, signResult.Headers, "Webhook-Signature")
	assert.Contains(t, signResult.Headers, "Webhook-Timestamp")
	assert.Contains(t, signResult.Headers, "Webhook-Id")

	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Signature: signResult.Signature,
		Timestamp: &now,
		MessageID: signResult.MessageID,
	})
	require.NoError(t, err)
	assert.True(t, verifyResult.Valid)
}

func TestSignAndVerify_Stripe(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeStripe, AlgorithmHMACSHA256)
	payload := []byte(`{"type":"charge.succeeded","data":{"id":"ch_123"}}`)
	now := time.Now()

	signResult := signPayload(t, svc, "tenant-1", scheme.ID, payload, now)

	// Stripe format: t=timestamp,v1=signature
	assert.Contains(t, signResult.Signature, "t=")
	assert.Contains(t, signResult.Signature, "v1=")

	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Signature: signResult.Signature,
		Timestamp: &now,
	})
	require.NoError(t, err)
	assert.True(t, verifyResult.Valid)
}

func TestSignAndVerify_GitHub(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeGitHub, AlgorithmHMACSHA256)
	payload := []byte(`{"action":"opened","pull_request":{"number":1}}`)
	now := time.Now()

	signResult := signPayload(t, svc, "tenant-1", scheme.ID, payload, now)

	// GitHub format: sha256=hexdigest
	assert.True(t, strings.HasPrefix(signResult.Signature, "sha256="))

	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Signature: signResult.Signature,
		Timestamp: &now,
	})
	require.NoError(t, err)
	assert.True(t, verifyResult.Valid)
}

func TestSignAndVerify_Slack(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeSlack, AlgorithmHMACSHA256)
	payload := []byte(`token=xyzzy&command=/test`)
	now := time.Now()

	signResult := signPayload(t, svc, "tenant-1", scheme.ID, payload, now)

	// Slack format: v0=hexdigest
	assert.True(t, strings.HasPrefix(signResult.Signature, "v0="))

	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Signature: signResult.Signature,
		Timestamp: &now,
	})
	require.NoError(t, err)
	assert.True(t, verifyResult.Valid)
}

func TestSignAndVerify_CrossSchemeRejection(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	schemeA := createTestScheme(t, svc, "tenant-1", TypeGitHub, AlgorithmHMACSHA256)
	schemeB := createTestScheme(t, svc, "tenant-1", TypeSlack, AlgorithmHMACSHA256)

	payload := []byte(`{"event":"test"}`)
	now := time.Now()

	signResult := signPayload(t, svc, "tenant-1", schemeA.ID, payload, now)

	// Try to verify signature from scheme A with scheme B
	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  schemeB.ID,
		Payload:   payload,
		Signature: signResult.Signature,
		Timestamp: &now,
	})
	require.NoError(t, err)
	assert.False(t, verifyResult.Valid)
}

func TestSignAndVerify_CrossAlgorithmRejection(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme256 := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, AlgorithmHMACSHA256)
	payload := []byte(`{"event":"test"}`)
	now := time.Now()

	signResult := signPayload(t, svc, "tenant-1", scheme256.ID, payload, now)

	// Create a different scheme with SHA512 and try to verify
	scheme512 := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, AlgorithmHMACSHA512)

	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme512.ID,
		Payload:   payload,
		Signature: signResult.Signature,
		Timestamp: &now,
	})
	require.NoError(t, err)
	assert.False(t, verifyResult.Valid)
}

// ============================================================
// Key Rotation Tests
// ============================================================

func TestKeyRotation_Immediate(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")

	// Sign with original key
	payload := []byte(`{"event":"before-rotation"}`)
	now := time.Now()
	signBefore := signPayload(t, svc, "tenant-1", scheme.ID, payload, now)

	// Rotate key (immediate)
	rotation, err := svc.RotateKey(ctx, "tenant-1", scheme.ID, &RotateKeyRequest{
		Immediate: true,
		Reason:    "test rotation",
	})
	require.NoError(t, err)
	assert.Equal(t, RotationCompleted, rotation.Status)
	assert.Equal(t, "test rotation", rotation.Reason)
	assert.NotEmpty(t, rotation.OldKeyID)
	assert.NotEmpty(t, rotation.NewKeyID)
	assert.NotEqual(t, rotation.OldKeyID, rotation.NewKeyID)
	assert.NotNil(t, rotation.CompletedAt)
	assert.NotNil(t, rotation.StartedAt)
	assert.NotNil(t, rotation.OverlapUntil)

	// Old key should be in rotating state
	oldKey, _ := repo.GetKey(ctx, rotation.OldKeyID)
	assert.Equal(t, KeyRotating, oldKey.Status)

	// New key should be primary
	newKey, _ := repo.GetKey(ctx, rotation.NewKeyID)
	assert.Equal(t, KeyPrimary, newKey.Status)

	// Old signature should still verify (old key in rotating state)
	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Signature: signBefore.Signature,
		Timestamp: &now,
	})
	require.NoError(t, err)
	assert.True(t, verifyResult.Valid)

	// New signatures should also verify
	payload2 := []byte(`{"event":"after-rotation"}`)
	now2 := time.Now()
	signAfter := signPayload(t, svc, "tenant-1", scheme.ID, payload2, now2)

	verifyResult2, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   payload2,
		Signature: signAfter.Signature,
		Timestamp: &now2,
	})
	require.NoError(t, err)
	assert.True(t, verifyResult2.Valid)
}

func TestKeyRotation_Scheduled(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")

	scheduledTime := time.Now().Add(24 * time.Hour)
	rotation, err := svc.RotateKey(ctx, "tenant-1", scheme.ID, &RotateKeyRequest{
		Reason:     "scheduled rotation",
		ScheduleAt: &scheduledTime,
	})
	require.NoError(t, err)
	assert.Equal(t, RotationScheduledStatus, rotation.Status)
	assert.Nil(t, rotation.CompletedAt)
}

func TestKeyRotation_CustomOverlapHours(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")

	rotation, err := svc.RotateKey(ctx, "tenant-1", scheme.ID, &RotateKeyRequest{
		Immediate:    true,
		OverlapHours: 48,
	})
	require.NoError(t, err)
	require.NotNil(t, rotation.OverlapUntil)

	expectedOverlap := rotation.ScheduledAt.Add(48 * time.Hour)
	assert.WithinDuration(t, expectedOverlap, *rotation.OverlapUntil, time.Second)
}

func TestKeyRotation_DefaultOverlapHours(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")

	rotation, err := svc.RotateKey(ctx, "tenant-1", scheme.ID, &RotateKeyRequest{
		Immediate: true,
	})
	require.NoError(t, err)
	require.NotNil(t, rotation.OverlapUntil)

	expectedOverlap := rotation.ScheduledAt.Add(24 * time.Hour)
	assert.WithinDuration(t, expectedOverlap, *rotation.OverlapUntil, time.Second)
}

func TestKeyRotation_MultipleRotations(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")

	for i := 0; i < 3; i++ {
		_, err := svc.RotateKey(ctx, "tenant-1", scheme.ID, &RotateKeyRequest{
			Immediate: true,
			Reason:    fmt.Sprintf("rotation %d", i),
		})
		require.NoError(t, err)
	}

	keys, _ := repo.ListKeys(ctx, scheme.ID)
	assert.Len(t, keys, 4) // 1 initial + 3 rotations

	// Exactly one key should be primary
	primaryCount := 0
	for _, k := range keys {
		if k.Status == KeyPrimary {
			primaryCount++
		}
	}
	assert.Equal(t, 1, primaryCount)
}

func TestKeyRotation_NonexistentScheme(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.RotateKey(ctx, "tenant-1", "nonexistent", &RotateKeyRequest{
		Immediate: true,
	})
	require.Error(t, err)
}

func TestKeyRotation_WrongTenant(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")

	_, err := svc.RotateKey(ctx, "tenant-2", scheme.ID, &RotateKeyRequest{
		Immediate: true,
	})
	require.Error(t, err)
}

// ============================================================
// GetKeys Tests
// ============================================================

func TestGetKeys(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")

	keys, err := svc.GetKeys(ctx, "tenant-1", scheme.ID)
	require.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Equal(t, KeyPrimary, keys[0].Status)
}

func TestGetKeys_AfterRotation(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")

	_, err := svc.RotateKey(ctx, "tenant-1", scheme.ID, &RotateKeyRequest{Immediate: true})
	require.NoError(t, err)

	keys, err := svc.GetKeys(ctx, "tenant-1", scheme.ID)
	require.NoError(t, err)
	assert.Len(t, keys, 2)
}

func TestGetKeys_WrongTenant(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")

	_, err := svc.GetKeys(ctx, "tenant-2", scheme.ID)
	require.Error(t, err)
}

func TestGetKeys_NonexistentScheme(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.GetKeys(ctx, "tenant-1", "nonexistent")
	require.Error(t, err)
}

// ============================================================
// RevokeKey Tests
// ============================================================

func TestRevokeKey(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")

	keys, _ := repo.ListKeys(ctx, scheme.ID)
	require.Len(t, keys, 1)
	keyID := keys[0].ID

	err := svc.RevokeKey(ctx, "tenant-1", scheme.ID, keyID)
	require.NoError(t, err)

	revokedKey, _ := repo.GetKey(ctx, keyID)
	assert.Equal(t, KeyRevoked, revokedKey.Status)
}

func TestRevokeKey_CannotSignWithRevokedKey(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")

	keys, _ := repo.ListKeys(ctx, scheme.ID)
	err := svc.RevokeKey(ctx, "tenant-1", scheme.ID, keys[0].ID)
	require.NoError(t, err)

	// Sign should fail because no primary/active key exists
	_, err = svc.Sign(ctx, "tenant-1", &SignatureRequest{
		SchemeID: scheme.ID,
		Payload:  []byte("test"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no signing key available")
}

func TestRevokeKey_RevokedKeyCannotVerify(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")
	payload := []byte("test-revoke-verify")
	now := time.Now()

	signResult := signPayload(t, svc, "tenant-1", scheme.ID, payload, now)

	// Revoke the key that was used
	keys, _ := repo.ListKeys(ctx, scheme.ID)
	err := svc.RevokeKey(ctx, "tenant-1", scheme.ID, keys[0].ID)
	require.NoError(t, err)

	// Verify should fail since the only key is revoked
	verifyResult, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   payload,
		Signature: signResult.Signature,
		Timestamp: &now,
	})
	require.NoError(t, err)
	assert.False(t, verifyResult.Valid)
}

func TestRevokeKey_WrongTenant(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")
	keys, _ := repo.ListKeys(ctx, scheme.ID)

	err := svc.RevokeKey(ctx, "tenant-2", scheme.ID, keys[0].ID)
	require.Error(t, err)
}

// ============================================================
// GetRotations Tests
// ============================================================

func TestGetRotations(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")

	_, err := svc.RotateKey(ctx, "tenant-1", scheme.ID, &RotateKeyRequest{
		Immediate: true,
		Reason:    "first",
	})
	require.NoError(t, err)

	_, err = svc.RotateKey(ctx, "tenant-1", scheme.ID, &RotateKeyRequest{
		Immediate: true,
		Reason:    "second",
	})
	require.NoError(t, err)

	rotations, err := svc.GetRotations(ctx, "tenant-1", scheme.ID)
	require.NoError(t, err)
	assert.Len(t, rotations, 2)
}

func TestGetRotations_WrongTenant(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")

	_, err := svc.GetRotations(ctx, "tenant-2", scheme.ID)
	require.Error(t, err)
}

func TestGetRotations_Empty(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")

	rotations, err := svc.GetRotations(ctx, "tenant-1", scheme.ID)
	require.NoError(t, err)
	assert.Empty(t, rotations)
}

// ============================================================
// GetStats Tests
// ============================================================

func TestGetStats(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")
	now := time.Now()

	// Sign 3 times
	for i := 0; i < 3; i++ {
		signPayload(t, svc, "tenant-1", scheme.ID, []byte(fmt.Sprintf("payload-%d", i)), now)
	}

	// Verify once successfully
	signResult := signPayload(t, svc, "tenant-1", scheme.ID, []byte("verify-me"), now)
	_, err := svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   []byte("verify-me"),
		Signature: signResult.Signature,
		Timestamp: &now,
	})
	require.NoError(t, err)

	// One failed verify
	_, err = svc.Verify(ctx, "tenant-1", &VerifyRequest{
		SchemeID:  scheme.ID,
		Payload:   []byte("test"),
		Signature: "bad",
		Timestamp: &now,
	})
	require.NoError(t, err)

	stats, err := svc.GetStats(ctx, "tenant-1", scheme.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(4), stats.TotalSigned) // 3 + 1 (the one we verified)
	assert.Equal(t, int64(1), stats.TotalVerified)
	assert.Equal(t, int64(1), stats.TotalFailed)
}

func TestGetStats_WrongTenant(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")

	_, err := svc.GetStats(ctx, "tenant-2", scheme.ID)
	require.Error(t, err)
}

func TestGetStats_EmptyStats(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")

	stats, err := svc.GetStats(ctx, "tenant-1", scheme.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.TotalSigned)
	assert.Equal(t, int64(0), stats.TotalVerified)
	assert.Equal(t, int64(0), stats.TotalFailed)
}

// ============================================================
// GetSupportedSchemes Tests
// ============================================================

func TestGetSupportedSchemes(t *testing.T) {
	schemes := GetSupportedSchemes()
	require.NotEmpty(t, schemes)

	typeSet := make(map[SignatureType]bool)
	for _, s := range schemes {
		typeSet[s.Type] = true
		assert.NotEmpty(t, s.Name)
		assert.NotEmpty(t, s.Description)
		assert.NotEmpty(t, s.DefaultAlgorithm)
	}

	for _, expected := range []SignatureType{TypeStandardWebhooks, TypeStripe, TypeGitHub, TypeSlack, TypeCustomHMAC} {
		assert.True(t, typeSet[expected], "expected supported type %s", expected)
	}
}

func TestGetSupportedSchemes_ViaService(t *testing.T) {
	svc, _ := newTestService()

	schemes := svc.GetSupportedSchemes()
	require.NotEmpty(t, schemes)
	assert.Equal(t, GetSupportedSchemes(), schemes)
}

// ============================================================
// GetDefaultConfig Tests
// ============================================================

func TestGetDefaultConfig(t *testing.T) {
	tests := []struct {
		sigType              SignatureType
		expectedHeader       string
		expectedFormat       string
		expectedTemplate     string
		expectedToleranceSec int
		expectedIncludeTS    bool
	}{
		{TypeStandardWebhooks, "Webhook-Signature", "base64", "{id}.{timestamp}.{body}", 300, true},
		{TypeStripe, "Stripe-Signature", "hex", "{timestamp}.{body}", 300, true},
		{TypeGitHub, "X-Hub-Signature-256", "hex", "{body}", 0, false},
		{TypeSlack, "X-Slack-Signature", "hex", "v0:{timestamp}:{body}", 300, true},
	}

	for _, tc := range tests {
		t.Run(string(tc.sigType), func(t *testing.T) {
			cfg := GetDefaultConfig(tc.sigType)
			assert.Equal(t, tc.expectedHeader, cfg.SignatureHeader)
			assert.Equal(t, tc.expectedFormat, cfg.SignatureFormat)
			assert.Equal(t, tc.expectedTemplate, cfg.SignedPayloadTemplate)
			assert.Equal(t, tc.expectedToleranceSec, cfg.TimestampToleranceSec)
			assert.Equal(t, tc.expectedIncludeTS, cfg.IncludeTimestamp)
		})
	}
}

func TestGetDefaultConfig_UnknownType(t *testing.T) {
	cfg := GetDefaultConfig("unknown_type")
	assert.Equal(t, "X-Webhook-Signature", cfg.SignatureHeader)
	assert.Equal(t, "hex", cfg.SignatureFormat)
}

func TestGetDefaultKeyConfig(t *testing.T) {
	cfg := GetDefaultKeyConfig()
	assert.Equal(t, KeyTypeSymmetric, cfg.KeyType)
	assert.Equal(t, RotationManual, cfg.RotationPolicy)
	assert.Equal(t, 24*time.Hour, cfg.MinKeyAge)
	assert.Equal(t, 90*24*time.Hour, cfg.MaxKeyAge)
	assert.Equal(t, 24*time.Hour, cfg.OverlapPeriod)
	assert.False(t, cfg.AutoRotate)
	assert.True(t, cfg.NotifyOnRotation)
	assert.Equal(t, 32, cfg.KeyLength)
}

// ============================================================
// computeSignature Tests
// ============================================================

func TestComputeSignature_HMACSHA256(t *testing.T) {
	svc, _ := newTestService()
	secret := "test-secret"
	payload := []byte("test-payload")

	sig, err := svc.computeSignature(AlgorithmHMACSHA256, secret, payload)
	require.NoError(t, err)

	// Verify against standard library
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	assert.Equal(t, expected, sig)
}

func TestComputeSignature_HMACSHA512(t *testing.T) {
	svc, _ := newTestService()
	secret := "test-secret-512"
	payload := []byte("test-payload-512")

	sig, err := svc.computeSignature(AlgorithmHMACSHA512, secret, payload)
	require.NoError(t, err)

	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	assert.Equal(t, expected, sig)
	// SHA-512 produces 128 hex chars (64 bytes)
	assert.Len(t, sig, 128)
}

func TestComputeSignature_HMACSHA256_Len(t *testing.T) {
	svc, _ := newTestService()

	sig, err := svc.computeSignature(AlgorithmHMACSHA256, "key", []byte("data"))
	require.NoError(t, err)
	// SHA-256 produces 64 hex chars (32 bytes)
	assert.Len(t, sig, 64)
}

func TestComputeSignature_UnsupportedAlgorithm(t *testing.T) {
	svc, _ := newTestService()

	_, err := svc.computeSignature("unsupported-algo", "secret", []byte("payload"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported algorithm")
}

func TestComputeSignature_ED25519_Unsupported(t *testing.T) {
	svc, _ := newTestService()

	_, err := svc.computeSignature(AlgorithmED25519, "key", []byte("data"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported algorithm")
}

func TestComputeSignature_RSA_Unsupported(t *testing.T) {
	svc, _ := newTestService()

	_, err := svc.computeSignature(AlgorithmRSASHA256, "key", []byte("data"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported algorithm")
}

func TestComputeSignature_Deterministic(t *testing.T) {
	svc, _ := newTestService()
	secret := "deterministic-test"
	payload := []byte("same-payload")

	sig1, err := svc.computeSignature(AlgorithmHMACSHA256, secret, payload)
	require.NoError(t, err)

	sig2, err := svc.computeSignature(AlgorithmHMACSHA256, secret, payload)
	require.NoError(t, err)

	assert.Equal(t, sig1, sig2, "HMAC should be deterministic for same inputs")
}

func TestComputeSignature_DifferentSecretsDiffer(t *testing.T) {
	svc, _ := newTestService()
	payload := []byte("test")

	sig1, _ := svc.computeSignature(AlgorithmHMACSHA256, "secret-a", payload)
	sig2, _ := svc.computeSignature(AlgorithmHMACSHA256, "secret-b", payload)

	assert.NotEqual(t, sig1, sig2)
}

func TestComputeSignature_DifferentPayloadsDiffer(t *testing.T) {
	svc, _ := newTestService()
	secret := "same-secret"

	sig1, _ := svc.computeSignature(AlgorithmHMACSHA256, secret, []byte("payload-a"))
	sig2, _ := svc.computeSignature(AlgorithmHMACSHA256, secret, []byte("payload-b"))

	assert.NotEqual(t, sig1, sig2)
}

func TestComputeSignature_EmptyPayload(t *testing.T) {
	svc, _ := newTestService()

	sig, err := svc.computeSignature(AlgorithmHMACSHA256, "secret", []byte{})
	require.NoError(t, err)
	assert.NotEmpty(t, sig)
}

// ============================================================
// formatSignature Tests
// ============================================================

func TestFormatSignature_Hex(t *testing.T) {
	svc, _ := newTestService()

	scheme := &SignatureScheme{
		Type: TypeCustomHMAC,
		Config: SignatureConfig{
			SignatureFormat: "hex",
		},
	}

	result := svc.formatSignature(scheme, "abcdef1234", time.Now())
	assert.Equal(t, "abcdef1234", result)
}

func TestFormatSignature_HexWithPrefix(t *testing.T) {
	svc, _ := newTestService()

	scheme := &SignatureScheme{
		Type: TypeCustomHMAC,
		Config: SignatureConfig{
			SignatureFormat: "hex",
			SignaturePrefix: "sha256=",
		},
	}

	result := svc.formatSignature(scheme, "abcdef1234", time.Now())
	assert.Equal(t, "sha256=abcdef1234", result)
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

	hexSig := hex.EncodeToString([]byte("test-signature-data"))
	result := svc.formatSignature(scheme, hexSig, time.Now())

	assert.True(t, strings.HasPrefix(result, "v1="))

	b64Part := strings.TrimPrefix(result, "v1=")
	decoded, err := base64.StdEncoding.DecodeString(b64Part)
	require.NoError(t, err)
	assert.Equal(t, "test-signature-data", string(decoded))
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

	assert.Equal(t, "t=1234567890,v1=abcdef1234", result)
}

// ============================================================
// parseSignature Tests
// ============================================================

func TestParseSignature_Stripe(t *testing.T) {
	svc, _ := newTestService()

	scheme := &SignatureScheme{
		Type: TypeStripe,
		Config: SignatureConfig{
			SignaturePrefix: "v1=",
		},
	}

	parsed := svc.parseSignature(scheme, "t=1234567890,v1=abcdef1234")
	assert.Equal(t, "abcdef1234", parsed)
}

func TestParseSignature_WithPrefix(t *testing.T) {
	svc, _ := newTestService()

	scheme := &SignatureScheme{
		Type: TypeCustomHMAC,
		Config: SignatureConfig{
			SignaturePrefix: "sha256=",
		},
	}

	parsed := svc.parseSignature(scheme, "sha256=abcdef1234")
	assert.Equal(t, "abcdef1234", parsed)
}

func TestParseSignature_Base64Format(t *testing.T) {
	svc, _ := newTestService()

	original := []byte("test-data")
	b64 := base64.StdEncoding.EncodeToString(original)

	scheme := &SignatureScheme{
		Type: TypeCustomHMAC,
		Config: SignatureConfig{
			SignatureFormat: "base64",
		},
	}

	parsed := svc.parseSignature(scheme, b64)
	assert.Equal(t, hex.EncodeToString(original), parsed)
}

func TestParseSignature_NoPrefix(t *testing.T) {
	svc, _ := newTestService()

	scheme := &SignatureScheme{
		Type:   TypeCustomHMAC,
		Config: SignatureConfig{},
	}

	parsed := svc.parseSignature(scheme, "abcdef1234")
	assert.Equal(t, "abcdef1234", parsed)
}

// ============================================================
// buildSignedPayload Tests
// ============================================================

func TestBuildSignedPayload(t *testing.T) {
	svc, _ := newTestService()

	scheme := &SignatureScheme{
		Config: SignatureConfig{
			SignedPayloadTemplate: "{id}.{timestamp}.{body}",
		},
	}

	ts := time.Unix(1234567890, 0)
	result := svc.buildSignedPayload(scheme, []byte("hello"), ts, "msg-123")

	assert.Equal(t, "msg-123.1234567890.hello", string(result))
}

func TestBuildSignedPayload_NoTemplate(t *testing.T) {
	svc, _ := newTestService()

	scheme := &SignatureScheme{
		Config: SignatureConfig{},
	}

	payload := []byte("raw-payload")
	result := svc.buildSignedPayload(scheme, payload, time.Now(), "")

	assert.Equal(t, "raw-payload", string(result))
}

func TestBuildSignedPayload_BodyOnly(t *testing.T) {
	svc, _ := newTestService()

	scheme := &SignatureScheme{
		Config: SignatureConfig{
			SignedPayloadTemplate: "{body}",
		},
	}

	result := svc.buildSignedPayload(scheme, []byte("just-body"), time.Now(), "")
	assert.Equal(t, "just-body", string(result))
}

func TestBuildSignedPayload_SlackFormat(t *testing.T) {
	svc, _ := newTestService()

	scheme := &SignatureScheme{
		Config: SignatureConfig{
			SignedPayloadTemplate: "v0:{timestamp}:{body}",
		},
	}

	ts := time.Unix(1000000000, 0)
	result := svc.buildSignedPayload(scheme, []byte("data"), ts, "")
	assert.Equal(t, "v0:1000000000:data", string(result))
}

func TestBuildSignedPayload_TimestampDotBody(t *testing.T) {
	svc, _ := newTestService()

	scheme := &SignatureScheme{
		Config: SignatureConfig{
			SignedPayloadTemplate: "{timestamp}.{body}",
		},
	}

	ts := time.Unix(1700000000, 0)
	result := svc.buildSignedPayload(scheme, []byte(`{"key":"value"}`), ts, "")
	assert.Equal(t, `1700000000.{"key":"value"}`, string(result))
}

// ============================================================
// ProcessPendingRotations Tests
// ============================================================

func TestProcessPendingRotations_ScheduledToInProgress(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")

	// Create a scheduled rotation
	rotation, err := svc.RotateKey(ctx, "tenant-1", scheme.ID, &RotateKeyRequest{
		Reason: "scheduled test",
	})
	require.NoError(t, err)
	assert.Equal(t, RotationScheduledStatus, rotation.Status)

	// Process pending rotations
	err = svc.ProcessPendingRotations(ctx)
	require.NoError(t, err)

	// The rotation should now be in progress
	updatedRotation, err := repo.GetRotation(ctx, rotation.ID)
	require.NoError(t, err)
	assert.Equal(t, RotationInProgress, updatedRotation.Status)
	assert.NotNil(t, updatedRotation.StartedAt)
}

func TestProcessPendingRotations_InProgressToCompleted(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	scheme := createTestScheme(t, svc, "tenant-1", TypeCustomHMAC, "")

	// Create a scheduled rotation
	rotation, err := svc.RotateKey(ctx, "tenant-1", scheme.ID, &RotateKeyRequest{
		Reason: "overlap-expiry-test",
	})
	require.NoError(t, err)

	// First process: scheduled -> in_progress
	err = svc.ProcessPendingRotations(ctx)
	require.NoError(t, err)

	// Set overlap_until to the past to simulate expiry
	rot, _ := repo.GetRotation(ctx, rotation.ID)
	past := time.Now().Add(-1 * time.Hour)
	rot.OverlapUntil = &past
	repo.SaveRotation(ctx, rot)

	// Second process: in_progress -> completed
	err = svc.ProcessPendingRotations(ctx)
	require.NoError(t, err)

	finalRotation, _ := repo.GetRotation(ctx, rotation.ID)
	assert.Equal(t, RotationCompleted, finalRotation.Status)
	assert.NotNil(t, finalRotation.CompletedAt)
}

func TestProcessPendingRotations_NoPending(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	err := svc.ProcessPendingRotations(ctx)
	require.NoError(t, err)
}

// ============================================================
// Cross-Tenant Isolation Tests
// ============================================================

func TestCrossTenantIsolation_Schemes(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	schemeA := createTestScheme(t, svc, "tenant-a", TypeCustomHMAC, "")
	schemeB := createTestScheme(t, svc, "tenant-b", TypeCustomHMAC, "")

	// Each tenant can only see their own
	schemesA, _ := svc.ListSchemes(ctx, "tenant-a")
	assert.Len(t, schemesA, 1)
	assert.Equal(t, schemeA.ID, schemesA[0].ID)

	schemesB, _ := svc.ListSchemes(ctx, "tenant-b")
	assert.Len(t, schemesB, 1)
	assert.Equal(t, schemeB.ID, schemesB[0].ID)
}

func TestCrossTenantIsolation_Signing(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	schemeA := createTestScheme(t, svc, "tenant-a", TypeCustomHMAC, "")

	// tenant-b cannot sign with tenant-a's scheme
	_, err := svc.Sign(ctx, "tenant-b", &SignatureRequest{
		SchemeID: schemeA.ID,
		Payload:  []byte("test"),
	})
	require.Error(t, err)
}

func TestCrossTenantIsolation_Verification(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	schemeA := createTestScheme(t, svc, "tenant-a", TypeCustomHMAC, "")
	now := time.Now()

	signResult := signPayload(t, svc, "tenant-a", schemeA.ID, []byte("test"), now)

	// tenant-b cannot verify tenant-a's signature
	result, err := svc.Verify(ctx, "tenant-b", &VerifyRequest{
		SchemeID:  schemeA.ID,
		Payload:   []byte("test"),
		Signature: signResult.Signature,
		Timestamp: &now,
	})
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Equal(t, "SCHEME_NOT_FOUND", result.ErrorCode)
}

// ============================================================
// ID Generation Tests
// ============================================================

func TestGenerateSchemeID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateSchemeID()
		assert.NotEmpty(t, id)
		assert.False(t, ids[id], "duplicate scheme ID generated")
		ids[id] = true
	}
}

func TestGenerateKeyID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateKeyID()
		assert.NotEmpty(t, id)
		assert.False(t, ids[id], "duplicate key ID generated")
		ids[id] = true
	}
}

func TestGenerateRotationID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateRotationID()
		assert.NotEmpty(t, id)
		assert.False(t, ids[id], "duplicate rotation ID generated")
		ids[id] = true
	}
}

// ============================================================
// Benchmarks
// ============================================================

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
