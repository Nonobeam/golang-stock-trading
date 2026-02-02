-- Migration: 000001_initial_schema
-- Description: Create initial tables for trading bot
-- Created: 2026-01-11

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Table: positions
-- Stores user's stock positions (both open and closed)
CREATE TABLE positions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id BIGINT NOT NULL,
    symbol VARCHAR(10) NOT NULL,
    
    -- Entry details
    entry_date TIMESTAMP NOT NULL,
    entry_price DECIMAL(15, 2) NOT NULL,
    quantity INTEGER NOT NULL,
    
    -- Risk management
    stop_loss DECIMAL(15, 2) NOT NULL,
    target_1 DECIMAL(15, 2),
    target_2 DECIMAL(15, 2),
    target_3 DECIMAL(15, 2),
    
    -- Signal metadata
    signal_type VARCHAR(50),
    score INTEGER CHECK (score >= 0 AND score <= 13),
    notes TEXT,
    
    -- Exit details (NULL if position still open)
    is_closed BOOLEAN DEFAULT FALSE,
    exit_date TIMESTAMP,
    exit_price DECIMAL(15, 2),
    exit_reason VARCHAR(100),
    
    -- Performance metrics
    pnl DECIMAL(15, 2),
    pnl_percent DECIMAL(10, 4),
    r_multiple DECIMAL(10, 4),
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_positions_user_symbol ON positions(user_id, symbol);
CREATE INDEX idx_positions_is_closed ON positions(is_closed);
CREATE INDEX idx_positions_user_open ON positions(user_id, is_closed) WHERE is_closed = FALSE;

-- Table: watchlist
-- Stocks user wants to monitor for buy opportunities
CREATE TABLE watchlist (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id BIGINT NOT NULL,
    symbol VARCHAR(10) NOT NULL,
    
    -- Alert settings
    target_price DECIMAL(15, 2),
    notes TEXT,
    signal_types JSONB DEFAULT '[]'::jsonb, -- e.g., ["pullback", "breakout"]
    min_score INTEGER DEFAULT 7 CHECK (min_score >= 0 AND min_score <= 13),
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    alert_sent BOOLEAN DEFAULT FALSE,
    last_alert_at TIMESTAMP,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(user_id, symbol)
);

CREATE INDEX idx_watchlist_active ON watchlist(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_watchlist_user ON watchlist(user_id);

-- Table: signals_history
-- Historical record of all detected signals
CREATE TABLE signals_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    symbol VARCHAR(10) NOT NULL,
    
    -- Signal details
    signal_type VARCHAR(50) NOT NULL,
    score INTEGER NOT NULL CHECK (score >= 0 AND score <= 13),
    
    -- Trade setup
    entry_price DECIMAL(15, 2) NOT NULL,
    stop_loss DECIMAL(15, 2) NOT NULL,
    targets JSONB, -- Array of target prices
    position_size INTEGER,
    risk_amount DECIMAL(15, 2),
    
    -- Detection metadata
    detected_at TIMESTAMP NOT NULL,
    regime VARCHAR(20), -- "bull", "bear", "range"
    
    -- User interaction
    sent_to_user BOOLEAN DEFAULT FALSE,
    user_action VARCHAR(20), -- "taken", "ignored", "watchlisted", NULL
    user_id BIGINT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_signals_symbol_date ON signals_history(symbol, detected_at DESC);
CREATE INDEX idx_signals_score ON signals_history(score DESC);
CREATE INDEX idx_signals_user ON signals_history(user_id) WHERE user_id IS NOT NULL;

-- Table: user_config
-- Per-user trading configuration
CREATE TABLE user_config (
    user_id BIGINT PRIMARY KEY,
    telegram_chat_id BIGINT UNIQUE NOT NULL,
    
    -- Capital management
    initial_capital DECIMAL(15, 2) DEFAULT 100000000,
    current_balance DECIMAL(15, 2),
    
    -- Trading parameters
    max_positions INTEGER DEFAULT 3 CHECK (max_positions > 0),
    risk_per_trade DECIMAL(5, 4) DEFAULT 0.02 CHECK (risk_per_trade > 0 AND risk_per_trade <= 0.1),
    min_signal_score INTEGER DEFAULT 7 CHECK (min_signal_score >= 0 AND min_signal_score <= 13),
    
    -- Notification settings
    notification_enabled BOOLEAN DEFAULT TRUE,
    daily_report_enabled BOOLEAN DEFAULT TRUE,
    daily_report_time TIME DEFAULT '08:00:00',
    timezone VARCHAR(50) DEFAULT 'Asia/Ho_Chi_Minh',
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_config_chat_id ON user_config(telegram_chat_id);

-- Table: trade_journal (optional, for future)
-- Detailed notes and analysis for each trade
CREATE TABLE trade_journal (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    position_id UUID REFERENCES positions(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    
    -- Pre-trade
    pre_trade_plan TEXT,
    pre_trade_checklist JSONB, -- {"trend_confirmed": true, "stop_placed": true, ...}
    
    -- During trade
    trade_notes TEXT,
    emotional_state VARCHAR(50),
    
    -- Post-trade
    post_trade_review TEXT,
    lessons_learned TEXT,
    mistakes TEXT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_trade_journal_position ON trade_journal(position_id);
CREATE INDEX idx_trade_journal_user ON trade_journal(user_id);

-- Function: Update updated_at timestamp automatically
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers: Auto-update updated_at
CREATE TRIGGER update_positions_updated_at
    BEFORE UPDATE ON positions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_watchlist_updated_at
    BEFORE UPDATE ON watchlist
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_config_updated_at
    BEFORE UPDATE ON user_config
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_trade_journal_updated_at
    BEFORE UPDATE ON trade_journal
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments for documentation
COMMENT ON TABLE positions IS 'Tracks all user stock positions (open and closed)';
COMMENT ON TABLE watchlist IS 'Stocks being monitored for buy opportunities';
COMMENT ON TABLE signals_history IS 'Historical log of all detected trading signals';
COMMENT ON TABLE user_config IS 'Per-user trading preferences and settings';
COMMENT ON TABLE trade_journal IS 'Detailed trade notes and post-trade reviews';

COMMENT ON COLUMN positions.r_multiple IS 'Actual gain/loss divided by initial risk';
COMMENT ON COLUMN watchlist.signal_types IS 'JSON array of signal types to watch for';
COMMENT ON COLUMN signals_history.targets IS 'JSON array of target prices [T1, T2, T3]';
