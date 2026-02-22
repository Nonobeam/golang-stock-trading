# Design: Enhanced Signal Generator Integration

## Architecture Overview

This change integrates the `EnhancedSignalGenerator` into the operational `daily_signals.py` workflow to activate Vietnamese market protections (transaction costs, floor-hit risk, liquidity constraints) that are already implemented but not currently used in production.

## Current State

```
┌─────────────────────────────────────────────────────────────┐
│                     Daily Workflow                          │
│                   (daily_signals.py)                        │
└────────────────────┬────────────────────────────────────────┘
                     │
                     │ imports
                     ▼
          ┌──────────────────────┐
          │  SignalGenerator     │  ← Basic generator (no validations)
          │  (generator.py)      │
          └──────────────────────┘
                     │
                     │ generates
                     ▼
               ┌──────────┐
               │  Signal  │  ⚠️ May ignore:
               └──────────┘    - Transaction costs
                               - Floor-hit risk
                               - Liquidity caps
```

**Unused Components**:

- `EnhancedSignalGenerator` (enhanced_generator.py) - **Fully implemented, not wired**
- `FloorHitClassifier` (floor_hit_classifier.py) - **Ready, not called**
- `TransactionCostModule` (transaction_costs.py) - **Ready, not called**
- `LiquidityManager` (liquidity_manager.py) - **Ready, not called**

## Target State

```
┌─────────────────────────────────────────────────────────────┐
│                     Daily Workflow                          │
│                   (daily_signals.py)                        │
└────────────────────┬────────────────────────────────────────┘
                     │
                     │ imports
                     ▼
          ┌──────────────────────────────────┐
          │  EnhancedSignalGenerator         │  ← Enhanced generator
          │  (enhanced_generator.py)         │
          └──────────┬───────────────────────┘
                     │
                     │ delegates to
                     ▼
     ┌───────────────────────────────────────────────┐
     │           Validation Modules                  │
     ├───────────────────────────────────────────────┤
     │  • TransactionCostModule                      │
     │    └─> Fee-adjusted return calculation        │
     │  • FloorHitClassifier                         │
     │    └─> Circuit breaker probability            │
     │  • LiquidityManager                           │
     │    └─> Position size caps                     │
     │  • PositionManager                            │
     │    └─> Stop-loss/target tracking (existing)   │
     └───────────────────────────────────────────────┘
                     │
                     │ generates
                     ▼
          ┌──────────────────────────┐
          │  Signal + Validation     │  ✓ Includes:
          │  Metadata                │    - Fee-adjusted returns
          └──────────────────────────┘    - Floor-hit probability
                                          - Liquidity constraints
                                          - Validation results
```

## Data Flow

### Input Requirements

1. **Predictions** (existing):
   - Multi-horizon forecasts: 1d, 5d, 10d
   - Quantiles: p10, p50, p90
   - Confidence scores

2. **Current Features** (new requirement):

   ```python
   current_features = {
       'momentum_5d': float,       # 5-day return
       'volume_surge': float,      # volume_ratio_5d
       'consecutive_down': int,    # streak of negative days
       'distance_from_support': float,  # (price - sma_20) / sma_20
       'volatility_5d': float,
       'relative_strength': float, # return_5d - return_20d
       'rsi_14': float
   }
   ```

   **Source**: Fetch from `features` table via new `get_current_features()` helper

3. **Uncertainty Metrics** (optional for now):
   ```python
   uncertainty_metrics = {
       'epistemic_uncertainty': float,  # Model disagreement
       'aleatoric_uncertainty': float,  # Market randomness
       'total_uncertainty': float,
       'confidence_score': float
   }
   ```

### Signal Generation Flow

```
1. Fetch Predictions
   ↓
2. Fetch Current Features ← NEW
   ↓
3. Generate Signal (Enhanced Generator)
   ├─> Check Stop-Loss (existing)
   ├─> Check Targets (existing)
   ├─> Check Floor-Hit Risk ← NEW
   │   └─> If > 20%: REJECT
   ├─> Check Transaction Costs ← NEW
   │   └─> If net_return < threshold: REJECT
   ├─> Check Liquidity ← NEW
   │   └─> If volume < 100K: REJECT
   ├─> Check Uncertainty ← NEW (optional)
   │   └─> If epistemic > 5%: REJECT
   └─> Apply Decision Logic
   ↓
4. Return Signal + Validation Metadata ← NEW
   ↓
5. Format Report with Validations ← NEW
```

## Component Interfaces

### EnhancedSignalGenerator

**Constructor**:

```python
EnhancedSignalGenerator(user_id: int = 1, exchange: str = 'HOSE')
```

**Key Method**:

```python
def generate_signal(
    ticker: str,
    predictions: Dict[int, Dict[str, float]],
    current_price: float,
    current_features: Dict,  # NEW
    uncertainty_metrics: Dict = None,  # NEW
    db_connection = None,
    user_id: int = None
) -> Tuple[str, float, str, Dict]:
    """
    Returns: (signal, strength, reason, validation_metadata)
    """
```

**Validation Metadata Structure**:

```python
validation_metadata = {
    'validations_passed': ['confidence', 'transaction_costs', 'liquidity'],
    'validations_failed': [],
    'warnings': ['Moderate floor risk (12%)'],
    'floor_hit_probability': 0.12,
    'fee_adjusted_return': 0.021,
    'gross_return': 0.025,
    'liquidity_score': 7,
    'max_position_shares': 15000
}
```

### Feature Fetching Helper

**New Function**:

```python
def get_current_features(conn, ticker: str, target_date: str) -> Dict:
    """
    Fetch latest technical features from database.

    Returns:
        Dict with keys: momentum_5d, volume_surge, consecutive_down,
        distance_from_support, volatility_5d, relative_strength, rsi_14

    Raises:
        None (returns None if features unavailable)
    """
```

**Database Query**:

```sql
SELECT
    return_5d,
    volume_ratio_5d,
    volatility_5d,
    rsi_14,
    sma_5,
    sma_20,
    return_1d,
    return_20d
FROM features
WHERE ticker = %s
  AND date <= %s
  AND features_complete = TRUE
ORDER BY date DESC
LIMIT 1
```

**Feature Calculation**:

```python
# Direct mappings
features['momentum_5d'] = row['return_5d']
features['volume_surge'] = row['volume_ratio_5d']
features['volatility_5d'] = row['volatility_5d']
features['rsi_14'] = row['rsi_14']

# Derived features
features['distance_from_support'] = (current_price - row['sma_20']) / row['sma_20']
features['relative_strength'] = row['return_5d'] - row['return_20d']

# Consecutive down days (requires window query or state tracking)
features['consecutive_down'] = calculate_consecutive_down(conn, ticker, target_date)
```

## Report Output Enhancement

### Before (Basic Generator)

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 VCI
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

ML Predictions:
  5d: +2.3% return (confidence: 70%, risk: -2% to +7%)

🟢 SIGNAL: BUY_MORE (strength: 0.75)
Reason: Strong 5d outlook: 2.3% return - Add to position
```

### After (Enhanced Generator)

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 VCI
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

ML Predictions:
  5d: +2.3% return (confidence: 70%, risk: -2% to +7%)

🟢 SIGNAL: BUY_MORE (strength: 0.75)
Reason: Strong 5d outlook: 1.9% (net after fees) - Add to position

Validation Results:
  ✓ Confidence: 70% (threshold: 60%)
  ✓ Transaction Costs: Net return 1.9% exceeds 1.5% minimum
  ✓ Liquidity: Score 7/10, max position 15,000 shares
  ✓ Floor Risk: 8% probability (acceptable)

Metrics:
  Gross Return: 2.3% → Net Return: 1.9% (after 0.4% fees)
  Floor-Hit Probability: 8%
  Max Position: 15,000 shares (1% of 20d avg volume)
```

## Error Handling Strategy

### Degradation Levels

1. **Features Unavailable**:
   - Log: `WARNING: Features not found for {ticker} on {date}, skipping floor-hit validation`
   - Behavior: Pass `current_features=None`, enhanced generator skips floor check
   - Other validations continue

2. **Floor-Hit Model Missing**:
   - Log: `WARNING: Floor-hit classifier not trained for {ticker}`
   - Behavior: FloorHitClassifier catches FileNotFoundError, returns None
   - Other validations continue

3. **Liquidity Data Unavailable**:
   - Log: `WARNING: Liquidity check failed: {error}`
   - Behavior: LiquidityManager returns default (tradeable=True)
   - Signal not rejected

### Failure Modes

| Component          | Failure            | Impact                   | Mitigation                        |
| ------------------ | ------------------ | ------------------------ | --------------------------------- |
| Features fetch     | DB error           | No floor-hit check       | Log warning, continue             |
| FloorHitClassifier | Model not found    | Skip floor validation    | Log warning, continue             |
| LiquidityManager   | Volume query fails | No liquidity cap         | Default to safe (tradeable)       |
| EnhancedGenerator  | Internal error     | Fall back to basic logic | Catch exception, log, return HOLD |

## Testing Strategy

### Unit Tests

1. **Feature Fetching**:
   - Mock database with feature rows
   - Verify correct feature dict returned
   - Test missing features handling

2. **Enhanced Generator**:
   - Test each validation path independently
   - Verify metadata structure
   - Test rejection scenarios

3. **Report Formatting**:
   - Test validation display
   - Test warning formatting
   - Test metric display

### Integration Tests

1. **End-to-End with Real Position**:
   - Run `daily_signals.py` with VCI position
   - Verify complete report format
   - Confirm all validations executed

2. **Rejection Scenarios**:
   - High floor risk (>20%) → HOLD_NONE
   - Low net return (<1.5%) → HOLD
   - Low liquidity → HOLD_NONE

3. **Regression Tests**:
   - Stop-loss triggering unchanged
   - Target detection unchanged
   - P&L calculation unchanged

## Migration Path

### Phase 1: Integration (Day 1)

- Update imports in `daily_signals.py`
- Add `get_current_features()` helper
- Modify `generate_and_save_signal()` call

### Phase 2: Enhancement (Day 2)

- Update report formatting
- Add validation result display
- Add warning/failure highlighting

### Phase 3: Validation (Day 3)

- Unit tests
- Integration tests
- Regression tests
- Documentation updates

## Rollback Strategy

If critical issues arise:

1. **Quick Rollback**:

   ```python
   # Change one line in daily_signals.py
   from signals.generator import SignalGenerator  # Revert to basic
   ```

2. **Verification**:
   - Run `daily_signals.py` with test position
   - Confirm basic functionality restored

3. **Investigation**:
   - Review logs for errors
   - Fix enhanced generator issues
   - Re-deploy

## Performance Considerations

### Additional Database Queries

- 1 extra query per position: `get_current_features()`
- Impact: ~50ms per position
- Total overhead for 5 positions: ~250ms (negligible)

### Floor-Hit Classification

- Model inference: ~10ms per prediction
- Impact: Minimal (in-memory XGBoost)

### Memory Usage

- Validation metadata: ~1KB per signal
- Impact: Negligible

## Security & Privacy

No new security concerns:

- Uses existing database connections
- No new external API calls
- No sensitive data exposure

## Future Enhancements

After integration, can add:

1. **Uncertainty Quantification**:
   - Feed `uncertainty_metrics` to enhanced generator
   - Reject signals with high epistemic uncertainty

2. **Regime Awareness**:
   - Detect market regime (bull/bear/volatile)
   - Apply regime-specific models

3. **Cross-Stock Validation**:
   - Validate model architecture across 10 stocks
   - Ensure predictions robust to ticker choice
