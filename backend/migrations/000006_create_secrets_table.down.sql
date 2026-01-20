-- Remove encrypted payload columns from delivery attempts
ALTER TABLE delivery_attempts 
DROP COLUMN IF EXISTS encrypted_payload,
DROP COLUMN IF EXISTS payload_encryption_key_id;

-- Drop secrets table
DROP TABLE IF EXISTS secret_versions;