-- Migration: Remove current_balance column from user_config
-- The current_balance will be calculated dynamically from positions

ALTER TABLE user_config DROP COLUMN IF EXISTS current_balance;

COMMENT ON TABLE user_config IS 'Per-user trading preferences and settings. Balance calculated from positions.';
