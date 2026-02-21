-- Add exit tracking fields to positions table for graduated exit strategy

ALTER TABLE positions ADD COLUMN IF NOT EXISTS target1_filled BOOLEAN DEFAULT FALSE;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS target2_filled BOOLEAN DEFAULT FALSE;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS trailing_stop_active BOOLEAN DEFAULT FALSE;

ALTER TABLE positions ADD COLUMN IF NOT EXISTS target1_exit_price DECIMAL(10,2);
ALTER TABLE positions ADD COLUMN IF NOT EXISTS target2_exit_price DECIMAL(10,2);

ALTER TABLE positions ADD COLUMN IF NOT EXISTS target1_exit_date TIMESTAMP;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS target2_exit_date TIMESTAMP;

-- Vietnamese market-specific fields
ALTER TABLE positions ADD COLUMN IF NOT EXISTS ceiling_hit_date DATE;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS ceiling_lock_days INT DEFAULT 0;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS floor_hit_days INT DEFAULT 0;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS last_floor_date DATE;

-- Index for querying open positions with unfilled targets
CREATE INDEX IF NOT EXISTS idx_positions_exit_tracking 
ON positions(status, target1_filled, target2_filled) 
WHERE status = 'OPEN';

COMMENT ON COLUMN positions.target1_filled IS 'True if Target 1 exit (30% at +15%) has been executed';
COMMENT ON COLUMN positions.target2_filled IS 'True if Target 2 exit (30% at +25%) has been executed';
COMMENT ON COLUMN positions.trailing_stop_active IS 'True if trailing stop is active for remaining 40%';
COMMENT ON COLUMN positions.ceiling_hit_date IS 'Date when stock hit +7% ceiling (Vietnamese market)';
COMMENT ON COLUMN positions.floor_hit_days IS 'Consecutive days stock hit -7% floor (triggers emergency exit at 3+)';
