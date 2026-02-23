// Package e2ee implements end-to-end encryption for webhook payloads.
//
// It provides per-endpoint asymmetric key pairs (X25519 for key exchange,
// XChaCha20-Poly1305 for payload encryption), automatic key rotation,
// and an audit log for all cryptographic operations.
package e2ee
