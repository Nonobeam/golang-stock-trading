# Vietnamese Stock Prediction System

<div align="center">

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Python](https://img.shields.io/badge/Python-3.10+-3776AB?logo=python&logoColor=white)](https://www.python.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![XGBoost](https://img.shields.io/badge/XGBoost-2.0.3-FF6600)](https://xgboost.readthedocs.io/)
[![License: MIT](https://img.shields.io/badge/license-MIT-green)](LICENSE)

### **Predict Returns. Quantify Risk. Trade with Confidence.**

Full-stack algorithmic trading system for Vietnamese stock markets. A Go service delivers real-time WebSocket signal scanning across a 50-stock curated universe, while a Python ML service provides multi-horizon XGBoost forecasting, weekly portfolio selection, and automated retraining.

[Quick Start](#quick-start) • [Architecture](#architecture) • [Capabilities](#system-capabilities) • [Algorithms](#algorithms--equations) • [Roadmap](#roadmap)

</div>

---

## What This System Does

A production-ready algorithmic trading framework for the HOSE and HNX markets. The Go service streams real-time 1-minute bars via WebSocket for all 50 stocks in the curated `stock_universe`, detects trading signals, and fires Telegram alerts. The Python ML service trains XGBoost quantile models weekly and runs portfolio selection every Monday morning.

| Layer      | Feature             | Detail                                                       |
| ---------- | ------------------- | ------------------------------------------------------------ |
| **Go**     | Real-Time Scanning  | 1-min OHLC WebSocket stream for all 50 universe stocks       |
| **Go**     | Signal Detection    | Entry/exit signals with position sizing and stop-loss        |
| **Go**     | Telegram Bot        | `/scan`, `/train`, `/watchlist`, `/portfolio` commands       |
| **Python** | Prediction          | 1-day, 5-day, 10-day return forecasts (p10/p50/p90)          |
| **Python** | Risk Management     | Floor-hit probability, drawdown protection, circuit breakers |
| **Python** | Portfolio Selection | Weekly top-5 stock selection scored by ML predictions        |
| **Python** | Scheduler           | Auto-retrain Sat/Sun 22:00 → portfolio run Mon 07:30 ICT     |
| **Both**   | Fee-Adjusted        | All metrics net of 0.4% round-trip transaction costs         |

---

## Quick Start

### Prerequisites

- Go 1.22+
- Python 3.10+
- PostgreSQL 15+ with `stock-trading` schema
- 500+ days of historical data per ticker
- DNSE account for real-time WebSocket data

### Go Service

```bash
# Build and run the main trading service
go build -o app.exe ./cmd/app
./app.exe
```

Configured via `.env`:

```env
DB_HOST=localhost
DB_PORT=5432
DB_NAME=stock-trading
DB_USER=postgres
DB_PASSWORD=your_password
TELEGRAM_BOT_TOKEN=your_bot_token
```

### Python ML Service

```bash
cd ml-service
python -m venv venv
venv\Scripts\activate      # Windows
pip install -r requirements.txt
```

Configure via `ml-service/.env.ml`:

```env
DB_HOST=localhost
DB_PORT=5432
DB_NAME=stock-trading
DB_USER=postgres
DB_PASSWORD=your_password
```

### Train Models

```bash
# Train a single ticker
python -m daily.run_daily_features      # generate/update features
python -m daily.run_daily_predictions   # generate predictions
```

### Start the Automated Scheduler

```bash
# Long-running process — retrain Sat/Sun 22:00, portfolio Mon 07:30 ICT
python ml-service/scripts/scheduler.py

# Manual immediate trigger (full retrain + portfolio selection)
python ml-service/scripts/scheduler.py run-now
```

---

## System Capabilities

<table>
<tr>
<td width="50%" valign="top">

### Prediction Engine

**Multi-Horizon Forecasting**

- 1-day, 5-day, and 10-day return predictions
- Three quantile models per horizon (p10, p50, p90)
- 9 total XGBoost models per stock

**Uncertainty Quantification**

- Aleatoric uncertainty: inherent market randomness
- Epistemic uncertainty: model disagreement across bootstraps
- Confidence scoring combining precision and calibration

**Calibration Monitoring**

- Verifies p10 actually covers 10% of outcomes
- Verifies p90 actually covers 90% of outcomes
- Triggers automatic recalibration when drift exceeds 3%

</td>
<td width="50%" valign="top">

### Vietnamese Market Protections

**Circuit Breaker Defense**

- Estimates floor-hit probability using logistic regression
- Blocks all BUY signals when P(floor) > 20%
- Triggers emergency SELL when P(floor) > 40%
- Exit before trap: fires at P(floor) > 20% and loss > -3%

**Fee-Adjusted Metrics**

- 0.15% brokerage each direction
- 0.1% selling tax on exit
- 0.4% total round-trip cost
- Minimum return thresholds: 1.0% (1d), 1.5% (5d), 2.0% (10d)

**Liquidity Enforcement**

- Excludes stocks below 100,000 shares daily volume
- Caps positions at 1% of average daily volume
- Splits large orders to minimize market impact

</td>
</tr>
<tr>
<td width="50%" valign="top">

### Portfolio & Risk Management

**Position-Aware Signals**

- BUY_NEW: Initiating a new position
- BUY_MORE: Adding to an existing position
- SELL: Closing entirely
- SELL_PARTIAL: Taking partial profits at target levels
- HOLD: Keeping current position
- HOLD_NONE: Staying in cash

**Drawdown Protection**

- Halves all position sizes at -10% portfolio drawdown
- Halts all trading at -15% drawdown
- Tracks peak equity continuously

**Stop Loss & Target Tracking**

- Monitors your stop loss daily against current price
- Auto-generates SELL when stop is breached
- Tracks target_1, target_2, target_3 profit levels
- SELL_PARTIAL when price reaches any target

</td>
<td width="50%" valign="top">

### Walk-Forward Validation

**Robustness Testing**

- Rolls 252-day training windows across the full history
- Tests on held-out 20-day periods to simulate real trading
- Reports fee-adjusted profit factor as primary metric
- Cross-validates across 10 most liquid Vietnamese stocks

**Key Performance Metrics**

- MAE (Mean Absolute Error): target < 3%
- IC (Information Coefficient): target > 0.10
- Directional Accuracy: target > 55%
- Profit Factor (net of fees): target > 1.5
- Quantile Coverage: p10 and p90 within ±3% of expected

**Feature Stability Tracking**

- Monitors which indicators consistently predict
- Identifies regime shifts requiring model retraining
- Adaptive feature selection across market conditions

</td>
</tr>
</table>

---

## Architecture

### System Overview

```
  DNSE WebSocket (1-min OHLCV)
         │
         ▼
  ┌─────────────────────┐
  │  Go Service          │   LiveScanner subscribes to all 50
  │  LiveScanner         │   stock_universe tickers in real time.
  │  (Signal Detection)  │   Fires Telegram alerts on high-score signals.
  └──────────┬──────────┘
             │ writes
             ▼
  ┌─────────────────────────────────────────────┐
  │              PostgreSQL (stock-trading)       │
  │  daily_bars │ features │ predictions          │
  │  stock_universe │ weekly_portfolio_selection  │
  │  signal_history │ positions │ watchlist        │
  └───────────────────────┬─────────────────────┘
                          │ reads
                          ▼
  ┌─────────────────────────────────────────────┐
  │  Python ML Service                           │
  │                                             │
  │  Sat/Sun 22:00 → Feature gen + Retrain      │
  │  Mon 07:30    → Portfolio selection run     │
  └─────────────────────────────────────────────┘
```

### Automated Pipeline Schedule (ICT)

| Time  | Day          | Job                                                                 |
| ----- | ------------ | ------------------------------------------------------------------- |
| 22:00 | Sat & Sun    | Feature backfill + XGBoost retrain for all universe tickers         |
| 07:30 | Monday       | Weekly portfolio selection (top-5 from 50-stock universe)           |
| Live  | Market hours | Go `LiveScanner` streams real-time bars, detects and alerts signals |

### Project Structure

```
.
├── cmd/app/               # Go main service (scanner, bot, API)
├── cmd/exit-evaluator/    # Stop-loss / take-profit monitoring daemon
├── internal/
│   ├── service/scanner/   # LiveScanner — real-time WebSocket signal detection
│   ├── service/telegram/  # Telegram bot command handlers
│   ├── db/repository/     # Go DB repositories incl. StockUniverseRepository
│   └── signals/           # Signal scoring logic
└── ml-service/
    ├── daily/             # Feature generation & prediction runners
    ├── portfolio/         # Weekly portfolio selection (filter, selector, universe)
    ├── models/            # XGBoost trainer + floor-hit classifier
    ├── scripts/scheduler.py  # Automated weekly pipeline scheduler
    └── tests/             # Test suite
```

### Database Schema

| Table                        | Contents                                                              |
| ---------------------------- | --------------------------------------------------------------------- |
| `stock_universe`             | 50 curated Vietnamese stocks eligible for scanning (`is_active` flag) |
| `daily_bars`                 | Raw OHLCV data per ticker per date                                    |
| `features`                   | 27 technical indicators per ticker per date                           |
| `predictions`                | p10/p50/p90 ML outputs per ticker, horizon, and date                  |
| `model_metadata`             | Model registry: file path, quantile, `in_production` flag             |
| `floor_hit_probabilities`    | Classifier outputs — P(floor hit) per ticker                          |
| `weekly_portfolio_selection` | History of weekly top-5 recommendations with scores                   |
| `positions`                  | Holdings: entry price, quantity, stop loss, profit targets            |
| `signal_history`             | Log of all signals detected by the live scanner                       |
| `watchlist`                  | Per-user manual watchlist (separate from scanner universe)            |

### Model Versioning

```
models/saved/
└── {TICKER}/
    ├── 20250119_094523/
    │   ├── model_{TICKER}_p10.json
    │   ├── model_{TICKER}_p50.json
    │   └── model_{TICKER}_p90.json
    └── 20250126_143022/     ← Active (in_production = TRUE)
        ├── model_{TICKER}_p10.json
        ├── model_{TICKER}_p50.json
        └── model_{TICKER}_p90.json
```

One version active per ticker via `in_production` flag. Old versions retained for rollback.

---

## Algorithms & Equations

### Quantile Regression Loss

Three separate XGBoost models per horizon use asymmetric loss to bracket true outcomes:

```
L_α(y, ŷ) = Σ ρ_α(y - ŷ)
where ρ_α(u) = u × (α - I(u < 0))

α = 0.10 → penalizes over-prediction (lower bound)
α = 0.50 → balanced (median)
α = 0.90 → penalizes under-prediction (upper bound)
```

### Confidence Scoring

```
precision    = 1.0 - min(1.0, (p90 - p10) / 0.20)
calibration  = coverage / 0.80
confidence   = precision × calibration
```

### Position Sizing

Range-based fractional sizing with horizon multiplier:

```
f = f_base × m_confidence × m_horizon

f_base = 0.10 (10% base allocation)

m_confidence:
  1.5  if (p90 - p10) < 0.05    → tight range, high confidence
  1.0  if 0.05 ≤ (p90 - p10) ≤ 0.15
  0.5  if (p90 - p10) > 0.15    → wide range, low confidence

m_horizon:
  0.8  for 1-day
  1.0  for 5-day
  1.2  for 10-day

shares = floor(account_value × f / current_price)
final_shares = min(shares, average_daily_volume × 0.01)
```

Hard cap: position may not exceed 20% of account value.

### Vietnamese Transaction Costs

```
entry_fee      = purchase_value × 0.0015
exit_fee       = sale_value × 0.0025    (brokerage + selling tax)
round_trip     = 0.004 (0.4%)

net_return     = gross_return - 0.004
profit_factor  = Σ(net_winning_trades) / |Σ(net_losing_trades)|
```

### Floor-Hit Probability

```
P(floor_hit) = 1 / (1 + e^(-z))

z = β₀ + β₁×momentum + β₂×volume_surge
      + β₃×distance_from_support + β₄×consecutive_down_days

P > 0.20 → reject all BUY signals
P > 0.40 → generate immediate SELL
P > 0.20 AND loss > -3% → emergency exit
```

### Drawdown Control

```
drawdown = (current_equity - peak_equity) / peak_equity

drawdown < -0.10 → position_multiplier = 0.5
drawdown < -0.15 → position_multiplier = 0.0 (halt trading)
```

---

## Feature Engineering

27 technical indicators calculated from OHLCV data:

**Returns:** 1d, 5d, 20d, 60d

**Trend:** SMA (10, 20, 50), EMA (12, 26), MACD + signal + histogram

**Momentum:** RSI-14, Williams %R

**Volatility:** Bollinger Bands (width, position), ATR-14, rolling std dev (5d, 20d)

**Volume:** ratio to 20-day average, 5d vs 20d trend, OBV

**Market Context:** rolling beta, correlation to VN-Index

---

## Training & Validation

### Training Windows

```python
TRAINING_WINDOW   = 252   # 1 year of trading days
VALIDATION_WINDOW = 80    # ~3 months
TEST_WINDOW       = 20    # ~1 month
```

### Walk-Forward Protocol

```
for t in range(start_date, end_date, step=20_days):
    train  = data[t - 252 : t]
    test   = data[t : t + 20]
    model  = train_xgboost(train)
    preds  = model.predict(test)
    net    = actual_returns - 0.004
    metrics[t] = evaluate(preds, net)
```

### Retraining Schedule

Initial training uses full history. Models retrain weekly (Saturday and Sunday at 22:00 ICT via `scheduler.py`) on a rolling 252-day window, automatically versioned and registered in `model_metadata`. The floor-hit classifier is also refreshed during each retraining run.

---

## Usage

### Telegram Bot Commands

| Command              | Description                                             |
| -------------------- | ------------------------------------------------------- |
| `/scan`              | Trigger an immediate portfolio scan and display results |
| `/train all`         | Retrain ML models for all universe tickers              |
| `/watchlist`         | Show your personal watchlist                            |
| `/watchlist add HPG` | Add a stock to your watchlist                           |
| `/portfolio`         | Show the latest weekly portfolio selection              |

### Check Universe

```sql
SELECT ticker, sector, exchange
FROM "stock-trading".stock_universe
WHERE is_active = TRUE
ORDER BY sector, ticker;
```

### Check Active Model Performance

```sql
SELECT ticker, quantile,
       metrics->>'test_mae'  AS mae,
       metrics->>'coverage'  AS coverage,
       training_date
FROM "stock-trading".model_metadata
WHERE in_production = TRUE
ORDER BY ticker, quantile;
```

### View Latest Weekly Portfolio

```sql
SELECT ticker, rank, composite_score, is_selected
FROM "stock-trading".weekly_portfolio_selection
WHERE week_start = (
    SELECT MAX(week_start) FROM "stock-trading".weekly_portfolio_selection
)
ORDER BY rank;
```

### Run Python Tests

```bash
pytest ml-service/tests/ -v
pytest ml-service/tests/ --cov=.
```

### Run Go Tests

```bash
go test ./...
```

---

## What the System Cannot Do

- Predict black swan events, regulatory changes, or news outside historical patterns
- Guarantee profits — all predictions are probabilistic
- Execute trades automatically — orders are placed manually through your broker
- Trade stocks below 100,000 shares daily volume
- Detect fundamental risks like bankruptcy or delisting from technical indicators alone

---

## Roadmap

- [ ] REST API for external prediction queries (FastAPI)
- [ ] LSTM ensemble model for sequential pattern capture
- [ ] Automated hyperparameter tuning (Bayesian)
- [ ] FTD regime-specific model variants
- [ ] Live monitoring dashboard with portfolio analytics
- [x] Real-time WebSocket scanning over stock universe
- [x] Weekly portfolio selection pipeline
- [x] Telegram bot with `/scan`, `/train`, `/portfolio` commands
- [x] Automated Sat/Sun retrain + Mon portfolio scheduler

---

## Configuration Reference

```python
# config.py
HYPERPARAMETERS = {
    "max_depth": 5,
    "learning_rate": 0.05,
    "n_estimators": 200,
    "subsample": 0.8,
    "colsample_bytree": 0.8,
    "reg_lambda": 5,
}

MIN_RETURN_THRESHOLDS = {
    "1d": 0.010,
    "5d": 0.015,
    "10d": 0.020,
}

FLOOR_HIT_THRESHOLDS = {
    "reject_buy": 0.20,
    "force_sell": 0.40,
    "emergency_exit_loss": -0.03,
}
```

---

## References

- XGBoost: Chen & Guestrin (2016), https://arxiv.org/abs/1603.02754
- Quantile Regression: Koenker & Bassett (1978)
- Technical Analysis: Murphy, _Technical Analysis of the Financial Markets_ (1999)

---

## License

MIT License — see [LICENSE](LICENSE) for details.

---

<div align="center">

### **Predict Returns. Quantify Risk. Trade with Confidence.**

</div>
