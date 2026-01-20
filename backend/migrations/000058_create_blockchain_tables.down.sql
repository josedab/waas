-- Drop blockchain tables

DROP TRIGGER IF EXISTS trigger_chain_configs_updated ON chain_configs;
DROP TRIGGER IF EXISTS trigger_contract_monitors_updated ON contract_monitors;

DROP TABLE IF EXISTS blockchain_blocks;
DROP TABLE IF EXISTS chain_configs;
DROP TABLE IF EXISTS blockchain_events;
DROP TABLE IF EXISTS contract_monitors;
