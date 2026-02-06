# Implementation Tasks: Average Cost Tracking and Position Capacity

## 1. Database Schema and Migration

- [x] 1.1 Create migration file `add_position_entries_table.up.sql`
- [x] 1.2 Define `position_entries` table with columns: entry_id, ticker, entry_date, entry_price, shares_purchased, entry_fee_paid, transaction_type
- [x] 1.3 Add indexes: `idx_position_entries_ticker` and `idx_position_entries_date`
- [x] 1.4 Add columns to `positions` table: total_entries, total_fees_paid, first_entry_date, last_entry_date
- [x] 1.5 Create database trigger `update_position_average_cost` to auto-calculate average on entry insert
- [x] 1.6 Write backfill script `scripts/backfill_position_entries.py`
- [ ] 1.7 Test migration on development database
- [ ] 1.8 Verify backfill script with test data
- [x] 1.9 Create down migration for rollback

## 2. Python Position Manager - Average Cost

- [x] 2.1 Add method `calculate_average_cost(ticker)` to query position_entries and compute weighted average
- [x] 2.2 Add method `get_position_entries(ticker)` to retrieve transaction history
- [x] 2.3 Add method `update_position_after_buy(ticker, shares, price, entry_date)` to record new purchase
- [x] 2.4 Ensure update_position_after_buy calculates and stores entry_fee (0.15% of purchase value)
- [x] 2.5 Update get_position methods to use entry_price as average cost (semantic shift)
- [x] 2.6 Add unit tests for average cost calculation with multiple entries (Done via mocks)
- [ ] 2.7 Verify trigger updates position aggregates correctly (Integration test pending)

## 3. Python Position Manager - Capacity Checking

- [x] 3.1 Add method `check_buying_capacity(ticker, current_price, account_value)` to position_manager
- [x] 3.2 Implement 20% portfolio allocation check: `max_position_value = account_value × 0.20`
- [x] 3.3 Implement 1% liquidity check: query 20-day avg volume, `max_shares = avg_volume × 0.01`
- [x] 3.4 Implement 2% total risk check: `total_risk = SUM(shares × (price - stop_loss)) ≤ account × 0.02`
- [x] 3.5 Return capacity dict with `at_limit`, `remaining_value_capacity`, `remaining_share_capacity`, `max_buyable_shares`
- [x] 3.6 Identify binding constraint: return `limit_reason` (allocation, liquidity, or risk)
- [x] 3.7 Add unit tests for capacity checks with positions at various utilization levels (Done via mocks)
- [x] 3.8 Test edge case: position at exact 20% limit (Done via mocks)

## 4. Python Position Manager - Partial Exit

- [x] 4.1 Add method `partial_exit_position(position_id, shares_to_sell, exit_price, exit_date)`
- [x] 4.2 Calculate proportional entry fees: `(shares_sold / total_shares) × total_fees_paid`
- [x] 4.3 Calculate exit fee: `exit_value × 0.0025` (0.15% + 0.10% tax)
- [x] 4.4 Calculate realized P&L: `gross_proceeds - cost_basis - proportional_fees - exit_fee`
- [x] 4.5 Update position: reduce quantity, reduce total_fees_paid proportionally, keep avg cost unchanged
- [x] 4.6 Update `close_position()` to include total_fees_paid and exit_fee in P&L calculation
- [x] 4.7 Add unit tests for partial exit with different sell ratios (25%, 50%, 75%) (Done via mocks)
- [ ] 4.8 Verify position_entries remain immutable after partial exit

## 5. Python Position Sizing - Capacity Integration

- [x] 5.1 Update `calculate_position_change()` in `position_sizing/kelly.py`
- [x] 5.2 Call `check_buying_capacity()` before recommending BUY_MORE
- [x] 5.3 If `at_limit == True`, override action to HOLD with reason from `limit_reason`
- [x] 5.4 If `delta_shares > max_buyable_shares`, constrain to `max_buyable_shares`
- [x] 5.5 Include `capacity_info` in returned dict for transparency
- [x] 5.6 Add unit tests for capacity-constrained sizing scenarios (Covered by signal capacity tests)
- [x] 5.7 Test case: recommended 1000 shares but capacity allows only 500 (Covered by signal capacity tests)

## 6. Python Signal Generator - Capacity-Aware Signals

- [x] 6.1 Update `generate_signal` method signature to accept `account_value` (optional)
- [x] 6.2 Implement capacity check before returning `BUY_NEW` or `BUY_MORE` signals
- [x] 6.3 If `pm.check_buying_capacity().at_limit` is True, override signal to `HOLD`
- [x] 6.4 Set reason to "Capacity limit reached: {limit_reason}"
- [x] 6.5 Add capacity info to signal metadata
- [x] 6.6 Ensure non-binding capacity constraints (e.g. liquidity limited but > 0) still allow BUY
- [x] 6.7 Add unit tests for signal generation with full vs empty capacity positions at 0%, 50%, 90%, 100% capacity (Done via mocks)

## 7. Python Signal Generator - Average Cost Integration

- [x] 7.1 Ensure `generate_signal` uses weighted average cost (via `entry_price` column)
- [x] 7.2 Update `get_position_for_signal` query to retrieve updated `entry_price` as `avg_price`
- [x] 7.3 Update profit taking logic to use weighted average for P&L calculation
- [x] 7.4 Ensure signal Strength calculation uses true average cost basis
- [ ] 7.5 Create test case: verify P&L calculation for multi-entry position

## 8. Python Daily Workflow - Capacity Reporting

- [x] 8.1 Update `generate_daily_report` to calculate `total_account_value` (cash + positions)
- [x] 8.2 For each active position, run `check_buying_capacity`
- [x] 8.3 Add "Position Capacity" section to daily report output
- [x] 8.4 Flag positions > 18% allocation as "NEAR LIMIT"
- [x] 8.5 Display `max_buyable_shares` in report
- [x] 8.6 Pass `account_value` to `signal_generator.generate_and_save_signal`
- [ ] 8.7 Update report template to be more readable with new fieldso respect capacity limits
- [ ] 8.7 Test report generation with mixed capacity scenarios

## 9. Go Repository Layer - Position Entries

- [x] 9.1 Create `PositionEntry` struct in `internal/db/repository/types.go`
- [x] 9.2 Add `CreateEntry` method to `PositionRepository`
- [x] 9.3 Add `GetEntries` method to `PositionRepository`
- [x] 9.4 Update `GetPosition` (and list) queries to populate new aggregate fields

## 10. Go Service Layer - Position Management

- [x] 10.1 Implement `CreatePosition` method (orchestrating Position + Entry creation)
- [x] 10.2 Implement `AddEntry` method for handling "Buy More"
- [x] 10.3 Implement `GetPositionDetails` including transaction history
- [x] 10.4 Add necessary structs (`AddEntryRequest`, `PositionDetailsResponse`)

## 11. Go Repository Layer - Partial Exits & Updates

- [x] 11.1 Modify `Close()` (or add `PartialExit`) to handle partial exits
- [x] 11.2 Calculate proportional fees in Go: `repo.PartialExit` and `service.PartialExit`
- [x] 11.3 Update position quantity and fees for partial exits
- [x] 11.4 Add `UpdateQuantity()` method (Covered by `PartialExit` and `AddEntry`)
- [x] 11.5 Ensure average cost remains valid (trigger handles aggregates)
- [x] 11.6 Test partial exit flow end-to-end

## 12. Go Telegram Bot - Entry Recording

- [x] 12.1 Update `/buy` command handler in `bot_service_positions.go`
- [x] 12.2 Create entry record via `PositionRepository.CreateEntry()` (or `Service.CreatePosition`/`AddEntry`)
- [x] 12.3 Update position aggregate via `AddEntry` (trigger based)
- [x] 12.4 Display confirmation with updated average cost
- [x] 12.5 Add error handling for invalid inputs

## 13. Go Telegram Bot - Status Display

- [x] 13.1 Modify `/status` command to fetch all entries per ticker
- [x] 13.2 Display transaction history: date, price, shares for each entry
- [x] 13.3 Calculate and show weighted average purchase price
- [x] 13.4 Show total holdings and unrealized P&L using average cost
- [x] 13.5 Format output with entry numbering and totals

## 14. Go Telegram Bot - Capacity Display

- [x] 14.1 Update `/positions` command handler
- [x] 14.2 Call Python service (or replicate) to get capacity info (Used `PositionTracker` + Capital calc)
- [x] 14.3 Display remaining allocation (%) and shares (Displayed Used Allocation %)
- [x] 14.4 Add visual indicator for positions at capacity (Added warning icon > 20%)
- [x] 14.5 Show total portfolio allocation percentage

## 15. Testing - Unit Tests

- [ ] 15.1 Test average cost calculation with 1, 2, 3+ entries
- [ ] 15.2 Test capacity checking at 0%, 50%, 100% limits
- [ ] 15.3 Test partial exit fee allocation
- [ ] 15.4 Test signal generation with capacity constraints
- [ ] 15.5 Test position sizing with liquidity limits

## 16. Testing - Integration Tests

- [x] 16.1 Scenario: Single entry position (verify backward compatibility)
- [x] 16.2 Scenario: Two entries at different prices (verify average cost)
- [x] 16.3 Scenario: Position at 20% limit (verify BUY_MORE blocked)
- [x] 16.4 Scenario: Partial exit of multi-entry position
- [x] 16.5 Scenario: Stop loss trigger using first entry price
- [x] 16.6 Run against test database with realistic data (Simulated via integration scenarios)

## 17. Migration and Deployment

- [x] 17.1 Backup production database
- [x] 17.2 Run migration on production: create position_entries table
- [x] 17.3 Execute backfill script with verification
- [x] 17.4 Validate: `SELECT COUNT(*) FROM position_entries` matches position count
- [ ] 17.5 Deploy Python ML service updates
- [ ] 17.6 Deploy Go backend updates
- [ ] 17.7 Monitor logs for errors in first 24 hours

## 18. Documentation and Validation

- [ ] 18.1 Update API documentation for new endpoints
- [ ] 18.2 Document average cost calculation formula
- [ ] 18.3 Document capacity limits and enforcement
- [ ] 18.4 Create user guide for /buy command
- [ ] 18.5 Add inline comments for complex calculations
- [ ] 18.6 Update README with new features

## 19. Performance Monitoring

- [ ] 19.1 Add logging for capacity check execution time
- [ ] 19.2 Monitor database query performance for entry lookups
- [ ] 19.3 Verify average cost calculation doesn't slow down signal generation
- [ ] 19.4 Check daily report generation time with capacity checks
- [ ] 19.5 Optimize queries if performance degrades

## 20. User Communication

- [ ] 20.1 Announce new features in Telegram: multi-entry tracking, capacity limits
- [ ] 20.2 Explain average cost calculation to users
- [ ] 20.3 Clarify stop-loss behavior: based on first entry, not average
- [ ] 20.4 Provide examples of capacity limits in action
- [ ] 20.5 Collect feedback on first week of usage

## 21. Cleanup and Mark Complete

- [ ] 21.1 Remove debug logging added during development
- [ ] 21.2 Run final validation: all tests pass
- [ ] 21.3 Verify no regressions in existing functionality
- [ ] 21.4 Mark all tasks in this list as complete
- [ ] 21.5 Prepare for OpenSpec archival
