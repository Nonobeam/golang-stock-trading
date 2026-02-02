-- Migration rollback: 000010_add_position_indexes
-- Description: Remove indexes and constraints added for position management

-- Remove constraints
ALTER TABLE positions 
DROP CONSTRAINT IF EXISTS chk_positions_entry_price_positive;

ALTER TABLE positions 
DROP CONSTRAINT IF EXISTS chk_positions_quantity_positive;

-- Remove indexes
DROP INDEX IF EXISTS idx_positions_user_active;
DROP INDEX IF EXISTS idx_positions_active_symbol;
