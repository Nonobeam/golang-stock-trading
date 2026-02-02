-- Add multi-horizon target columns to features table
ALTER TABLE "stock-trading".features 
ADD COLUMN IF NOT EXISTS target_return_5d DECIMAL(10,6),
ADD COLUMN IF NOT EXISTS target_return_10d DECIMAL(10,6);

-- Create signals table with simplified schema using JSONB metadata
CREATE TABLE IF NOT EXISTS "stock-trading".signals (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    signal_date DATE NOT NULL,
    
    -- Signal Output
    signal VARCHAR(20) NOT NULL, -- BUY, SELL, HOLD
    strength DECIMAL(4,3),       -- Signal strength 0.0-1.0
    reason TEXT,                 -- Human-readable explanation
    
    -- Store all predictions and context in JSONB for flexibility
    metadata JSONB DEFAULT '{}'::JSONB,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE (ticker, signal_date)
);

-- Create position_recommendations table
CREATE TABLE IF NOT EXISTS "stock-trading".position_recommendations (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    recommendation_date DATE NOT NULL,
    
    -- Position Sizing Calculations
    position_fraction DECIMAL(6,4),      -- Recommended allocation (0.0-1.0)
    confidence_multiplier DECIMAL(4,2),  -- Confidence adjustment
    horizon_multiplier DECIMAL(4,2),     -- Horizon adjustment
    
    -- Position Sizing Inputs/Outputs
    account_value DECIMAL(15,2),
    current_position_value DECIMAL(15,2),
    target_position_value DECIMAL(15,2),
    recommendation_shares INTEGER,       -- positive=buy, negative=sell
    recommendation_value DECIMAL(15,2),
    
    -- Store detailed calculation in JSONB
    calculation_details JSONB DEFAULT '{}'::JSONB,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE (ticker, recommendation_date)
);

-- Add indexes for performance
CREATE INDEX IF NOT EXISTS idx_signals_ticker_date ON "stock-trading".signals(ticker, signal_date DESC);
CREATE INDEX IF NOT EXISTS idx_recs_ticker_date ON "stock-trading".position_recommendations(ticker, recommendation_date DESC);

-- Add JSONB GIN index for efficient querying of metadata
CREATE INDEX IF NOT EXISTS idx_signals_metadata ON "stock-trading".signals USING GIN (metadata);

-- Add horizon to model_metadata
ALTER TABLE "stock-trading".model_metadata
ADD COLUMN IF NOT EXISTS horizon INTEGER DEFAULT 1;

-- Add calibration_score to model_metadata (default 0.70 for new models)
ALTER TABLE "stock-trading".model_metadata
ADD COLUMN IF NOT EXISTS calibration_score DECIMAL(4,3) DEFAULT 0.70;

COMMENT ON COLUMN "stock-trading".model_metadata.calibration_score IS 'Coverage ratio / 0.80 - measures if p10-p90 range captures actual outcomes. Updated weekly during retraining.';
COMMENT ON COLUMN "stock-trading".signals.metadata IS 'JSONB storing predictions, position context, and other flexible data. Query with metadata->>''key''';
