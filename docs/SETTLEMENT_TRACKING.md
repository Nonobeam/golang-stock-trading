# T+2 Settlement Risk Management Guide

## Overview

The T+2 Settlement Risk Management system tracks the settlement status of stock positions and prevents excessive exposure to locked capital risk. In the Vietnamese stock market, shares cannot be sold for 3 trading days after purchase (T+0, T+1, T+2), becoming sellable on T+3. During this settlement period, stop losses cannot be executed, creating uncontrolled risk exposure.

This guide explains how the system works, how to configure it, and how to use it effectively.

## Core Concepts

### Settlement Status Lifecycle

When you purchase shares, they go through the following settlement status progression:

```
Purchase Day (T+0) → LOCKED_T0
1 Trading Day Later (T+1) → LOCKED_T1
2 Trading Days Later (T+2) → LOCKED_T2
3 Trading Days Later (T+3) → LIQUID (can sell)
```

**Key Points:**
- Trading days exclude weekends and Vietnamese holidays
- Only LIQUID shares can be sold (including stop loss execution)
- LOCKED shares have uncontrolled downside risk

### Locked Risk Budget

Since stop losses cannot execute on LOCKED shares, each locked position carries worst-case "floor hit" risk. The system calculates and enforces a locked risk budget to prevent overexposure.

**Risk Calculation Formula:**
```
Locked Risk = Position Value × Exchange Risk Multiplier

Where Exchange Risk Multiplier:
- HOSE (Ho Chi Minh Stock Exchange): 20% (7% floor limit)
- HNX (Hanoi Stock Exchange): 30% (10% floor limit)
- UPCOM (Unlisted Public Company Market): 40% (15% floor limit)
```

**Example:**
```
Position: 100 shares of VNM at 50,000 VND/share
Exchange: HOSE
Position Value: 100 × 50,000 = 5,000,000 VND
Locked Risk: 5,000,000 × 0.20 = 1,000,000 VND
```

### Locked Risk Threshold

The system enforces a maximum percentage of your account value that can be held in locked risk.

**Default Threshold:** 10% of account value
**Configurable Range:** 5% - 20%

**Example:**
```
Account Value: 100,000,000 VND
Threshold: 10%
Max Locked Risk: 10,000,000 VND
```

If total locked risk across all positions exceeds this threshold, new BUY signals will be rejected until existing positions settle.

### Entry Day Restrictions

Purchasing shares on Thursday or Friday extends the settlement period over the weekend, increasing locked risk duration. The system automatically applies position size reductions:

**Entry Day Multipliers:**
- Monday - Wednesday: 100% (full position size)
- Thursday: 50% (half position size)
- Friday: 50% (half position size)

**Rationale:**
- Thursday purchase → Liquid on Tuesday (5 calendar days)
- Friday purchase → Liquid on Wednesday (5 calendar days)
- This 2x longer calendar duration increases risk exposure

## Database Schema

### Position Table Extensions

The `positions` table includes the following settlement tracking columns:

```sql
settlement_status   settlement_status  -- Current settlement status
purchase_date       TIMESTAMP          -- Date of purchase
settlement_date     TIMESTAMP          -- T+2 settlement date
can_sell_date       TIMESTAMP          -- T+3 sellable date
locked_capital      DECIMAL(15, 2)     -- Capital locked in settlement
liquid_capital      DECIMAL(15, 2)     -- Capital available to sell
exchange            VARCHAR(10)        -- HOSE, HNX, or UPCOM
```

### Settlement Tracking Table

The `position_settlement_tracking` table records daily snapshots:

```sql
CREATE TABLE position_settlement_tracking (
    id BIGSERIAL PRIMARY KEY,
    position_id BIGINT NOT NULL REFERENCES positions(id),
    snapshot_date DATE NOT NULL,
    settlement_status settlement_status NOT NULL,
    locked_capital DECIMAL(15, 2),
    liquid_capital DECIMAL(15, 2),
    days_until_liquid INT
);
```

### Theoretical Stop Breaches

The `theoretical_stop_breaches` table tracks stop losses that were triggered but couldn't execute:

```sql
CREATE TABLE theoretical_stop_breaches (
    id BIGSERIAL PRIMARY KEY,
    position_id BIGINT NOT NULL REFERENCES positions(id),
    breach_date TIMESTAMP NOT NULL,
    stop_price DECIMAL(15, 2) NOT NULL,
    actual_price DECIMAL(15, 2) NOT NULL,
    settlement_status settlement_status NOT NULL,
    days_until_executable INT NOT NULL
);
```

## Configuration

### User Configuration

Add the locked risk threshold to the `user_config` table:

```sql
ALTER TABLE user_config
ADD COLUMN locked_risk_threshold DECIMAL(4, 3) DEFAULT 0.10
CHECK (locked_risk_threshold BETWEEN 0.05 AND 0.20);
```

**Setting Your Threshold:**
- Conservative: 5% (only 1-2 locked positions at a time)
- Moderate: 10% (default, balanced approach)
- Aggressive: 15-20% (higher risk tolerance)

**Example:**
```sql
UPDATE user_config
SET locked_risk_threshold = 0.15
WHERE user_id = 1;
```

### System Configuration

The settlement calculator uses Vietnamese trading calendar:

**Holidays Included (2024-2026):**
- New Year's Day (Jan 1)
- Tết (Lunar New Year) - 5-7 days
- Hung Kings' Day (10th day of 3rd lunar month)
- Reunification Day (Apr 30)
- International Workers' Day (May 1)
- National Day (Sep 2)

## API Reference

### Go APIs

#### Settlement Status Calculation

```go
import "github.com/nonobeam/golang-stock-trading/internal/vn"

// Calculate settlement status from dates
status := vn.CalculateSettlementStatusFromDates(purchaseDate, currentDate)

// Check if position is locked
isLocked := position.IsLocked()

// Check if position is liquid
isLiquid := position.IsLiquid()

// Get days until liquid
days := vn.GetDaysUntilLiquid(purchaseDate, currentDate)
```

#### Locked Risk Calculation

```go
import "github.com/nonobeam/golang-stock-trading/internal/risk"

// Calculate locked risk for a position
lockedRisk := risk.CalculateLockedRisk(shares, price, exchange)

// Check if user can afford locked risk
calc := risk.NewLockedRiskCalculator(db)
canAfford, message, err := calc.CanAffordLockedRisk(
    ctx, userID, ticker, shares, price, accountValue, userConfig,
)
```

#### Signal Validation

```go
import "github.com/nonobeam/golang-stock-trading/internal/signals"

validator := signals.NewSettlementValidator(db)
result, err := validator.ValidateSignal(
    ctx, userID, ticker, positionSize, entryPrice, accountValue, userConfig,
)

if !result.Approved {
    // Signal rejected due to locked risk
    log.Warn().Str("reason", result.Message).Msg("Signal rejected")
}
```

#### Stop Loss Validation

```go
import "github.com/nonobeam/golang-stock-trading/internal/service/position"

validator := position.NewStopLossValidator(db)
result, err := validator.CanExecuteStopLoss(ctx, position)

if !result.CanExecute {
    // Stop loss cannot execute (shares locked)
    log.Warn().Str("reason", result.Reason).Msg("Stop loss blocked")
}
```

### Python APIs

#### Locked Risk Calculator

```python
from position_sizing.locked_risk import LockedRiskCalculator

calc = LockedRiskCalculator(db_connection)

# Calculate locked risk
locked_risk = calc.calculate_locked_risk(shares=100, price=50000, exchange='HOSE')

# Check budget
approved, message = calc.check_locked_risk_budget(
    user_id=1,
    ticker='VNM',
    shares=100,
    price=50000,
    account_value=100_000_000,
    threshold=0.10
)

if not approved:
    print(f"Purchase rejected: {message}")
```

#### Signal Validation Integration

```python
from signals.enhanced_generator import EnhancedSignalGenerator

generator = EnhancedSignalGenerator(db_connection)

# Generate signals with settlement risk validation
signal, confidence, reason, metadata = generator.generate_signal_for_ticker(
    ticker='VNM',
    user_id=1,
    account_value=100_000_000,
    current_price=50000
)

# Check if rejected due to settlement risk
if "T+2 Settlement Risk" in reason:
    print(f"Signal blocked: {reason}")
```

## Daily Operations

### Settlement Status Update Job

Run daily to update all position settlement statuses:

```bash
# Go implementation
./cmd/tools/settlement-update/main.go

# Or via service
internal/jobs/settlement_update.go
```

**What it does:**
1. Queries all open positions
2. Calculates current settlement status based on purchase_date
3. Updates `locked_capital` and `liquid_capital`
4. Records daily snapshot in `position_settlement_tracking`
5. Detects theoretical stop breaches

**Recommended Schedule:** Run daily at market close (15:00 Vietnam time)

### Monitoring Reports

Generate daily settlement monitoring reports:

```bash
python ml-service/monitoring/settlement_monitor.py
```

**Report Includes:**
- Settlement status distribution (LOCKED_T0/T1/T2 vs LIQUID)
- Locked risk utilization by user
- Theoretical stop breach summary (last 7 days)
- Settlement transition validation

## Telegram Bot Commands

### Position Status Command

```
/position VNM
```

**Output:**
```
📊 Position: VNM

Entry: 50,000 VND
Current: 52,000 VND (+4.00%)
Quantity: 100 shares

Settlement Status: LOCKED_T1
Days Until Liquid: 2

Capital Breakdown:
• Locked: 5,000,000 VND (cannot sell)
• Liquid: 0 VND

⚠️ Stop Loss: 48,000 VND
Cannot execute stop loss - shares in settlement

Purchase: 2026-02-10
Can Sell From: 2026-02-14
```

### Settlement Alerts

The bot automatically sends alerts for:

1. **Position Becomes Liquid:**
```
✅ Position Now Liquid

Symbol: VNM
Quantity: 100 shares
Value: 5,200,000 VND

Your shares are now sellable.
Stop loss protection is active.
```

2. **Locked Risk Threshold Alert:**
```
⚠️ Locked Risk Alert

Your locked capital risk is approaching the threshold:
• Current: 9,500,000 VND
• Threshold: 10,000,000 VND
• Usage: 95.0%

New purchases may be restricted until existing positions settle.
```

3. **Theoretical Stop Breach:**
```
🚨 Stop Loss Breach (Non-Executable)

Symbol: VNM
Stop Loss: 48,000 VND
Current Price: 47,500 VND

⚠️ Cannot execute: Shares in settlement (LOCKED_T1)
Days until executable: 2
Can sell from: 2026-02-14

⚠️ Price has hit your stop loss, but shares are locked in settlement period.
Monitor the price closely. If it recovers before settlement, you may avoid the loss.
```

4. **Entry Day Warning:**
```
⚠️ Entry Day Warning

Symbol: VNM
Entry Day: Thursday

⚡ Position size recommendation: 50%

Reason: Thursday entries extend settlement period over the weekend,
increasing the locked risk duration. Consider reducing position size to manage risk.
```

## Data Migration

### Backfilling Existing Positions

For existing positions without settlement data, use the migration script:

```bash
# Dry run (shows what would be done)
python scripts/migrate_settlement_data.py --dry-run

# Execute migration
python scripts/migrate_settlement_data.py

# Validate migration
python scripts/migrate_settlement_data.py --validate-only
```

**Migration Strategy:**
- All existing open positions → `LIQUID` status
- `locked_capital` → 0
- `liquid_capital` → `quantity × entry_price`
- `settlement_date` → `entry_date + 3 days` (conservative)
- `can_sell_date` → `entry_date + 4 days`
- `exchange` → Inferred from ticker (defaults to HOSE)

## Testing

### Unit Tests

#### Go Tests
```go
// Test settlement status calculation
func TestCalculateSettlementStatus(t *testing.T) {
    purchaseDate := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)

    // T+0
    currentT0 := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
    assert.Equal(t, "LOCKED_T0", vn.CalculateSettlementStatusFromDates(purchaseDate, currentT0))

    // T+3 (liquid)
    currentT3 := time.Date(2026, 2, 13, 0, 0, 0, 0, time.UTC)
    assert.Equal(t, "LIQUID", vn.CalculateSettlementStatusFromDates(purchaseDate, currentT3))
}
```

#### Python Tests
```python
# Run settlement tracking tests
pytest ml-service/tests/test_settlement_tracking.py -v
```

**Test Coverage:**
- Settlement status calculations (T+0 through LIQUID)
- Locked risk calculations (all exchanges)
- Entry day restrictions (Thu/Fri half-size)
- Vietnamese holiday handling
- Weekend skip logic
- Budget validation

### Integration Tests

Test complete signal validation flow:

```python
# Test signal rejection due to locked risk budget
def test_signal_rejected_locked_risk_exceeded():
    # Create locked positions exceeding threshold
    # Generate new BUY signal
    # Assert signal rejected with proper message
```

## Best Practices

### Risk Management

1. **Start Conservative:** Use 10% threshold initially, adjust based on experience
2. **Monitor Utilization:** Keep locked risk below 80% of threshold
3. **Avoid Thursday/Friday Entries:** Unless conviction is very high
4. **Diversify Entry Days:** Spread purchases across week to stagger settlement

### Position Sizing

1. **Account for Locked Risk:** Signal generator automatically reduces size if needed
2. **Entry Day Multiplier:** System applies 50% reduction on Thu/Fri automatically
3. **Exchange Risk:** Higher risk on HNX/UPCOM → smaller positions

### Monitoring

1. **Daily Reports:** Review settlement monitor output daily
2. **Telegram Alerts:** Enable notifications for threshold warnings
3. **Theoretical Stops:** Monitor breaches to understand execution lag risk
4. **Transition Tracking:** Watch positions moving LOCKED → LIQUID

### Edge Cases

1. **Holidays:** System accounts for Vietnamese holidays automatically
2. **Weekend Purchases:** Not possible (market closed), but system handles edge cases
3. **Position Closing:** Closing position resets settlement tracking
4. **Partial Exits:** Not yet supported (TODO: track per-entry settlement)

## Troubleshooting

### Common Issues

**Issue:** Signal rejected with "Locked risk budget exceeded"
- **Solution:** Wait for existing positions to become LIQUID, or increase threshold

**Issue:** Stop loss not executing
- **Solution:** Check position settlement status - may be LOCKED

**Issue:** Settlement status not updating
- **Solution:** Ensure daily settlement update job is running

**Issue:** Incorrect exchange inferred
- **Solution:** Manually set exchange in position record, enhance ticker-to-exchange mapping

### Validation Queries

```sql
-- Check locked risk utilization
SELECT
    user_id,
    SUM(
        CASE
            WHEN exchange = 'HOSE' THEN locked_capital * 0.20
            WHEN exchange = 'HNX' THEN locked_capital * 0.30
            WHEN exchange = 'UPCOM' THEN locked_capital * 0.40
            ELSE locked_capital * 0.20
        END
    ) as total_locked_risk
FROM positions
WHERE is_closed = FALSE
  AND settlement_status IN ('LOCKED_T0', 'LOCKED_T1', 'LOCKED_T2')
GROUP BY user_id;

-- Check positions stuck in locked status
SELECT id, symbol, settlement_status, purchase_date, can_sell_date
FROM positions
WHERE is_closed = FALSE
  AND settlement_status IN ('LOCKED_T0', 'LOCKED_T1', 'LOCKED_T2')
  AND can_sell_date < NOW();
```

## Future Enhancements

### Planned Features

1. **Per-Entry Tracking:** Track settlement status for each purchase separately (partial exits)
2. **Dynamic Thresholds:** Adjust threshold based on market volatility
3. **Advanced Alerts:** Predict when locked risk will free up
4. **Portfolio Optimization:** Suggest optimal entry days to maintain liquidity
5. **Exchange Auto-Detection:** Real-time ticker-to-exchange mapping via market data API

### Known Limitations

1. Exchange inference defaults to HOSE (needs production enhancement)
2. No partial exit tracking (all-or-nothing position closure)
3. Holiday calendar requires manual updates for 2027+
4. No intraday settlement status updates (daily batch only)

## References

### Related Documentation
- `openspec/changes/implement-t2-settlement-risk/proposal.md` - Original proposal
- `PROJECT_STRUCTURE.md` - System architecture
- `db/migrations/000014_add_settlement_tracking.up.sql` - Database schema

### Code Locations
- **Go Settlement Logic:** `internal/vn/settlement.go`
- **Go Risk Calculation:** `internal/risk/locked_risk.go`
- **Go Signal Validation:** `internal/signals/settlement_validator.go`
- **Python Risk Calculator:** `ml-service/position_sizing/locked_risk.py`
- **Python Integration:** `ml-service/signals/enhanced_generator.py`
- **Telegram Alerts:** `internal/service/telegram/settlement_alerts.go`
- **Daily Job:** `internal/jobs/settlement_update.go`
- **Monitoring:** `ml-service/monitoring/settlement_monitor.py`

---

**Version:** 1.0
**Last Updated:** 2026-02-06
**Status:** Production Ready
