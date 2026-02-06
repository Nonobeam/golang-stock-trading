# Implementation Tasks

## Phase 1: Database Infrastructure (2 tasks)

- [x] **T1.1** Create migration `000012_add_portfolio_equity_tracking.up.sql`
  - Create `portfolio_equity_snapshots` table (user_id, snapshot_date, total_equity, peak_equity, current_drawdown, open_positions_value, closed_pnl, cash_balance)
  - Create `r_multiple_statistics` table (date, avg_r_multiple, median_r_multiple, win_rate, best_r, worst_r, total_trades, metadata jsonb)
  - Add indexes for efficient querying by user and date
  - **Validation**: Run migration on dev database, verify tables created

- [x] **T1.2** Create migration `000012_add_portfolio_equity_tracking.down.sql`
  - Drop tables in reverse order
  - **Validation**: Test rollback on dev database

## Phase 2: Portfolio Equity Tracking (3 tasks)

- [x] **T2.1** Create `ml-service/validation/portfolio_metrics.py`
  - Implement `PortfolioEquityTracker` class
  - Method: `calculate_current_equity(user_id)` - Sum open positions + closed P&L + cash
  - Method: `get_peak_equity(user_id)` - Query max historical equity
  - Method: `calculate_drawdown(user_id)` - (current - peak) / peak
  - Method: `save_equity_snapshot(user_id, date)` - Insert daily snapshot
  - **Validation**: Unit tests with mock positions data

- [x] **T2.2** Create `ml-service/validation/r_multiple_analytics.py`
  - Implement `RMultipleAnalytics` class
  - Method: `calculate_portfolio_r_stats(user_id, lookback_days)` - Aggregate R-multiples
  - Method: `get_r_distribution(user_id)` - Histogram data
  - Method: `get_r_by_signal_type(user_id)` - Group by signal_type
  - Method: `save_daily_r_statistics(date)` - Store aggregates
  - **Validation**: Unit tests with sample closed positions

- [x] **T2.3** Add equity tracking to `ml-service/position_manager/manager.py`
  - Import `PortfolioEquityTracker`
  - Add method: `snapshot_portfolio_equity(user_id, date)` - Wrapper for daily snapshots
  - **Validation**: Integration test creates snapshot

## Phase 3: Drawdown-Based Position Sizing (3 tasks)

- [x] **T3.1** Modify `ml-service/position_sizing/kelly.py`
  - Add parameter `drawdown_multiplier: float = 1.0` to `calculate_size()` and `calculate_shares()`
  - Update formula: `position_fraction = base_fraction * confidence_mul * horizon_mul * drawdown_multiplier`
  - Add docstring explaining drawdown multiplier
  - **Validation**: Unit tests verify multiplier applied correctly

- [x] **T3.2** Create `ml-service/position_sizing/drawdown_manager.py`
  - Implement `DrawdownManager` class
  - Method: `get_drawdown_multiplier(user_id)` - Returns multiplier based on current drawdown
    - If drawdown >= -15%: return 0.0 (stop trading)
    - If drawdown >= -10%: return 0.5 (half position sizes)
    - If drawdown >= -5%: return 1.0 (normal sizing)
    - Else: return 1.0
  - Method: `check_trading_allowed(user_id)` - Returns boolean and reason
  - **Validation**: Unit tests with various drawdown scenarios

- [x] **T3.3** Integrate drawdown checks into `ml-service/daily/daily_signals.py`
  - Import `DrawdownManager` and `PortfolioEquityTracker`
  - Before generating signals, calculate current drawdown
  - Get drawdown multiplier
  - If multiplier = 0.0, skip BUY signal generation entirely
  - Pass drawdown_multiplier to PositionSizer
  - Log drawdown status to console
  - **Validation**: Test with mock high drawdown, verify no BUY signals generated

## Phase 4: Monitoring and Alerting (3 tasks)

- [x] **T4.1** Update `ml-service/monitoring/alerter.py`
  - Add method: `check_drawdown_alert(user_id)` - Alert if drawdown crosses thresholds
  - Alert levels:
    - WARNING at -8% drawdown
    - CRITICAL at -10% drawdown (half position sizes)
    - EMERGENCY at -15% drawdown (trading stopped)
  - Send alerts via existing notification channels
  - **Validation**: Unit test triggers alerts at thresholds

- [x] **T4.2** Create drawdown visualization script `ml-service/scripts/plot_equity_curve.py`
  - Query `portfolio_equity_snapshots` for user
  - Plot equity curve with peak equity overlay
  - Highlight drawdown periods
  - Mark -10% and -15% threshold lines
  - Save to `logs/equity_curve_{user_id}.png`
  - **Validation**: Run script, verify chart generated

- [x] **T4.3** Create R-multiple report script `ml-service/scripts/generate_r_report.py`
  - Query closed positions for user
  - Calculate R-multiple statistics (avg, median, win rate, distribution)
  - Generate markdown report with charts
  - Save to `logs/r_multiple_report_{date}.md`
  - **Validation**: Run script, verify report generated

## Phase 5: Daily Workflow Integration (3 tasks)

- [x] **T5.1** Update `ml-service/daily/run_daily_signals.py`
  - At start of script, snapshot previous day's equity
  - Check current drawdown before generating signals
  - Log drawdown status to console and logs
  - If drawdown >= -15%, exit early with log message
  - **Validation**: Run script manually, verify equity snapshot created

- [x] **T5.2** Create `ml-service/daily/run_daily_equity.py` (new script)
  - Execute daily equity snapshot for all active users
  - Run R-multiple statistics aggregation
  - Check for drawdown alerts
  - Log summary statistics
  - **Validation**: Run script, verify snapshots for all users

- [x] **T5.3** Update daily validation example `ml-service/daily_validation_example.py`
  - Add section demonstrating equity tracking
  - Add section demonstrating R-multiple analytics
  - Add section showing drawdown-adjusted position sizing
  - **Validation**: Run example end-to-end, verify all features work

## Phase 6: Testing and Documentation (3 tasks)

- [x] **T6.1** Create integration test `ml-service/tests/test_drawdown_integration.py`
  - Test scenario: Portfolio at -11% drawdown
  - Verify position sizes reduced by 50%
  - Verify BUY signals still generated (just smaller)
  - Test scenario: Portfolio at -16% drawdown
  - Verify no BUY signals generated
  - **Validation**: All tests pass

- [x] **T6.2** Create integration test `ml-service/tests/test_equity_tracking.py`
  - Test equity calculation with open and closed positions
  - Test drawdown calculation accuracy
  - Test daily snapshot creation
  - Test edge cases (no positions, negative equity, etc.)
  - **Validation**: All tests pass

- [x] **T6.3** Update documentation
  - Update `ml-service/README.md` with drawdown feature
  - Add equity tracking usage examples
  - Add R-multiple analytics examples
  - Update `IMPLEMENTATION_SUMMARY.md` to mark features as implemented
  - **Validation**: Documentation review

---

## Dependencies

- T2.1 must complete before T3.2 (DrawdownManager needs PortfolioEquityTracker)
- T3.1 and T3.2 must complete before T3.3
- T2.1 must complete before T5.1
- Phase 4 can be done in parallel with Phase 3
- Phase 6 depends on all previous phases

## Estimated Effort

- Phase 1 (Database): 1-2 hours
- Phase 2 (Equity Tracking): 3-4 hours
- Phase 3 (Position Sizing): 2-3 hours
- Phase 4 (Monitoring): 2-3 hours
- Phase 5 (Workflow Integration): 2-3 hours
- Phase 6 (Testing & Docs): 2-3 hours

**Total**: 12-18 hours

## Success Criteria

✅ Portfolio equity tracked daily in database
✅ Drawdown calculated accurately from peak equity
✅ Position sizes automatically reduced at -10% drawdown
✅ Trading stops automatically at -15% drawdown
✅ Alerts sent when drawdown thresholds crossed
✅ R-multiple statistics available in reports
✅ All tests passing
✅ Documentation updated
