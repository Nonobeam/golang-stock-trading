-- Fix numeric overflow for high-priced Vietnamese stocks (e.g. GAS ~100,000 VND)
-- MACD/signal/hist are price-derived (not ratios), so they need room for 6-digit prices.
-- DECIMAL(10,6) max = 9999.999999 → too small.
-- Widening to DECIMAL(16,6) max = 9,999,999,999.999999 → safe for any stock price.

ALTER TABLE "stock-trading".features
    ALTER COLUMN macd         TYPE DECIMAL(16,6),
    ALTER COLUMN macd_signal  TYPE DECIMAL(16,6),
    ALTER COLUMN macd_hist    TYPE DECIMAL(16,6);

-- Also widen return columns just in case (extreme returns on penny stocks)
-- DECIMAL(10,6) allows -9999.999999 which is fine for returns, but use (12,6) for safety
ALTER TABLE "stock-trading".features
    ALTER COLUMN return_1d    TYPE DECIMAL(12,6),
    ALTER COLUMN return_5d    TYPE DECIMAL(12,6),
    ALTER COLUMN return_20d   TYPE DECIMAL(12,6),
    ALTER COLUMN return_60d   TYPE DECIMAL(12,6);
