-- Migration: 000012_add_portfolio_equity_tracking
-- Description: Add portfolio-level equity tracking and R-multiple analytics tables
-- Created: 2026-02-03

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Table: portfolio_equity_snapshots
-- Tracks daily portfolio equity values for drawdown calculations
CREATE TABLE portfolio_equity_snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id BIGINT NOT NULL,
    snapshot_date DATE NOT NULL,
    
    -- Equity components
    total_equity DECIMAL(15, 2) NOT NULL,
    peak_equity DECIMAL(15, 2) NOT NULL,
    current_drawdown DECIMAL(10, 4) NOT NULL,
    
    -- Breakdown for debugging and analysis
    open_positions_value DECIMAL(15, 2),
    closed_pnl DECIMAL(15, 2),
    cash_balance DECIMAL(15, 2),
    
    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure one snapshot per user per day
    UNIQUE(user_id, snapshot_date)
);

-- Indexes for efficient querying
CREATE INDEX idx_equity_snapshots_user_date ON portfolio_equity_snapshots(user_id, snapshot_date DESC);
CREATE INDEX idx_equity_snapshots_date ON portfolio_equity_snapshots(snapshot_date DESC);
CREATE INDEX idx_equity_snapshots_drawdown ON portfolio_equity_snapshots(current_drawdown) WHERE current_drawdown < -0.05;

-- Table: r_multiple_statistics
-- Daily portfolio-level R-multiple aggregations
CREATE TABLE r_multiple_statistics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id BIGINT NOT NULL,
    calculation_date DATE NOT NULL,
    
    -- Aggregate statistics
    avg_r_multiple DECIMAL(10, 4),
    median_r_multiple DECIMAL(10, 4),
    stddev_r_multiple DECIMAL(10, 4),
    
    -- Distribution metrics
    best_r_multiple DECIMAL(10, 4),
    worst_r_multiple DECIMAL(10, 4),
    
    -- Performance metrics
    win_rate DECIMAL(5, 4), -- % of trades with R > 0
    total_trades INTEGER,
    profitable_trades INTEGER,
    
    -- Breakdown by signal type (JSON)
    r_by_signal_type JSONB,
    
    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure one statistics record per user per day
    UNIQUE(user_id, calculation_date)
);

-- Indexes for efficient querying
CREATE INDEX idx_r_stats_user_date ON r_multiple_statistics(user_id, calculation_date DESC);
CREATE INDEX idx_r_stats_date ON r_multiple_statistics(calculation_date DESC);
CREATE INDEX idx_r_stats_avg_r ON r_multiple_statistics(avg_r_multiple DESC);

-- Comments for documentation
COMMENT ON TABLE portfolio_equity_snapshots IS 'Daily portfolio equity snapshots for drawdown calculation and risk management';
COMMENT ON TABLE r_multiple_statistics IS 'Daily aggregated R-multiple statistics for performance tracking';

COMMENT ON COLUMN portfolio_equity_snapshots.total_equity IS 'Total portfolio equity = open positions value + closed P&L + cash';
COMMENT ON COLUMN portfolio_equity_snapshots.peak_equity IS 'Historical maximum equity (running max)';
COMMENT ON COLUMN portfolio_equity_snapshots.current_drawdown IS 'Drawdown from peak = (current - peak) / peak';
COMMENT ON COLUMN r_multiple_statistics.r_by_signal_type IS 'JSON breakdown of R-multiple by signal type';
