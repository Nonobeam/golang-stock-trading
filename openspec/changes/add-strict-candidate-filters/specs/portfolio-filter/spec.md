## ADDED Requirements

### Requirement: Trend Direction Filter

The system SHALL reject any candidate stock that is not in an uptrend or neutral
trend as defined by all three of the following conditions holding simultaneously:

1. The 20-day simple moving average (SMA-20) of close prices is greater than the
   60-day simple moving average (SMA-60).
2. The current price (last close) is above the SMA-20.
3. The slope of the SMA-20 over the last 10 days is flat or positive (i.e., the
   SMA-20 computed from close[−10] to close[−1] is non-decreasing).

A stock that satisfies one or two but not all three conditions SHALL be eliminated.

If fewer than 60 bars of price history are available for a ticker, the ticker
SHALL be eliminated with reason "Insufficient history for trend filter".

#### Scenario: Stock passes all three trend conditions

- **WHEN** SMA-20 > SMA-60, price > SMA-20, and SMA-20 slope ≥ 0
- **THEN** the stock passes the trend filter and proceeds to the next check

#### Scenario: SMA-20 below SMA-60

- **WHEN** SMA-20 < SMA-60 regardless of current price
- **THEN** the stock is eliminated with reason containing "trend"

#### Scenario: Price below SMA-20 even though SMA-20 above SMA-60

- **WHEN** SMA-20 > SMA-60 but current price < SMA-20
- **THEN** the stock is eliminated with reason containing "trend"

#### Scenario: SMA-20 curling downward

- **WHEN** SMA-20 > SMA-60 and price > SMA-20 but SMA-20 slope over last 10 days < 0
- **THEN** the stock is eliminated with reason containing "trend"

---

### Requirement: Price Momentum Quality Filter

The system SHALL reject any candidate whose recent realized returns fall below
minimum thresholds:

1. Realized return over the past 20 trading days must be greater than −5 %.
   (`(close[-1] / close[-20]) − 1 > −0.05`)
2. Realized return over the past 60 trading days must be greater than −10 %.
   (`(close[-1] / close[-60]) − 1 > −0.10`)

Both conditions must hold.

If fewer than 60 bars are available for a ticker, the ticker SHALL be eliminated
with reason "Insufficient history for momentum filter".

#### Scenario: Returns within acceptable range

- **WHEN** 20-day return is −3 % and 60-day return is −7 %
- **THEN** the stock passes the momentum filter

#### Scenario: 20-day return worse than −5 %

- **WHEN** 20-day return is −6 %
- **THEN** the stock is eliminated with reason containing "momentum" or "20d return"

#### Scenario: 60-day return worse than −10 %

- **WHEN** 60-day return is −12 % even if 20-day return is +1 %
- **THEN** the stock is eliminated with reason containing "momentum" or "60d return"

---

### Requirement: Volume Confirmation Filter

The system SHALL reject any candidate where trading volume has quietly dried up,
defined as:

`avg_volume(past 10 days) / avg_volume(past 30 days) < 0.80`

If fewer than 30 bars of volume history are available, the ticker SHALL be
eliminated with reason "Insufficient history for volume filter".

#### Scenario: Volume stable or growing

- **WHEN** 10-day avg volume is 90 % of 30-day avg volume
- **THEN** the stock passes the volume confirmation filter

#### Scenario: Volume declining sharply

- **WHEN** 10-day avg volume is only 70 % of 30-day avg volume
- **THEN** the stock is eliminated with reason containing "volume"

#### Scenario: Volume exactly at threshold

- **WHEN** ratio equals exactly 0.80
- **THEN** the stock passes (boundary is inclusive on the pass side)

---

### Requirement: RSI Health Check Filter

The system SHALL reject any candidate whose 14-period RSI is outside the range
[35, 75] (both endpoints inclusive).

RSI-14 calculation uses standard Wilder smoothing with a minimum of 15 bars.

If fewer than 15 bars of close history are available, the ticker SHALL be
eliminated with reason "Insufficient history for RSI filter".

#### Scenario: RSI within healthy range

- **WHEN** RSI-14 is 55
- **THEN** the stock passes the RSI filter

#### Scenario: RSI oversold (below 35) — active selling pressure

- **WHEN** RSI-14 is 28
- **THEN** the stock is eliminated with reason containing "RSI" or "oversold"

#### Scenario: RSI overbought (above 75) — risk of pull-back

- **WHEN** RSI-14 is 80
- **THEN** the stock is eliminated with reason containing "RSI" or "overbought"

#### Scenario: RSI exactly at boundary values

- **WHEN** RSI-14 is 35 or 75
- **THEN** the stock passes (boundaries are inclusive)

---

### Requirement: Higher High / Higher Low Structure Filter

The system SHALL reject any candidate that has not shown buyer accumulation
confirmed by price structure over the past 40 trading days.

The 40-day window is split into two 20-day half-periods:

- **Current period**: close[−20:] (most recent 20 days)
- **Prior period**: close[−40:−20] (preceding 20 days)

The stock passes this filter if and only if BOTH:

1. `max(high[current period]) > max(high[prior period])` — at least one higher high
2. `min(low[current period]) > min(low[prior period])` — at least one higher low

If fewer than 40 bars of OHLCV history are available, the ticker SHALL be
eliminated with reason "Insufficient history for HH/HL filter".

#### Scenario: Clear uptrend with higher highs and lows

- **WHEN** the current 20-day high is above prior 20-day high AND current 20-day low is above prior 20-day low
- **THEN** the stock passes

#### Scenario: Higher high but lower low — not confirmed

- **WHEN** max high of current period exceeds prior period but min low of current period is below prior period
- **THEN** the stock is eliminated with reason containing "higher low" or "structure"

#### Scenario: Lower high (failed rally)

- **WHEN** max high of current period is below max high of prior period
- **THEN** the stock is eliminated with reason containing "higher high" or "structure"

---

### Requirement: No Recent Sharp Drop Filter

The system SHALL reject any candidate that experienced a single-day close-to-close
decline exceeding 5 % within the past 10 trading days.

`any((close[i] / close[i-1]) − 1 < −0.05 for i in last 10 days)`

If fewer than 11 bars of close history are available, the ticker SHALL be
eliminated with reason "Insufficient history for sharp-drop filter".

#### Scenario: No large single-day drop recently

- **WHEN** all daily returns over the past 10 days are within [−5 %, ...]
- **THEN** the stock passes

#### Scenario: One sharp drop within the window

- **WHEN** day t has a close-to-close return of −6 % within the past 10 days
- **THEN** the stock is eliminated with reason containing "sharp drop" and the percentage

#### Scenario: Sharp drop exactly at boundary

- **WHEN** a day has −5.0 % return (exactly at threshold)
- **THEN** the stock passes (strictly less than −5 % is the rejection criterion)

---

### Requirement: Distance From 52-Week High Filter

The system SHALL reject any candidate trading more than 30 % below its 52-week
high (approximately 252 trading days):

`current_price / max(close[−252:]) >= 0.70`

If fewer than 63 bars of close history are available (less than approximately
one quarter), the ticker SHALL be eliminated with reason "Insufficient history
for 52-week high filter".

If between 63 and 251 bars are available, the system SHALL use the available
history as the lookback period for the high calculation.

#### Scenario: Stock near annual high

- **WHEN** current price is 85 % of 52-week high
- **THEN** the stock passes

#### Scenario: Stock deeply discounted from annual high

- **WHEN** current price is 60 % of 52-week high
- **THEN** the stock is eliminated with reason containing "52-week" or "high"

#### Scenario: Boundary case — exactly 70 %

- **WHEN** current price is exactly 70 % of 52-week high
- **THEN** the stock passes (≥ 0.70 is the inclusive pass condition)

---

### Requirement: Positive Days Ratio Filter

The system SHALL reject any candidate where fewer than 45 % of the past 20
trading days closed positive or flat (i.e., close ≥ prior close).

`(count of days where close[i] >= close[i-1]) / 20 >= 0.45`

If fewer than 21 bars of close history are available, the ticker SHALL be
eliminated with reason "Insufficient history for positive-days filter".

#### Scenario: Majority of recent days positive

- **WHEN** 12 of the past 20 days closed up or flat (60 %)
- **THEN** the stock passes

#### Scenario: Persistent down closes — active distribution

- **WHEN** only 8 of the past 20 days closed up or flat (40 %)
- **THEN** the stock is eliminated with reason containing "positive days" or "distribution"

#### Scenario: Exactly at threshold

- **WHEN** exactly 9 of 20 days are positive-or-flat (45 %)
- **THEN** the stock passes (≥ 0.45 is inclusive)

---

### Requirement: Technical Filter Data Loading

The system SHALL load OHLCV price history (date, open, high, low, close, volume)
for all universe tickers with a lookback of 252 trading days before running
candidate filtering.

The data SHALL be loaded from the `daily_bars` table using the same database
connection used by existing data-loading helpers in `selector.py`.

Tickers with zero rows returned by the history query SHALL be treated as having
insufficient history and SHALL fail all technical filters with reason "No price
history available".

#### Scenario: History loaded successfully for all tickers

- **WHEN** 252 daily bars are available for a ticker
- **THEN** all eight technical filters can execute and the ticker is evaluated normally

#### Scenario: Ticker has no history

- **WHEN** the daily_bars query returns zero rows for a ticker
- **THEN** the ticker is eliminated with reason "No price history available"

---

### Requirement: Cash Discipline When Candidates Are Scarce

The filter layer SHALL NOT force a full portfolio when fewer candidates survive
than the target portfolio size.

When `len(candidates) < portfolio_size`, the system SHALL:

1. Log a warning with the exact count.
2. Pass the reduced candidate list to the optimizer unchanged.
3. The optimizer then selects from the smaller pool (which may result in fewer
   than `portfolio_size` stocks being held — holding cash for the remainder is
   the correct outcome).

The system SHALL NOT relax any filter threshold to artificially inflate the
candidate count.

#### Scenario: Only 3 candidates survive all filters

- **WHEN** 47 of 50 universe stocks fail one or more of the eight technical filters
- **THEN** exactly 3 candidates are passed to the optimizer and a warning is logged

#### Scenario: Zero candidates survive

- **WHEN** all stocks fail filters (e.g., broad market correction)
- **THEN** an empty candidates list is passed to the optimizer and a warning is logged
