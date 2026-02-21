-- Rollback exit tracking fields

ALTER TABLE positions DROP COLUMN IF EXISTS target1_filled;
ALTER TABLE positions DROP COLUMN IF EXISTS target2_filled;
ALTER TABLE positions DROP COLUMN IF EXISTS trailing_stop_active;

ALTER TABLE positions DROP COLUMN IF EXISTS target1_exit_price;
ALTER TABLE positions DROP COLUMN IF EXISTS target2_exit_price;

ALTER TABLE positions DROP COLUMN IF EXISTS target1_exit_date;
ALTER TABLE positions DROP COLUMN IF EXISTS target2_exit_date;

ALTER TABLE positions DROP COLUMN IF EXISTS ceiling_hit_date;
ALTER TABLE positions DROP COLUMN IF EXISTS ceiling_lock_days;
ALTER TABLE positions DROP COLUMN IF EXISTS floor_hit_days;
ALTER TABLE positions DROP COLUMN IF EXISTS last_floor_date;

DROP INDEX IF EXISTS idx_positions_exit_tracking;
