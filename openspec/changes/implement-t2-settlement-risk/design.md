# Design: T+2 Settlement Risk Management

## Context

The Vietnamese stock market operates on T+2 settlement, where purchased shares are not immediately sellable. The system currently lacks proper tracking of settlement status, leading to three critical issues:

1. **Uncontrolled Risk During Settlement**: Shares purchased cannot have stop losses executed for 2-3 days, creating a period of uncontrolled risk.

2. **Incorrect Risk Calculations**: The system treats all positions as having executable stop losses, when in reality locked positions have worst-case floor-hit risk.

3. **Over-Leverage Risk**: Without tracking locked capital, the system may allow excessive purchases that trap too much capital in unsellable positions.

### Stakeholders

- **Primary Users**: Vietnamese retail traders using the system for automated trading
- **Risk Managers**: Need visibility into locked vs liquid capital
- **Developers**: Go backend team, Python ML service team

### Constraints

- Must maintain backward compatibility with existing positions (migration required)
- Settlement date calculation must account for Vietnamese holidays (lunar calendar)
- Daily settlement update must run reliably at market close (16:30)
- Cannot modify shares during settlement period (exchange restriction)

## Goals / Non-Goals

### Goals

1. Track settlement status for every position (LOCKED_T0/T1/T2 vs LIQUID)
2. Calculate locked risk using worst-case floor-hit scenarios
3. Enforce locked risk budget to prevent over-leverage during settlement
4. Validate settlement status before executing stop losses
5. Apply entry day restrictions to reduce weekend lock risk
6. Provide visibility into settlement timeline for each position

### Non-Goals

- Simulating limit order fills during settlement period (separate feature)
- Automated execution of stop losses (remains manual)
- Integration with brokerage APIs for settlement verification
- Real-time settlement status tracking (daily update sufficient)

## Decisions

### Decision 1: Settlement Status State Machine

**Choice**: Use explicit enum states (LOCKED_T0, LOCKED_T1, LOCKED_T2, LIQUID) rather than calculating on-the-fly.

**Rationale**:
- Explicit states make debugging easier
- Daily snapshot provides audit trail
- Faster queries (no date calculation at query time)
- Easier to add alerts/notifications on state transitions

**Alternatives Considered**:
- Calculate status on-the-fly from purchase_date: Would require complex date logic in every query, harder to debug.
- Binary locked/liquid flag: Less granular, harder to track progression toward liquid state.

**Trade-off**: Requires daily cron job to update statuses, but provides better observability.

### Decision 2: Locked Risk Calculation Formula

**Choice**: Use exchange-specific percentages (HOSE: 20%, HNX: 30%, UPCOM: 40%) based on floor limit plus margin of safety.

**Rationale**:
- HOSE has 7% floor limit → 20% accounts for floor hit plus slippage/fees
- HNX has 10% floor limit → 30% accounts for floor hit plus slippage/fees
- UPCOM has 15% floor limit → 40% accounts for floor hit plus slippage/fees
- Conservative approach prevents catastrophic loss scenarios

**Alternatives Considered**:
- Use exact floor percentage (7%, 10%, 15%): Too optimistic, doesn't account for slippage/fees/continued decline.
- Use uniform 25% across all exchanges: Oversimplifies, doesn't reflect actual market structure.
- Calculate from historical worst-case scenarios: Too complex, requires significant backtesting.

**Trade-off**: May be overly conservative, but safety is paramount during uncontrollable risk period.

### Decision 3: Locked Risk Budget Threshold

**Choice**: Default to 10% of account value, configurable from 5% to 20%.

**Rationale**:
- 10% is conservative enough to prevent catastrophic loss
- If all locked positions hit floor, maximum loss is ~2-4% of account (10% locked * 20-40% floor risk)
- Configurable to allow users to tune based on risk tolerance
- Upper limit of 20% prevents excessive leverage

**Alternatives Considered**:
- Fixed 15% threshold: Less flexible, doesn't accommodate different risk profiles.
- No threshold: Dangerous, could lead to 100% of capital locked in settlement.
- Dynamic threshold based on market volatility: Too complex for initial implementation.

**Trade-off**: May limit trading opportunities during high conviction periods, but protects from settlement lock disasters.

### Decision 4: Entry Day Restrictions

**Choice**: Reduce position size to 50% for Thursday/Friday entries.

**Rationale**:
- Thursday purchase: locked through Friday + weekend + Monday/Tuesday = 4-5 days
- Friday purchase: locked through weekend + Monday/Tuesday/Wednesday = 5-6 days
- Longer lock period = more uncertainty = smaller position
- Still allows trading, just with appropriate risk adjustment

**Alternatives Considered**:
- Block Thursday/Friday entries entirely: Too restrictive, eliminates legitimate opportunities.
- No restriction: Ignores extended weekend lock risk.
- Dynamic multiplier based on market conditions: Too complex, hard to explain to users.

**Trade-off**: May miss some Friday breakouts, but reduces weekend risk exposure.

### Decision 5: Daily Settlement Update Timing

**Choice**: Run settlement update at 16:30 (after market close) via cron job.

**Rationale**:
- Market closes at 15:00, settlement status shouldn't change until next trading day
- 16:30 provides buffer for end-of-day processing
- Aligns with typical end-of-day workflow
- Vietnamese market: no after-hours trading to complicate status

**Alternatives Considered**:
- Run at midnight: Misses same-day transition for positions that became liquid.
- Real-time updates on every trade: Overkill, settlement status only changes once per day.
- Manual updates: Unreliable, prone to human error.

**Trade-off**: Requires reliable cron infrastructure, but provides consistent daily updates.

### Decision 6: Stop Loss Execution Validation

**Choice**: Block execution for locked positions, log as theoretical breach.

**Rationale**:
- Attempting to sell locked shares will fail at brokerage level
- Recording theoretical breaches provides performance analytics
- Allows post-settlement analysis: "if we could have sold, what would have happened?"
- Prevents misleading "stop loss worked" metrics

**Alternatives Considered**:
- Allow execution attempt (fail at brokerage): Wastes API calls, creates noise.
- Don't track theoretical breaches: Loses valuable performance data.
- Auto-execute when position becomes liquid: Too risky if price has recovered.

**Trade-off**: Requires additional tracking table, but provides crucial performance insights.

## Data Model

### Positions Table Extensions

```sql
ALTER TABLE positions ADD COLUMN settlement_status VARCHAR(20) CHECK (settlement_status IN ('LOCKED_T0', 'LOCKED_T1', 'LOCKED_T2', 'LIQUID'));
ALTER TABLE positions ADD COLUMN purchase_date TIMESTAMP;
ALTER TABLE positions ADD COLUMN settlement_date TIMESTAMP;
ALTER TABLE positions ADD COLUMN can_sell_date TIMESTAMP;
ALTER TABLE positions ADD COLUMN locked_capital DECIMAL(15,2);
ALTER TABLE positions ADD COLUMN liquid_capital DECIMAL(15,2);
ALTER TABLE positions ADD COLUMN exchange VARCHAR(10) CHECK (exchange IN ('HOSE', 'HNX', 'UPCOM'));
```

### Position Settlement Tracking Table

```sql
CREATE TABLE position_settlement_tracking (
    tracking_id UUID PRIMARY KEY,
    position_id UUID NOT NULL REFERENCES positions(id),
    check_date DATE NOT NULL,
    settlement_status VARCHAR(20),
    days_until_liquid INTEGER,
    locked_value DECIMAL(15,2),
    locked_risk DECIMAL(15,2),
    risk_classification VARCHAR(30) CHECK (risk_classification IN ('HIGH_RISK_LOCKED', 'MODERATE_RISK_NEAR_LIQUID', 'LOW_RISK_LIQUID')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_settlement_tracking_position ON position_settlement_tracking(position_id, check_date DESC);
```

### Theoretical Stop Breaches Table

```sql
CREATE TABLE theoretical_stop_breaches (
    breach_id UUID PRIMARY KEY,
    position_id UUID NOT NULL REFERENCES positions(id),
    breach_date TIMESTAMP NOT NULL,
    stop_price DECIMAL(15,2),
    actual_price DECIMAL(15,2),
    settlement_status VARCHAR(20),
    days_until_executable INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Workflows

### Workflow 1: New Position Entry

1. User receives BUY_NEW signal from ML service
2. ML service calculates:
   - Proposed position size
   - Entry day multiplier (1.0 Mon-Wed, 0.5 Thu-Fri)
   - Adjusted position size
   - Locked risk for proposed purchase: `shares * price * exchange_multiplier`
3. ML service queries current locked risk from database
4. ML service checks: `current_locked + new_locked <= account_value * locked_risk_threshold`
5. If budget exceeded: Reject signal with reason "Locked risk budget exceeded"
6. If budget OK: Issue BUY signal with adjusted position size
7. Upon execution, Go backend creates position with:
   - settlement_status = LOCKED_T0
   - purchase_date = current_date
   - settlement_date = purchase_date + 2 trading days
   - can_sell_date = settlement_date + 1 trading day
   - exchange = inferred from ticker
   - locked_capital = shares * entry_price
   - liquid_capital = 0

### Workflow 2: Daily Settlement Status Update

1. Cron job triggers at 16:30
2. Go backend queries all active positions
3. For each position:
   - Calculate current settlement status based on purchase_date and current_date
   - Update settlement_status in positions table
   - Update locked_capital and liquid_capital
   - Insert snapshot into position_settlement_tracking
4. Detect positions transitioning to LIQUID
5. Send Telegram notification: "Position [ticker] now liquid, stop loss executable"
6. Log summary: X positions transitioned, Y still locked

### Workflow 3: Stop Loss Breach Handling

1. Price monitoring detects: current_price <= stop_loss
2. Go backend queries position settlement_status
3. If settlement_status in (LOCKED_T0, LOCKED_T1, LOCKED_T2):
   - Do NOT execute stop loss order
   - Insert record into theoretical_stop_breaches table
   - Calculate days_until_executable
   - Send notification: "Stop loss breached at [price], cannot execute until [can_sell_date]"
   - Log for performance tracking
4. If settlement_status = LIQUID:
   - Proceed with normal stop loss execution
   - Log as executable stop

### Workflow 4: Locked Risk Reporting

1. User requests risk dashboard
2. Go backend queries:
   - All positions with settlement_status in (LOCKED_T0, LOCKED_T1, LOCKED_T2)
   - Calculate locked_risk per position: `shares * entry_price * exchange_multiplier`
   - Sum to get total_locked_risk
3. Query all LIQUID positions
   - Calculate liquid_risk per position: `shares * (entry_price - stop_loss)`
   - Sum to get total_liquid_risk
4. Calculate metrics:
   - locked_risk_pct = (total_locked_risk / account_value) * 100
   - budget_remaining = (account_value * locked_risk_threshold) - total_locked_risk
5. Return dashboard with:
   - Locked capital breakdown by position
   - Days until liquid per position
   - Locked risk utilization percentage
   - Budget remaining for new purchases

## Risks / Trade-offs

### Risk 1: Settlement Date Calculation Errors

**Risk**: Lunar calendar holidays (Tết, Hung Kings) may be calculated incorrectly, leading to wrong settlement dates.

**Mitigation**:
- Maintain explicit holiday lookup table for 2024-2026
- Add admin interface to manually adjust settlement dates
- Monitor for positions stuck in LOCKED status beyond expected date
- Validate against historical settlement calendars

**Residual Risk**: Low. Most holidays are predictable, and manual override exists for edge cases.

### Risk 2: Overly Conservative Locked Risk

**Risk**: Using 20-40% locked risk multipliers may be too conservative, rejecting profitable trades.

**Mitigation**:
- Make multipliers configurable in user_config
- Start with conservative defaults, allow users to tune
- Track signal rejection metrics: if >30% rejections, reevaluate
- Implement A/B test: compare performance with/without locked risk budget

**Residual Risk**: Medium. Balance between safety and opportunity is subjective.

### Risk 3: Cron Job Failure

**Risk**: Daily settlement update cron job fails, leaving statuses stale.

**Mitigation**:
- Implement health check: if settlement update >24 hours old, alert
- Add manual trigger endpoint for settlement update
- Log all cron executions with success/failure status
- Add retry logic with exponential backoff

**Residual Risk**: Low. Monitoring and manual override provide safety nets.

### Risk 4: Performance Impact of Daily Snapshots

**Risk**: Inserting daily snapshot for every position may bloat position_settlement_tracking table.

**Mitigation**:
- Only snapshot positions with status changes (not all positions daily)
- Add data retention policy: delete snapshots older than 6 months
- Partition table by check_date for efficient queries
- Index on position_id + check_date for fast lookups

**Residual Risk**: Low. Table growth is predictable and manageable.

## Migration Plan

### Phase 1: Schema Migration (Maintenance Window)

1. Run migration `000014_add_settlement_tracking.up.sql`
2. Add new columns to positions table (all nullable initially)
3. Create position_settlement_tracking table
4. Create theoretical_stop_breaches table
5. Add locked_risk_threshold to user_config with default 0.10

### Phase 2: Data Backfill (Offline)

1. Query all existing positions with is_closed = FALSE
2. For each position:
   - Assume already settled (set settlement_status = LIQUID)
   - Set purchase_date = entry_date
   - Set settlement_date = entry_date + 3 days (conservative)
   - Set can_sell_date = settlement_date
   - Infer exchange from ticker symbol:
     - Tickers ending in "E": HOSE
     - Tickers ending in "N": HNX
     - Others: UPCOM
   - Set locked_capital = 0 (already liquid)
   - Set liquid_capital = quantity * entry_price
3. Validate: all active positions have non-null settlement_status

### Phase 3: Code Deployment (Zero Downtime)

1. Deploy Go backend with settlement logic (backward compatible)
2. Deploy Python ML service with locked risk checks (backward compatible)
3. Settlement status fields populated, locked risk logic inactive
4. Monitor for errors in new code paths

### Phase 4: Feature Activation (Gradual)

1. Enable locked risk budget checks with threshold = 20% (conservative)
2. Monitor signal rejections for 1 week
3. If rejection rate <20%, lower threshold to 15%
4. Monitor for 1 week
5. If rejection rate still <20%, lower to default 10%
6. Enable entry day restrictions after threshold tuned

### Phase 5: Settlement Cron Activation

1. Deploy cron job to staging, run manually for 1 week
2. Validate settlement transitions match expected behavior
3. Deploy cron job to production with 16:30 schedule
4. Monitor for missed executions or errors
5. After 2 weeks stable operation, consider feature complete

### Rollback Plan

If critical issues discovered:

1. Disable locked risk budget enforcement (allows all signals)
2. Disable entry day restrictions (full position sizes)
3. Disable settlement cron job (statuses frozen)
4. Positions remain functional with new fields nullable
5. Revert code deployment if needed
6. Run down migration if schema changes cause issues

## Open Questions

1. **Q**: Should we implement manual override for locked risk budget on a per-trade basis?
   **A**: Yes, add `override_locked_risk` flag to signal generation, requires explicit user confirmation.

2. **Q**: How should we handle positions that span multiple purchases with different settlement dates?
   **A**: Use the most recent purchase_date for settlement status, or track per-entry with position_entries table (already exists).

3. **Q**: Should we automatically execute stop loss when position becomes LIQUID if price still below stop?
   **A**: No, too risky. Price may have recovered. Notify user and let them decide.

4. **Q**: How to handle exchange changes (e.g., stock moves from HNX to HOSE)?
   **A**: Rare event. Log warning if detected, require manual position close/reopen.

5. **Q**: Should locked risk budget be per-stock or portfolio-wide?
   **A**: Portfolio-wide to prevent concentration risk in locked capital.

## Success Criteria

After 3 months of production operation:

1. **Zero settlement lock failures**: No cases where stop loss failed to execute due to missing settlement tracking
2. **Locked risk never exceeded 12%**: Total locked risk stays within configured threshold + 2% buffer
3. **No Thursday/Friday entries at full size**: Entry day restrictions enforced (unless manually overridden)
4. **Theoretical stop breach tracking**: At least 10 instances recorded for performance analysis
5. **Settlement transition accuracy**: 99%+ of positions transition to LIQUID on expected can_sell_date
6. **Signal rejection <25%**: Locked risk budget rejects <25% of otherwise valid signals

## Future Enhancements (Out of Scope)

- Real-time settlement status updates via WebSocket
- Predictive locked risk forecasting based on upcoming signals
- Automated rebalancing when locked risk approaches threshold
- Integration with brokerage API to verify actual settlement dates
- Machine learning to optimize locked risk multipliers per exchange
