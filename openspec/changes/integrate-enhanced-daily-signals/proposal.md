# Proposal: Integrate Enhanced Signal Generator into Daily Workflow

## Summary

Wire the `EnhancedSignalGenerator` (with transaction cost filtering, floor-hit risk assessment, and liquidity constraints) into the `daily_signals.py` operational workflow to ensure daily recommendations incorporate all Vietnamese market protections.

## Problem Statement

The ML service has two signal generators:

1. **Basic `SignalGenerator`** (`signals/generator.py`) - Position-aware but ignores:
   - Transaction costs (0.4% round-trip fees)
   - Floor-hit risk (circuit breaker traps)
   - Liquidity constraints (volume caps)

2. **`EnhancedSignalGenerator`** (`signals/enhanced_generator.py`) - Includes all protections but **not used** in production workflows

The `daily_signals.py` script (operational workflow) currently imports the **basic generator**, meaning:

- Buy signals may not overcome transaction costs (false profitability)
- Users can enter positions with high circuit breaker trap risk (>20%)
- Position sizes may exceed liquidity caps (execution disasters)

This creates a **misalignment** between implemented capabilities and operational reality.

## Proposed Solution

### Core Change

Replace `SignalGenerator` with `EnhancedSignalGenerator` in `daily_signals.py` to activate all validation layers.

### Implementation Steps

1. **Update `daily_signals.py` imports**:
   - Change: `from signals.generator import SignalGenerator`
   - To: `from signals.enhanced_generator import EnhancedSignalGenerator`

2. **Modify signal generation calls** to provide required inputs:
   - `current_features` (dict) - Technical indicators for floor-hit risk
   - `uncertainty_metrics` (dict) - Optional epistemic/aleatoric uncertainty

3. **Fetch current features** from database:
   - Query latest `features` table row for ticker
   - Extract: `momentum_5d`, `volume_surge`, `consecutive_down`, `distance_from_support`, `volatility_5d`, `relative_strength`, `rsi_14`

4. **Handle validation metadata** in report output:
   - Display `validations_passed` and `validations_failed` counts
   - Show critical warnings (floor risk, liquidity issues)
   - Include fee-adjusted vs gross returns in reasoning

5. **Backward compatibility**:
   - Enhanced generator extends basic generator interface
   - Existing position logic (stop-loss, targets) unchanged
   - Returns same signal types: `BUY_NEW`, `BUY_MORE`, `SELL`, `SELL_PARTIAL`, `HOLD`, `HOLD_NONE`

## Success Criteria

- [ ] `daily_signals.py` uses `EnhancedSignalGenerator`
- [ ] Daily reports show validation results (passed/failed checks)
- [ ] Floor-hit risk warnings appear when probability >10%
- [ ] Buy signals filtered when floor risk >20%
- [ ] Fee-adjusted returns displayed in signal reasoning
- [ ] Liquidity caps enforced (max position = 1% of 20d avg volume)
- [ ] No regression in existing position monitoring features

## Impact

### User-Visible Changes

- **Signal Reduction**: 20-40% fewer buy signals (only profitable after fees)
- **Enhanced Reasoning**: Reports now explain WHY signals were rejected
  - Example: "Expected return 1.2% below 1.5% threshold after fees"
  - Example: "CRITICAL: High floor-hit risk (23%) - Circuit breaker likely"
- **Liquidity Warnings**: Alerts when requested position exceeds safe limits

### System Benefits

- **Risk Mitigation**: Prevents entry into circuit breaker traps
- **Profitability**: Ensures all trades overcome 0.4% fee drag
- **Execution Quality**: Position sizes respect market liquidity

## Dependencies

- Existing `EnhancedSignalGenerator` implementation (complete)
- Database `features` table with current technical indicators
- Floor-hit classifier models trained for target tickers
- No new Python packages required

## Risks & Mitigations

| Risk                                 | Impact                         | Mitigation                                                       |
| ------------------------------------ | ------------------------------ | ---------------------------------------------------------------- |
| **Signal frequency drops sharply**   | Fewer trading opportunities    | Acceptable - quality over quantity is design goal                |
| **Features missing for new tickers** | Validation warnings in reports | Graceful degradation - validator logs warnings but doesn't crash |
| **Floor-hit model not trained**      | Risk assessment skipped        | System logs warning and continues with other validations         |

## Timeline

- **Day 1**: Update `daily_signals.py` imports and signal generation calls
- **Day 1**: Add feature fetching from database
- **Day 2**: Enhance report formatting to show validation metadata
- **Day 2**: Test with real positions (VCI, HPG, FPT)
- **Day 3**: Validation and documentation updates

Estimated effort: **2-3 days**

## Testing Strategy

1. **Unit Testing**: Verify enhanced generator integration
2. **Integration Testing**: Run `daily_signals.py` on historical dates with known positions
3. **Validation Checks**:
   - Compare basic vs enhanced signal counts
   - Verify fee-adjusted returns displayed
   - Confirm floor-hit warnings appear
   - Check liquidity caps enforced

## Related Changes

- Extends: `enhance-ml-prediction-validation` (provides the enhanced generator)
- Complements: Position management integration (existing)
- Enables: Future integration of uncertainty quantification metrics
