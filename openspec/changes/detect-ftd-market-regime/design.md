# Follow-Through Day Detection: Technical Design

## Architecture Overview

### System Context

FTD detection operates as a **market regime monitoring system** that runs continuously during market hours, analyzing VN-Index data to identify transition points from bearish to bullish sentiment.

### Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                   VN-Index Data Stream                       │
│              (WebSocket + Daily OHLCV API)                   │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│         internal/regime/ftd/detector.go                      │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  1. DowntrendDetector                                 │  │
│  │     - RSI < 30 monitoring                            │  │
│  │     - Support level tests                            │  │
│  │     - New low tracking                               │  │
│  └────────────────────┬─────────────────────────────────┘  │
│                       │                                      │
│  ┌────────────────────▼─────────────────────────────────┐  │
│  │  2. RallyAttemptTracker (State Machine)              │  │
│  │     Day 1: First close higher after new low          │  │
│  │     Days 2-3: Price hold + volume depletion check    │  │
│  │     Days 4-7: FTD confirmation window                │  │
│  └────────────────────┬─────────────────────────────────┘  │
│                       │                                      │
│  ┌────────────────────▼─────────────────────────────────┐  │
│  │  3. FTDConfirmation                                  │  │
│  │     - Price gain > 1.2% check                        │  │
│  │     - Volume surge validation                        │  │
│  │     - Market breadth confirmation                    │  │
│  └────────────────────┬─────────────────────────────────┘  │
│                       │                                      │
│  ┌────────────────────▼─────────────────────────────────┐  │
│  │  4. FTDScorer                                         │  │
│  │     Score = Price(30) + Volume(30) +                │  │
│  │             Breadth(20) + Leaders(20)               │  │
│  └────────────────────┬─────────────────────────────────┘  │
└───────────────────────┼─────────────────────────────────────┘
                        │
        ┌───────────────┴──────────────┐
        │                              │
        ▼                              ▼
┌──────────────────┐          ┌──────────────────┐
│  PostgreSQL DB   │          │ PortfolioManager │
│  - regime data   │          │ Risk Adjustment  │
│  - ftd_events    │          │ - Size: 1.0x→1.5x│
│  - breadth data  │          │ - Positions: 6→8 │
└──────────────────┘          │ - Corr: .85→.90  │
                              └────────┬─────────┘
                                       │
                                       ▼
                              ┌──────────────────┐
                              │  Telegram Alerts │
                              │  - Day 1 detected│
                              │  - FTD confirmed │
                              │  - /ftd_status   │
                              └──────────────────┘
```

## Data Model

### market_regime_tracking

```sql
CREATE TABLE market_regime_tracking (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL UNIQUE,
    index_value DECIMAL(10,2) NOT NULL,
    volume BIGINT NOT NULL,
    volume_vs_avg_20d DECIMAL(5,2), -- Percentage
    rally_attempt_day INT, -- NULL, 1, 2, 3, 4-7
    is_ftd BOOLEAN DEFAULT FALSE,
    ftd_strength VARCHAR(20), -- 'weak', 'moderate', 'strong'
    breadth_ratio DECIMAL(5,2), -- advancing / declining
    leader_participation BOOLEAN,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### ftd_events

```sql
CREATE TABLE ftd_events (
    id SERIAL PRIMARY KEY,
    event_date DATE NOT NULL,
    rally_attempt_start_date DATE NOT NULL,
    days_to_ftd INT NOT NULL, -- How many days from Day 1 to FTD
    ftd_strength_score INT NOT NULL, -- 0-100
    pattern_type VARCHAR(50), -- 'island_reversal', 'double_bottom', 'supply_test', etc.
    success_7d DECIMAL(5,2), -- % gain 7 days later
    success_14d DECIMAL(5,2),
    success_30d DECIMAL(5,2),
    is_validated BOOLEAN DEFAULT FALSE, -- Updated after 30 days
    invalidated_by_distribution BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### market_breadth_daily

```sql
CREATE TABLE market_breadth_daily (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL UNIQUE,
    advancing_stocks INT NOT NULL,
    declining_stocks INT NOT NULL,
    unchanged_stocks INT,
    new_highs INT,
    new_lows INT,
    sector_leaders JSONB, -- {"banks": +2.3%, "securities": +1.8%, "steel": +0.5%}
    created_at TIMESTAMP DEFAULT NOW()
);
```

## State Machine Logic

### Rally Attempt States

```
[DOWNTREND] --(new low)--> [WATCHING]
    │
    └─(close higher)─> [DAY_1]
                          │
                          ├─(hold above Day 1 low)─> [DAY_2]
                          │                             │
                          │                             └─(hold + volume↓)─> [DAY_3]
                          │                                                     │
                          │                                                     └─> [DAY_4_TO_7_WINDOW]
                          │                                                            │
                          │                                                            ├─(FTD criteria met)─> [FTD_CONFIRMED]
                          │                                                            │
                          │                                                            └─(criteria not met by Day 7)─> [FAILED]
                          │
                          └─(break Day 1 low)─> [RESET to WATCHING]
```

## FTD Scoring Algorithm

### Components (Total: 100 points)

```go
type FTDScore struct {
    PriceScore      int // 0-30 points
    VolumeScore     int // 0-30 points
    BreadthScore    int // 0-20 points
    LeaderScore     int // 0-20 points
    Total           int // 0-100
    Strength        string // "weak", "moderate", "strong"
}

func CalculateFTDScore(data FTDData) FTDScore {
    score := FTDScore{}

    // Price Score (30 points max)
    priceGain := data.CloseToday/data.CloseYesterday - 1.0
    switch {
    case priceGain >= 0.020: // 2%+
        score.PriceScore = 30
    case priceGain >= 0.015: // 1.5%-2%
        score.PriceScore = 25
    case priceGain >= 0.012: // 1.2%-1.5%
        score.PriceScore = 20
    default:
        score.PriceScore = 0
    }

    // Volume Score (30 points max)
    volumeRatio := data.VolumeToday / data.AvgVolume20d
    switch {
    case volumeRatio >= 2.0: // 2x average
        score.VolumeScore = 30
    case volumeRatio >= 1.5: // 1.5x average
        score.VolumeScore = 25
    case volumeRatio >= 1.2: // 1.2x average
        score.VolumeScore = 20
    case volumeRatio > 1.0: // Above average
        score.VolumeScore = 10
    }

    // Breadth Score (20 points max)
    breadthRatio := float64(data.AdvancingStocks) / float64(data.DecliningStocks)
    switch {
    case breadthRatio >= 3.0: // 3:1 or better
        score.BreadthScore = 20
    case breadthRatio >= 2.0: // 2:1
        score.BreadthScore = 15
    case breadthRatio >= 1.5: // 1.5:1
        score.BreadthScore = 10
    default:
        score.BreadthScore = 0
    }

    // Leader Participation (20 points max)
    leaderCount := 0
    sectors := []string{"banks", "securities", "steel"}
    for _, sector := range sectors {
        if data.SectorPerformance[sector] > 0.01 { // Sector up >1%
            leaderCount++
        }
    }
    score.LeaderScore = leaderCount * (20 / len(sectors))

    // Total and Strength
    score.Total = score.PriceScore + score.VolumeScore + score.BreadthScore + score.LeaderScore
    switch {
    case score.Total >= 80:
        score.Strength = "strong"
    case score.Total >= 60:
        score.Strength = "moderate"
    default:
        score.Strength = "weak"
    }

    return score
}
```

## Integration Points

### 1. Portfolio Manager Risk Adjustment

```go
// In internal/risk/portfolio_manager.go
func (pm *PortfolioManager) OnFTDConfirmed(ftdScore int) {
    pm.ftdActive = true
    pm.ftdStrength = ftdScore

    // Adjust risk limits based on FTD strength
    if ftdScore >= 80 { // Strong FTD
        pm.positionSizeMultiplier = 1.5
        pm.Limits.MaxPositions = 8
        pm.Limits.MaxPairwiseCorrelation = 0.90
    } else if ftdScore >= 60 { // Moderate FTD
        pm.positionSizeMultiplier = 1.25
        pm.Limits.MaxPositions = 7
        pm.Limits.MaxPairwiseCorrelation = 0.87
    }

    logger.Info().
        Int("ftd_score", ftdScore).
        Float64("multiplier", pm.positionSizeMultiplier).
        Msg("FTD confirmed - risk parameters adjusted")
}

func (pm *PortfolioManager) OnFTDInvalidated() {
    pm.ftdActive = false
    pm.positionSizeMultiplier = 1.0
    pm.Limits = DefaultRiskLimits() // Revert to defaults

    logger.Warn().Msg("FTD invalidated by distribution - reverting to conservative mode")
}
```

### 2. ML Service Regime Indicator

FTD status can be passed to Python ML service as a regime indicator via gRPC:

```protobuf
message MarketRegime {
    string regime_type = 1; // "bull", "bear", "range", "ftd_confirmed"
    int32 ftd_strength = 2; // 0-100 if FTD active
    bool ftd_active = 3;
}
```

## Performance Considerations

### Efficiency

- FTD detection runs **once per day** after market close (not real-time during trading)
- Database queries use indexes on `date` column
- State machine stored in memory, persisted to DB at EOD

### Scalability

- Single VN-Index monitoring (not per-stock), minimal compute overhead
- Breadth calculation limited to watchlist stocks (~50-100)
- Historical validation runs in background batch job

## Error Handling

### Data Availability

- If market breadth data unavailable, FTD score capped at 60 (no breadth/leader points)
- Missing volume data → postpone FTD check to next day
- VN-Index data gap → reset rally attempt counter

### False Positives

- Require consecutive validation: FTD remains "pending" for 2 trading days
- Monitor for distribution days (down day on volume) → auto-invalidate FTD

## Testing Strategy

### Unit Tests

- State machine transitions (all edge cases)
- FTD scorer with various input combinations
- Pattern recognizers (island reversal, double bottom)

### Integration Tests

- End-to-end rally attempt tracking over 10-day window
- Risk parameter adjustment on FTD events
- Database persistence and retrieval

### Historical Validation

- Backtest on 2023-2025 VN-Index data
- Calculate precision/recall for FTD detection
- Measure average post-FTD gains
