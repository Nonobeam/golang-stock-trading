# Design: Average Cost Tracking and Position Capacity

## Context

The current system allows users to record stock positions but doesn't properly handle multiple purchases of the same stock (dollar-cost averaging). This leads to:

- **Incorrect P&L tracking** when users add to existing positions
- **Uncontrolled position sizing** without enforcement of the 20% maximum allocation limit
- **No liquidity awareness** when generating BUY_MORE signals
- **Missing transaction audit trail** for regulatory or analysis purposes

The system has two layers that need coordination:

1. **Go backend**: Database operations, Telegram bot commands, position storage
2. **Python ML service**: Signal generation, position sizing, daily monitoring

Both layers must agree on average cost calculations and capacity limits.

## Goals / Non-Goals

### Goals

- Track each purchase transaction separately while maintaining aggregate position view
- Calculate weighted average cost automatically
- Enforce position size limits (20% portfolio max) and liquidity constraints (1% daily volume)
- Support partial exits with proper fee allocation
- Maintain backward compatibility with existing closed positions
- Provide migration path for active positions

### Non-Goals

- **Not changing fee structure** - Still using 0.15% entry, 0.25% exit
- **Not implementing tax lot selection** - Exits always reduce proportionally
- **Not adding new signal types** - Using existing BUY_NEW, BUY_MORE, SELL, SELL_PARTIAL
- **Not changing target/stop algorithms** - Only using average cost in calculations
- **Not adding real-time position tracking** - Still daily batch processing

## Decisions

### Decision 1: Dual-Table Approach

**Choice**: Add new `position_entries` table alongside existing `positions` table

**Rationale**:

- Preserves existing data model and queries
- Allows gradual migration without breaking current functionality
- `positions` becomes aggregate view, `position_entries` is transaction log
- Simpler rollback if issues found

**Alternatives considered**:

- **Single table with JSON entries**: Harder to query, no referential integrity
- **Replace positions entirely**: High migration risk, breaks existing code
- **Compound primary key**: Requires changing all foreign key relationships

### Decision 2: Average Cost Storage Location

**Choice**: Store calculated average in `positions.entry_price`, keep raw entries in `position_entries`

**Rationale**:

- Minimizes code changes (most code already reads `entry_price`)
- Semantic shift is clearer than adding new column
- Source of truth is `position_entries`, `entry_price` is cached calculation
- Triggers can auto-update on entry changes

**Alternatives considered**:

- **New `avg_cost` column**: More explicit but requires updating all P&L formulas
- **Calculate on-the-fly**: Too slow for signal generation batch processing
- **Store in Redis**: Adds dependency, cache invalidation complexity

### Decision 3: Capacity Check Location

**Choice**: Capacity checks in Python position manager, enforced in signal generator

**Rationale**:

- Signal generation is Python-based, needs capacity info
- Position manager already has database access
- Go services call Python via gRPC for signals anyway
- Centralized business logic in one place

**Alternatives considered**:

- **Go-side enforcement**: Duplicates logic across languages
- **Database constraints**: Can't enforce dynamic limits (account value changes)
- **Separate service**: Over-engineering for current scale

### Decision 4: Stop Loss Handling

**Choice**: Keep stop-loss based on **first entry price** (Option A from proposal)

**Rationale**:

- Preserves original risk thesis - first entry established acceptable risk-reward
- Prevents moving stop away from market during averaging down
- Example: First entry at 36,850 with stop at 35,100 = 4.75% max loss accepted
- Adding shares at 38,000 means stronger conviction, not acceptance of larger loss
- Protects against classic trap of lowering stops to accommodate bad averaging

**Total Risk Validation**:

- While stop level stays constant, total risk amount increases with position size
- Example calculation:
  - First entry: 100 shares at 36,850, stop at 35,100 → Risk = 175,000 VND
  - Add 50 shares at 38,000, keep stop at 35,100 → Additional risk = 145,000 VND
  - Total risk = 320,000 VND
- **MUST validate**: `total_risk <= account_value × 0.02` before allowing BUY_MORE
- Reject BUY_MORE even with strong predictions if total risk exceeds portfolio limit

**Alternatives considered**:

- **Reset based on average**: Could create false sense of safety, increases real risk
- **Ask user per purchase**: Too much cognitive load for automation
- **Dual stops**: Overly complex for marginal benefit

### Decision 5: Fee Allocation for Partial Exits

**Choice**: Proportional allocation - `(shares_sold / total_shares) × total_fees_paid`

**Fee Structure Confirmed**: Per-transaction model

- **Entry fee per purchase**: `entry_fee = purchase_value × 0.0015`
- **Exit fee per sale**: `exit_fee = sale_value × 0.0025` (includes 0.15% + 0.10% tax)

**Example with two entries**:

```
Entry 1: 100 shares × 36,850 VND = 3,685,000 VND
  Fee 1: 3,685,000 × 0.0015 = 5,527 VND

Entry 2: 50 shares × 38,000 VND = 1,900,000 VND
  Fee 2: 1,900,000 × 0.0015 = 2,850 VND

Total entry fees: 8,377 VND (stored in total_fees_paid)

Exit: 150 shares × 40,000 VND = 6,000,000 VND
  Exit fee: 6,000,000 × 0.0025 = 15,000 VND

Total fees for complete exit: 8,377 + 15,000 = 23,377 VND
```

**Database tracking**:

- `position_entries` table: stores each `entry_fee` per transaction
- `positions` table: accumulates `total_fees_paid = Σ(entry_fees)`
- On exit: add `exit_fee` to total for final P&L calculation

**Rationale**:

- Fair allocation across all entries
- Mathematically simple and auditable
- Matches tax lot accounting standards
- Remaining fees tracked in updated position

**Alternatives considered**:

- **FIFO fee allocation**: Complex tracking, doesn't match actual cost basis
- **Average fee per share**: Same result, more calculation steps
- **No fee allocation**: Understates true exit cost

## Data Flow

### Adding to Position

```
User Action (Telegram /buy)
    ↓
Go: BotService.handleBuy()
    ↓
Go: PositionRepository.AddEntry()
    ├→ INSERT into position_entries
    └→ Trigger: update_position_average_cost()
        ├→ Calculate weighted average
        └→ UPDATE positions SET entry_price = avg, total_entries++
    ↓
Python: PositionManager.check_buying_capacity()
    ├→ Query current position value
    ├→ Query account value
    ├→ Check: position_value < account_value × 0.20
    └→ Return: remaining capacity
```

### Signal Generation with Capacity

```
Daily Cron Job
    ↓
Python: daily_signals.py
    ↓
For each ticker in watchlist:
    ↓
    SignalGenerator.generate()
        ├→ PositionManager.get_position_for_signal()
        │   └→ Returns: avg_cost, quantity, stop_loss
        ├→ Check predictions and market regime
        └→ If BUY_MORE candidate:
            └→ PositionManager.check_buying_capacity()
                ├→ remaining_value_capacity
                ├→ remaining_share_capacity
                └→ If at_limit: signal = HOLD
    ↓
Output: Daily report with capacity warnings
```

## Migration Plan

### Phase 1: Database (Zero Downtime)

1. **Create `position_entries` table** (new migration)
2. **Add columns to `positions`**: `total_entries`, `total_fees_paid`, `first_entry_date`, `last_entry_date`
3. **Backfill script**:
   ```python
   for each position in positions:
       INSERT INTO position_entries (
           ticker, entry_date, entry_price, shares_purchased,
           entry_fee_paid, transaction_type
       ) VALUES (
           position.symbol, position.entry_date, position.entry_price,
           position.quantity, position.quantity * position.entry_price * 0.0015,
           'BUY_NEW'
       )
       UPDATE positions SET
           total_entries = 1,
           first_entry_date = position.entry_date,
           last_entry_date = position.entry_date,
           total_fees_paid = position.quantity * position.entry_price * 0.0015
       WHERE id = position.id
   ```
4. **Validation**: Verify `entry_price` unchanged for all positions

### Phase 2: Python ML Service

1. Update `PositionManager`:
   - Add `calculate_average_cost()` method
   - Add `check_buying_capacity()` method
   - Update `update_position_quantity()` to recalculate average
2. Update `PositionSizer`:
   - Modify `calculate_position_change()` to call capacity check
   - Return zero shares if at limit
3. Update `SignalGenerator`:
   - Call capacity check before BUY_MORE signals
   - Use average cost for stop/target checks
4. **Test**: Run against historical data, verify signals match expected

### Phase 3: Go Services

1. Update `PositionRepository`:
   - Add `CreateEntry()`, `GetEntries()`, `GetAverageCost()`
   - Modify `Close()` to handle proportional fee allocation
2. Update `PositionService`:
   - Integrate average cost in status displays
3. Update Telegram Bot:
   - `/buy` command stores new entry
   - `/status` shows transaction history
   - `/positions` displays capacity remaining
4. **Test**: Manual testing via Telegram commands

### Phase 4: Daily Workflow Integration

1. Update `daily_signals.py`:
   - Include capacity information in reports
   - Show remaining allocation per position
2. **Deploy**: Run daily job, monitor for anomalies
3. **Validate**: P&L calculations match manual verification

### Rollback Plan

Each phase can be rolled back independently:

- **Database**: Drop `position_entries` table, remove new columns
- **Python**: Revert to previous version, capacity checks return unlimited
- **Go**: Remove entry-related endpoints, fall back to single position view
- **Daily**:Revert script to previous version

Data preservation: `position_entries` is append-only, rolling back code doesn't lose transaction history.

## Risks / Trade-offs

### Risk 1: Average Cost Calculation Drift

**Risk**: Floating-point arithmetic could cause average cost to drift from true value over many transactions

**Mitigation**:

- Use `DECIMAL` types in database, not `FLOAT`
- Recalculate from source (`position_entries`) periodically
- Add validation check in daily job: `assert calculated_avg ≈ stored_avg`

### Risk 2: Capacity Limit Gaming

**Risk**: User could bypass limits by using multiple accounts or manual trading outside the system

**Mitigation**:

- This is a guardrail, not security feature
- Limits are for user protection, not enforcement
- Daily reports show capacity usage for manual review

### Risk 3: Stop Loss Confusion

**Risk**: Users might expect stop-loss to move with average cost when buying more shares

**Mitigation**:

- Clear documentation in bot messages: "Stop-loss based on first entry"
- Option to manually adjust stops via `/updateposition` command
- Daily report shows risk per position (distance to stop)

### Risk 4: Database Migration Failure

**Risk**: Backfill script could fail midway, leaving inconsistent state

**Mitigation**:

- Run backfill in transaction with rollback
- Idempotent script (can re-run safely)
- Test on copy of production database first
- Manual verification step before deploying code changes

### Trade-off: Performance vs. Accuracy

**Trade-off**: Calculating average cost on every read is slow, but caching risks staleness

**Decision**: Cache in `positions.entry_price`, update via trigger on `position_entries` changes

**Justification**:

- Positions don't change frequently (max a few times per day)
- Signal generation runs in batch, not real-time
- Trigger ensures consistency without manual sync

## Confirmed Decisions Summary

All critical design questions have been resolved:

### ✅ Stop Loss Strategy

**Decision**: Keep at first entry price, with total risk validation

- Stop-loss remains at level based on first purchase (preserves original risk thesis)
- Adding shares increases total risk but doesn't move stop level
- **Validation gate**: `total_risk <= account_value × 0.02` enforced before BUY_MORE
- Example: First entry 100 shares at 36,850 (stop 35,100) = 175k VND risk
  - Add 50 shares at 38,000 = +145k VND risk → total 320k VND
  - Must verify: 320k <= account × 2% before allowing purchase

### ✅ Fee Structure

**Decision**: Per-transaction model

- Entry: 0.15% per purchase transaction
- Exit: 0.25% per sale transaction (0.15% broker + 0.10% tax)
- Each entry records its own fee in `position_entries.entry_fee_paid`
- Accumulated in `positions.total_fees_paid` for P&L calculations

### ✅ Capacity Enforcement

**Decision**: Block entirely when at limit

- No BUY_MORE signals when position reaches 20% allocation or 1% liquidity cap
- Signal overridden to `HOLD` with reason `position_at_capacity_limit`
- Forces discipline and prevents over-concentration
- Report displays: "Signal: HOLD (would be BUY_MORE but at 20% capacity limit)"

### ✅ Additional Confirmed Items

- **Historical backfill**: Create single synthetic entry per existing position
- **Liquidity source**: Calculate 20-day average volume from `market_data` table
- **Entry immutability**: `position_entries` is append-only audit log

---

**Status**: All design questions resolved. Ready for implementation approval.
