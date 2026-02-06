# Change: Enforce Stop-Loss Preservation and Complete Risk Management

## Why

Analysis of the current implementation reveals three critical gaps in position risk management:

1. **Stop-loss drift** - No enforcement preventing stop_loss from being modified after position entry. The system calculates risk assuming stop_loss equals the first entry stop level, but nothing prevents manual updates that invalidate this assumption.

2. **Incomplete risk calculation** - Current risk formula uses simplified calculation `current_risk = current_shares * (first_entry_price - stop_loss)` which doesn't account for multiple entries at different prices against the same stop level.

3. **Missing soft capacity warnings** - System only checks hard limits (20% value, 1% volume). No warnings when approaching limits at 18% value or 0.8% volume, reducing trader awareness.

These gaps allow positions to accumulate excessive risk through multiple entries or stop-loss modifications, undermining the 2% per-trade risk limit that protects the account.

## What Changes

### Database Schema

**Modified Table: `positions`**

- Add `first_entry_stop_loss` (DECIMAL) - Immutable stop level set at first purchase
- Add `stop_loss_locked` (BOOLEAN) - Prevents modifications (default TRUE)
- Add `stop_loss_override_reason` (TEXT) - Required if lock removed
- Add `stop_loss_last_modified` (TIMESTAMP) - Audit trail

**Rationale**: Separate immutable first entry stop from current stop_loss enables validation while preserving flexibility for edge cases.

### Risk Calculation (Python)

**Position Manager** (`ml-service/position_manager/manager.py`)

- Add `calculate_total_position_risk()` - Entry-by-entry risk aggregation
- Modify `check_buying_capacity()` - Use per-entry risk calculation instead of simplified formula
- Add stop-loss validation before capacity checks

**Formula Change**:

```
BEFORE:
  total_risk = total_shares * (first_entry_price - stop_loss)

AFTER:
  total_risk = sum(entry.shares * (entry.price - first_entry_stop_loss) for each entry)
```

**Example**: First entry 100 shares at 36,850, second entry 50 shares at 38,000, both with stop at 35,100:
- Correct: (100 × 1,750) + (50 × 2,900) = 320,000 VND
- Incorrect (current): 150 × average_price_delta = varies based on average

### Capacity Warning Tiers (Python)

**Position Sizing** (`ml-service/position_sizing/kelly.py`)

- Add three-tier capacity status: NORMAL, APPROACHING_LIMIT, AT_HARD_LIMIT
- Soft limits: 18% portfolio value, 0.8% daily volume
- Constrain recommendations when approaching limits

**Signal Generation** (`ml-service/signals/generator.py`)

- Use three-tier capacity status in signal logic
- Warn when approaching limits (allow constrained BUY_MORE)
- Block when at hard limits (override to HOLD)

### Stop-Loss Protection (Go + Python)

**Repository Validation** (`internal/db/repository/position_repository.go`)

- Validate stop_loss changes against `first_entry_stop_loss`
- Reject updates unless `stop_loss_locked = FALSE`
- Require `stop_loss_override_reason` if unlocked

**Daily Validation** (`ml-service/daily/daily_signals.py`)

- Pre-flight check: verify all stop_losses match first entry stops
- Alert on deviations exceeding 1% tolerance
- Block signal generation until resolved

### Configuration

**New Config File** (`ml-service/config.py`)

```python
STOP_LOSS_PERCENT = 0.0475  # 4.75% below entry
STOP_LOSS_TOLERANCE = 0.01  # 1% deviation allowed

POSITION_LIMITS = {
    'value_hard_limit': 0.20,
    'value_soft_limit': 0.18,
    'liquidity_hard_limit': 0.01,
    'liquidity_soft_limit': 0.008,
    'risk_percent_limit': 0.02
}

STOP_LOSS_POLICY = {
    'lock_at_first_entry': True,
    'allow_manual_override': False,
    'require_override_reason': True,
    'validate_daily': True
}
```

### Migration Strategy

1. Add new columns to positions table
2. Backfill `first_entry_stop_loss` from first entry in position_entries
3. Set `stop_loss_locked = TRUE` for all positions
4. Validate backfilled values match expected stop levels (within 1%)
5. Flag positions with deviations for manual review

## Impact

### Breaking Changes

> [!WARNING]
> **Stop-Loss Modification**: Existing code or manual processes that update `stop_loss` will be rejected unless `stop_loss_locked` is explicitly set to FALSE with a reason.

> [!WARNING]
> **Risk Calculation Change**: Capacity checks will use per-entry risk aggregation instead of simplified formula. This may reduce buying capacity for positions with multiple entries above average cost.

### Affected Components

**Database:**
- Migration: `add_stop_loss_protection_columns`
- Backfill script for existing positions

**Go Services:**
- `internal/db/repository/position_repository.go` - Update() validation
- `internal/db/repository/types.go` - New Position fields

**Python ML Service:**
- `ml-service/position_manager/manager.py` - Risk calculation, capacity warnings
- `ml-service/position_sizing/kelly.py` - Three-tier capacity status
- `ml-service/signals/generator.py` - Capacity-aware signals
- `ml-service/daily/daily_signals.py` - Stop-loss integrity checks
- `ml-service/config.py` - New configuration constants

**Tests:**
- Go: Stop-loss modification rejection
- Python: Multi-entry risk calculation accuracy
- Python: Soft/hard limit enforcement
- Python: Daily integrity validation

### Benefits

- ✅ Prevents accidental stop-loss widening that increases risk
- ✅ Accurate multi-entry position risk calculation
- ✅ Early warnings before hitting capacity limits
- ✅ Daily integrity checks catch manual modifications
- ✅ Audit trail for any stop-loss overrides
- ✅ Enforces 2% risk limit across multiple entries

### Risk Mitigation

- Backfill validation ensures data correctness before enforcement
- Stop-loss override mechanism available for edge cases
- Three-tier capacity (instead of binary) reduces false positives
- Gradual rollout: database → validation → enforcement
- Rollback: set `stop_loss_locked = FALSE` globally if needed
