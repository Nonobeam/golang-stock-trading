# Design: Enhance ML Prediction Validation

## Context

The current ML prediction system lacks production-grade validation and does not account for Vietnamese market microstructure. This creates risks:

- Models may be overfit to training data without robust out-of-sample testing
- Quantile predictions cannot be trusted without calibration verification
- Transaction costs (0.4% round-trip) make marginal predictions unprofitable
- Circuit breakers (±7% HOSE, ±10% HNX) can trap positions
- Thin liquidity causes execution slippage on large orders

This change introduces comprehensive validation infrastructure to ensure models are trustworthy before risking capital.

## Goals / Non-Goals

**Goals:**

- Implement walk-forward validation to prove model works across multiple time periods
- Verify quantile predictions are properly calibrated using empirical coverage
- Track feature stability to identify reliable vs noisy predictors
- Account for Vietnamese market frictions (fees, price limits, liquidity) in all validations
- Quantify prediction uncertainty separately for market randomness vs model limitations
- Provide regime-aware models for bull/bear/volatile market conditions
- Generate comprehensive validation reports for daily monitoring

**Non-Goals:**

- Real-time intraday validation (daily batch sufficient for swing trading)
- Automated retraining based on validation results (manual review required)
- Multi-exchange support beyond Vietnamese market
- Alternative ML frameworks beyond XGBoost
- Historical position reconstruction

## Architectural Decisions

### Decision 1: Validation as Separate Module, Not Integrated into Models

**What:** Create `ml-service/validation/` as standalone module distinct from `models/` and `features/`

**Why:**

- Validation is a cross-cutting concern used by training, inference, and monitoring
- Allows testing validation logic independently of model implementations
- Follows existing ML service pattern (separate `features/`, `models/`, `signals/` modules)
- Easier to add new validation techniques without modifying model code

**Alternatives Considered:**

- Embed validation in `models/trainer.py` → Rejected: violates single responsibility
- Create validation as part of `monitoring/` → Rejected: monitoring is for production alerts, validation is for model quality assessment

---

### Decision 2: Walk-Forward Validation with Expanding Window (Not Rolling)

**What:** Use expanding window where training set grows over time, not rolling window with fixed size

**Formula:**

```
Period i:
  Train on [start_date, start_date + 252 days + i*20 days]
  Test on [train_end + 1, train_end + 20 days]
```

**Why:**

- More data for later periods improves model quality (realistic production scenario)
- Vietnamese market evolving rapidly - need full history to capture regime shifts
- Rolling window discards old data which may contain valuable crisis patterns
- Matches how models will be retrained in production (always use all available data)

**Alternatives Considered:**

- Rolling window (fixed 252 days) → Rejected: waste historical data, especially for young Vietnamese market with limited history
- Anchored forward testing (no retraining) → Rejected: unrealistic, production models will retrain monthly

**Implementation:**

```python
for i in range(num_periods):
    train_end = start_date + timedelta(days=252 + i*20)
    test_end = train_end + timedelta(days=20)

    train_data = data[data.date <= train_end]
    test_data = data[(data.date > train_end) & (data.date <= test_end)]

    model.fit(train_data)
    predictions = model.predict(test_data)
    metrics = calculate_metrics(predictions, test_data.actuals)
```

---

### Decision 3: Empirical Coverage Formula for Calibration

**What:** Use indicator function to count proportion of actuals falling within prediction intervals

**Verified Formula:**
$$\text{Empirical Coverage}_q = \frac{1}{N} \sum_{i=1}^{N} \mathbb{I}(y_i \in [Q_{0.1}(x_i), Q_{0.9}(x_i)])$$

For 80% prediction interval (p10 to p90):

- Expected coverage: 0.80
- Acceptable range: 0.75 to 0.85
- Alert threshold: < 0.75 or > 0.85

**Why:**

- Standard statistical formula for evaluating quantile regression
- Directly interpretable (what percent of actual outcomes fall within predicted range)
- Robust to distribution assumptions (non-parametric)
- Widely used in financial forecasting literature

**Source:** Journal of Machine Learning Research (Conformalized Quantile Regression)

---

### Decision 4: Information Coefficient using Pearson Correlation

**What:** Calculate IC as Pearson correlation between predicted returns and actual returns

**Verified Formula:**
$$IC = \rho(R_{predicted}, R_{actual}) = \frac{Cov(R_{predicted}, R_{actual})}{\sigma_{predicted} \cdot \sigma_{actual}}$$

Where:

- $\rho$ is Pearson correlation coefficient
- $Cov$ is covariance between predicted and actual returns
- $\sigma$ is standard deviation

**Threshold:** IC > 0.15 considered acceptable for financial predictions

**Why:**

- Standard metric in quantitative finance for forecast skill
- Range [-1, 1] easy to interpret (0 = no skill, 1 = perfect)
- More robust than MSE/MAE for financial returns (captures directional accuracy)
- Used by major quantitative hedge funds (AQR, Bridgewater)

**Alternatives Considered:**

- Spearman rank correlation → Rejected: loses magnitude information, only captures rank order
- Mean Squared Error → Rejected: penalizes large errors excessively, not robust to outliers in financial data

**Source:** Multiple industry sources (Bajaj AMC, Cockatoo Quant)

---

### Decision 5: Kelly Criterion for Position Sizing

**What:** Use Kelly formula to calculate optimal position size based on win rate and reward/risk ratio

**Verified Formula:**
$$K\% = W - \frac{(1 - W)}{R}$$

Where:

- $K\%$ = Kelly percentage (fraction of capital to risk)
- $W$ = Win rate (probability of profitable trade)
- $R$ = Win/loss ratio (average gain / average loss)

**Modification:** Use Half-Kelly in production ($K\% / 2$) to reduce volatility

**Why:**

- Maximizes long-term geometric growth rate of capital (mathematically optimal)
- Self-adjusting based on strategy edge
- Prevents over-betting and ruin risk
- Half-Kelly reduces drawdowns by ~50% with only 25% reduction in growth

**Example:**

```
Win rate = 0.60 (60% of trades profitable)
Avg gain = 2.5%, Avg loss = 1.5%
R = 2.5 / 1.5 = 1.67

K% = 0.60 - (0.40 / 1.67) = 0.60 - 0.24 = 0.36 (36% of capital)
Half-Kelly = 18% of capital
```

**Source:** Investopedia, BlackwellGlobal, Wikipedia (John Kelly Jr., 1956)

---

### Decision 6: Sharpe Ratio for Risk-Adjusted Returns

**What:** Calculate Sharpe ratio to compare strategies adjusting for risk

**Verified Formula:**
$$\text{Sharpe Ratio} = \frac{R_p - R_f}{\sigma_p}$$

Where:

- $R_p$ = Expected portfolio return
- $R_f$ = Risk-free rate (Vietnamese 10-year government bond ~2-3%)
- $\sigma_p$ = Standard deviation of portfolio returns (volatility)

**Threshold:** Sharpe > 1.0 required for production trading

**Why:**

- Industry standard for risk-adjusted performance
- Accounts for volatility (high return with high volatility is less attractive)
- Comparable across strategies and time periods
- Nobel Prize-winning metric (William Sharpe, 1990)

**Vietnamese Market Adjustment:**
Use VN government bond yield (currently ~2.5%) as risk-free rate instead of US Treasury

**Source:** Wall Street Prep, Corporate Finance Institute

---

### Decision 7: Mean Absolute Error for Regression Metrics

**What:** Use MAE as primary regression metric alongside IC

**Verified Formula:**
$$MAE = \frac{1}{n} \sum_{i=1}^{n} |y_i - \hat{y}_i|$$

Where:

- $n$ = number of predictions
- $y_i$ = actual return
- $\hat{y}_i$ = predicted return

**Threshold:** MAE < 3% for acceptable model

**Why:**

- Expressed in same units as target (percentage returns) - easy to interpret
- Robust to outliers (unlike MSE which squares errors)
- Directly translates to expected prediction error
- Standard metric for regression evaluation

**Source:**GeeksforGeeks, Deepchecks, scikit-learn

---

### Decision 8: Bootstrap Ensembles for Uncertainty Quantification

**What:** Train 10 XGBoost models on different bootstrap samples to separate aleatoric and epistemic uncertainty

**Implementation:**

```python
# Train ensemble
for i in range(10):
    bootstrap_sample = resample(train_data, replace=True, n_samples=len(train_data))
    models[i].fit(bootstrap_sample)

# Predict with uncertainty
predictions = [model.predict(X) for model in models]
epistemic_uncertainty = np.std(predictions, axis=0)  # Model disagreement
aleatoric_uncertainty = np.mean([pred.p90 - pred.p10 for pred in predictions])
total_uncertainty = np.sqrt(epistemic**2 + aleatoric**2)
```

**Definitions:**

- **Aleatoric Uncertainty:** Inherent market randomness (irreducible)
- **Epistemic Uncertainty:** Model parameter uncertainty (reducible with more data)

**Why:**

- Separate sources of uncertainty require different responses
  - High epistemic → Retrain with more data
  - High aleatoric → Reduce position size (market unpredictable)
- Bootstrap ensembles provide reliable uncertainty estimates
- Computationally efficient (10x models still tractable)

**Source:** Medium (Uncertainty Quantification in ML), NSF Research

---

### Decision 9: XGBoost Feature Importance using Gain Metric

**What:** Track feature importance using gain-based metric (not weight/frequency)

**Formula:**
$$\text{Gain}_{\text{feature}} = \sum_{\text{all splits using feature}} \text{improvement in accuracy}$$

**Why:**

- Gain measures actual contribution to prediction accuracy
- Weight (frequency) only counts usage, not impact
- Cover measures affected samples, but doesn't reflect performance
- Gain aligns with model objective (minimize prediction error)

**Stability Metric:**
$$\text{Stability} = \frac{1}{\text{CV}(\text{Gain})} = \frac{\text{mean}(\text{Gain})}{\text{std}(\text{Gain})}$$

Coefficient of variation < 0.3 considered stable

**Source:** Medium (XGBoost Feature Importance), Stack Exchange

---

## Vietnamese Market-Specific Considerations

### Transaction Cost Formula

**Verified Rates:**

- Brokerage fee: 0.15% (both buy and sell)
- Securities transaction tax: 0.1% (sell only)
- **Round-trip cost:** 0.15% + (0.15% + 0.1%) = 0.40%

**Fee-Adjusted Return:**
$$R_{\text{net}} = R_{\text{gross}} - 0.004$$

**Minimum Profitable Return:**

- 1-day horizon: 1.0% (covers fees + minimum edge)
- 5-day horizon: 1.5% (reduces fee impact via longer holding)
- 10-day horizon: 2.0% (further fee amortization)

---

### Circuit Breaker Rules

**HOSE (Ho Chi Minh Stock Exchange):** ±7% daily limit  
**HNX (Hanoi Stock Exchange):** ±10% daily limit

When stock hits floor price:

- All bids removed
- Only sell orders at floor price accepted
- Liquidity disappears
- Cannot execute stop-loss

**Risk:** Position trapped for multiple consecutive days if panic continues

---

### Liquidity Constraint Formula

**Position Cap:**
$$\text{Max Shares} = \text{Avg Daily Volume}_{20d} \times 0.01$$

**Liquidity Score:**

- Score 10: > 5M shares/day (VNM, HPG, VIC)
- Score 5: 500K - 5M shares/day
- Score 1: 100K - 500K shares/day (risky)
- **Exclude:** < 100K shares/day (untradeable)

**Execution Strategy:**
If position = 1% of daily volume, split into 10 orders over 2-3 hours to minimize market impact

---

## Database Schema Design

### walk_forward_results

```sql
CREATE TABLE walk_forward_results (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    model_horizon INT NOT NULL,  -- 1, 5, 10 days
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    mae DECIMAL(8,4),            -- Mean Absolute Error
    ic DECIMAL(6,4),              -- Information Coefficient (-1 to 1)
    directional_accuracy DECIMAL(6,4),  -- Proportion correct direction
    sharpe_ratio DECIMAL(8,4),
    fee_adjusted_sharpe DECIMAL(8,4),  -- After 0.4% fees
    num_predictions INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_ticker_horizon (ticker, model_horizon),
    INDEX idx_period (period_start, period_end)
);
```

### calibration_reports

```sql
CREATE TABLE calibration_reports (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    model_horizon INT NOT NULL,
    quantile_level VARCHAR(5) NOT NULL,  -- 'p10', 'p25', 'p50', 'p75', 'p90'
    expected_coverage DECIMAL(6,4) NOT NULL,  -- 0.10 for p10
    actual_coverage DECIMAL(6,4) NOT NULL,    -- Empirical coverage observed
    calibration_error DECIMAL(6,4) NOT NULL,  -- actual - expected
    num_samples INT NOT NULL,
    check_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL,  -- 'OK', 'WARNING', 'ERROR'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_ticker_horizon (ticker, model_horizon),
    INDEX idx_date (check_date)
);
```

(Additional tables follow same pattern...)

---

## Risk Mitigation

### Risk 1: Computational Cost

- **Problem:** Walk-forward validation takes 4-6 hours per stock
- **Mitigation:** Run weekly, not daily; parallelize across stocks; use cloud GPU instances

### Risk 2: Storage Growth

- **Problem:** Validation tables grow 50MB/year
- **Mitigation:** Minimal impact; implement rolling retention (keep 2 years)

### Risk 3: Signal Reduction

- **Problem:** Stricter filters reduce trade frequency 40-60%
- **Mitigation:** Acceptable if win rate and Sharpe improve; track both metrics

---

## Open Questions

1. **Risk-free rate:** Use Vietnamese 10-year bond (2.5%) or USD equivalent (4.0%)?
   - **Recommendation:** Use VND rate for local currency returns
2. **Calibration tolerance:** Is 75-85% coverage too wide? Tighten to 77-83%?
   - **Defer:** Start with wider range, tighten after observing real calibration

3. **Ensemble size:** 10 models sufficient or increase to 20/30?
   - **Recommendation:** Start with 10, increase if uncertainty estimates unreliable
