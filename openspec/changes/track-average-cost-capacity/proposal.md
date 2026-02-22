# Change: Track Average Cost and Position Capacity

## Why

Currently, the system treats each stock purchase as a single position entry, with no support for tracking multiple purchases of the same stock at different prices. This creates several problems:

1. **Inaccurate P&L calculation** - When users buy more shares of a stock they already own (dollar-cost averaging), the system doesn't calculate weighted average cost, leading to incorrect profit/loss calculations
2. **No capacity management** - The system doesn't enforce position size limits (20% max per stock) or liquidity constraints (1% of daily volume), allowing users to accumulate oversized positions
3. **Missing transaction history** - Users can't see individual purchase transactions and their entry quality relative to the current average
4. **Signal generation issues** - BUY_MORE signals don't check if the position is already at capacity limits

This change implements comprehensive average cost tracking and position capacity management to enable proper multi-entry position handling.

## What Changes

### Database Schema

**New Table: `position_entries`**

- Tracks each individual purchase transaction
- Columns: `entry_id`, `ticker`, `entry_date`, `entry_price`, `shares_purchased`, `entry_fee_paid`, `transaction_type`
- Enables reconstruction of average cost and transaction history

**Modified Table: `positions`**

- Add aggregated tracking fields: `total_entries`, `total_fees_paid`, `first_entry_date`, `last_entry_date`
- Maintain `entry_price` as average cost (semantic shift from first entry to weighted average)
- All existing P&L calculations will use average cost instead of first entry price

### Position Management (Go + Python)

**Go Repository Layer** (`internal/db/repository/`)

- New methods for position entry CRUD operations
- Update position quantity calculations to use weighted averages
- Modify P&L calculations to include fee allocation

**Python Position Manager** (`ml-service/position_manager/`)

- `calculate_average_cost()` - Compute weighted average from all entries
- `update_position_after_buy()` - Recalculate averages on new purchases
- `check_buying_capacity()` - Validate against 20% allocation and 1% liquidity limits
- Support partial exits with proportional fee allocation

### Position Sizing (Python)

**Modified Equations** (`ml-service/position_sizing/`)

- Replace ideal sizing with capacity-constrained sizing
- Check limits before recommending BUY_MORE
- Return zero shares if position is at capacity

### Signal Generation (Python)

**Updated Logic** (`ml-service/signals/`)

- Check capacity before generating BUY_MORE signals
- Use average cost for stop-loss and target checks
- Include capacity information in signal recommendations

### Migration Strategy

- Backfill `position_entries` from existing `positions` records
- Verify average cost calculations match current positions
- Update all code references from entry_price to avg_cost semantics

## Impact

### Breaking Changes

> [!WARNING]  
> **Semantic Change**: The `entry_price` column in the `positions` table will change meaning from "first purchase price" to "weighted average cost". Existing code that relies on first entry price semantics must be updated.

> [!IMPORTANT]  
> **Stop Loss Recalculation**: Choose one approach:
>
> - **Option A** (Recommended): Keep stop-loss based on first entry price
> - **Option B**: Reset stop-loss based on new average cost after each purchase
>
> This decision affects risk management behavior and should be validated with user preference.

### Affected Components

**Database:**

- New migration: `add_position_entries_table`
- Modified table: `positions` (new columns, no data loss)

**Go Services:**

- `internal/db/repository/position_repository.go` - New entry methods, updated calculations
- `internal/service/position/position_service.go` - Average cost integration
- `internal/service/telegram/bot_service_positions.go` - Display entry history

**Python ML Service:**

- `ml-service/position_manager/manager.py` - Average cost and capacity checks
- `ml-service/position_sizing/kelly.py` - Capacity-constrained sizing
- `ml-service/signals/generator.py` - Position-aware signal logic
- `ml-service/daily/daily_signals.py` - Updated reporting with capacity info

### Benefits

- ✅ Accurate P&L tracking for multi-entry positions
- ✅ Automatic position size limit enforcement (20% max, 1% liquidity)
- ✅ Complete transaction history per stock
- ✅ Entry quality analysis (purchases above/below average)
- ✅ Prevents oversized positions via signal blocking
- ✅ Supports dollar-cost averaging strategy
- ✅ Fee-aware profit calculations (proportional allocation)

### Risk Mitigation

- Backfill existing positions as single entries (preserves data)
- Validation step confirms average cost matches stored values
- Gradual rollout: database → Python → Go → signals
- Rollback plan: revert migrations and code changes independently
