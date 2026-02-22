# Market Regime Tracking

## ADDED Requirements

### MRT-001: Daily Market Data Collection

**Priority:** Must Have

The system MUST collect and store daily VN-Index market data including OHLCV, volume metrics, and market breadth indicators.

#### Scenario: Daily Data Ingestion

**Given** the market has closed for the day  
**When** the daily data collection process runs  
**Then** the system SHALL fetch VN-Index OHLCV data from DNSE API, calculate 20-day and 50-day average volume, store data in market_regime_tracking table, and calculate volume_vs_avg_20d as percentage

#### Scenario: Market Breadth Calculation

**Given** daily stock data is available  
**When** market breadth is calculated  
**Then** the system SHALL count advancing stocks, count declining stocks, calculate breadth ratio, and store in market_breadth_daily table

### MRT-002: Downtrend Detection

**Priority:** Must Have

The system MUST identify when VN-Index is in a downtrend state as a prerequisite for FTD detection.

#### Scenario: Bearish Market Identification

**Given** VN-Index daily data for the last 60 days  
**When** downtrend analysis runs  
**Then** the system SHALL flag downtrend if RSI less than 30 for 3 plus consecutive days, price breaks below 50-day SMA with volume, or new 20-day low is made

#### Scenario: Support Level Testing

**Given** the market is in a downtrend  
**When** price approaches previous support  
**Then** the system SHALL identify support level, track if price tests support within 2 percent proximity, and mark support test in regime tracking table

### MRT-003: Rally Attempt Day 1 Detection

**Priority:** Must Have

The system MUST detect Day 1 of a rally attempt when price closes higher after making a new low.

#### Scenario: Day 1 Identification

**Given** the market made a new 20-day low yesterday  
**When** today close is higher than yesterday close  
**Then** the system SHALL mark rally_attempt_day as 1 in tracking table, store the low price as rally_attempt_baseline, and log Day 1 detection with timestamp

#### Scenario: Day 1 Alert

**Given** Day 1 has been detected  
**When** the detection is confirmed  
**Then** the system SHALL send Telegram notification with rally attempt details including VN-Index close, gain percentage, and low price

### MRT-004: Days 2-3 Validation

**Priority:** Must Have

The system MUST validate that price holds above Day 1 low during Days 2-3 with volume depletion.

#### Scenario: Price Holding Validation

**Given** currently on Day 2 or Day 3 of rally attempt  
**When** daily close is checked  
**Then** the system SHALL verify close greater than Day 1 low price, reset rally_attempt_day to NULL if broken, or increment rally_attempt_day if holds

#### Scenario: Volume Depletion Check

**Given** on Day 2 or Day 3  
**When** volume is compared to Day 1  
**Then** the system SHALL calculate volume ratio, mark as supply test passed if volume less than 50 percent of Day 1, and log supply test status

### MRT-005: FTD Confirmation Days 4-7

**Priority:** Must Have

The system MUST detect Follow-Through Day during Days 4-7 based on price gain, volume surge, and breadth confirmation.

#### Scenario: FTD Criteria Check

**Given** rally attempt is on Day 4, 5, 6, or 7  
**When** FTD detection runs  
**Then** the system SHALL verify price gain at least 1.2 percent, volume greater than previous day, volume greater than 20-day average, and market breadth ratio at least 1.5

#### Scenario: FTD Scoring

**Given** FTD criteria are met  
**When** FTD score is calculated  
**Then** the system SHALL assign Price Score (0-30 points), assign Volume Score (0-30 points), assign Breadth Score (0-20 points), assign Leader Score (0-20 points), and calculate Total Score as sum (0-100)

#### Scenario: FTD Strength Classification

**Given** FTD score has been calculated  
**When** strength is determined  
**Then** the system SHALL classify as strong if score at least 80, moderate if score 60 to 79, or weak if score less than 60

### MRT-006: Leader Participation Validation

**Priority:** Must Have

The system MUST validate that market leadership sectors participate in the FTD.

#### Scenario: Sector Performance Check

**Given** FTD candidate day  
**When** leader participation is evaluated  
**Then** the system SHALL check Bank sector, Securities sector, and Steel sector performance, and assign points based on number of sectors up more than 1 percent

### MRT-007: FTD Confirmation Alert

**Priority:** Must Have

The system MUST send alerts when FTD is confirmed with strength classification.

#### Scenario: Strong FTD Alert

**Given** FTD strength equals strong with score 80-100  
**When** FTD is confirmed  
**Then** the system SHALL send Telegram notification with score, VN-Index close, gain percentage, volume ratio, breadth ratio, and risk parameter changes

#### Scenario: Moderate FTD Alert

**Given** FTD strength equals moderate with score 60-79  
**When** FTD is confirmed  
**Then** the system SHALL send Telegram notification with score, price, gain, and position sizing multiplier

### MRT-008: Risk Parameter Adjustment on FTD

**Priority:** Must Have

The system MUST adjust portfolio risk parameters when FTD is confirmed.

#### Scenario: Strong FTD Risk Adjustment

**Given** FTD with score at least 80 is confirmed  
**When** risk parameters are applied  
**Then** the system SHALL set position_size_multiplier to 1.5, increase max_positions from 6 to 8, and increase max_pairwise_correlation from 0.85 to 0.90

#### Scenario: Moderate FTD Risk Adjustment

**Given** FTD with score 60-79 is confirmed  
**When** risk parameters are applied  
**Then** the system SHALL set position_size_multiplier to 1.25, increase max_positions from 6 to 7, and increase max_pairwise_correlation from 0.85 to 0.87

#### Scenario: FTD Invalidation by Distribution

**Given** FTD was confirmed in the last 2 trading days  
**When** a distribution day occurs (price down at least 0.2 percent on volume greater than average)  
**Then** the system SHALL revert risk parameters to defaults, mark FTD as invalidated in ftd_events table, and send Telegram alert

### MRT-009: FTD Status Command

**Priority:** Must Have

The system MUST provide a Telegram command to check current FTD status.

#### Scenario: FTD Status Query

**Given** user sends /ftd_status command  
**When** command is processed  
**Then** the system SHALL respond with market regime status including VN-Index close, daily change, rally attempt day, FTD detected status, strength, score, risk multiplier, max positions, last FTD date, and success rate

#### Scenario: No Active Rally Status

**Given** no rally attempt is in progress  
**When** /ftd_status is sent  
**Then** the system SHALL respond with monitoring status, VN-Index current close, waiting status, and last FTD date

### MRT-010: Historical FTD Performance Tracking

**Priority:** Should Have

The system SHOULD track and validate FTD success rates over time.

#### Scenario: 7 Day Success Tracking

**Given** an FTD event occurred 7 trading days ago  
**When** the 7-day window completes  
**Then** the system SHALL calculate success_7d percentage gain and update ftd_events table with the value

#### Scenario: 30 Day Success Tracking

**Given** an FTD event occurred 30 calendar days ago  
**When** the 30-day window completes  
**Then** the system SHALL calculate success_30d percentage, update ftd_events table, and recalculate overall FTD success rate

#### Scenario: FTD Performance Dashboard

**Given** user requests FTD history  
**When** /ftd_history command is sent  
**Then** the system SHALL display total FTD events, success rate for 7-day gains above 2 percent, average 7-day gain, average 30-day gain, false positives count, and recent events with their performance
