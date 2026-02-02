-- Rollback migration for validation infrastructure

DROP INDEX IF EXISTS idx_floor_hit_exchange;
DROP INDEX IF EXISTS idx_floor_hit_date;
DROP INDEX IF EXISTS idx_floor_hit_ticker;
DROP TABLE IF EXISTS floor_hit_probabilities;

DROP INDEX IF EXISTS idx_feature_stability_horizon;
DROP INDEX IF EXISTS idx_feature_stability_date;
DROP INDEX IF EXISTS idx_feature_stability_ticker_feature;
DROP TABLE IF EXISTS feature_stability;

DROP INDEX IF EXISTS idx_coverage_actual_date;
DROP INDEX IF EXISTS idx_coverage_prediction_date;
DROP INDEX IF EXISTS idx_coverage_ticker_horizon;
DROP TABLE IF EXISTS prediction_coverage;

DROP INDEX IF EXISTS idx_calibration_status;
DROP INDEX IF EXISTS idx_calibration_date;
DROP INDEX IF EXISTS idx_calibration_ticker_horizon;
DROP TABLE IF EXISTS calibration_reports;

DROP INDEX IF EXISTS idx_walk_forward_period;
DROP INDEX IF EXISTS idx_walk_forward_ticker_horizon;
DROP TABLE IF EXISTS walk_forward_results;
