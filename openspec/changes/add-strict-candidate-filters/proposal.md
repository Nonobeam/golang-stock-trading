# Change: Add Strict Candidate Filters for Vietnamese Market

## Why

The current `portfolio/filter.py` runs five ML-centric checks (predictions exist,
floor-hit probability, average volume, expected return, model confidence). These
checks ignore market structure entirely: a stock in a six-month downtrend with
declining volume and persistently negative daily closes can pass every existing
filter while being genuinely untradeable.

This change adds eight technical/market-structure filters that run **before** ML
scoring. The goal is to ensure every candidate presented to the optimizer is
technically sound — not just statistically interesting to the model.

## What Changes

- **`portfolio/filter.py`** — extend `filter_candidates()` with eight new hard
  rules that require OHLCV price history (close, high, low, volume).
- **`portfolio/selector.py`** — load/pass a `price_history` map so `filter.py`
  can compute technical indicators inline.
- **`config.py`** — add configurable thresholds for all eight filters under
  `PORTFOLIO_CONFIG`.
- **`tests/test_portfolio_filter.py`** — cover all eight new filters with
  pass/fail boundary scenarios.

## Impact

- Affected specs: `portfolio-filter` (new capability)
- Affected code: `ml-service/portfolio/filter.py`, `ml-service/portfolio/selector.py`,
  `ml-service/config.py`, `ml-service/tests/test_portfolio_filter.py`
- Expected outcome: 60–70 % of the 50-stock universe eliminated before ML scoring
  in normal market conditions; near-zero candidates in broad corrections (correct
  behavior — hold cash).
- No changes to the optimizer, scorer, or report layers.
- No database schema changes.
