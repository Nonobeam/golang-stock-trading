# Spec: Walk-Forward Validation

Walk-forward validation provides time-series cross-validation to prove models work out-of-sample across multiple distinct time periods, simulating real production conditions.

## ADDED Requirements

### Requirement: Expanding Window Validation

The system SHALL perform rolling time-series validation with an expanding training window to simulate realistic model retraining.

#### Scenario: Validate VCI model across 2024-2025

**Given** VCI has daily bars from 2024-01-01 to 2025-12-31  
**And** walk-forward validator configured with 252-day initial training, 20-day test windows  
**When** validator.validate('VCI', '2024-01-01', '2025-12-31') is called  
**Then** system should:

- Train model on [2024-01-01, 2024-12-31] (252 days)
- Test on [2025-01-01, 2025-01-20] (20 days)
- Train model on [2024-01-01, 2025-01-20] (272 days)
- Test on [2025-01-21, 2025-02-10] (20 days)
- Repeat sliding forward by 20 days until end date
- Store all test period metrics in database

#### Scenario: Calculate aggregate metrics across all periods

**Given** walk-forward validation completed for VCI 5-day horizon  
**And** 18 separate test periods exist (360 days / 20-day windows)  
**When** aggregate metrics are calculated  
**Then** system should compute:

- Mean MAE across all 18 periods
- Mean IC (Information Coefficient) across all periods
- Mean directional accuracy (% correct up/down predictions)
- Mean Sharpe ratio
- Fee-adjusted Sharpe ratio (after 0.4% transaction costs)
- Standard deviation of metrics across periods (consistency check)

**And** metrics should meet thresholds:

- Mean MAE < 3.0%
- Mean IC > 0.15
- Mean directional accuracy > 0.55 (55%)
- Mean Sharpe > 1.0
- Fee-adjusted Sharpe > 1.0

---

### Requirement: Information Coefficient Calculation

The system MUST correctly calculate IC using Pearson correlation between predicted and actual returns.

**Formula:**
$$IC = \rho(R_{predicted}, R_{actual}) = \frac{Cov(R_{predicted}, R_{actual})}{\sigma_{predicted} \cdot \sigma_{actual}}$$

#### Scenario: Calculate IC for one test period

**Given** test period has 20 predictions  
**And** predicted returns: [2.3%, 1.8%, -0.5%, ...]  
**And** actual returns: [2.1%, 2.2%, -0.8%, ...]  
**When** IC is calculated  
**Then** system should:

- Calculate covariance between predicted and actual arrays
- Calculate standard deviation of predicted returns
- Calculate standard deviation of actual returns
- Compute IC = Cov / (σ_pred × σ_actual)
- Return value in range [-1, 1]

**And** IC > 0.15 indicates acceptable predictive skill

---

### Requirement: Mean Absolute Error Calculation

The system MUST calculate MAE as the average of absolute differences between predictions and actuals.

**Formula:**
$$MAE = \frac{1}{n} \sum_{i=1}^{n} |y_i - \hat{y}_i|$$

#### Scenario: Calculate MAE for test period

**Given** 20 predictions in test period  
**And** predicted median (p50) returns: [2.5%, 1.8%, 3.1%, ...]  
**And** actual returns: [2.1%, 2.2%, 2.8%, ...]  
**When** MAE is calculated  
**Then** system should:

- Compute absolute difference for each prediction: |actual - predicted|
- Sum all absolute differences
- Divide by number of predictions
- Express result as percentage

**Example:**
|Predicted|Actual|Absolute Error|
|---------|------|--------------|
|2.5%|2.1%|0.4%|
|1.8%|2.2%|0.4%|
|3.1%|2.8%|0.3%|

MAE = (0.4 + 0.4 + 0.3) / 3 = 0.367%

---

### Requirement: Directional Accuracy Calculation

The system MUST calculate the percentage of predictions that correctly predicted the direction (up/down) of price movement.

#### Scenario: Calculate directional accuracy

**Given** 20 predictions in test period  
**And** 12 predictions forecasted positive returns, actual was positive (correct)  
**And** 5 predictions forecasted negative returns, actual was negative (correct)  
**And** 3 predictions forecasted wrong direction  
**When** directional accuracy is calculated  
**Then** accuracy = (12 + 5) / 20 = 0.85 (85%)

**And** accuracy > 0.55 indicates useful directional signal

---

### Requirement: Fee-Adjusted Sharpe Ratio

The system MUST calculate Sharpe ratio after subtracting 0.4% transaction costs from each trade.

**Base Formula:**
$$\text{Sharpe Ratio} = \frac{R_p - R_f}{\sigma_p}$$

**Fee-Adjusted:**
$$R_{p,net} = R_{p,gross} - 0.004$$

#### Scenario: Calculate fee-adjusted Sharpe for test period

**Given** test period generated 10 BUY signals  
**And** gross returns from signals: [2.5%, 1.8%, 3.1%, -1.2%, ...]  
**And** mean gross return = 1.8%  
**And** std dev of returns = 2.1%  
**And** Vietnamese risk-free rate = 0.025 (2.5% annual)  
**When** fee-adjusted Sharpe is calculated  
**Then** system should:

- Subtract 0.4% from every trade return
- Calculate net returns: [2.1%, 1.4%, 2.7%, -1.6%, ...]
- Calculate mean net return = 1.4%
- Keep same std dev = 2.1% (volatility unchanged)
- Compute Sharpe = (0.014 - 0.025) / 0.021 = 0.52

**Note:** If gross Sharpe was 0.95, fee-adjusted might drop to 0.52 due to 0.4% drag

---

### Requirement: Database Storage for Results

The system MUST persist all validation results for historical tracking and analysis.

#### Scenario: Store validation results

**Given** walk-forward validation completed for VCI 5-day horizon  
**And** period 2025-01-01 to 2025-01-20 has metrics  
**When** results are stored  
**Then** system should insert into `walk_forward_results`:

- ticker = 'VCI'
- model_horizon = 5
- period_start = '2025-01-01'
- period_end = '2025-01-20'
- mae = 2.85
- ic = 0.17
- directional_accuracy = 0.56
- sharpe_ratio = 1.15
- fee_adjusted_sharpe = 0.68
- num_predictions = 20
- created_at = current_timestamp

**And** record should be queryable for reporting

---

### Requirement: Parallel Validation Across Horizons

The system MUST validate all 3 horizons (1-day, 5-day, 10-day) independently.

#### Scenario: Validate VCI for all horizons

**Given** VCI model exists for 1-day, 5-day, and 10-day predictions  
**When** full validation is run  
**Then** system should:

- Run walk-forward for 1-day horizon
- Run walk-forward for 5-day horizon
- Run walk-forward for 10-day horizon
- Store separate results for each horizon
- Compare performance across horizons

**And** execution can be parallelized (3 separate processes)
