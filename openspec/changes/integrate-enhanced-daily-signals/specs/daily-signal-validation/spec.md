# Daily Signal Validation Capability

## Overview

This spec defines the requirements for integrating the `EnhancedSignalGenerator` into the `daily_signals.py` operational workflow to enable Vietnamese market-specific validations (transaction costs, floor-hit risk, liquidity constraints) in daily position monitoring and signal generation.

## Why

The ML service has comprehensive validation modules (transaction costs, floor-hit risk, liquidity constraints) implemented in `EnhancedSignalGenerator`, but the operational `daily_signals.py` workflow uses the basic `SignalGenerator` which bypasses all protections. This creates a critical misalignment where:

1. **False Profitability**: Buy signals may not overcome 0.4% Vietnamese transaction costs
2. **Circuit Breaker Risk**: Users can enter positions with >20% probability of being trapped at -7% floor limits
3. **Liquidity Disasters**: Position sizes may exceed safe execution limits (1% of daily volume)

Wiring the enhanced generator into the daily workflow ensures all recommendations respect Vietnamese market realities and protects users from preventable losses.

---

## ADDED Requirements

### Requirement: Enhanced Signal Generator Integration

The daily signals script MUST use `EnhancedSignalGenerator` instead of the basic `SignalGenerator` to activate all validation modules.

#### Scenario: Daily workflow uses enhanced generator

**Given** the daily signals script is executed for an active position  
**When** the script generates trading signals  
**Then** all Vietnamese market validations MUST be applied:

- Transaction cost filtering (0.4% round-trip fees)
- Floor-hit risk assessment (circuit breaker probability)
- Liquidity constraint enforcement (1% volume cap)

**And** validation results MUST be included in the signal metadata

#### Scenario: Backward compatibility maintained

**Given** an existing position with stop-loss and targets configured  
**When** the enhanced generator processes the position  
**Then** all existing position management features MUST work unchanged:

- Stop-loss triggering and alerts
- Target level detection and partial sells
- Unrealized P&L calculation
- Position-aware signal differentiation (BUY_NEW vs BUY_MORE)

---

### Requirement: Feature Fetching for Floor-Hit Assessment

The daily signals script MUST fetch current technical features from the database to enable floor-hit risk prediction.

#### Scenario: Features successfully retrieved

**Given** a ticker with complete feature data in the database  
**When** the script fetches current features for the ticker  
**Then** a feature dictionary MUST be returned containing:

- `momentum_5d`: 5-day return percentage
- `volume_surge`: Current volume relative to 20-day average
- `consecutive_down`: Count of consecutive negative return days
- `distance_from_support`: Distance to SMA20 support level
- `volatility_5d`: 5-day rolling volatility
- `relative_strength`: 5-day vs 20-day return comparison
- `rsi_14`: 14-period Relative Strength Index

#### Scenario: Features unavailable graceful degradation

**Given** a ticker with no feature data in the database  
**When** the script attempts to fetch current features  
**Then** a warning MUST be logged indicating features are unavailable  
**And** the script MUST continue with `current_features=None`  
**And** the enhanced generator MUST skip floor-hit validation  
**And** other validations (transaction costs, liquidity) MUST still execute

---

### Requirement: Transaction Cost Validation

All buy signals MUST be validated for profitability after Vietnamese market transaction costs (0.4% round-trip).

#### Scenario: Buy signal profitable after fees

**Given** a predicted return of 2.5% for 5-day horizon  
**When** transaction cost validation is applied  
**Then** the gross return of 2.5% MUST be reduced by 0.4% fees  
**And** the net return of 2.1% MUST exceed the 1.5% minimum threshold for 5-day trades  
**And** the signal MUST pass transaction cost validation  
**And** validation metadata MUST record:

```python
{
  'gross_return': 0.025,
  'fee_adjusted_return': 0.021,
  'validations_passed': ['transaction_costs']
}
```

#### Scenario: Buy signal unprofitable after fees

**Given** a predicted return of 1.2% for 5-day horizon  
**When** transaction cost validation is applied  
**Then** the net return of 0.8% MUST be below the 1.5% minimum threshold  
**And** the signal MUST be rejected with `HOLD` or `HOLD_NONE`  
**And** the rejection reason MUST state: "Expected return 1.2% below 1.5% threshold after fees"  
**And** validation metadata MUST record:

```python
{
  'validations_failed': [{
    'check': 'transaction_costs',
    'gross_return': 0.012,
    'min_threshold': 0.015,
    'horizon': 5
  }]
}
```

---

### Requirement: Floor-Hit Risk Assessment

Buy signals MUST be blocked when circuit breaker (floor-hit) probability exceeds 20%.

#### Scenario: Acceptable floor risk

**Given** current features indicate moderate downside momentum  
**When** floor-hit classifier predicts 8% probability  
**Then** the signal MUST pass floor risk validation  
**And** validation metadata MUST record:

```python
{
  'floor_hit_probability': 0.08,
  'validations_passed': ['floor_risk']
}
```

#### Scenario: High floor risk rejection

**Given** current features indicate strong bearish signals (momentum_5d: -4.2%, consecutive_down: 3)  
**When** floor-hit classifier predicts 23% probability  
**Then** the signal MUST be rejected with `HOLD_NONE`  
**And** the rejection reason MUST state: "CRITICAL: High floor-hit risk (23%) - Circuit breaker likely"  
**And** validation metadata MUST record:

```python
{
  'floor_hit_probability': 0.23,
  'validations_failed': [{
    'check': 'floor_risk',
    'probability': 0.23,
    'threshold': 0.20
  }]
}
```

#### Scenario: Moderate floor risk warning

**Given** floor-hit classifier predicts 12% probability  
**When** floor risk is assessed  
**Then** the signal MAY proceed (below 20% rejection threshold)  
**But** a warning MUST be added to validation metadata:

```python
{
  'warnings': ['Moderate floor risk (12%) - Consider reducing position']
}
```

---

### Requirement: Liquidity Constraint Enforcement

Buy signals MUST respect market liquidity limits to prevent execution disasters.

#### Scenario: Sufficient liquidity

**Given** a ticker with 20-day average volume of 1,500,000 shares  
**When** liquidity validation is performed  
**Then** the maximum allowed position MUST be 15,000 shares (1% of average volume)  
**And** the liquidity score MUST be calculated (1-10 scale)  
**And** validation metadata MUST record:

```python
{
  'liquidity_score': 7,
  'avg_volume_20d': 1500000,
  'max_position_shares': 15000,
  'validations_passed': ['liquidity']
}
```

#### Scenario: Insufficient liquidity rejection

**Given** a ticker with 20-day average volume of 80,000 shares (below 100K minimum)  
**When** liquidity validation is performed  
**Then** the signal MUST be rejected with `HOLD_NONE`  
**And** the rejection reason MUST state: "Liquidity too low (score: 2/10)"  
**And** validation metadata MUST record:

```python
{
  'validations_failed': [{
    'check': 'liquidity',
    'liquidity_score': 2,
    'min_score': 2,
    'avg_volume': 80000
  }]
}
```

---

### Requirement: Validation Result Reporting

Daily reports MUST display validation results and rejection reasons for all positions.

#### Scenario: Successful signal with validations

**Given** a buy signal passes all validations  
**When** the daily report is generated  
**Then** the report MUST display:

- Signal type and strength
- **NEW**: Validation summary showing passed checks
- **NEW**: Fee-adjusted return vs gross return
- **NEW**: Floor-hit probability
- **NEW**: Liquidity metrics

**Example output**:

```
🟢 SIGNAL: BUY_MORE (strength: 0.75)
Reason: Strong 5d outlook: 1.9% (net after fees) - Add to position

Validation Results:
  ✓ Confidence: 70%
  ✓ Transaction Costs: Net 1.9% exceeds 1.5% minimum
  ✓ Liquidity: Score 7/10, max 15,000 shares
  ✓ Floor Risk: 8% probability

Metrics:
  Gross Return: 2.3% → Net Return: 1.9%
  Floor-Hit Probability: 8%
  Max Position: 15,000 shares
```

#### Scenario: Rejected signal with explanations

**Given** a buy signal fails floor risk validation  
**When** the daily report is generated  
**Then** the report MUST display:

- **NEW**: Clear rejection reason
- **NEW**: Failed validation details
- **NEW**: Specific metrics that triggered rejection

**Example output**:

```
⚪ SIGNAL: HOLD_NONE (strength: 0.00)
Reason: CRITICAL: High floor-hit risk (23%) - Circuit breaker likely

Validation Results:
  ✓ Confidence: 68%
  ✓ Transaction Costs: Net 1.8% exceeds 1.5% minimum
  ✓ Liquidity: Score 6/10
  ✗ Floor Risk: 23% probability (threshold: 20%)

Warning: Stock shows strong bearish momentum with 3 consecutive down days
```

#### Scenario: Warnings displayed

**Given** a signal has validation warnings but passes  
**When** the daily report is generated  
**Then** the report MUST display all warnings with clear indicators

**Example output**:

```
Validation Results:
  ✓ All checks passed

⚠ Warnings:
  - Moderate floor risk (12%) - Consider reducing position
  - High market unpredictability (16%)
```

---

### Requirement: Error Handling and Graceful Degradation

The system MUST handle validation failures gracefully without blocking the daily workflow.

#### Scenario: Missing floor-hit model

**Given** the floor-hit classifier model file does not exist for a ticker  
**When** the enhanced generator attempts floor-hit validation  
**Then** a warning MUST be logged: "Floor-hit classifier not trained for {ticker}"  
**And** floor-hit validation MUST be skipped  
**And** other validations MUST continue  
**And** the signal generation MUST proceed (not crash)

#### Scenario: Database connection failure during feature fetch

**Given** the database is temporarily unavailable  
**When** the script attempts to fetch current features  
**Then** an error MUST be logged  
**And** `current_features=None` MUST be passed to the generator  
**And** floor-hit validation MUST be skipped with warning  
**And** transaction cost and liquidity validations MUST attempt to proceed

#### Scenario: Liquidity manager exception

**Given** the liquidity manager encounters an error querying volume data  
**When** liquidity validation is attempted  
**Then** the error MUST be logged  
**And** a warning MUST be added to validation metadata  
**And** the signal MUST NOT be automatically rejected (safe default: tradeable=True)

---

## MODIFIED Requirements

### Requirement: Daily Signal Generation Interface

The `generate_and_save_signal()` call in `daily_signals.py` MUST be updated to provide enhanced inputs.

#### Scenario: Enhanced signal generation call

**Given** predictions and current price are available  
**When** the script calls `generate_and_save_signal()`  
**Then** the call MUST include:

- `ticker`: Stock symbol
- `predictions`: Multi-horizon forecast dict
- `date`: Signal date (YYYY-MM-DD)
- `current_price`: Current market price
- **NEW**: `current_features`: Technical feature dict (or None)
- `db_connection`: Database connection for position queries
- `user_id`: User identifier

**And** the return value MUST be unpacked as:

```python
signal_dict, save_success = sg.generate_and_save_signal(
    ticker=ticker,
    predictions=predictions,
    date=report_date,
    current_price=current_price,
    current_features=features,  # NEW
    db_connection=conn,
    user_id=user_id
)
```

**And** `signal_dict` MUST contain the new validation metadata

---

## Implementation Notes

### Feature Calculation Details

**Consecutive Down Days**: Requires counting the streak of negative returns leading up to the current date. Implementation options:

1. **Window Query** (recommended):

```sql
WITH recent_returns AS (
  SELECT date, return_1d
  FROM features
  WHERE ticker = %s AND date <= %s
  ORDER BY date DESC
  LIMIT 10
)
SELECT COUNT(*) AS consecutive_down
FROM recent_returns
WHERE return_1d < 0
ORDER BY date DESC
```

2. **Iterative Calculation**: Loop through last N days until first positive return

### Threshold Reference

| Horizon | Min Net Return | Gross Equivalent |
| ------- | -------------- | ---------------- |
| 1 day   | 1.0%           | 1.4%             |
| 5 day   | 1.5%           | 1.9%             |
| 10 day  | 2.0%           | 2.4%             |

Floor-hit probability thresholds:

- **Critical (reject)**: > 20%
- **Warning**: 10-20%
- **Acceptable**: < 10%

Liquidity score thresholds:

- **Reject**: < 2/10
- **Warning**: 2-4/10
- **Acceptable**: > 4/10

---

## Dependencies

- `EnhancedSignalGenerator` (signals/enhanced_generator.py) - Must be already implemented
- `FloorHitClassifier` (models/floor_hit_classifier.py) - Must have trained models
- `TransactionCostModule` (validation/transaction_costs.py) - Must be functional
- `LiquidityManager` (validation/liquidity_manager.py) - Must be functional
- Database `features` table - Must contain technical indicators

---

## Success Metrics

- [ ] Daily reports show validation results for all positions
- [ ] Buy signals filtered when floor risk > 20% (measured via rejected signal count)
- [ ] Fee-adjusted returns displayed in all signal reasoning
- [ ] Liquidity caps enforced (max position = 1% of 20d volume)
- [ ] No regression in existing features (stop-loss, targets, P&L)
- [ ] Signal rejection rate: 20-40% (quality over quantity)
