# Design: Strict Candidate Filter Layer

## Context

Vietnamese equities exhibit margin-call cascades, thin liquidity, and forced
selling pressure that make oversold stocks continue to fall — behaviors the ML
model cannot reliably predict. The existing filters address model quality (do
we have good predictions?) but not market structure (is the stock worth trading
regardless of what the model says?).

The eight filters below operate purely on OHLCV price history and are
intentionally strict. When the market is broadly weak, most candidates should
be eliminated. That is the intended, correct behavior.

## Data Requirements

All eight filters derive from daily OHLCV bars. The selector already fetches a
`vol_map` (90-day avg volume). We extend this to fetch a `price_history` map:

```
{ticker: [{date, open, high, low, close, volume}, ...]}
```

Lookback required: 252 trading days (≈ 1 year) to support the 52-week high
filter. For the remaining seven filters a 60-day window is sufficient; loading
252 days gives us both without a second DB call.

## Filters (ordered by cheapest-to-compute first)

| #   | Name                     | Key Computation                                                  | Threshold           |
| --- | ------------------------ | ---------------------------------------------------------------- | ------------------- |
| 1   | Trend Direction          | SMA-20 > SMA-60; price > SMA-20; SMA-20 slope ≥ 0 over last 10 d | Must pass all three |
| 2   | Price Momentum Quality   | Realized return -20 d > −5 %; realized return -60 d > −10 %      | Both conditions     |
| 3   | Volume Confirmation      | avg_vol_10d / avg_vol_30d ≥ 0.80                                 | Ratio ≥ 0.80        |
| 4   | RSI Health Check         | RSI-14 within [35, 75]                                           | Inclusive           |
| 5   | Higher High / Higher Low | Current 20-d window vs prior 20-d window has ≥ 1 HH and ≥ 1 HL   | Both                |
| 6   | No Recent Sharp Drop     | No single-day drop > 5 % in past 10 d                            | Reject if any       |
| 7   | Distance From 52-Wk High | current_price / max(close[-252:]) ≥ 0.70                         | Ratio ≥ 0.70        |
| 8   | Positive Days Ratio      | Fraction of up/flat closes in past 20 d ≥ 0.45                   | Ratio ≥ 0.45        |

## Execution Order

Filters run in the order listed above, short-circuiting on first failure. This
minimises the computation cost for stocks that fail early filters.

## Implementation Approach

1. Add a `_load_price_history(tickers, lookback_days, date)` helper to
   `selector.py` (mirrors the existing `get_volume_map` pattern).
2. Pass `price_history` into `filter_candidates()` as an additional argument.
3. Add a private `_technical_filter(ticker, bars)` function inside `filter.py`
   that returns `(passed: bool, reason: str)`.
4. Call `_technical_filter` after the existing prediction/floor-prob/volume/
   confidence checks (so stocks without predictions are rejected first without
   wasting compute on technical calculations).
5. All thresholds sourced from `PORTFOLIO_CONFIG` — zero hard-coded magic numbers.

## Goals / Non-Goals

- **Goals**: Eliminate structurally weak stocks; make cash holding disciplined
  (if < `portfolio_size` candidates survive, do not force-fill the portfolio).
- **Non-Goals**: Predict direction; replace ML scoring; backtest filter
  performance; change optimizer or report layers.

## Risks / Trade-offs

- **Overfitting to current conditions**: Thresholds (SMA windows, RSI bands)
  are calibrated for normal Vietnamese market conditions. During prolonged
  bull markets fewer stocks are eliminated; during deep corrections nearly all
  are — both are correct behaviors.
- **Data availability**: Filters require 252 bars. Stocks listed < 1 year ago
  will have insufficient history; they are treated as failing the 52-week high
  filter and eliminated. This is conservative but appropriate.
- **Compute**: Loading 252-bar history for 50 tickers is cheap (< 1 s on a
  typical Postgres instance).

## Open Questions

None — thresholds are fully specified by the user.
