-- Migration: 000010_add_position_indexes
-- Description: Add indexes and constraints to positions table for position management
-- Created: 2026-02-02

-- Add index for fast active position lookup by symbol
CREATE INDEX IF NOT EXISTS idx_positions_active_symbol 
ON positions(symbol, is_closed) 
WHERE is_closed = FALSE;

-- Add index for user's active positions (multi-position queries)
CREATE INDEX IF NOT EXISTS idx_positions_user_active 
ON positions(user_id, is_closed, symbol);

-- Add constraint to ensure quantity is always positive
ALTER TABLE positions 
ADD CONSTRAINT chk_positions_quantity_positive 
CHECK (quantity > 0);

-- Add constraint to ensure entry_price is always positive
ALTER TABLE positions 
ADD CONSTRAINT chk_positions_entry_price_positive 
CHECK (entry_price > 0);

-- Comments for documentation
COMMENT ON INDEX idx_positions_active_symbol IS 'Fast lookup for active positions by symbol';
COMMENT ON INDEX idx_positions_user_active IS 'Efficient queries for all user active positions';
