-- Drop indexes
DROP INDEX IF EXISTS "stock-trading".idx_recs_ticker_date;
DROP INDEX IF EXISTS "stock-trading".idx_signals_ticker_date;

-- Drop tables
DROP TABLE IF EXISTS "stock-trading".position_recommendations;
DROP TABLE IF EXISTS "stock-trading".signals;

-- Remove columns from features table
ALTER TABLE "stock-trading".features 
DROP COLUMN IF EXISTS target_return_10d,
DROP COLUMN IF EXISTS target_return_5d;

-- Remove horizon from model_metadata
ALTER TABLE "stock-trading".model_metadata
DROP COLUMN IF EXISTS horizon;

