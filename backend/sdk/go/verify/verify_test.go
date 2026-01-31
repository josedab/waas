package verify

import (
	"strconv"
	"testing"
	"time"
)

func TestVerifyValidSignature(t *testing.T) {
	secret := "whsec_test_secret_key"
	v := New(secret)

	payload := []byte(`{"event":"order.created","data":{"id":"123"}}`)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sig := v.Sign(payload, ts)

	if err := v.Verify(payload, sig, ts); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestVerifyInvalidSignature(t *testing.T) {
	v := New("secret1")

	payload := []byte(`{"event":"test"}`)
	ts := strconv.FormatInt(time.Now().Unix(), 10)

	if err := v.Verify(payload, "v1=deadbeef", ts); err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestVerifyMissingSignature(t *testing.T) {
	v := New("secret")
	if err := v.Verify([]byte("test"), "", ""); err != ErrMissingSignature {
		t.Fatalf("expected ErrMissingSignature, got %v", err)
	}
}

func TestVerifyExpiredTimestamp(t *testing.T) {
	v := New("secret", WithTimestampTolerance(1*time.Second))

	payload := []byte(`{"event":"test"}`)
	ts := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
	sig := v.Sign(payload, ts)

	if err := v.Verify(payload, sig, ts); err != ErrTimestampExpired {
		t.Fatalf("expected ErrTimestampExpired, got %v", err)
	}
}

func TestKeyRotation(t *testing.T) {
	oldSecret := "old_secret"
	newSecret := "new_secret"
	v := New(newSecret, WithSecrets(oldSecret))

	payload := []byte(`{"event":"test"}`)
	ts := strconv.FormatInt(time.Now().Unix(), 10)

	// Sign with old secret
	oldV := New(oldSecret)
	sig := oldV.Sign(payload, ts)

	// Verify with new verifier that knows both secrets
	if err := v.Verify(payload, sig, ts); err != nil {
		t.Fatalf("expected nil for old secret, got %v", err)
	}
}
