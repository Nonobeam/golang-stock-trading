# Design: Drawdown-Based Risk Management System

## Overview

This design implements portfolio-level risk management through automated drawdown-based position sizing adjustments and enhanced R-multiple tracking. The system preserves capital during losing streaks by automatically reducing or halting trading when portfolio equity drops significantly from its peak.

## Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    Daily Workflow Entry Point                    │
│                   (daily/run_daily_signals.py)                   │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ├──► 1. Snapshot Previous Day Equity
                            │    (PortfolioEquityTracker)
                            │
                            ├──► 2. Calculate Current Drawdown
                            │    (PortfolioEquityTracker)
                            │
                            ├──► 3. Get Drawdown Multiplier
                            │    (DrawdownManager)
                            │
                            ├──► 4. Check Trading Allowed
                            │    If multiplier == 0.0, exit early
                            │
                            └──► 5. Generate Signals with Adjusted Sizing
                                 (PositionSizer receives drawdown_multiplier)

┌─────────────────────────────────────────────────────────────────┐
│                      Database Layer                              │
├─────────────────────────────────────────────────────────────────┤
│  portfolio_equity_snapshots                                     │
│  - snapshot_date, total_equity, peak_equity, current_drawdown   │
│                                                                   │
│  r_multiple_statistics (daily aggregates)                       │
│  - avg_r, median_r, win_rate, best/worst                        │
│                                                                   │
│  positions (existing, has r_multiple column)                    │
└─────────────────────────────────────────────────────────────────┘
```

## Key Design Decisions

### 1. Portfolio Equity Calculation

**Formula**:

```python
total_equity = open_positions_market_value + closed_positions_total_pnl + cash_balance

where:
  open_positions_market_value = Σ(quantity × current_price) for all open positions
  closed_positions_total_pnl = Σ(pnl) for all closed positions
  cash_balance = initial_capital - Σ(invested_capital_open_positions)
```

**Rationale**:

- Provides real-time view of entire portfolio value
- Accounts for unrealized gains/losses on open positions
- Includes realized P&L from closed positions
- Maintains cash balance to prevent over-allocation

### 2. Drawdown Measurement

**Formula**:

```python
peak_equity = max(total_equity for all historical snapshots)
current_drawdown = (current_equity - peak_equity) / peak_equity

Example:
  Peak equity: 100,000,000 VND
  Current equity: 88,000,000 VND
  Drawdown: (88M - 100M) / 100M = -12% (crosses -10% threshold)
```

**Design Choice**: Use running maximum (peak equity) rather than period-based high

- **Why**: Aligns with industry standard (Van Tharp, Turtle Traders methodology)
- **Benefit**: More conservative than rolling window drawdown
- **Trade-off**: Takes longer to recover from historical peaks

### 3. Drawdown Thresholds and Multipliers

**Decision Table**:

| Current Drawdown | Multiplier | Action                  | Rationale                            |
| ---------------- | ---------- | ----------------------- | ------------------------------------ |
| `> -5%`          | `1.0`      | Normal trading          | Healthy portfolio performance        |
| `-5% to -10%`    | `1.0`      | Normal trading          | Minor fluctuation, acceptable        |
| `-10% to -15%`   | `0.5`      | **Half position sizes** | Warning: Reduce risk exposure        |
| `< -15%`         | `0.0`      | **Stop all trading**    | Critical: Preserve remaining capital |

**Rationale for Thresholds**:

- **-10%**: Industry standard for "WARNING" level (Dalio, Soros methodologies)
- **-15%**: Conservative circuit breaker to prevent catastrophic losses
- **Multiplier of 0.5**: Allows continued trading but with reduced risk

**Alternative Considered**: Smooth multiplier function

```python
# Rejected: Too complex and hard to reason about
multiplier = max(0, 1 - (abs(drawdown) - 0.05) / 0.10)
```

**Why Rejected**: Step function is simpler, more predictable, easier to backtest

### 4. Position Sizing Integration

**Modified Formula**:

```python
# Original (range-based)
position_fraction = f_base × m_confidence × m_horizon

# New (with drawdown adjustment)
position_fraction = f_base × m_confidence × m_horizon × m_drawdown

where m_drawdown is the drawdown multiplier from DrawdownManager
```

**Integration Point**: `PositionSizer.calculate_size()`

- Add optional parameter: `drawdown_multiplier: float = 1.0`
- Multiply final allocation by this parameter
- Default to 1.0 for backward compatibility

**Example**:

```python
# Normal conditions (drawdown = -3%)
f = 0.10 × 1.5 × 1.0 × 1.0 = 0.15 (15%)

# High drawdown (-12%)
f = 0.10 × 1.5 × 1.0 × 0.5 = 0.075 (7.5%)

# Critical drawdown (-16%)
f = 0.10 × 1.5 × 1.0 × 0.0 = 0.00 (0%, no trade)
```

### 5. R-Multiple Enhanced Tracking

**What's Already Implemented**:

- `positions.r_multiple` column exists
- Calculated as: `(exit_price - entry_price) / (entry_price - stop_loss)`
- Stored when position is closed

**What's Missing**:

- Portfolio-level aggregation
- Historical tracking over time
- Comparison by signal type
- Visibility in daily reports

**New Data Model**:

```sql
CREATE TABLE r_multiple_statistics (
    id UUID PRIMARY KEY,
    user_id BIGINT NOT NULL,
    calculation_date DATE NOT NULL,

    -- Aggregate statistics
    avg_r_multiple DECIMAL(10, 4),
    median_r_multiple DECIMAL(10, 4),
    stddev_r_multiple DECIMAL(10, 4),

    -- Distribution
    best_r_multiple DECIMAL(10, 4),
    worst_r_multiple DECIMAL(10, 4),

    -- Performance
    win_rate DECIMAL(5, 4), -- % of R > 0
    total_trades INTEGER,
    profitable_trades INTEGER,

    -- Breakdown by signal type (JSON)
    r_by_signal_type JSONB,

    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(user_id, calculation_date)
);
```

**Analytics Queries**:

```python
# Get 90-day average R-multiple
SELECT AVG(avg_r_multiple)
FROM r_multiple_statistics
WHERE user_id = ? AND calculation_date >= NOW() - INTERVAL '90 days'

# Best performing signal types
SELECT
    signal_type,
    AVG(r_multiple) as avg_r,
    COUNT(*) as trades
FROM positions
WHERE user_id = ? AND is_closed = TRUE
GROUP BY signal_type
ORDER BY avg_r DESC
```

### 6. Daily Equity Snapshot Workflow

**Timing**: End of each trading day (after market close)

**Process**:

1. Calculate current equity for all users
2. Determine peak equity (compare with historical max)
3. Calculate current drawdown
4. Store snapshot in `portfolio_equity_snapshots`
5. Trigger alerts if drawdown crosses thresholds

**Pseudo-code**:

```python
def snapshot_daily_equity():
    for user_id in active_users:
        # Calculate equity components
        open_value = sum(pos.quantity * current_price
                        for pos in get_open_positions(user_id))
        closed_pnl = sum(pos.pnl for pos in get_closed_positions(user_id))
        cash = user.initial_capital - sum(invested)

        total_equity = open_value + closed_pnl + cash

        # Get peak
        peak = get_peak_equity(user_id)
        if total_equity > peak:
            peak = total_equity

        # Calculate drawdown
        drawdown = (total_equity - peak) / peak if peak > 0 else 0

        # Save snapshot
        save_snapshot(user_id, date, total_equity, peak, drawdown,
                     open_value, closed_pnl, cash)

        # Check alerts
        if drawdown <= -0.10 and drawdown > -0.15:
            send_alert("WARNING: Portfolio at {}% drawdown".format(drawdown*100))
        elif drawdown <= -0.15:
            send_alert("CRITICAL: Trading stopped at {}% drawdown".format(drawdown*100))
```

## Error Handling and Edge Cases

### Edge Case 1: No Historical Data (New User)

**Scenario**: User just started, no equity snapshots exist
**Solution**: Use `initial_capital` as peak equity

```python
peak_equity = get_peak_equity(user_id) or user.initial_capital
```

### Edge Case 2: Negative Equity

**Scenario**: User loses more than initial capital (shouldn't happen with stops, but theoretically possible)
**Solution**: Set drawdown to -100% maximum, trigger emergency stop

```python
if total_equity <= 0:
    drawdown = -1.0  # -100%
    drawdown_multiplier = 0.0
    send_emergency_alert("Portfolio equity negative!")
```

### Edge Case 3: No Positions Ever Opened

**Scenario**: User watched stocks but never traded
**Solution**: Equity = initial capital, drawdown = 0%

```python
if no_positions_exist(user_id):
    total_equity = user.initial_capital
    drawdown = 0.0
```

### Edge Case 4: Market Data Unavailable

**Scenario**: Cannot get current price for open position
**Solution**: Use last known price, log warning

```python
current_price = get_latest_price(symbol) or position.entry_price
if not get_latest_price(symbol):
    logger.warning(f"No price data for {symbol}, using entry price")
```

## Performance Considerations

### Database Query Optimization

**Equity Calculation Query** (runs daily per user):

```sql
-- Optimized single query to get all components
WITH open_positions AS (
    SELECT SUM(quantity * entry_price) as invested,
           ARRAY_AGG((symbol, quantity)) as holdings
    FROM positions
    WHERE user_id = ? AND is_closed = FALSE
),
closed_pnl AS (
    SELECT SUM(pnl) as total_pnl
    FROM positions
    WHERE user_id = ? AND is_closed = TRUE
)
SELECT
    op.invested,
    op.holdings,
    cp.total_pnl
FROM open_positions op
CROSS JOIN closed_pnl cp
```

**Index Requirements**:

```sql
CREATE INDEX idx_positions_user_open_equity
    ON positions(user_id, is_closed, quantity, entry_price)
    WHERE is_closed = FALSE;

CREATE INDEX idx_positions_user_closed_pnl
    ON positions(user_id, pnl)
    WHERE is_closed = TRUE;

CREATE INDEX idx_equity_snapshots_user_date
    ON portfolio_equity_snapshots(user_id, snapshot_date DESC);
```

### Caching Strategy

**Cache Peak Equity**: Store in memory for current trading session

```python
class DrawdownManager:
    def __init__(self):
        self._peak_cache = {}  # {user_id: (peak_value, timestamp)}

    def get_peak_equity(self, user_id):
        # Cache for 1 hour to reduce DB hits
        if user_id in self._peak_cache:
            peak, ts = self._peak_cache[user_id]
            if (datetime.now() - ts).seconds < 3600:
                return peak

        # Query DB and cache
        peak = query_peak_from_db(user_id)
        self._peak_cache[user_id] = (peak, datetime.now())
        return peak
```

## Testing Strategy

### Unit Tests

1. **PortfolioEquityTracker**:
   - Test equity calculation with various position combinations
   - Test peak equity tracking (monotonic increase)
   - Test drawdown calculation accuracy
   - Test edge cases (negative equity, no positions, etc.)

2. **DrawdownManager**:
   - Test multiplier thresholds (-10%, -15%)
   - Test trading permission logic
   - Test recovery scenarios (drawdown improving)

3. **PositionSizer** (modified):
   - Test drawdown multiplier integration
   - Test backward compatibility (multiplier = 1.0)

### Integration Tests

1. **End-to-End Drawdown Scenario**:
   - Simulate portfolio losing 12% from peak
   - Verify position sizes automatically halved
   - Verify alerts sent
   - Verify BUY signals still generated (but smaller)

2. **Circuit Breaker Test**:
   - Simulate portfolio losing 16% from peak
   - Verify no BUY signals generated
   - Verify trading stopped
   - Verify critical alerts sent

3. **Recovery Test**:
   - Start with -13% drawdown (half sizing)
   - Simulate profitable trades bringing drawdown to -4%
   - Verify normal position sizing resumes

## Deployment Considerations

### Rollout Plan

1. **Phase 1**: Deploy database migration (low risk)
2. **Phase 2**: Enable equity tracking WITHOUT position sizing changes (monitor for 1 week)
3. **Phase 3**: Enable drawdown multiplier with logging only (dry-run mode for 3 days)
4. **Phase 4**: Enable full drawdown-based position sizing

### Monitoring Metrics

Track these metrics post-deployment:

- Daily equity snapshot success rate
- Drawdown calculation latency
- Average drawdown multiplier applied
- Number of trading halts (multiplier = 0)
- Number of risk reductions (multiplier = 0.5)
- R-multiple distribution changes over time

### Rollback Plan

If issues detected:

1. Set `ENABLE_DRAWDOWN_CONTROLS=false` environment variable
2. Drawdown manager returns multiplier = 1.0 (normal sizing)
3. System operates as before
4. Database snapshots continue (data collection not affected)

## Future Enhancements

Potential improvements for future iterations:

1. **Dynamic Thresholds**: Adjust based on market regime
   - Bull market: -10%/-15% thresholds (current)
   - Bear market: -7%/-12% thresholds (more conservative)

2. **Time-Based Recovery**: Require X days above threshold before resuming
   - Prevent whipsaw on volatile equity curves

3. **User-Configurable Thresholds**: Allow per-user risk tolerance
   - Conservative: -5%/-10%
   - Moderate: -10%/-15% (default)
   - Aggressive: -15%/-20%

4. **Bayesian Drawdown Prediction**: Forecast probability of hitting thresholds
   - Use historical volatility to estimate risk

5. **Cross-User Benchmarking**: Compare R-multiple vs. other users
   - Identify top performers for strategy sharing

---

**Summary**: This design provides a robust, battle-tested approach to portfolio risk management through automated drawdown-based controls and enhanced R-multiple analytics, aligned with professional trading risk management standards.
