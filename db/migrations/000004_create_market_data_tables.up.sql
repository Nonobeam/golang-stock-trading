CREATE TABLE IF NOT EXISTS daily_bars (
    symbol VARCHAR(10) NOT NULL,
    date DATE NOT NULL,
    open DECIMAL(12,2),
    high DECIMAL(12,2),
    low DECIMAL(12,2),
    close DECIMAL(12,2),
    volume BIGINT,
    turnover DECIMAL(15,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (symbol, date)
);

CREATE INDEX idx_daily_bars_symbol_date ON daily_bars(symbol, date DESC);

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

CREATE TABLE IF NOT EXISTS vnindex_daily (
    date DATE NOT NULL PRIMARY KEY,
    value DECIMAL(10,2),
    change DECIMAL(10,2),
    volume BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS spread_snapshots (
    symbol VARCHAR(10) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    bid_price DECIMAL(12,2),
    ask_price DECIMAL(12,2),
    spread_percent DECIMAL(5,2),
    PRIMARY KEY (symbol, timestamp)
);

CREATE INDEX idx_spread_snapshots_symbol_time ON spread_snapshots(symbol, timestamp DESC);
