# Change: Add Drawdown-Based Risk Controls

## Why

The ML trading system currently lacks portfolio-level risk management based on overall account performance. Two critical risk management features identified in the algorithms documentation are missing:

1. **Global Drawdown Reduction** - When portfolio equity experiences significant drawdowns (-10% or -15%), position sizes should automatically reduce or stop entirely to preserve capital and prevent catastrophic losses

2. **Enhanced R-Multiple Tracking** - While R-multiple is calculated per position, there's no portfolio-level tracking of average R-multiple, no reporting in daily workflows, and no integration with position sizing decisions

Without these features:

- System cannot automatically reduce risk during losing streaks
- Portfolio could experience deeper drawdowns than acceptable
- No visibility into risk-adjusted performance metrics
- Position sizing doesn't adapt to recent performance

These gaps create unnecessary risk exposure and reduce the system's ability to preserve capital during adverse market conditions.

## What Changes

### New Capabilities

1. **Portfolio Equity Tracking**
   - Create `portfolio_equity_snapshots` table to track daily equity curves
   - Calculate daily equity = open positions market value + closed positions P&L + remaining cash
   - Store peak equity (running maximum) for drawdown calculations
   - Track current drawdown percentage from peak

2. **Drawdown-Based Position Sizing Adjustment**
   - Add drawdown multiplier to position sizing logic
   - Reduce all position sizes by 50% when drawdown reaches -10%
   - Stop all new positions (reduce to 0%) when drawdown reaches -15%
   - Reset to normal sizing when equity recovers above -5% drawdown

3. **Enhanced R-Multiple Reporting**
   - Add portfolio-level R-multiple statistics (average, median, best, worst)
   - Include R-multiple distribution in daily validation reports
   - Track R-multiple by signal type to identify best-performing patterns
   - Add R-multiple charts to validation dashboards

### Modified Components

**ML Service (Python):**

- `ml-service/position_sizing/kelly.py` - Add drawdown_multiplier parameter
- `ml-service/position_manager/manager.py` - Add portfolio equity tracking methods
- `ml-service/daily/daily_signals.py` - Integrate drawdown checks before position sizing
- `ml-service/validation/` - New module `portfolio_metrics.py` for equity tracking
- `ml-service/monitoring/` - Update alerter to warn on high drawdowns

**Database:**

- New migration: `000012_add_portfolio_equity_tracking.up.sql`
- New table: `portfolio_equity_snapshots`
- New table: `r_multiple_statistics` (daily aggregates)

**Daily Workflow:**

- Update daily signal generation to check drawdown before generating BUY signals
- Add equity snapshot at end of each trading day
- Generate R-multiple performance reports

## Impact

### Affected Components

- `ml-service/position_sizing/kelly.py` - Position sizer modifications
- `ml-service/position_manager/manager.py` - Equity tracking
- `ml-service/daily/daily_signals.py` - Drawdown integration
- `ml-service/validation/portfolio_metrics.py` - New module
- `ml-service/monitoring/alerter.py` - Drawdown alerts
- `db/migrations/` - New migration files
- Daily workflow scripts - Equity snapshots and reporting

### Benefits

- ✅ Automatic risk reduction during losing streaks
- ✅ Capital preservation through drawdown-based circuit breakers
- ✅ Complete visibility into portfolio equity curve over time
- ✅ Enhanced R-multiple analytics for strategy refinement
- ✅ Early warning system for performance deterioration
- ✅ Aligns implementation with documented risk management algorithms

### Risks and Trade-offs

- ⚠️ **Performance Impact**: Daily equity calculations add minimal overhead
- ⚠️ **False Positives**: Short-term drawdowns could unnecessarily restrict trading
- ⚠️ **Complexity**: Adds portfolio-level state management
- **Mitigation**: Thresholds (-10%/-15%) are conservative and battle-tested

### Non-Breaking Changes

- Existing position sizing logic remains unchanged when drawdown < 10%
- R-multiple calculations already implemented, just adding reporting
- No changes to existing database tables (only additions)
- Backward compatible with current workflow
