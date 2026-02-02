-- Remove min_signal_score from user_config (moving to per-stock preferences)
-- Create new table for per-stock signal score preferences

-- Remove old column
ALTER TABLE "stock-trading".user_config DROP COLUMN IF EXISTS min_signal_score;

-- Create new preferences table
CREATE TABLE "stock-trading".stock_signal_preferences (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    min_signal_score INT NOT NULL CHECK (min_signal_score >= 0 AND min_signal_score <= 13),
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- One preference per stock per user
    UNIQUE(user_id, symbol)
);

-- Indexes for performance
CREATE INDEX idx_stock_signal_prefs_user ON "stock-trading".stock_signal_preferences(user_id);
CREATE INDEX idx_stock_signal_prefs_symbol ON "stock-trading".stock_signal_preferences(symbol);
CREATE INDEX idx_stock_signal_prefs_user_symbol ON "stock-trading".stock_signal_preferences(user_id, symbol);

-- Comments
COMMENT ON TABLE "stock-trading".stock_signal_preferences IS 'Per-stock minimum signal score preferences for users';
COMMENT ON COLUMN "stock-trading".stock_signal_preferences.min_signal_score IS 'Minimum signal score (0-13) required to act on signals for this stock';
COMMENT ON COLUMN "stock-trading".stock_signal_preferences.notes IS 'User notes explaining why this threshold was set';
