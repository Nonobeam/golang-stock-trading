-- Migration: Add validation infrastructure tables
-- Created: 2026-02-02
-- Purpose: Support walk-forward validation, calibration checking, coverage tracking,
--          feature stability analysis, and floor-hit risk prediction

-- Walk-forward validation results
CREATE TABLE IF NOT EXISTS walk_forward_results (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    model_horizon INT NOT NULL,  -- 1, 5, or 10 days
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    mae DECIMAL(8,4),            -- Mean Absolute Error (%)
    ic DECIMAL(6,4),              -- Information Coefficient (-1 to 1)
    directional_accuracy DECIMAL(6,4),  -- Proportion correct direction (0 to 1)
    sharpe_ratio DECIMAL(8,4),
    fee_adjusted_sharpe DECIMAL(8,4),  -- After 0.4% transaction costs
    num_predictions INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_walk_forward_ticker_horizon ON walk_forward_results(ticker, model_horizon);
CREATE INDEX idx_walk_forward_period ON walk_forward_results(period_start, period_end);

-- Quantile calibration reports
CREATE TABLE IF NOT EXISTS calibration_reports (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    model_horizon INT NOT NULL,
    quantile_level VARCHAR(5) NOT NULL,  -- 'p10', 'p25', 'p50', 'p75', 'p90'
    expected_coverage DECIMAL(6,4) NOT NULL,  -- 0.10 for p10, 0.90 for p90
    actual_coverage DECIMAL(6,4) NOT NULL,    -- Empirical coverage observed
    calibration_error DECIMAL(6,4) NOT NULL,  -- actual - expected
    num_samples INT NOT NULL,
    check_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL,  -- 'OK', 'WARNING', 'ERROR'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_calibration_ticker_horizon ON calibration_reports(ticker, model_horizon);
CREATE INDEX idx_calibration_date ON calibration_reports(check_date);
CREATE INDEX idx_calibration_status ON calibration_reports(status);

-- Prediction interval coverage tracking
CREATE TABLE IF NOT EXISTS prediction_coverage (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    model_horizon INT NOT NULL,
    prediction_date DATE NOT NULL,
    actual_date DATE NOT NULL,
    predicted_p10 DECIMAL(8,4),
    predicted_p50 DECIMAL(8,4),
    predicted_p90 DECIMAL(8,4),
    actual_return DECIMAL(8,4),
    within_range BOOLEAN NOT NULL,  -- TRUE if actual in [p10, p90]
    below_p10 BOOLEAN NOT NULL,
    above_p90 BOOLEAN NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_coverage_ticker_horizon ON prediction_coverage(ticker, model_horizon);
CREATE INDEX idx_coverage_prediction_date ON prediction_coverage(prediction_date);
CREATE INDEX idx_coverage_actual_date ON prediction_coverage(actual_date);

-- Feature importance stability tracking
CREATE TABLE IF NOT EXISTS feature_stability (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    model_horizon INT NOT NULL,
    feature_name VARCHAR(100) NOT NULL,
    importance_gain DECIMAL(10,6),     -- XGBoost gain metric
    importance_weight DECIMAL(10,6),   -- XGBoost weight metric
    importance_cover DECIMAL(10,6),    -- XGBoost cover metric
    rank_position INT,                  -- Rank by gain (1 = most important)
    training_date DATE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_feature_stability_ticker_feature ON feature_stability(ticker, feature_name);
CREATE INDEX idx_feature_stability_date ON feature_stability(training_date);
CREATE INDEX idx_feature_stability_horizon ON feature_stability(model_horizon);

-- Floor/ceiling hit probability predictions
CREATE TABLE IF NOT EXISTS floor_hit_probabilities (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    exchange VARCHAR(10) NOT NULL,  -- 'HOSE' (±7%) or 'HNX' (±10%)
    prediction_date DATE NOT NULL,
    floor_probability DECIMAL(6,4) NOT NULL,  -- Probability of hitting floor limit
    ceiling_probability DECIMAL(6,4) NOT NULL,  -- Probability of hitting ceiling limit
    actual_hit_floor BOOLEAN,  -- TRUE if actually hit floor next day (filled next day)
    actual_hit_ceiling BOOLEAN,  -- TRUE if actually hit ceiling next day
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_floor_hit_ticker ON floor_hit_probabilities(ticker);
CREATE INDEX idx_floor_hit_date ON floor_hit_probabilities(prediction_date);
CREATE INDEX idx_floor_hit_exchange ON floor_hit_probabilities(exchange);
