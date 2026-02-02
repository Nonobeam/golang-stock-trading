-- Rollback: Restore current_balance column to user_config

ALTER TABLE user_config ADD COLUMN IF NOT EXISTS current_balance DECIMAL(15, 2);

COMMENT ON TABLE user_config IS 'Per-user trading preferences and settings';
