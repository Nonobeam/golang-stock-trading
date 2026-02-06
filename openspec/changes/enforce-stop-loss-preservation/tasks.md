## 1. Database Migration

- [ ] 1.1 Create migration `000014_add_stop_loss_protection.up.sql`
  - Add `first_entry_stop_loss DECIMAL(15,2)` to positions table
  - Add `stop_loss_locked BOOLEAN DEFAULT TRUE` to positions table
  - Add `stop_loss_override_reason TEXT` to positions table
  - Add `stop_loss_last_modified TIMESTAMP` to positions table

- [ ] 1.2 Create down migration `000014_add_stop_loss_protection.down.sql`
  - Drop new columns in reverse order

- [ ] 1.3 Create backfill script `scripts/backfill_first_entry_stops.py`
  - For each position, get first entry from position_entries
  - Calculate `first_entry_stop_loss = first_entry_price * (1 - 0.0475)`
  - Update positions.first_entry_stop_loss
  - Set stop_loss_locked = TRUE
  - Validate against current stop_loss (flag if deviation > 1%)

- [ ] 1.4 Run backfill and validate results
  - Execute backfill script
  - Verify all positions have first_entry_stop_loss populated
  - Review flagged positions with deviations
  - Document any manual corrections needed

## 2. Configuration Setup

- [ ] 2.1 Create `ml-service/config.py` with constants
  - STOP_LOSS_PERCENT = 0.0475
  - STOP_LOSS_TOLERANCE = 0.01
  - POSITION_LIMITS dict (hard/soft thresholds)
  - STOP_LOSS_POLICY dict (lock settings)

## 3. Python Position Manager Updates

- [ ] 3.1 Add `calculate_total_position_risk()` method
  - Query all position_entries for ticker
  - Sum entry-by-entry risk: shares × (entry_price - first_entry_stop_loss)
  - Return total risk amount

- [ ] 3.2 Modify `check_buying_capacity()` at line 274
  - Replace simplified risk formula (line 358) with calculate_total_position_risk()
  - Add stop-loss validation before capacity checks
  - Verify stop_loss matches first_entry_stop_loss (within tolerance)
  - Return error if locked and mismatched

- [ ] 3.3 Add three-tier capacity status to `check_buying_capacity()`
  - Calculate position_percent and volume_percent
  - Determine at_hard_limit (≥20% or ≥1%)
  - Determine at_soft_limit (≥18% or ≥0.8% but below hard)
  - Add status fields to return dict: at_hard_limit, at_soft_limit, warnings

- [ ] 3.4 Update return value of `check_buying_capacity()`
  - Add 'capacity_status': NORMAL | APPROACHING_LIMIT | AT_HARD_LIMIT
  - Add 'distance_to_hard_limit_value' and 'distance_to_hard_limit_volume'
  - Add 'warnings' list for soft limit messages

## 4. Python Position Sizing Updates

- [ ] 4.1 Modify `calculate_position_change()` in kelly.py:98
  - Update capacity handling to use three-tier status
  - At AT_HARD_LIMIT: return HOLD with reason
  - At APPROACHING_LIMIT: allow constrained BUY_MORE with warning
  - At NORMAL: standard behavior

- [ ] 4.2 Add capacity warning messages
  - When approaching_limit, include distance to hard limit in reason
  - Show max additional shares before hitting cap

## 5. Python Signal Generation Updates

- [ ] 5.1 Modify `generate_signal()` in generator.py:36
  - Update capacity check (line 142-146) to use three-tier status
  - Add soft limit warning branch
  - Include capacity details in signal metadata

- [ ] 5.2 Update `generate_and_save_signal()` metadata
  - Add capacity_status field
  - Add warnings field if approaching limits
  - Include distance to hard limits

## 6. Daily Validation Workflow

- [ ] 6.1 Create `ml-service/validation/stop_loss_integrity.py`
  - Function `validate_stop_loss_integrity(positions)`
  - For each position, check stop_loss vs first_entry_stop_loss
  - Identify deviations > tolerance
  - Return list of violations

- [ ] 6.2 Integrate validation into `ml-service/daily/daily_signals.py`
  - Call validate_stop_loss_integrity() before signal generation
  - Log violations as errors
  - Option: block signal generation if violations found
  - Send alert notification for violations

## 7. Go Repository Validation

- [ ] 7.1 Update `internal/db/repository/types.go`
  - Add FirstEntryStopLoss field to Position struct
  - Add StopLossLocked field
  - Add StopLossOverrideReason field
  - Add StopLossLastModified field

- [ ] 7.2 Modify `Update()` in position_repository.go:266
  - Add validation before updating stop_loss
  - Check if stop_loss_locked = TRUE
  - If locked and stop_loss changed, reject with error
  - If unlocked, require stop_loss_override_reason
  - Update stop_loss_last_modified timestamp

- [ ] 7.3 Add `ValidateStopLoss()` helper function
  - Compare new stop_loss vs first_entry_stop_loss
  - Check tolerance (within 1%)
  - Return validation error with details

## 8. Testing

- [ ] 8.1 Go unit tests in `test/stop_loss_protection_test.go`
  - Test: Reject stop_loss update when locked
  - Test: Allow stop_loss update when unlocked with reason
  - Test: Reject stop_loss update without reason when unlocked
  - Test: Timestamp updated on successful change

- [ ] 8.2 Python unit tests in `ml-service/tests/test_risk_calculation.py`
  - Test: Multi-entry risk calculation accuracy
  - Test: Single entry matches simplified formula
  - Test: Risk aggregation with varying entry prices
  - Test: Stop-loss validation catches deviations

- [ ] 8.3 Python unit tests in `ml-service/tests/test_capacity_warnings.py`
  - Test: Hard limit blocks BUY_MORE
  - Test: Soft limit allows constrained BUY_MORE
  - Test: Normal capacity allows full recommendation
  - Test: Warning messages include correct thresholds

- [ ] 8.4 Integration test for daily validation
  - Test: Clean positions pass validation
  - Test: Modified stop_loss flagged as violation
  - Test: Tolerance allows small rounding differences
  - Test: Validation blocks signal generation

## 9. Documentation

- [ ] 9.1 Update README with stop-loss policy
  - Document lock-at-first-entry policy
  - Explain override procedure
  - List capacity tier thresholds

- [ ] 9.2 Add migration guide
  - Steps to backfill existing positions
  - How to review flagged deviations
  - Rollback procedure if needed

- [ ] 9.3 Update capacity formulas documentation
  - Document per-entry risk calculation
  - Explain soft vs hard limits
  - Provide examples

## 10. Deployment

- [ ] 10.1 Pre-deployment validation
  - Run backfill script in test environment
  - Verify all positions backfilled correctly
  - Test stop-loss modification rejection
  - Confirm daily validation runs cleanly

- [ ] 10.2 Production deployment sequence
  - Deploy database migration
  - Run backfill script
  - Deploy Python changes (position manager, sizing, signals)
  - Deploy Go changes (repository validation)
  - Monitor for validation errors

- [ ] 10.3 Post-deployment verification
  - Check daily validation runs without violations
  - Verify capacity warnings appear correctly
  - Confirm risk calculations use new formula
  - Test stop-loss modification rejection in production
