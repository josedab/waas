-- Rollback Zero-Trust Tables

DROP INDEX IF EXISTS idx_certificate_rotations_endpoint;
DROP INDEX IF EXISTS idx_signature_verifications_delivery;
DROP INDEX IF EXISTS idx_security_profiles_endpoint;
DROP INDEX IF EXISTS idx_signing_keys_key_id;
DROP INDEX IF EXISTS idx_signing_keys_tenant;
DROP INDEX IF EXISTS idx_endpoint_certificates_expiry;
DROP INDEX IF EXISTS idx_endpoint_certificates_fingerprint;
DROP INDEX IF EXISTS idx_endpoint_certificates_endpoint;

DROP TABLE IF EXISTS certificate_rotations;
DROP TABLE IF EXISTS signature_verifications;
DROP TABLE IF EXISTS endpoint_security_profiles;
DROP TABLE IF EXISTS signing_keys;
DROP TABLE IF EXISTS endpoint_certificates;
