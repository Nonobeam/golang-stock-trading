-- Rollback: Remove stock_signal_preferences table and restore min_signal_score to user_config

-- Drop indexes
DROP INDEX IF EXISTS "stock-trading".idx_stock_signal_prefs_user_symbol;
DROP INDEX IF EXISTS "stock-trading".idx_stock_signal_prefs_symbol;
DROP INDEX IF EXISTS "stock-trading".idx_stock_signal_prefs_user;

-- Drop table
DROP TABLE IF EXISTS "stock-trading".stock_signal_preferences CASCADE;

-- Restore column to user_config
ALTER TABLE "stock-trading".user_config 
ADD COLUMN IF NOT EXISTS min_signal_score INTEGER DEFAULT 7 CHECK (min_signal_score >= 0 AND min_signal_score <= 13);
