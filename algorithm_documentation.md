# Algorithm and Approach Documentation

## Introduction

This document details the algorithms, methodologies, and system architecture used in the Nonobeam Stock Trading project. It is a living document that is updated as we scan different parts of the codebase.

## Table of Contents

1. [ML Service Algorithms](#ml-service-algorithms)
2. [Golang Trading Logic](#golang-trading-logic)
3. [System Architecture](#system-architecture)

---

## ML Service Algorithms

### 1. Overview & Architecture

The ML Service is a Python-based gRPC application (`server/grpc_server.py`) that provides:

1.  **Price Predictions:** Future return distributions (p10, p50, p90).
2.  **Risk Assessment:** Floor-hit probabilities and uncertainty quantification.
3.  **Signal Generation:** Converting predictions into actionable trading signals.

**Execution Model:**

- **Trigger:** The service listens on port `50051`. Training is triggered via the `TriggerTraining` RPC, likely utilized by the Telegram Bot (`internal/service/telegram/bot_service.go`) or a scheduler.
- **Storage:**
  - **Models:** Trained XGBoost models are saved as JSON files in `models/saved/`.
  - **Metadata:** Training runs, metrics, and paths are stored in the `model_metadata` PostgreSQL table.
  - **Predictions:** Stored in `stock-trading.signals` table.

### 2. Feature Engineering

Located in `features/calculator.py`, the system generates over 40 technical indicators from OHLC data:

- **Trend:** SMA/EMA (5, 10, 20, 50, 200), MACD, Bollinger Bands (`bb_width` for volatility).
- **Momentum:** RSI (14, 28), Returns (1d, 5d, 20d, 60d).
- **Volume:** Volume Ratios (vs 5d/20d avg), Volume Trend, OBV.
- **Volatility:** Rolling volatility, ATR, Coefficient of Variation.
- **Market Context:** Distance from support (SMA20), Relative Strength vs Market, **Market Regime (FTD/Uptrend/Downtrend status)**.

### 3. Predictive Models

The core engine uses **XGBoost** for two distinct tasks:

#### A. Quantile Regression (Price Prediction)

- **Goal:** Predict the 10th (p10), 50th (p50), and 90th (p90) percentile of future returns.
- **Horizons:** 1-day, 5-day, and 10-day.
- **Algorithm:** XGBoost with `objective='reg:quantileerror'`.
- **Key Logic (`models/trainer.py`):**
  - Trains separate models for each quantile and horizon.
  - Validates using **MAE** and **Calibration Coverage** (e.g., p10 should cover ~10% of data).
  - Enforces quantile ordering (p10 < p50 < p90) during validation.

#### B. Floor-Hit Classifier (Risk)

- **Goal:** Predict probability of a stock hitting the "floor" (circuit breaker limit: -7% HOSE, -10% HNX).
- **Algorithm:** XGBoost Binary Classifier (`binary:logistic`).
- **Features:** Specialized risk features like `volume_surge`, `consecutive_down`, `distance_from_support`.
- **Thresholds:** `>20%` probability triggers a "CRITICAL" warning (Signal Rejection).

### 4. Signal Generation & Value Creation

Raw predictions are transformed into trading decisions in `signals/enhanced_generator.py`. This is the "Brain" that ensures value and safety.

**The Signal Pipeline:**

1.  **Validation Gates (Rejection Criteria):**
    - **Stop Loss:** Checks if current price < stored stop loss.
    - **Floor Risk:** Rejects if Floor Prob > 20% (Circuit Breaker risk).
    - **Confidence:** Rejects if avg model confidence < 60%.
    - **Uncertainty:** Rejects if "Epistemic Uncertainty" (model disagreement) > **1%**.
      - **Formula:** `σ_epistemic = std([pred_1, ..., pred_10])` across bootstrap ensemble
      - `σ_aleatoric = mean(p90 - p10)` for market noise
      - `σ_total = sqrt(σ_epistemic² + σ_aleatoric²)`
      - `Confidence = 1 / (σ_total + ε)`
    - **Liquidity:** Rejects if estimated position > 1% of 20d Avg Volume.
    - **Settlement Risk:** Checks T+2 constraints (e.g., reduces size on Thu/Fri).

2.  **Profitability Check:**
    - **Fee Structure:** Brokerage 0.15% (buy & sell) + Tax 0.10% (sell only) = **0.4% total round-trip**.
    - **Minimum Thresholds (Net of Fees):**
      - 1-Day: **1.0%**
      - 5-Day: **1.5%**
      - 10-Day: **2.0%**
    - **Liquidity Cap:** Position size ≤ **1% of 20-day average volume**.
    - **Liquidity Tiers** (score 1-10):
      - Tier 10: >5M shares/day
      - Tier 7: 1M-5M
      - Tier 5: 500K-1M
      - Tier 3: 250K-500K
      - Tier 1: 100K-250K
      - **<100K = Untradeable** (rejected).

3.  **Signal Output:**
    - **BUY_NEW / BUY_MORE:** Strong `p50` + Low `p10` risk + All Validations Passed.
    - **SELL:**
      - **Target Reached:** Profit taking at configured levels.
      - **Negative Outlook:** `p50` < -1%.
      - **Stop Loss:** Price variation.
    - **HOLD:** Neutral outlook or validation failure (safe state).

**Value Realization:**
The system only generates a BUY signal when the **probability of profit** (after fees) is high and **tail risk** (floor hit/crash) is low.

## Golang Trading Logic

The Golang service acts as the orchestration layer, managing user interactions, risk controls, and real-time monitoring.

### 1. Active Components

#### A. Risk Management (`internal/risk`)

The system enforces strict risk controls on every trade:

- **Position Sizing (`position_sizing.go`):**
  - **FTD / Market Regime Risk — Automated Scanner (`live_scanner.go`):**
    - **FTD Confirmed:** **2.0%** risk per trade (Aggressive)
    - **Rally Attempt (In Progress):** **1.5%** risk per trade (Moderate)
    - **Downtrend/Defensive:** **0.5%** risk per trade (Conservative)
    - _Note: This dynamic risk is used by `CalculateSimple()` and overrides score-based risk in the scanner._

  - **Base Risk by Score — `GetRiskPercentByScore()` (Full Sizing Calculator):**
    - Score ≥11: **2.0%** | Score 9-10: **1.5%** | Score 7-8: **1.0%** | Score <7: **0%** (skip trade)

  - **Score Position Size Multiplier — `GetScoreMultiplier()`:**
    - Score ≤6: **0.5x** | Score 7-8: **1.0x** | Score 9-10: **1.25x** | Score ≥11: **1.5x**

- **Market Regime Limits (`internal/risk/dynamic_limits.go`):**
  - **Aggressive (FTD/Bull):** Max Rank 8%, Max Pos 8, Daily Loss 2.5%, Max Size 25%.
  - **Default:** Max Risk 6%, Max Pos 8, Daily Loss 2.0%, Max Size 20%.

- **Volatility Multiplier (ATR %):**
  - <3%: **1.2x** (low)
  - 3-5%: **1.0x** (normal)
  - 5-8%: **0.8x** (high)
  - > 8%: **0.6x** (extreme)
- **Correction Multiplier:**
  - ρ ≥0.85: **0.0x** (reject)
  - ρ 0.7-0.85: **0.5x**
  - ρ 0.5-0.7: **0.8x**
  - ρ <0.5: **1.0x**
- **Gap Risk (Vietnam):** All sizes ÷ **3.25** for 3-day floor buffer.
- **Stop-Loss Methods (`stop_loss.go`):**
  - ATR-Based, Percentage, Technical Level, Support/MA, Swing Low, Floor-Aware (multi-day).
  - **Pre-emptive Alerts:** 50% and 70% distance to stop.
- **Locked Risk (`locked_risk.go`):**
  - **T+2 Settlement:** `LockedRisk = Shares × Price × ExchangeMultiplier`
  - **Exchange Multipliers:** HOSE = **20%** | HNX = **30%** | UPCOM = **40%** (worst-case floor exposure)
  - **Budget:** Configurable threshold per user (validated against total locked risk before each new purchase).
  - **Entry Day:** Thursday/Friday entries reduced by **50%** risk budget (weekend lock-up).
- **Portfolio Limits (`portfolio_manager.go`):**
  - **Max Positions:** 8 (6 for Vietnam Conservative)
  - **Max Position Size:** 20% of capital
  - **Max Total Risk:** 6% (4% Vietnam)
  - **Max Risk/Trade:** 2%
  - **Loss Limits:** Daily 2%, Weekly 5%, Monthly 10%
  - **Max Drawdown:** 20%
  - **Sector Exposure:** Max 40% per sector, 3 positions max
  - **Correlation:** Max pairwise 0.85, avg portfolio 0.60

#### B. Price Monitor (`internal/service/monitor`)

A real-time service that watches **watchlist symbols** for price alerts:

- **Trigger:** Runs every 1 minute during market hours.
- **Alert Cooldown:** 30 minutes per symbol per alert type.
- **Logic:**
  - Fetches real-time price from DNSE WebSocket via `MarketDataService`.
  - Sends Telegram alert on **Price Drop (-3% from reference)** or **Gain (+5% from reference)**.
  - Broadcasts **Market Open / Market Closed / Weekend** messages on session transitions.
- **Note:** Does NOT call the Exit Engine. Exit evaluation is decoupled from the monitor loop.

#### C. Exit Engine (`internal/position/exit_engine.go`)

A prioritized logic engine that determines when to sell (partially or fully):

1.  **Emergency Exit (Priority 1):**
    - **Trigger:** Floor-hit probability > **30%** OR **3+ consecutive days** hitting floor.
    - **Action:** **Sell 100%**.

2.  **Target 1 (Priority 2):**
    - **Trigger:** Profit ≥ **15%** OR Price ≥ Target 1 (Resistance).
    - **Action:** **Sell 30%** (Lock in initial profit).

3.  **Target 2 (Priority 3):**
    - **Trigger:** Profit ≥ **25%** AND Price ≥ Target 2 AND Target 1 already filled.
    - **Action:** **Sell 30%** (Scale out).

4.  **Trailing Stop / Target 3 (Priority 4):**
    - **Trigger:** Trailing stop hit (after T1/T2 filled). _(Integration with `targets_trailing.go` pending.)_
    - **Action:** **Sell 40%** (remainder of position after T1/T2 exits).

#### D. Telegram Bot (`internal/service/telegram`)

The primary user interface for the system:

- **Commands:**
  - `/train <SYMBOL>`: Triggers ML training loop.
  - `/predict <SYMBOL>`: Fetches ML predictions via gRPC.
  - `/risk` & `/limits`: Displays current risk exposure.
- **Notifications:** Specific alerts for price movements and trade confirmations.

### 2. Live Scanner (Now Active)

**Important Update:** The `LiveScanner` service (`internal/service/scanner/live_scanner.go`) has been **activated** in the main application.

- **Function:** Continuously scans watchlist symbols using 4 strategies.
- **Strategies & Parameters:**
  - **Pullback:** Rally 5-20%, pullback 2-10 days, within 3% of EMA20, ADX ≥25, RSI 40-60.
  - **Breakout:** Consolidation 20-60 days, range 8-25%, 2-3 resistance tests, volume ≥90th percentile.
  - **Crossover:** EMA20 crosses EMA50, pullback 2-7 days, within 3% of EMA20, ADX ≥20, min 2 triggers.
  - **Mean Reversion:** ADX <20 (ranging), RSI <30 (oversold) or >70 (overbought), within 3% of S/R, min **3** range tests.
- **Scoring System:**
  - **Confidence → Score:** Very High=10, High=9, Moderate=7, Low=6
  - **Priority:** Pullback > Breakout > Crossover > Mean Reversion
  - **Trade Threshold:** Score ≥7; Alert Threshold: Score ≥9
- **Execution:** Currently **Alert Only**.

## System Architecture

### 1. High-Level Flow

1.  **Data Ingestion:**
    - Real-time: WebSocket connection to DNSE (Go Service).
    - Historical: Manual import via Telegram (Excel files) -> Python Service.
2.  **Analysis:**
    - **Python ML Service:** Generates return predictions (p10/p50/p90) and risk assessments.
    - **Go Service:** Monitors price thresholds and manages portfolio risk.
3.  **Execution & Notification:**
    - **Telegram Bot:** Acts as the central command center for alerts and manual execution.
    - **Database:** PostgreSQL stores all state (positions, signals, model metadata).

### 2. Key Integration Points

- **gRPC:** The Go service calls the Python ML service for `Predict` and `TriggerTraining`.
- **Shared Database:** Both services access the same PostgreSQL instance, though they manage different tables (Python: `model_metadata`, Go: `positions`, `users`, `market_regime`).
- **T+2 Settlement:** Both layers incorporate Vietnam's T+2 settlement cycle (ML in predictions, Go in risk management).
- **Purchase Tracking:** Go service tracks individual entry transactions (`position_entries`) to support accurate cost-basis and settlement tracking for partial fills.
