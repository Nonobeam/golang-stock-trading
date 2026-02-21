CREATE TABLE IF NOT EXISTS market_regime_tracking (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL UNIQUE,
    index_value DECIMAL(10,2) NOT NULL,
    volume BIGINT NOT NULL,
    volume_vs_avg_20d DECIMAL(5,2),
    rally_attempt_day INT CHECK (rally_attempt_day BETWEEN 1 AND 7),
    rally_attempt_baseline DECIMAL(10,2),
    is_ftd BOOLEAN DEFAULT FALSE,
    ftd_strength VARCHAR(20) CHECK (ftd_strength IN ('weak', 'moderate', 'strong')),
    ftd_score INT CHECK (ftd_score BETWEEN 0 AND 100),
    breadth_ratio DECIMAL(5,2),
    leader_participation_score INT CHECK (leader_participation_score BETWEEN 0 AND 20),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_market_regime_date ON market_regime_tracking(date DESC);
CREATE INDEX idx_rally_attempt ON market_regime_tracking(rally_attempt_day) WHERE rally_attempt_day IS NOT NULL;

CREATE TABLE IF NOT EXISTS ftd_events (
    id SERIAL PRIMARY KEY,
    event_date DATE NOT NULL,
    rally_attempt_start_date DATE NOT NULL,
    days_to_ftd INT NOT NULL CHECK (days_to_ftd BETWEEN 4 AND 7),
    ftd_strength VARCHAR(20) NOT NULL,
    ftd_score INT NOT NULL CHECK (ftd_score BETWEEN 0 AND 100),
    pattern_type VARCHAR(50),
    price_gain_pct DECIMAL(5,2),
    volume_ratio DECIMAL(5,2),
    breadth_ratio DECIMAL(5,2),
    leader_score INT,
    success_7d DECIMAL(5,2),
    success_14d DECIMAL(5,2),
    success_30d DECIMAL(5,2),
    is_validated BOOLEAN DEFAULT FALSE,
    invalidated_by_distribution BOOLEAN DEFAULT FALSE,
    invalidation_date DATE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_ftd_event_date ON ftd_events(event_date DESC);

CREATE TABLE IF NOT EXISTS market_breadth_daily (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL UNIQUE,
    advancing_stocks INT NOT NULL,
    declining_stocks INT NOT NULL,
    unchanged_stocks INT,
    new_highs INT,
    new_lows INT,
    sector_leaders JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_breadth_date ON market_breadth_daily(date DESC);
