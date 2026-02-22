# Change: Implement T+2 Settlement Risk Management and Vietnamese Market Reality

## Why

The Vietnamese stock market operates on T+2 settlement, where shares purchased are not immediately sellable. The current system lacks proper tracking of settlement status, creating three critical gaps:

1. **T+2 Settlement Lock**: After purchasing shares, they are locked for 2 trading days before settlement, then require an additional day before they can be sold. Stop losses cannot be executed during this period, creating uncontrolled risk.

2. **Locked Capital Risk**: Shares in settlement (T+0 to T+2) cannot have stop losses executed. If the price hits floor during this period, the position becomes trapped with worst-case floor loss (7% for HOSE, 10% for HNX, 15% for UPCOM).

3. **Entry Day Risk**: Buying late in the week (Thursday/Friday) extends the locked risk period over the weekend, creating additional exposure.

Without proper settlement tracking, the system cannot differentiate between:
- Locked positions (where stop losses are theoretical only)
- Liquid positions (where stop losses are executable)

This leads to incorrect risk calculations and potential catastrophic losses during the settlement period.

## What Changes

### Database Schema

- Add settlement status tracking to positions table
- Add fields: `settlement_status`, `purchase_date`, `settlement_date`, `can_sell_date`, `locked_capital`, `liquid_capital`
- Create `position_settlement_tracking` table with daily settlement state snapshots
- Add settlement status enum: `LOCKED_T0`, `LOCKED_T1`, `LOCKED_T2`, `LIQUID`

### Risk Calculation Logic

- Implement two-tier risk calculation:
  - **Locked Risk**: For shares T+0 to T+2, calculate worst-case floor-hit risk (7%-15% depending on exchange)
  - **Liquid Risk**: For shares T+3+, calculate normal controlled risk (entry - stop)
- Add portfolio constraint: Total locked risk must not exceed 10% of account value
- Reject new purchases if they would push locked risk over 10% threshold

### Position Entry Rules

- Add day-of-week restrictions:
  - Monday-Wednesday: 100% position size allowed
  - Thursday-Friday: 50% position size allowed (compensate for weekend lock risk)
- Calculate settlement dates accounting for weekends and Vietnam holidays
- Track days until position becomes liquid

### Signal Generation Updates

- Before BUY_NEW or BUY_MORE signals, check locked risk budget
- Calculate locked risk for new purchase: `new_shares * price * floor_loss_pct`
- Reject signal if `current_locked_risk + new_locked_risk > 10% account_value`
- Generate message explaining rejection reason

### Stop Loss Execution Logic

- Check settlement status before attempting stop loss execution
- If `settlement_status` is LOCKED: Generate warning "Stop loss cannot be executed, shares in settlement until [date]"
- If LIQUID: Proceed with normal stop loss execution
- Track separately: theoretical stops vs executable stops

### Settlement Status Updates

- Implement daily cron job (16:30 after market close) to update settlement statuses
- Transition logic:
  - T+0 (Purchase Date): Status = `LOCKED_T0`
  - T+1: Status = `LOCKED_T1`
  - T+2: Status = `LOCKED_T2` (shares arrive after hours)
  - T+3: Status = `LIQUID` (can sell)
- Account for weekends and holidays in settlement date calculations

## Impact

**BREAKING**: Changes risk calculation logic and may reject signals that were previously accepted.

### Affected Specs

This change introduces NEW capabilities (no existing specs to modify):
- Settlement Risk Management (new)
- Locked Capital Tracking (new)
- Entry Day Restrictions (new)

### Affected Code

**Go Backend:**
- `internal/vn/settlement.go` - Already has T+2 calculation, extend with settlement status
- `internal/db/repository/position_repository.go` - Add settlement status queries
- `internal/db/repository/types.go` - Add settlement status fields to Position struct
- `internal/risk/position_sizing.go` - Add locked risk calculation
- `internal/service/position/position_service.go` - Add settlement status management
- `db/migrations/` - New migration for settlement tracking tables

**Python ML Service:**
- `ml-service/signals/generator.py` - Add locked risk budget check before BUY signals
- `ml-service/position_manager/manager.py` - Add settlement status queries
- `ml-service/position_sizing/` - Add locked risk calculation module
- `ml-service/daily/` - Add daily settlement status update script

### Migration Path

1. Add new database columns and tables (backward compatible - nullable fields)
2. Backfill existing positions with settlement status (all existing = LIQUID)
3. Deploy new risk calculation logic
4. Enable locked risk budget enforcement
5. Monitor for 1 week before enforcing Thursday/Friday restrictions

### Risk

- **High**: Risk calculation changes may be too conservative, reducing trading opportunities
- **Mitigation**: Make 10% locked risk threshold configurable, start at 15% and tighten after validation
- **Medium**: Settlement date calculation may be incorrect for holidays
- **Mitigation**: Comprehensive testing with 2024-2026 holiday calendar

### Success Metrics

After 3 months of operation:
- Zero stop loss failures due to settlement lock
- Locked risk never exceeded configured threshold (10% default)
- No positions entered Thursday/Friday unless manual override
- Track: number of signals rejected due to locked risk budget
