-- Migration Rollback: 000012_add_portfolio_equity_tracking
-- Description: Remove portfolio equity tracking and R-multiple analytics tables
-- Created: 2026-02-03

-- Drop tables in reverse order of creation
DROP TABLE IF EXISTS r_multiple_statistics;
DROP TABLE IF EXISTS portfolio_equity_snapshots;
