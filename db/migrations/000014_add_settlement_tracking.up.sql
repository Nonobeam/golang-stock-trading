-- Migration: 000014_add_settlement_tracking
-- Description: Add T+2 settlement tracking for Vietnamese market reality
-- Created: 2026-02-06

-- Create settlement_status enum
CREATE TYPE settlement_status AS ENUM ('LOCKED_T0', 'LOCKED_T1', 'LOCKED_T2', 'LIQUID');

-- Add settlement tracking columns to positions table
ALTER TABLE positions ADD COLUMN IF NOT EXISTS settlement_status settlement_status DEFAULT 'LIQUID';
ALTER TABLE positions ADD COLUMN IF NOT EXISTS purchase_date TIMESTAMP;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS settlement_date TIMESTAMP;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS can_sell_date TIMESTAMP;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS locked_capital DECIMAL(15, 2) DEFAULT 0;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS liquid_capital DECIMAL(15, 2) DEFAULT 0;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS exchange VARCHAR(10) CHECK (exchange IN ('HOSE', 'HNX', 'UPCOM'));

-- Create position_settlement_tracking table for daily settlement snapshots
CREATE TABLE position_settlement_tracking (
    tracking_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    position_id UUID NOT NULL REFERENCES positions(id) ON DELETE CASCADE,
    check_date DATE NOT NULL,
    settlement_status settlement_status NOT NULL,
    days_until_liquid INTEGER NOT NULL DEFAULT 0,
    locked_value DECIMAL(15, 2) NOT NULL DEFAULT 0,
    locked_risk DECIMAL(15, 2) NOT NULL DEFAULT 0,
    risk_classification VARCHAR(30) CHECK (risk_classification IN ('HIGH_RISK_LOCKED', 'MODERATE_RISK_NEAR_LIQUID', 'LOW_RISK_LIQUID')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Ensure one snapshot per position per day
    UNIQUE(position_id, check_date)
);

-- Indexes for settlement tracking
CREATE INDEX idx_settlement_tracking_position ON position_settlement_tracking(position_id, check_date DESC);
CREATE INDEX idx_settlement_tracking_date ON position_settlement_tracking(check_date DESC);
CREATE INDEX idx_settlement_tracking_status ON position_settlement_tracking(settlement_status, check_date DESC);

-- Create theoretical_stop_breaches table
-- Tracks stop losses that were triggered but could not be executed due to settlement lock
CREATE TABLE theoretical_stop_breaches (
    breach_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    position_id UUID NOT NULL REFERENCES positions(id) ON DELETE CASCADE,
    breach_date TIMESTAMP NOT NULL,
    stop_price DECIMAL(15, 2) NOT NULL,
    actual_price DECIMAL(15, 2) NOT NULL,
    settlement_status settlement_status NOT NULL,
    days_until_executable INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for theoretical stop breaches
CREATE INDEX idx_theoretical_breaches_position ON theoretical_stop_breaches(position_id, breach_date DESC);
CREATE INDEX idx_theoretical_breaches_date ON theoretical_stop_breaches(breach_date DESC);

-- Add locked_risk_threshold to user_config table
ALTER TABLE user_config ADD COLUMN IF NOT EXISTS locked_risk_threshold DECIMAL(3, 2) DEFAULT 0.10
    CHECK (locked_risk_threshold >= 0.05 AND locked_risk_threshold <= 0.20);

-- Comments for documentation
COMMENT ON TYPE settlement_status IS 'T+2 settlement status: LOCKED_T0 (purchase day), LOCKED_T1 (T+1), LOCKED_T2 (T+2), LIQUID (T+3+, can sell)';

COMMENT ON COLUMN positions.settlement_status IS 'Current settlement status for this position';
COMMENT ON COLUMN positions.purchase_date IS 'Date of most recent purchase (used for settlement calculation)';
COMMENT ON COLUMN positions.settlement_date IS 'Date shares settle (T+2 from purchase)';
COMMENT ON COLUMN positions.can_sell_date IS 'Date shares become sellable (T+3 from purchase)';
COMMENT ON COLUMN positions.locked_capital IS 'Value of shares currently in settlement (cannot sell)';
COMMENT ON COLUMN positions.liquid_capital IS 'Value of shares already settled (can sell)';
COMMENT ON COLUMN positions.exchange IS 'Stock exchange: HOSE (7% floor), HNX (10% floor), UPCOM (15% floor)';

COMMENT ON TABLE position_settlement_tracking IS 'Daily snapshots of settlement status for risk tracking and audit trail';
COMMENT ON COLUMN position_settlement_tracking.days_until_liquid IS 'Days remaining until position becomes sellable';
COMMENT ON COLUMN position_settlement_tracking.locked_risk IS 'Worst-case floor-hit risk for locked shares (exchange-dependent)';
COMMENT ON COLUMN position_settlement_tracking.risk_classification IS 'HIGH_RISK_LOCKED: T+0-T+1, MODERATE_RISK_NEAR_LIQUID: T+2, LOW_RISK_LIQUID: T+3+';

COMMENT ON TABLE theoretical_stop_breaches IS 'Stop losses that were triggered but could not be executed due to settlement lock';
COMMENT ON COLUMN theoretical_stop_breaches.days_until_executable IS 'Days until stop loss would become executable';

COMMENT ON COLUMN user_config.locked_risk_threshold IS 'Maximum locked capital risk as % of account (default 10%, range 5-20%)';
