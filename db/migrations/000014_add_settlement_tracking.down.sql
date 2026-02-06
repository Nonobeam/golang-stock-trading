-- Migration: 000014_add_settlement_tracking (DOWN)
-- Description: Rollback T+2 settlement tracking
-- Created: 2026-02-06

-- Drop theoretical stop breaches table
DROP TABLE IF EXISTS theoretical_stop_breaches;

-- Drop position settlement tracking table
DROP TABLE IF EXISTS position_settlement_tracking;

-- Remove settlement columns from positions table
ALTER TABLE positions DROP COLUMN IF EXISTS settlement_status;
ALTER TABLE positions DROP COLUMN IF EXISTS purchase_date;
ALTER TABLE positions DROP COLUMN IF EXISTS settlement_date;
ALTER TABLE positions DROP COLUMN IF EXISTS can_sell_date;
ALTER TABLE positions DROP COLUMN IF EXISTS locked_capital;
ALTER TABLE positions DROP COLUMN IF EXISTS liquid_capital;
ALTER TABLE positions DROP COLUMN IF EXISTS exchange;

-- Remove locked_risk_threshold from user_config
ALTER TABLE user_config DROP COLUMN IF EXISTS locked_risk_threshold;

-- Drop settlement_status enum type
DROP TYPE IF EXISTS settlement_status;
