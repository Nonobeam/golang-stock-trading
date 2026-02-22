## ADDED Requirements

### Requirement: Weekly Scheduled Portfolio Report

The system SHALL automatically run the portfolio selection process every Monday morning (target: 07:30 Vietnam time ICT) after the weekly ML model retraining is complete. The sequence MUST be:

1. Weekly model retraining (Saturday–Sunday)
2. Universe-wide predictions generation (Monday pre-07:30)
3. Portfolio selection run (Monday 07:30)
4. Telegram report delivery to the configured chat

The scheduled run MUST complete and deliver the Telegram report before Vietnamese market open at 09:00 ICT.

#### Scenario: Scheduled Monday run triggers

- **WHEN** Monday 07:30 ICT arrives and the ML model for the latest week is available
- **THEN** the portfolio selector runs automatically
- **AND** the Telegram report is delivered to the configured chat by 08:00 ICT at the latest

#### Scenario: Predictions not available at run time

- **WHEN** the Monday 07:30 run finds that fewer than 10 universe stocks have predictions for today's date
- **THEN** the run aborts with an error message sent to Telegram: "⚠️ Portfolio scan aborted: insufficient predictions available (N/50). Check ML pipeline."

### Requirement: Manual Scan Command

The system SHALL provide a `/scan` Telegram bot command that triggers an ad-hoc portfolio selection run on demand. The command MUST:

- Accept an optional date parameter (`/scan YYYY-MM-DD`) to run for a specific date using historical predictions; default to today.
- Respond within 30 seconds (or send an acknowledgement within 5 seconds and deliver the report as a follow-up message if computation takes longer).
- Be restricted to the configured bot owner chat ID (same restriction as other privileged commands).

#### Scenario: Ad-hoc scan triggered

- **WHEN** the bot owner sends `/scan`
- **THEN** the bot acknowledges "🔍 Running portfolio scan for today..."
- **AND** within 30 seconds delivers the full portfolio report as one or more Telegram messages

#### Scenario: Historical date scan

- **WHEN** the bot owner sends `/scan 2026-01-13`
- **THEN** the scanner uses predictions stored for 2026-01-13
- **AND** the report header clearly shows "📅 Portfolio Scan — 2026-01-13 (Historical)"

### Requirement: Portfolio Report Content

The system SHALL format the weekly portfolio report as a structured plain-text Telegram message containing the following sections in order:

1. **Header**: Date, run timestamp, universe size, candidates after filtering.
2. **Recommended Portfolio (5 stocks)**: For each: rank, ticker, sector, composite score, p10/p50/p90 (10d), brief selection reason.
3. **Current Holdings Comparison**: Overlap count, rotation suggestions with exit cost estimates.
4. **Diversification Summary**: Sector breakdown of the recommended 5 and average pairwise correlation.
5. **Near-Misses / Warnings**: Up to 3 near-miss stocks with reason excluded; stocks eliminated at filter stage (count per filter rule).

If the total message exceeds Telegram's 4096-character limit, the system MUST split it into sequential messages (Part 1/2, etc.).

#### Scenario: Full report delivered

- **WHEN** the portfolio selection completes with 5 stocks selected
- **THEN** the Telegram message contains all five sections
- **AND** each selected stock shows its p10/p50/p90 and composite score

#### Scenario: Long report split

- **WHEN** the formatted report text exceeds 4096 characters
- **THEN** the system sends multiple messages in sequence
- **AND** each message is labelled "Portfolio Report (1/N)", "Portfolio Report (2/N)" etc.

#### Scenario: Diversification summary shown

- **WHEN** the recommended 5 are VCB (Banking), TCB (Banking), FPT (Technology), HPG (Steel), VNM (Consumer)
- **THEN** the report shows: Banking: 2, Technology: 1, Steel: 1, Consumer: 1
- **AND** shows the average pairwise correlation of the 5 (e.g., "Avg correlation: 0.38")

### Requirement: Universe-Wide Batch Prediction

The system SHALL be able to generate ML predictions for all 50 universe stocks in a single batch call. The existing `PredictionGenerator` MUST be extended with a `run_universe_predictions(date: str)` method that:

- Loads active tickers from `stock_universe`
- Calls `generate_daily_predictions()` for each ticker
- Logs per-ticker success/failure
- Returns a summary dict: `{success: N, failed: [...tickers...]}`

Tickers for which predictions cannot be generated (insufficient history, missing model) MUST be logged and excluded from the portfolio selection run rather than causing the entire run to fail.

#### Scenario: Batch prediction for all 50 tickers

- **WHEN** `run_universe_predictions("2026-02-24")` is called
- **THEN** predictions are attempted for all 50 active universe tickers
- **AND** the method returns `{"success": 48, "failed": ["PDR", "DXG"]}` if 2 fail

#### Scenario: Partial failure does not abort

- **WHEN** 5 tickers fail prediction generation due to insufficient history
- **THEN** the remaining 45 predictions are generated and saved normally
- **AND** the 5 failed tickers are excluded from that week's portfolio selection
