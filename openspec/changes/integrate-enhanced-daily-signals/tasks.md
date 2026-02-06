# Implementation Tasks

## Overview

Wire `EnhancedSignalGenerator` into `daily_signals.py` to activate transaction cost filtering, floor-hit risk assessment, and liquidity constraints in the daily operational workflow.

---

## Phase 1: Core Integration (Day 1)

### Task 1.1: Update Signal Generator Import

- [ ] **File**: `ml-service/daily/daily_signals.py`
- [ ] Change import from `signals.generator.SignalGenerator` to `signals.enhanced_generator.EnhancedSignalGenerator`
- [ ] Update instantiation: `sg = EnhancedSignalGenerator(user_id=user_id, exchange='HOSE')`
- [ ] **Validation**: Script imports without errors

### Task 1.2: Add Feature Fetching Function

- [ ] **File**: `ml-service/daily/daily_signals.py`
- [ ] Create `get_current_features(conn, ticker, target_date)` function
- [ ] Query latest features row from database:
  ```sql
  SELECT momentum_5d, volume_ratio_5d, volatility_5d, rsi_14,
         sma_5, sma_20, return_1d, return_5d, return_20d
  FROM features
  WHERE ticker = %s AND date = %s AND features_complete = TRUE
  ```
- [ ] Calculate derived features:
  - `volume_surge = volume_ratio_5d`
  - `consecutive_down` (count negative return_1d streak)
  - `distance_from_support = (current_price - sma_20) / sma_20`
  - `relative_strength = return_5d - return_20d`
- [ ] Return dict with keys: `momentum_5d`, `volume_surge`, `consecutive_down`, `distance_from_support`, `volatility_5d`, `relative_strength`, `rsi_14`
- [ ] **Validation**: Function returns valid feature dict for test ticker

### Task 1.3: Modify Signal Generation Call

- [ ] **File**: `ml-service/daily/daily_signals.py` (line ~238)
- [ ] Update `generate_and_save_signal()` call to include:
  - `current_features=features_dict`
  - `uncertainty_metrics=None` (optional for now)
- [ ] Handle return value with 4 elements: `signal_dict, save_success` (enhanced generator returns validation metadata)
- [ ] **Validation**: Signal generation succeeds with enhanced inputs

---

## Phase 2: Report Enhancement (Day 2)

### Task 2.1: Display Validation Results

- [ ] **File**: `ml-service/daily/daily_signals.py`
- [ ] Extract validation metadata from signal response
- [ ] Add validation summary to report after signal:
  ```
  Validations: ✓ confidence ✓ transaction_costs ✓ liquidity ✓ floor_risk
  Fee-adjusted return: 2.1% (gross: 2.5%)
  ```
- [ ] **Validation**: Report shows validation checks for all positions

### Task 2.2: Format Warnings and Failures

- [ ] **File**: `ml-service/daily/daily_signals.py`
- [ ] Add warning section if `validation_metadata['warnings']` not empty:
  ```
  ⚠ Warnings:
    - Moderate floor risk (12%) - Consider reducing position
    - High market unpredictability (16%)
  ```
- [ ] Highlight failed validations in red/bold
- [ ] **Validation**: Failed validations clearly visible in reports

### Task 2.3: Update Report Format

- [ ] **File**: `ml-service/daily/daily_signals.py`
- [ ] Modify signal display to show:
  - Gross vs net returns
  - Floor-hit probability (if available)
  - Liquidity score and max position
- [ ] Add emoji indicators for validation status (✓/⚠/❌)
- [ ] **Validation**: Enhanced report format is readable and informative

---

## Phase 3: Error Handling & Edge Cases (Day 2-3)

### Task 3.1: Graceful Feature Degradation

- [ ] **File**: `ml-service/daily/daily_signals.py`
- [ ] Handle missing features gracefully:
  - If features query returns None, log warning
  - Pass `current_features=None` to enhanced generator
  - Generator will skip floor-hit validation but continue
- [ ] **Validation**: Script doesn't crash when features unavailable

### Task 3.2: Handle Missing Floor-Hit Models

- [ ] **File**: `ml-service/daily/daily_signals.py`
- [ ] Catch exception if floor-hit classifier not trained
- [ ] Log warning: "Floor-hit risk not assessed - model not trained for {ticker}"
- [ ] Continue with other validations
- [ ] **Validation**: Missing floor model doesn't block signal generation

### Task 3.3: Liquidity Manager Exception Handling

- [ ] **File**: Already handled in `EnhancedSignalGenerator._check_liquidity()`
- [ ] Verify that liquidity check failures don't reject signals
- [ ] **Validation**: Liquidity errors logged but don't crash workflow

---

## Phase 4: Testing & Validation (Day 3)

### Task 4.1: Unit Tests

- [ ] **File**: `ml-service/tests/test_daily_signals.py` (create if not exists)
- [ ] Test feature fetching with mock database
- [ ] Test signal generation with/without features
- [ ] Test validation metadata parsing
- [ ] **Validation**: All tests pass

### Task 4.2: Integration Test - VCI Position

- [ ] Run `daily_signals.py` with real VCI position
- [ ] Verify output shows:
  - Fee-adjusted returns
  - Floor-hit probability
  - Liquidity caps
  - Validation checks
- [ ] **Validation**: Report matches expected format

### Task 4.3: Regression Test - Basic Features

- [ ] Test stop-loss triggering still works
- [ ] Test target level detection still works
- [ ] Test position P&L calculation unchanged
- [ ] **Validation**: No regression in existing features

### Task 4.4: Compare Signal Counts

- [ ] Run enhanced generator on historical dates (last 30 days)
- [ ] Compare signal counts: basic vs enhanced
- [ ] Document reduction rate and rejection reasons
- [ ] **Validation**: Signal reduction within expected range (20-40%)

---

## Phase 5: Documentation (Day 3)

### Task 5.1: Update README

- [ ] **File**: `ml-service/daily/README.md`
- [ ] Document enhanced signal generator usage
- [ ] Add examples of validation output
- [ ] Explain rejection reasons
- [ ] **Validation**: README accurately describes new behavior

### Task 5.2: Update IMPLEMENTATION_SUMMARY

- [ ] **File**: `ml-service/IMPLEMENTATION_SUMMARY.md`
- [ ] Mark integration as complete
- [ ] Document daily workflow enhancements
- [ ] **Validation**: Summary reflects current state

### Task 5.3: Add Usage Examples

- [ ] **File**: `ml-service/daily/README.md`
- [ ] Example command with output
- [ ] Example of rejected signal with reasoning
- [ ] Example of warning handling
- [ ] **Validation**: Examples are actionable

---

## Acceptance Criteria

- [ ] `daily_signals.py` uses `EnhancedSignalGenerator`
- [ ] Reports display validation results (passed/failed checks)
- [ ] Floor-hit warnings appear when probability >10%
- [ ] Buy signals blocked when floor risk >20%
- [ ] Fee-adjusted returns shown in signal reasoning
- [ ] Liquidity caps enforced and displayed
- [ ] No regression in stop-loss/target/P&L tracking
- [ ] Graceful handling of missing features/models
- [ ] Documentation updated

---

## Estimated Effort

- **Phase 1 (Integration)**: 4 hours
- **Phase 2 (Reporting)**: 3 hours
- **Phase 3 (Error Handling)**: 2 hours
- **Phase 4 (Testing)**: 4 hours
- **Phase 5 (Documentation)**: 2 hours

**Total**: 15 hours (~2 days)
