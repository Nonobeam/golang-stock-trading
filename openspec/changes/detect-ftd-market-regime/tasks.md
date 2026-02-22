# Implementation Tasks

## Phase 1: Data Foundation (4 tasks)

- [ ] Create `market_regime_tracking` table with columns: date, index_value, volume, volume_vs_avg_20d, rally_attempt_day, is_ftd, ftd_strength, breadth_ratio, leader_participation
- [ ] Create `ftd_events` table with columns: event_date, rally_attempt_start_date, days_to_ftd, ftd_strength_score, success_7d, success_30d
- [ ] Create `market_breadth_daily` table with columns: date, advancing_stocks, declining_stocks, unchanged_stocks, new_highs, new_lows, sector_leaders (JSON)
- [ ] Add VN-Index daily data ingestion hook to populate market_regime_tracking

## Phase 2: FTD Detection Logic (8 tasks)

- [ ] Create `internal/regime/ftd/` package structure
- [ ] Implement `DowntrendDetector` - monitors RSI <30, support tests, new lows
- [ ] Implement `RallyAttemptTracker` - state machine for Day 1-7 counting
- [ ] Implement Day 1 detection - first close higher after new low
- [ ] Implement Days 2-3 validation - price holds above Day 1 low, volume depletion check
- [ ] Implement FTD confirmation (Days 4-7) - price gain >1.2%, volume surge, breadth validation
- [ ] Implement `FTDScorer` - calculates 0-100 score based on price(30), volume(30), breadth(20), leaders(20)
- [ ] Implement false FTD filters - leader participation <60%, volume not exceeding previous day, Day 1-3 suspect signals

## Phase 3: Pattern Recognition (4 tasks)

- [ ] Implement Island Reversal detection - gap-down exhaustion + sideways + gap-up breakout
- [ ] Implement Double Bottom detection - first bottom storage + second test comparison + neckline break
- [ ] Implement Supply Test detection - Days 2-3 volume trend calculation, volume <50% of Day 1 flag
- [ ] Add pattern metadata to ftd_events table for analysis

## Phase 4: Risk Parameter Integration (5 tasks)

- [ ] Add `OnFTDConfirmed()` method to `PortfolioManager` accepting FTD strength score
- [ ] Implement position sizing multiplier adjustment - 1.5x for strong FTD (score 80-100), 1.0x for moderate (60-79)
- [ ] Implement max positions increase - 6→8 for strong FTD
- [ ] Implement correlation tolerance increase - 0.85→0.90 for FTD periods
- [ ] Implement auto-revert on FTD invalidation (distribution day within 2 sessions)

## Phase 5: Telegram Integration (4 tasks)

- [ ] Add `/ftd_status` command showing current rally attempt day, FTD detection status, strength score
- [ ] Implement Day 1 detection alert - "Rally Attempt Day 1 detected at X,XXX on VN-Index"
- [ ] Implement FTD confirmation alert with strength rating - "🚨 FOLLOW-THROUGH DAY CONFIRMED (Score: XX/100)"
- [ ] Add daily market regime summary including FTD status

## Phase 6: Monitoring & Validation (5 tasks)

- [ ] Create `GetFTDHistory()` API endpoint returning last 20 FTD events with success rates
- [ ] Implement historical success tracking - calculate 7d/14d/30d gains post-FTD
- [ ] Add FTD performance dashboard metrics - total events, success rate, avg gain, false positive rate
- [ ] Backtest FTD detection on 2023-2025 VN-Index data, validate 80%+ accuracy
- [ ] Add unit tests for FTD scorer, rally tracker, and pattern recognizers

## Phase 7: Documentation (4 tasks)

- [ ] Document FTD detection algorithm in `algorithm_documentation.md`
- [ ] Add FTD configuration options to config.yaml (enable/disable, score thresholds, risk multipliers)
- [ ] Create FTD detection flowchart diagram
- [ ] Update Telegram bot help text with `/ftd_status` command

## Total: 34 tasks across 7 phases
