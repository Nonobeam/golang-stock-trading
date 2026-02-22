## Implementation Checklist

### 1. Config

- [x] 1.1 Add threshold keys to `PORTFOLIO_CONFIG` in `config.py`:
  - `trend_sma_short` (20), `trend_sma_long` (60), `trend_slope_days` (10)
  - `momentum_20d_min` (−0.05), `momentum_60d_min` (−0.10)
  - `volume_ratio_min` (0.80), `volume_short_window` (10), `volume_long_window` (30)
  - `rsi_period` (14), `rsi_lower` (35), `rsi_upper` (75)
  - `hhhl_window` (20)
  - `sharp_drop_window` (10), `sharp_drop_threshold` (−0.05)
  - `high52w_lookback` (252), `high52w_ratio_min` (0.70)
  - `posdays_window` (20), `posdays_ratio_min` (0.45)

### 2. Data Loading

- [x] 2.1 Add `_load_price_history(tickers, lookback_days, date)` to
      `selector.py` — query `daily_bars` returning `(date, open, high, low,
close, volume)` ordered by `(ticker, date)`.
- [x] 2.2 Call `_load_price_history(tickers, 252, run_date)` in `selector.run()`
      between steps 4 (vol_map) and 5 (filter_candidates).
- [x] 2.3 Pass the resulting `price_history: Dict[str, List[Dict]]` into
      `filter_candidates()` as a new keyword argument.

### 3. Filter Implementation

- [x] 3.1 Add private helper `_compute_sma(closes, period)` in `filter.py`.
- [x] 3.2 Add private helper `_compute_rsi(closes, period)` using Wilder
      smoothing.
- [x] 3.3 Implement `_technical_filter(ticker, bars, cfg)` in `filter.py`
      returning `(passed: bool, reason: str)`. Filters execute in order (short-circuit on first failure):
  1. No price history guard
  2. Trend Direction (SMA-20 > SMA-60, price > SMA-20, slope ≥ 0)
  3. Price Momentum Quality (20-day and 60-day return thresholds)
  4. Volume Confirmation (10-day / 30-day ratio ≥ 0.80)
  5. RSI Health Check (RSI-14 in [35, 75])
  6. Higher High / Higher Low Structure (current vs prior 20-day window)
  7. No Recent Sharp Drop (no single day < −5 % in last 10 days)
  8. Distance From 52-Week High (price ≥ 70 % of 252-day max)
  9. Positive Days Ratio (≥ 45 % of past 20 days up or flat)
- [x] 3.4 Updated `filter_candidates()` signature to accept `price_history`
      (optional, backward-compatible) and call `_technical_filter`.
- [x] 3.5 Extended audit trail entries with `technical_reason` field.

### 4. Tests

- [x] 4.1 Added `_make_bars(n, trend, base)` factory helper with realistic
      zigzag pattern (RSI ~60 for "up", ~30-35 for "down").
- [x] 4.2 For each of the eight new filters: pass, fail, boundary, and
      insufficient-history scenarios (47 tests total).
- [x] 4.3 Scenario confirming ML-centric rule short-circuits before technical.
- [x] 4.4 Integration scenario with 5 stocks (2 fail ML, 2 fail technical, 1 passes).

### 5. Validation

- [x] 5.1 `openspec validate add-strict-candidate-filters --strict` — PASSED
- [x] 5.2 `pytest tests/test_portfolio_filter.py` — 47 passed, 0 failed
- [ ] 5.3 Live portfolio scan with real data (manual, requires DB connection)
