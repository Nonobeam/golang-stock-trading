-- Migration: Remove unused market data snapshot tables
-- These tables have no code references and are not being populated

-- Drop spread_snapshots table
DROP TABLE IF EXISTS spread_snapshots CASCADE;
DROP INDEX IF EXISTS idx_spread_snapshots_symbol_time;

-- Drop stock_info_snapshots table  
DROP TABLE IF EXISTS stock_info_snapshots CASCADE;
DROP INDEX IF EXISTS idx_stock_info_snapshots_symbol_time;

-- Comments
COMMENT ON SCHEMA public IS 'Removed spread_snapshots and stock_info_snapshots tables - no code usage found';
