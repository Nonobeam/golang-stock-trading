# Proposal: Implement Graduated Exit Decision Engine

## Change ID

`implement-graduated-exits`

## Why

The trading system calculates target prices and has trailing stop logic, but lacks automated exit execution:

- ✅ Target prices (Target1, Target2, Target3) are calculated and tracked in positions
- ✅ Trailing stops are fully implemented (`internal/risk/targets_trailing.go`)
- ✅ Target hit notifications exist in Telegram bot
- ❌ **No automated exit execution** - positions require manual selling
- ❌ **No graduated position scaling** - no 30%/30%/40% allocation logic
- ❌ **No exit decision engine** - no workflow to evaluate exit conditions

**Impact:** Traders must manually monitor positions and execute exits, missing optimal profit-taking opportunities and risking drawdowns from delayed exits.

## Proposed Solution

Implement an **Exit Decision Engine** that automates graduated profit-taking with three levels:

### Target System

1. **Target 1 (30% position):** Conservative exit at +15% profit OR first resistance level
2. **Target 2 (30% position):** Expected value exit at +25% profit AND price clears first resistance
3. **Target 3 (40% position):** Trailing stop exit using existing trailing stop system

### Components

**1. Exit Decision Engine** (`internal/position/exit_engine.go`)

- Evaluates positions against target thresholds
- Generates exit signals with reason codes
- Handles emergency exit conditions (floor-hit >30%, climax tops)

**2. Signal Types** (extend `internal/signals/types.go`)

- `SELL_TARGET1` - 30% exit at first target
- `SELL_TARGET2` - 30% exit at second target
- `SELL_TARGET3` - 40% trailing stop exit
- `SELL_EMERGENCY` - Immediate full exit

**3. Database Extensions**

```sql
ALTER TABLE positions ADD COLUMN target1_filled BOOLEAN DEFAULT FALSE;
ALTER TABLE positions ADD COLUMN target2_filled BOOLEAN DEFAULT FALSE;
ALTER TABLE positions ADD COLUMN trailing_stop_active BOOLEAN DEFAULT FALSE;
```

**4. Daily Workflow Integration** (`cmd/signals/main.go`)

- Add exit evaluation step before signal generation
- Check all open positions for exit conditions
- Execute partial sells via DNSE API

## User-Visible Changes

### Before

- User manually monitors Telegram alerts for target hits
- Manual execution of exits via trading platform
- No systematic profit-taking discipline

### After

- System automatically executes exits at targets
- Telegram notification: "✅ VCI: Sold 30% at Target 1 (36,500 VND, +15.2%)"
- Position tracking shows filled targets: `T1 ✓ at 36.5k | T2 pending | T3 trailing`
- Daily signal output includes exit recommendations

## Risks & Mitigations

| Risk                   | Mitigation                                                          |
| ---------------------- | ------------------------------------------------------------------- |
| API execution failures | Retry logic with exponential backoff, fallback to user notification |
| Partial fill issues    | Track actual shares sold vs intended, adjust remaining targets      |
| T+2 settlement impact  | Check if Thursday/Friday, adjust exit sizes per `take_profit.md`    |
| Liquidity constraints  | Respect 0.5% daily volume limit, split exits across days if needed  |

## Dependencies

- ✅ Existing trailing stop system (`targets_trailing.go`)
- ✅ Target calculation logic in signal generators
- ✅ DNSE trading API for order execution
- ⚠️ Need `migrate-telegram-multiuser` to access correct chat IDs for notifications

## Success Criteria

1. ✅ Exit engine evaluates all positions daily
2. ✅ Automated exits execute within 5 minutes of target hit
3. ✅ 95%+ execution success rate for exit orders
4. ✅ Telegram notifications sent for all exits
5. ✅ Position database accurately reflects target fills
6. ✅ Backtest shows improved profit capture vs manual exits

## Out of Scope

- ML prediction integration (Phase 3)
- Advanced pattern detection (Phase 2)
- Performance tracking dashboard (future enhancement)
- Custom target levels per user (uses fixed 15%/25%/trailing)

## Vietnamese Market Adaptations (2025-2026)

Based on current HOSE/HNX trading conditions:

### Critical Requirements

1. **Ceiling-Hit Exit Logic** - Stocks hitting +7% ceiling can lock for 3-5 days
   - Execute Target1 (30%) immediately when ceiling hit with 3x volume
   - Tighten trailing stop to 3% for remainder
2. **Board Lot Rounding**
   - HOSE: Round to 10-share lots
   - HNX: Round to 100-share lots
   - Handle odd lots on final exit
3. **Foreign Ownership Limit (FOL) Monitoring**
   - Emergency exit if FOL >95% of limit (49-100% depending on stock)
   - Accelerate exits if FOL >85%
4. **T+2 Intraday Settlement** (Post-KRX launch May 2025)
   - Settlement completes 12:30pm on T+2
   - Execute before 11am for same-day cash availability
   - Optimize for liquidity windows: 10am, 1pm, 2:15pm
5. **Vietnamese Emergency Exits**
   - Consecutive floor hits (3+ days) → Full exit
   - VN-Index drops >3% intraday → Evaluate all positions
   - Market regime = Bear + floor-hit >20% → Full exit

### Seasonal & Structural

6. **Tet Holiday Adjustment** - Reduce all positions to 50% week before Lunar New Year (7-10 day closure)

7. **SOE-Specific Rules** - State-owned enterprises have audit suspension risk
   - Adjusted allocation: 30%/40%/30% (vs normal 30%/30%/40%)
   - Take more profit earlier, less trailing exposure

8. **Psychological Price Levels** - Adjust targets to avoid Vietnamese round-number resistance (10k, 20k, 50k, 100k VND)

### Updated Success Criteria

- ✅ Board lot rounding errors <1% of intended size
- ✅ Zero exits missed due to FOL restrictions
- ✅ Ceiling-hit exits executed within 2 sessions
- ✅ Average slippage <0.3% (Vietnamese market standard)

## Implementation Estimate

**Effort:** 2-3 weeks (20-25 tasks)

**Phases:**

1. Database schema & exit engine core (5 days)
2. Daily workflow integration & API calls (5 days)
3. Testing & validation (3-5 days)
4. Documentation & deployment (2 days)

## Open Questions for User

1. **Emergency Exits:** Should floor-hit >30% override all targets and exit 100% immediately?
2. **Friday Exits:** T+2 now settles intraday (12:30pm) - still need 50% reduction on Thursday/Friday entries?
3. **Notifications:** Want detailed Telegram alerts per exit or daily summary only?
4. **Backtest Data:** Have CSV of historical trades to validate exit logic?
5. **Ceiling-Hit Strategy:** Execute 30% immediately when hitting +7% ceiling, or wait for next-day confirmation?
6. **FOL Monitoring:** Real-time API calls (expensive) or daily batch updates?
7. **Odd Lot Handling:** For HNX 100-share lots, accept poor liquidity on odd lots or force round multiples?
8. **SOE Identification:** Manual list of state-owned enterprises or automated ownership API check?

---

**Next Step:** Upon approval, create detailed `tasks.md` with 20-25 granular implementation tasks.
