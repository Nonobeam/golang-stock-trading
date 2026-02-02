-- Rollback migration: 000001_initial_schema
-- Description: Drop all initial tables

-- Drop triggers first
DROP TRIGGER IF EXISTS update_trade_journal_updated_at ON trade_journal;
DROP TRIGGER IF EXISTS update_user_config_updated_at ON user_config;
DROP TRIGGER IF EXISTS update_watchlist_updated_at ON watchlist;
DROP TRIGGER IF EXISTS update_positions_updated_at ON positions;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_trade_journal_user;
DROP INDEX IF EXISTS idx_trade_journal_position;
DROP INDEX IF EXISTS idx_user_config_chat_id;
DROP INDEX IF EXISTS idx_signals_user;
DROP INDEX IF EXISTS idx_signals_score;
DROP INDEX IF EXISTS idx_signals_symbol_date;
DROP INDEX IF EXISTS idx_watchlist_user;
DROP INDEX IF EXISTS idx_watchlist_active;
DROP INDEX IF EXISTS idx_positions_user_open;
DROP INDEX IF EXISTS idx_positions_is_closed;
DROP INDEX IF EXISTS idx_positions_user_symbol;

-- Drop tables (in reverse order of dependencies)
DROP TABLE IF EXISTS trade_journal CASCADE;
DROP TABLE IF EXISTS signals_history CASCADE;
DROP TABLE IF EXISTS watchlist CASCADE;
DROP TABLE IF EXISTS user_config CASCADE;
DROP TABLE IF EXISTS positions CASCADE;

-- Drop extension if no longer needed
-- Note: Only drop if no other migrations use UUID
-- DROP EXTENSION IF EXISTS "uuid-ossp";
