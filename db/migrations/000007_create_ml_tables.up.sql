-- Create ML-related tables in stock-trading schema

-- Features table: stores calculated technical indicators
CREATE TABLE IF NOT EXISTS "stock-trading".features (
    ticker VARCHAR(10) NOT NULL,
    date DATE NOT NULL,
    
    -- Returns (4 features)
    return_1d DECIMAL(10,6),
    return_5d DECIMAL(10,6),
    return_20d DECIMAL(10,6),
    return_60d DECIMAL(10,6),
    
    -- Moving Averages - SMA (5 features)
    sma_5 DECIMAL(12,2),
    sma_10 DECIMAL(12,2),
    sma_20 DECIMAL(12,2),
    sma_50 DECIMAL(12,2),
    sma_200 DECIMAL(12,2),
    
    -- Moving Averages - EMA (2 features)
    ema_12 DECIMAL(12,2),
    ema_26 DECIMAL(12,2),
    
    -- Technical Indicators (9 features)
    rsi_14 DECIMAL(6,2),
    rsi_28 DECIMAL(6,2),
    macd DECIMAL(10,6),
    macd_signal DECIMAL(10,6),
    macd_hist DECIMAL(10,6),
    bb_upper DECIMAL(12,2),
    bb_middle DECIMAL(12,2),
    bb_lower DECIMAL(12,2),
    bb_width DECIMAL(10,6),
    
    -- Volume Features (6 features)
    volume_ratio_5d DECIMAL(10,6),
    volume_ratio_20d DECIMAL(10,6),
    volume_trend DECIMAL(6,4),
    obv BIGINT,
    turnover_ratio_5d DECIMAL(10,6),
    turnover_ratio_20d DECIMAL(10,6),
    
    -- Volatility (4 features)
    volatility_5d DECIMAL(10,6),
    volatility_20d DECIMAL(10,6),
    atr_14 DECIMAL(12,2),
    coefficient_variation DECIMAL(10,6),
    
    -- Price Ratios (4 features)
    price_to_sma20 DECIMAL(6,4),
    price_to_sma50 DECIMAL(6,4),
    price_to_sma200 DECIMAL(6,4),
    range_to_close DECIMAL(6,4),
    
    -- Feature metadata
    features_complete BOOLEAN DEFAULT FALSE,
    feature_version VARCHAR(10) DEFAULT 'v1.0',
    calculated_at TIMESTAMP DEFAULT NOW(),
    
    PRIMARY KEY (ticker, date)
);

-- Predictions table: stores ML model predictions and outcomes
CREATE TABLE IF NOT EXISTS "stock-trading".predictions (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    prediction_date DATE NOT NULL,
    target_date DATE NOT NULL,
    horizon INTEGER DEFAULT 1,
    
    -- Quantile predictions
    p10 DECIMAL(10,6) NOT NULL,
    p50 DECIMAL(10,6) NOT NULL,
    p90 DECIMAL(10,6) NOT NULL,
    confidence DECIMAL(6,4),
    
    -- Actual outcomes (filled later)
    actual_return DECIMAL(10,6),
    error_p10 DECIMAL(10,6),
    error_p50 DECIMAL(10,6),
    error_p90 DECIMAL(10,6),
    
    -- Model metadata
    model_version VARCHAR(20) NOT NULL,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE (ticker, prediction_date, target_date, horizon)
);

-- Model metadata table: tracks trained models
CREATE TABLE IF NOT EXISTS "stock-trading".model_metadata (
    id SERIAL PRIMARY KEY,
    model_id VARCHAR(50) UNIQUE NOT NULL,
    ticker VARCHAR(10) NOT NULL,
    quantile DECIMAL(4,2) NOT NULL,
    
    -- Training period
    training_date DATE NOT NULL,
    train_start_date DATE NOT NULL,
    train_end_date DATE NOT NULL,
    train_days INTEGER NOT NULL,
    
    -- Validation period
    val_start_date DATE,
    val_end_date DATE,
    
    -- Configuration and metrics
    hyperparameters JSONB,
    metrics JSONB,
    
    -- Production status
    in_production BOOLEAN DEFAULT FALSE,
    
    -- Model file location
    file_path VARCHAR(255),
    
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_features_ticker_date ON "stock-trading".features(ticker, date DESC);
CREATE INDEX IF NOT EXISTS idx_features_complete ON "stock-trading".features(features_complete) WHERE features_complete = TRUE;
CREATE INDEX IF NOT EXISTS idx_features_ticker_recent ON "stock-trading".features(ticker, date DESC) WHERE features_complete = TRUE;

CREATE INDEX IF NOT EXISTS idx_predictions_ticker_target ON "stock-trading".predictions(ticker, target_date DESC);
CREATE INDEX IF NOT EXISTS idx_predictions_pending ON "stock-trading".predictions(ticker) WHERE actual_return IS NULL;
CREATE INDEX IF NOT EXISTS idx_predictions_ticker_date ON "stock-trading".predictions(ticker, prediction_date DESC);

CREATE INDEX IF NOT EXISTS idx_model_ticker_prod ON "stock-trading".model_metadata(ticker, in_production) WHERE in_production = TRUE;
CREATE INDEX IF NOT EXISTS idx_model_ticker_date ON "stock-trading".model_metadata(ticker, training_date DESC);
CREATE INDEX IF NOT EXISTS idx_model_quantile ON "stock-trading".model_metadata(ticker, quantile, in_production);
