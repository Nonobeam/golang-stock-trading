# Vietnamese Stock Prediction System

<div align="center">

[![Python](https://img.shields.io/badge/Python-3.10+-3776AB?logo=python&logoColor=white)](https://www.python.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![XGBoost](https://img.shields.io/badge/XGBoost-2.0.3-FF6600)](https://xgboost.readthedocs.io/)
[![License: MIT](https://img.shields.io/badge/license-MIT-green)](LICENSE)

### **Predict Returns. Quantify Risk. Trade with Confidence.**

XGBoost quantile regression system for Vietnamese stock markets with multi-horizon forecasting, circuit breaker protection, and full transaction cost accounting.

[Quick Start](#quick-start) • [Architecture](#architecture) • [Capabilities](#system-capabilities) • [Algorithms](#algorithms--equations) • [Roadmap](#roadmap)

</div>

---

## What This System Does

The Vietnamese Stock Prediction System is a production-ready ML trading framework built specifically for the HOSE and HNX markets. It predicts stock returns across three time horizons with uncertainty bounds, manages positions with circuit breaker awareness, and enforces fee-adjusted profitability thresholds so every signal has a real edge after costs.

| Feature | Detail |
|---------|--------|
| **Prediction** | 1-day, 5-day, 10-day return forecasts with p10/p50/p90 quantiles |
| **Risk Management** | Floor-hit probability, drawdown protection, stop loss monitoring |
| **Market-Specific** | HOSE -7% / HNX -10% circuit breaker awareness built in |
| **Fee-Adjusted** | All metrics net of 0.4% round-trip transaction costs |
| **Position-Aware** | 6 signal types accounting for your current holdings |
| **Validation** | Walk-forward testing across multiple stocks and time periods |

---

## Quick Start

### Prerequisites

- Python 3.10+
- PostgreSQL with `stock-trading` schema
- 500+ days of historical data per ticker

### Installation

```bash
cd ml-service
python -m venv venv
source venv/bin/activate      # Windows: venv\Scripts\activate
pip install -r requirements.txt
```

### Configuration

Create a `.env` file:

```env
DB_HOST=localhost
DB_PORT=5432
DB_NAME=stock-trading
DB_USER=postgres
DB_PASSWORD=your_password
```

### Train Your First Model

```bash
python train.py --ticker HPG
```

### Run Daily Signals

```bash
python daily_signals.py
```

This loads your active positions, fetches the latest predictions, checks floor risk, enforces fee thresholds, and outputs a full recommendation report.

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

### Data Flow

```
┌─────────────────────────────────────────────────────────┐
│                  Trading System                         │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────┐         ┌──────────────┐             │
│  │ Go Service   │────────▶│  PostgreSQL  │             │
│  │ (Data        │         │  daily_bars  │             │
│  │  Collection) │         └──────┬───────┘             │
│  └──────────────┘                │                      │
│                                  ▼                      │
│                         ┌──────────────┐               │
│                         │   Feature    │               │
│                         │ Calculator   │               │
│                         └──────┬───────┘               │
│                                │                        │
│                                ▼                        │
│                         ┌──────────────┐               │
│                         │   XGBoost    │               │
│                         │   Trainer    │               │
│                         │ (9 models)   │               │
│                         └──────┬───────┘               │
│                                │                        │
│              ┌─────────────────┼──────────────────┐    │
│              ▼                 ▼                  ▼    │
│       ┌────────────┐  ┌──────────────┐  ┌──────────┐  │
│       │   Saved    │  │  model_      │  │ predic-  │  │
│       │  Models    │  │  metadata    │  │  tions   │  │
│       └────────────┘  └──────────────┘  └──────────┘  │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### Project Structure

```
ml-service/
├── config.py              # Hyperparameters, DB settings, thresholds
├── train.py               # Main training script
├── daily_signals.py       # Daily workflow and signal generation
├── db/
│   ├── connection.py      # PostgreSQL connection pooling
│   └── queries.py         # Parameterized SQL queries
├── data/
│   └── loader.py          # Load features and targets from database
├── features/
│   └── calculator.py      # 27 technical indicator calculations
├── models/
│   ├── trainer.py         # XGBoost quantile regression training
│   └── saved/             # Versioned model files per ticker
└── tests/                 # 25-test suite
```

### Database Schema

| Table | Contents |
|-------|----------|
| `daily_bars` | Raw OHLCV data: symbol, date, open, high, low, close, volume, turnover |
| `features` | 27 technical indicators per ticker per date, with `features_complete` flag |
| `predictions` | p10/p50/p90 outputs per ticker, prediction date, target date, actual return |
| `model_metadata` | Model registry: file path, quantile, `in_production` flag, training date |
| `positions` | Your holdings: entry price, quantity, stop loss, profit targets |

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

Initial training uses full history. Models retrain weekly on a rolling 252-day window, automatically versioned and registered in `model_metadata`.

---

## Usage

### Train a Stock

```bash
python train.py --ticker VNM
```

### Train Multiple Stocks

```bash
for ticker in HPG VNM FPT VIC MSN; do
    python train.py --ticker $ticker
done
```

### Check Active Model Performance

```sql
SELECT ticker, quantile,
       metrics->>'test_mae'  AS mae,
       metrics->>'coverage'  AS coverage,
       training_date
FROM model_metadata
WHERE in_production = TRUE
ORDER BY ticker, quantile;
```

### Run Tests

```bash
pytest tests/ -v           # 25 tests covering DB, data, training, integration
pytest tests/ --cov=.      # with coverage report
```

---

## What the System Cannot Do

- Predict black swan events, regulatory changes, or company-specific news outside historical patterns
- Guarantee profits — all predictions are probabilistic
- Execute trades automatically — you still place orders manually through your broker
- Trade stocks below 100,000 shares daily volume
- Work intraday — designed for end-of-day bar data
- Detect fundamental risks like bankruptcy or delisting from technical indicators alone

---

## Roadmap

- [ ] REST API for real-time predictions (FastAPI)
- [ ] LSTM ensemble model for sequential pattern capture
- [ ] Multi-stock portfolio optimization
- [ ] Automated hyperparameter tuning (Bayesian)
- [ ] Regime detection (bull / bear / volatile) with regime-specific models
- [ ] Live monitoring dashboard
- [ ] Mobile alert integration for daily signals

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
- Technical Analysis: Murphy, *Technical Analysis of the Financial Markets* (1999)

---

## License

MIT License — see [LICENSE](LICENSE) for details.

---

<div align="center">

### **Predict Returns. Quantify Risk. Trade with Confidence.**

</div>