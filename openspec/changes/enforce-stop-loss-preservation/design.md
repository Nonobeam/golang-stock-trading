## Context

The system allows multiple purchases of the same stock (averaging in), but lacks enforcement to prevent stop-loss modifications after position entry. Risk calculations also use a simplified formula that doesn't accurately compute risk when multiple entries exist at different prices against a single stop level.

### Current Behavior

1. **Stop-Loss Drift**: `position_repository.go:Update()` allows stop_loss updates without validation
2. **Simplified Risk Formula**: `manager.py:358` uses `total_shares * (first_entry_price - stop_loss)` instead of per-entry aggregation
3. **Binary Capacity**: Only checks at_limit (true/false), no warnings before hitting limits

### Problems

- Users can accidentally widen stops, increasing risk beyond 2% limit
- Multi-entry positions calculate risk incorrectly (underestimates or overestimates depending on entry prices)
- No advance warning when position approaches 20% capacity limit

### Stakeholders

- **Traders**: Need protection from accidental risk increases
- **System**: Must enforce 2% per-trade risk limit
- **Compliance**: Audit trail for any stop-loss modifications

## Goals / Non-Goals

### Goals

1. **Lock stop-loss at first entry** - Immutable baseline stop level
2. **Accurate multi-entry risk** - Sum risk per entry, not averaged
3. **Three-tier capacity** - Normal, approaching (warning), at limit (block)
4. **Daily integrity checks** - Catch manual modifications
5. **Override mechanism** - Allow exceptions with audit trail

### Non-Goals

- Automatic stop-loss adjustment (trailing stops) - Future enhancement
- Position-level risk limits (portfolio-wide risk management) - Separate concern
- Dynamic capacity limits based on volatility - Out of scope
- Historical stop-loss tracking (audit log of all changes) - Only need current + reason

## Decisions

### Decision 1: Separate first_entry_stop_loss from stop_loss

**What**: Add `first_entry_stop_loss` column (immutable) separate from `stop_loss` (current/active)

**Why**:
- Preserves original risk thesis without preventing future flexibility
- Enables validation without breaking existing workflows
- Supports audit/review of any deviations

**Alternatives Considered**:
- Make `stop_loss` itself immutable → Too rigid, no edge case handling
- Store all stop changes in audit table → Over-engineering for current need

### Decision 2: Per-entry risk aggregation

**What**: Change risk formula from `shares * (avg_price - stop)` to `sum(entry_shares[i] * (entry_price[i] - stop))`

**Why**:
- Mathematically correct for multiple entries at different prices
- Example: 100 @ 36,850 + 50 @ 38,000, stop 35,100
  - Correct: (100 × 1,750) + (50 × 2,900) = 320,000
  - Incorrect: 150 × average_delta = varies, inaccurate

**Alternatives Considered**:
- Use average cost for all calculations → Simpler but mathematically wrong
- Track risk per entry separately → Complex, doesn't aggregate for limit checks

### Decision 3: Three-tier capacity (normal, soft warning, hard block)

**What**: Add APPROACHING_LIMIT tier at 18% value / 0.8% volume with warnings

**Why**:
- Gives traders advance notice before hitting limits
- Allows position building up to limit with awareness
- Prevents surprise blocking of BUY_MORE signals

**Alternatives Considered**:
- Binary (at limit or not) → Current state, too abrupt
- Graduated tiers (multiple warning levels) → Over-complex for benefit
- Configurable thresholds per stock → Adds complexity, minimal value

### Decision 4: Stop-loss lock with override

**What**: `stop_loss_locked = TRUE` by default, can be unlocked with required reason

**Why**:
- Protects against accidental modifications
- Allows intentional changes for edge cases (e.g., trailing stops manually)
- Audit trail via `stop_loss_override_reason`

**Alternatives Considered**:
- Fully immutable (no override) → Too rigid
- Warn but allow → Ineffective, users will ignore warnings
- Require admin approval → Over-process for single-user system

### Decision 5: Daily validation pre-flight check

**What**: Run `validate_stop_loss_integrity()` before signal generation, block if violations

**Why**:
- Catches manual database edits or bugs
- Prevents compounding bad data through automated signals
- Forces review/correction before continuing

**Alternatives Considered**:
- Alert only (don't block) → Risk of accumulating violations
- Validate on trade execution → Too late, position already sized wrong
- Real-time validation on every read → Performance overhead

## Risks / Trade-offs

### Risk 1: Backfill accuracy

**Risk**: Calculated `first_entry_stop_loss` may not match actual historical stop if manually changed

**Mitigation**:
- Backfill script validates against current `stop_loss` (within 1% tolerance)
- Flag deviations for manual review before enforcement
- Document expected vs actual for each flagged position

### Risk 2: Legitimate stop-loss changes blocked

**Trade-off**: Locking stops prevents both accidental AND intentional changes

**Mitigation**:
- Override mechanism with required reason
- Document common edge cases in README
- Low friction to unlock if needed (set boolean + provide reason)

### Risk 3: Risk calculation change reduces capacity

**Impact**: Positions with entries above average will show higher risk, reducing buying capacity

**Example**: First entry 36,850, second entry 40,000, avg 38,425, stop 35,100
- Old: 150 × (38,425 - 35,100) = 498,750 risk
- New: 100 × (36,850 - 35,100) + 50 × (40,000 - 35,100) = 420,000 risk
- Actually, new formula can be higher or lower depending on entry prices

**Mitigation**:
- Run simulation on existing positions to quantify impact
- If needed, adjust soft limit to compensate (e.g., 18.5% instead of 18%)
- Document change in migration guide

### Risk 4: Daily validation blocks trading

**Risk**: Spurious validation failures prevent signal generation

**Mitigation**:
- 1% tolerance for rounding differences
- Clear error messages indicating which position and deviation
- Manual override to skip validation if needed (emergency escape hatch)

## Migration Plan

### Phase 1: Database (Week 1)

1. Create and test migration in dev environment
2. Backfill script development and testing
3. Run backfill on production replica, review flagged positions
4. Deploy migration to production
5. Execute backfill, verify results

### Phase 2: Python Validation (Week 2)

1. Deploy risk calculation changes to ml-service
2. Deploy capacity warning tiers
3. Monitor for capacity warnings in daily signals
4. Validate risk calculations match expectations

### Phase 3: Go Enforcement (Week 3)

1. Deploy repository validation
2. Test stop-loss modification rejection in staging
3. Deploy to production
4. Monitor for validation errors

### Phase 4: Daily Checks (Week 3)

1. Deploy daily validation workflow
2. Run manually first to verify no false positives
3. Enable automatic blocking on violations
4. Set up alerts for violations

### Rollback

- **Database**: Run down migration (loses new columns but doesn't break existing)
- **Python**: Revert commits, old formula still works (may be inaccurate but functional)
- **Go**: Revert validation, allows stop changes again
- **Daily**: Disable validation check, signals continue

Emergency escape:
```sql
UPDATE positions SET stop_loss_locked = FALSE WHERE stop_loss_locked = TRUE;
```

## Open Questions

1. **Stop-loss tolerance**: Is 1% appropriate or should it be tighter (0.5%) or looser (2%)?
   - **Decision needed before**: Backfill validation

2. **Soft limit thresholds**: Are 18% value / 0.8% volume the right warning points?
   - **Decision needed before**: Python capacity updates

3. **Daily validation**: Should it block signals or just alert?
   - **Decision needed before**: Daily workflow integration

4. **Override workflow**: Should unlocking stop_loss require additional confirmation beyond reason text?
   - **Decision needed before**: Go repository validation

5. **Backfill conflicts**: How to handle positions where calculated first_entry_stop doesn't match current stop?
   - Options: (A) Use current stop as first_entry_stop, (B) Use calculated stop and flag for review, (C) Manual review all mismatches
   - **Decision needed before**: Running backfill script
