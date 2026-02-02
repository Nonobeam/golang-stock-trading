-- Rollback: Restore spread_snapshots and stock_info_snapshots tables

CREATE TABLE IF NOT EXISTS stock_info_snapshots (
    symbol VARCHAR(10) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    last_price DECIMAL(12,2),
    ceiling DECIMAL(12,2),
    floor DECIMAL(12,2),
    reference DECIMAL(12,2),
    spread_percent DECIMAL(5,2),
    foreign_buy_vol BIGINT,
    foreign_sell_vol BIGINT,
    hit_ceiling BOOLEAN,
    hit_floor BOOLEAN,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (symbol, timestamp)
);

CREATE INDEX idx_stock_info_snapshots_symbol_time ON stock_info_snapshots(symbol, timestamp DESC);

CREATE TABLE IF NOT EXISTS spread_snapshots (
    symbol VARCHAR(10) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    bid_price DECIMAL(12,2),
    ask_price DECIMAL(12,2),
    spread_percent DECIMAL(5,2),
    PRIMARY KEY (symbol, timestamp)
);

CREATE INDEX idx_spread_snapshots_symbol_time ON spread_snapshots(symbol, timestamp DESC);
