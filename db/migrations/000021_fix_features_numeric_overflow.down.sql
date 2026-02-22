-- Revert: restore original column widths
ALTER TABLE "stock-trading".features
    ALTER COLUMN macd         TYPE DECIMAL(10,6),
    ALTER COLUMN macd_signal  TYPE DECIMAL(10,6),
    ALTER COLUMN macd_hist    TYPE DECIMAL(10,6),
    ALTER COLUMN return_1d    TYPE DECIMAL(10,6),
    ALTER COLUMN return_5d    TYPE DECIMAL(10,6),
    ALTER COLUMN return_20d   TYPE DECIMAL(10,6),
    ALTER COLUMN return_60d   TYPE DECIMAL(10,6);
