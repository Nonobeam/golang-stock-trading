-- Migration: 000013_add_position_entries_tracking (DOWN)
-- Description: Rollback position_entries table and aggregate tracking columns
-- Created: 2026-02-03

-- Drop trigger first
DROP TRIGGER IF EXISTS trigger_update_average_cost ON position_entries;

-- Drop function
DROP FUNCTION IF EXISTS update_position_average_cost();

-- Drop indexes
DROP INDEX IF EXISTS idx_position_entries_ticker;
DROP INDEX IF EXISTS idx_position_entries_date;
DROP INDEX IF EXISTS idx_position_entries_user;

-- Drop table
DROP TABLE IF EXISTS position_entries;

-- Remove added columns from positions table
ALTER TABLE positions DROP COLUMN IF EXISTS total_entries;
ALTER TABLE positions DROP COLUMN IF EXISTS total_fees_paid;
ALTER TABLE positions DROP COLUMN IF EXISTS first_entry_date;
ALTER TABLE positions DROP COLUMN IF EXISTS last_entry_date;
