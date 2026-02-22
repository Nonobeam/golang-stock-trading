# Follow-Through Day (FTD) Detection

## Why

Without systematic detection of market turning points, the trading system remains in defensive mode too long after corrections end, missing optimal entry opportunities during market recoveries. FTD detection provides a proven technical signal for regime transitions, enabling the system to shift from conservative to aggressive positioning at precisely the right moment. This maximizes returns during bull market resumptions while maintaining discipline during ongoing downtrends.

## Problem

The system currently lacks the ability to detect market regime changes from bearish to bullish, specifically Follow-Through Days (FTD) - a proven technical pattern that signals the end of corrections and start of sustained rallies. This causes:

- Missing optimal entry timing during market recoveries
- Defensive positioning continuing too long after bottoms form
- Inability to shift from conservative to aggressive stance systematically

## Proposed Solution

Implement a Follow-Through Day detection system that:

1. **Tracks Market Downtrends:** Monitors VN-Index for oversold conditions and support tests
2. **Identifies Rally Attempts:** Detects Day 1 (first close higher after new low)
3. **Validates Supply Tests:** Monitors Days 2-3 for price holding and volume depletion
4. **Confirms FTD:** Detects Days 4-7 strong advance (>1.2%) on volume surge with breadth confirmation
5. **Adjusts Risk Parameters:** Increases position sizes, correlation tolerance, and max positions when FTD confirmed

## User Value

**Traders/Users:**

- Systematic detection of market turning points
- Clear signals to shift from defensive to aggressive
- Avoid missing rallies or buying too early in downtrends
- Confidence scoring (0-100) for FTD strength

**System:**

- Market regime awareness for ML predictions
- Dynamic risk parameter adjustment
- Historical FTD performance tracking for continuous improvement

## Success Criteria

- FTD events detected with 80%+ accuracy (validated historically)
- Average 7-day gain post-FTD > 2%
- False positive rate < 20%
- Integration with position sizing (1.5x multiplier on confirmed FTD)
- Telegram alerts within 1 minute of FTD confirmation

## Scope

**In Scope:**

- VN-Index daily data tracking (OHLCV + breadth)
- Rally attempt state machine (Days 1-7 tracking)
- FTD scoring algorithm (price/volume/breadth/leaders)
- Database schema for regime tracking and ftd_events
- Risk parameter adjustments on FTD
- Telegram `/ftd_status` command and alerts

**Out of Scope:**

- Individual stock FTD detection (focus on index only)
- Multi-timeframe FTD (daily only for now)
- ML model re-training triggers (future enhancement)
- Distribution day tracking (separate change)

## Dependencies

- Existing `internal/regime` package (market regime detection)
- VN-Index WebSocket data (already available)
- Market breadth data (may need new DNSE API endpoint or manual input)
- PostgreSQL for regime tracking tables

## Technical Approach

1. **New Package:** `internal/regime/ftd/` for FTD-specific logic
2. **Data Layer:** New tables `market_regime_tracking`, `ftd_events`, `market_breadth_daily`
3. **State Machine:** Rally attempt day counter with validation rules
4. **Scoring:** 0-100 FTD strength (price 30pts, volume 30pts, breadth 20pts, leaders 20pts)
5. **Integration:** Hook into `PortfolioManager` for risk adjustment on FTD events

## Open Questions

1. **Market Breadth Data:** Does DNSE API provide advancing/declining stocks count, or do we need to calculate from watchlist?
2. **Leader Participation:** Should we hard-code Bank/Securities/Steel sectors, or make configurable?
3. **Historical Validation:** Run backtest on 2023-2025 VN-Index data before deployment?
4. **FTD Invalidation:** If distribution day occurs <2 sessions after FTD, should we auto-revert risk parameters?
