# Implementation Tasks: Graduated Exit Decision Engine

## Phase 1: Database & Core Engine (Week 1)

### Database Schema

- [ ] Create migration `000XXX_add_exit_tracking.up.sql`
  - Add `target1_filled BOOLEAN DEFAULT FALSE` to positions
  - Add `target2_filled BOOLEAN DEFAULT FALSE` to positions
  - Add `trailing_stop_active BOOLEAN DEFAULT FALSE` to positions
  - Add `target1_exit_price DECIMAL(10,2)` to positions
  - Add `target2_exit_price DECIMAL(10,2)` to positions
  - Add `target1_exit_date TIMESTAMP` to positions
  - Add `target2_exit_date TIMESTAMP` to positions
- [ ] Create down migration `000XXX_add_exit_tracking.down.sql`
- [ ] Test migration up/down locally
- [ ] Update `internal/position/types.go` Position struct with new fields

### Exit Decision Engine Core

- [ ] Create `internal/position/exit_engine.go`
- [ ] Implement `ExitEvaluator` struct with config
- [ ] Add `EvaluatePosition()` method - checks one position for exit conditions
- [ ] Add `ShouldExitTarget1()` - checks profit >= 15% OR price >= resistance
- [ ] Add `ShouldExitTarget2()` - checks profit >= 25% AND cleared resistance
- [ ] Add `ShouldExitTarget3()` - checks if trailing stop hit
- [ ] Add `CheckEmergencyExit()` - floor-hit >30%, climax top detection
- [ ] Create `ExitDecision` type with reason, percentage, signal type

### Signal Types Extension

- [ ] Add `SELL_TARGET1`, `SELL_TARGET2`, `SELL_TARGET3`, `SELL_EMERGENCY` to `SignalType` enum in `internal/signals/types.go`
- [ ] Update signal type string mappings
- [ ] Add exit signal struct `ExitSignal` with target info

## Phase 2: Workflow Integration (Week 2)

### Daily Signal Workflow

- [ ] Update `cmd/signals/main.go` - add exit evaluation before signal generation
- [ ] Create `evaluateExits()` function - loops through all open positions
- [ ] Call `ExitEvaluator.EvaluatePosition()` for each position
- [ ] Generate exit signals and store in memory
- [ ] Add exit signals to daily output

### Position Partial Exit Logic

- [ ] Create `internal/position/scaler.go` for position scaling logic
- [ ] Implement `CalculateExitShares(position, targetLevel)` - computes 30%/30%/40%
- [ ] Handle rounding for share quantities
- [ ] Validate exit shares <= remaining shares

### API Order Execution

- [ ] Create `internal/trading/exit_executor.go`
- [ ] Implement `ExecuteExit(position, exitDecision)` method
- [ ] Call DNSE sell order API
- [ ] Handle partial fill responses
- [ ] Implement retry logic (3 attempts, exponential backoff)
- [ ] Log all execution attempts

### Database Updates

- [ ] Update position record after successful exit
  - Set `target1_filled = TRUE` and `target1_exit_price`, `target1_exit_date`
  - Reduce `current_shares` by exit quantity
  - Update `current_value` based on remaining shares
- [ ] Handle transaction rollback on API failures

## Phase 3: Enhancements & Validation (Week 2-3)

### Vietnamese Market Rules Integration

- [ ] Add T+2 settlement check - intraday settlement at 12:30pm on T+2
- [ ] Update liquidity constraint check - 0.5% of daily volume for most stocks, 1.5% for VN30
- [ ] Implement multi-day exit splitting for large positions
- [ ] Add consecutive floor-hit tracking (3+ days triggers full exit)
- [ ] Add VN-Index drop detector (>3% triggers position evaluation)
- [ ] Add market regime + floor-hit combination check (Bear + 20% → full exit)

### Vietnamese Seasonal & Structural Rules

- [ ] Create `internal/vn/tet_calendar.go` for Lunar New Year dates
- [ ] Implement `CheckTetHolidayAdjustment()` - returns reduction signal 7-10 days before Tet
- [ ] Add automated position reduction to 50% week before Tet
- [ ] Create `internal/vn/soe_classifier.go` for state-owned enterprise identification
- [ ] Implement SOE-specific allocation: 30%/40%/30% (vs normal 30%/30%/40%)
- [ ] Add psychological price level detector (10k, 20k, 50k, 100k VND)
- [ ] Implement target adjustment to stay below round-number resistance

### Telegram Notifications

- [ ] Create `internal/service/telegram/exit_alerts.go`
- [ ] Implement `NotifyTargetExit(position, target, price, percentGain)`
- [ ] Send notification on successful exit
- [ ] Include: symbol, target level, shares sold, price, profit %
- [ ] Add summary of remaining position and next target

### Testing

- [ ] Unit test `ExitEvaluator` with mock positions
  - Test target1 trigger at 15% profit
  - Test target1 trigger at resistance level
  - Test target2 requires both 25% AND cleared resistance
  - Test target3 trailing stop logic
  - Test emergency exit overrides all targets
- [ ] Integration test database migrations
- [ ] Integration test exit executor with DNSE API (use test mode)
- [ ] Test position scaling calculations for various share counts

### Validation & Backtest

- [ ] Create `test/backtest_exits.go` for historical validation
- [ ] Load CSV of past trades
- [ ] Simulate exit decisions using historical price data
- [ ] Compare automated exits vs actual manual exits
- [ ] Measure: profit capture %, average hold time, max drawdown
- [ ] Document results in `openspec/changes/implement-graduated-exits/backtest_results.md`

## Phase 4: Documentation & Deployment (Week 3)

### Documentation

- [ ] Update `README.md` - add Exit Decision Engine section
- [ ] Document exit configuration in `docs/CONFIGURATION.md`
- [ ] Create `docs/EXIT_STRATEGY.md` explaining graduated exit logic
- [ ] Add examples to `docs/EXAMPLES.md` showing exit workflow

### Deployment

- [ ] Run database migration in production
- [ ] Deploy updated `cmd/signals/main.go`
- [ ] Monitor first 5 days of automated exits
- [ ] Create alert for execution failures
- [ ] Validate Telegram notifications working

### OpenSpec Finalization

- [ ] Mark all tasks complete
- [ ] Run `openspec validate implement-graduated-exits --strict`
- [ ] Resolve any validation errors
- [ ] Archive change with `openspec archive implement-graduated-exits`

---

**Total Tasks:** 60 (increased from 45 to include Vietnamese market adaptations)
**Estimated Duration:** 3-4 weeks (increased from 2.5-3 weeks)
**Parallelizable:** Database/Core + Board Lot handling can be done simultaneously
