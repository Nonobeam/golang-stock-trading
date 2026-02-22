# Design: Graduated Exit Decision Engine

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                   Daily Signal Workflow                      │
│                  (cmd/signals/main.go)                       │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ├─→ 1. Fetch Open Positions
                         │
                         ├─→ 2. Exit Evaluation
                         │    ┌──────────────────────────┐
                         │    │   ExitEvaluator          │
                         │    │  (exit_engine.go)        │
                         │    │                          │
                         │    │  ┌──────────────────┐   │
                         │    │  │ Check Target 1   │   │
                         │    │  │  ≥15% OR resist  │   │
                         │    │  └──────────────────┘   │
                         │    │  ┌──────────────────┐   │
                         │    │  │ Check Target 2   │   │
                         │    │  │ ≥25% AND cleared │   │
                         │    │  └──────────────────┘   │
                         │    │  ┌──────────────────┐   │
                         │    │  │ Check Target 3   │   │
                         │    │  │ Trailing stop    │   │
                         │    │  └──────────────────┘   │
                         │    │  ┌──────────────────┐   │
                         │    │  │ Emergency exits  │   │
                         │    │  │ Floor/Climax     │   │
                         │    │  └──────────────────┘   │
                         │    └──────────────────────────┘
                         │                │
                         ├─→ 3. Execute Exits
                         │    ┌──────────────────────────┐
                         │    │   ExitExecutor           │
                         │    │  (exit_executor.go)      │
                         │    │                          │
                         │    │  • Calculate shares      │
                         │    │  • DNSE API sell order   │
                         │    │  • Handle partial fills  │
                         │    │  • Retry logic           │
                         │    └──────────────────────────┘
                         │                │
                         ├─→ 4. Update Database
                         │    ┌──────────────────────────┐
                         │    │  positions table         │
                         │    │  • target1_filled        │
                         │    │  • target2_filled        │
                         │    │  • current_shares        │
                         │    │  • exit prices/dates     │
                         │    └──────────────────────────┘
                         │                │
                         └─→ 5. Notify User
                              ┌──────────────────────────┐
                              │  Telegram Alerts         │
                              │  "✅ VCI: Sold 30% at    │
                              │   Target 1 (36.5k +15%)" │
                              └──────────────────────────┘
```

## Data Flow

### Position Evaluation Flow

```
Position
  ├─ entry_price: 30000
  ├─ current_price: 34500 (from market data)
  ├─ current_shares: 100
  ├─ target1: 34500 (+15%)
  ├─ target2: 37500 (+25%)
  ├─ target1_filled: false
  └─ target2_filled: false
       │
       ▼
ExitEvaluator.EvaluatePosition()
       │
       ├─ IF !target1_filled AND (profit >= 15% OR price >= resistance1)
       │    → ExitDecision{type: SELL_TARGET1, percentage: 30%, shares: 30}
       │
       ├─ IF target1_filled AND !target2_filled AND (profit >= 25% AND price > resistance1)
       │    → ExitDecision{type: SELL_TARGET2, percentage: 30%, shares: 30}
       │
       └─ IF target1_filled AND target2_filled AND trailing_stop_hit
            → ExitDecision{type: SELL_TARGET3, percentage: 100%, shares: 40}
```

### Emergency Exit Override

```
IF floor_hit_probability > 30%:
  → SELL_EMERGENCY (100% of position, ignore all targets)

IF climax_top_detected (gap_up + volume_spike + reversal_candle):
  → SELL_EMERGENCY (100% of position)
```

## Component Responsibilities

### 1. ExitEvaluator (`internal/position/exit_engine.go`)

**Responsibility:**決定是否應該退出以及退出多少

**Methods:**

- `EvaluatePosition(pos *Position, marketData *MarketData) *ExitDecision`
- `ShouldExitTarget1(pos *Position) bool`
- `ShouldExitTarget2(pos *Position) bool`
- `ShouldExitTarget3(pos *Position, trailingStop float64) bool`
- `CheckEmergencyExit(pos *Position, floorProb float64) *ExitDecision`

**Dependencies:**

- Reads: `Position`, market data, resistance levels
- Writes: `ExitDecision` structs (no DB writes)

### 2. ExitExecutor (`internal/trading/exit_executor.go`)

**Responsibility:** 執行退出決定 - API calls, retries, error handling

**Methods:**

- `ExecuteExit(decision *ExitDecision) error`
- `placeSellOrder(symbol string, shares int, price float64) (*OrderResponse, error)`
- `retryWithBackoff(fn func() error, maxRetries int) error`

**Dependencies:**

- DNSE trading API client
- Telegram notification service
- Database for position updates

### 3. PositionScaler (`internal/position/scaler.go`)

**Responsibility:** 計算退出股數 (30%/30%/40% logic)

**Methods:**

- `CalculateExitShares(position *Position, targetLevel int) int`
- `CalculateRemainingShares(position *Position) int`

**Logic:**

```go
func CalculateExitShares(pos *Position, targetLevel int) int {
    if targetLevel == 1 && !pos.Target1Filled {
        return int(float64(pos.InitialShares) * 0.30) // 30%
    }
    if targetLevel == 2 && pos.Target1Filled && !pos.Target2Filled {
        return int(float64(pos.InitialShares) * 0.30) // 30%
    }
    if targetLevel == 3 && pos.Target1Filled && pos.Target2Filled {
        return pos.CurrentShares // Remaining 40%
    }
    return 0
}
```

## Database Schema

### Positions Table Extensions

```sql
ALTER TABLE positions ADD COLUMN IF NOT EXISTS target1_filled BOOLEAN DEFAULT FALSE;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS target2_filled BOOLEAN DEFAULT FALSE;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS trailing_stop_active BOOLEAN DEFAULT FALSE;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS target1_exit_price DECIMAL(10,2);
ALTER TABLE positions ADD COLUMN IF NOT EXISTS target2_exit_price DECIMAL(10,2);
ALTER TABLE positions ADD COLUMN IF NOT EXISTS target1_exit_date TIMESTAMP;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS target2_exit_date TIMESTAMP;
```

### Example Position Record Lifecycle

```
Initial Entry:
  current_shares: 100
  target1_filled: false
  target2_filled: false

After Target 1 Exit:
  current_shares: 70        (100 - 30)
  target1_filled: true
  target1_exit_price: 34500
  target1_exit_date: 2026-02-15 10:30:00
  target2_filled: false

After Target 2 Exit:
  current_shares: 40        (70 - 30)
  target1_filled: true
  target2_filled: true
  target2_exit_price: 37500
  target2_exit_date: 2026-02-18 14:15:00

After Target 3 Exit (trailing stop):
  current_shares: 0         (40 - 40)
  target1_filled: true
  target2_filled: true
  trailing_stop_active: false
  status: CLOSED
```

## Error Handling Strategy

### API Execution Failures

```
ExecuteExit() failure scenarios:
  1. Network timeout
     → Retry 3x with exponential backoff (1s, 2s, 4s)
     → If all fail: Log error, send Telegram alert, DO NOT update DB

  2. Partial fill (requested 30 shares, only 20 filled)
     → Update DB with actual filled quantity
     → Recompute remaining targets proportionally
     → Notify user of partial execution

  3. Order rejected (insufficient balance, stock halted)
     → Log rejection reason
     → Send Telegram alert with reason
     → Mark position for manual review
```

### Database Transaction Safety

```go
tx := db.Begin()
defer tx.Rollback() // Rollback if not committed

// 1. Check current position state
pos := tx.FindPosition(id)

// 2. Execute API order
response, err := executor.ExecuteExit(decision)
if err != nil {
    return err // Rollback triggered
}

// 3. Update position with actual execution
pos.CurrentShares -= response.FilledShares
pos.Target1Filled = true
pos.Target1ExitPrice = response.AvgPrice
tx.Save(pos)

// 4. Commit transaction
tx.Commit()
```

## Vietnamese Market Constraints

### T+2 Settlement Impact

```go
func AdjustExitSize(decision *ExitDecision, entryDate time.Time) {
    dayOfWeek := time.Now().Weekday()

    // Thursday/Friday entries lock capital over weekend
    if dayOfWeek == time.Thursday || dayOfWeek == time.Friday {
        decision.Percentage *= 0.50 // Reduce to 50% of planned exit
        decision.Shares = int(float64(decision.Shares) * 0.50)
        decision.Reason += " (T+2 weekend adjustment)"
    }
}
```

### Liquidity Constraints

```go
func ValidateLiquidity(shares int, dailyVolume int64) error {
    maxShares := int(float64(dailyVolume) * 0.005) // 0.5% limit

    if shares > maxShares {
        return fmt.Errorf("exit size %d exceeds 0.5%% daily volume limit (%d)",
                          shares, maxShares)
    }
    return nil
}
```

## Configuration

### Exit Thresholds

```yaml
exit_engine:
  target1_profit_percent: 15.0
  target2_profit_percent: 25.0
  emergency_floor_threshold: 30.0
  max_retries: 3
  retry_backoff_ms: [1000, 2000, 4000]
  liquidity_limit_percent: 0.5
  t2_weekend_reduction: 0.50
```

## Success Metrics

**Operational:**

- Exit evaluation latency < 100ms per position
- API execution success rate > 95%
- Notification delivery < 5 seconds after exit

**Financial:**

- Profit capture ratio: (realized profit / max potential profit) > 70%
- Average hold time reduction vs manual exits: -15%
- Drawdown from peak: < 8% (vs manual ~12%)

## Future Enhancements (Out of Scope)

1. **Custom Target Levels:** Per-user configuration of target percentages
2. **ML-Based Targets:** Dynamic targets based on ML predictions
3. **Pattern-Adjusted Exits:** Earlier exits on bearish patterns
4. **Multi-Symbol Correlation:** Coordinate exits across correlated positions
