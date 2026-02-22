# Change: Add Weekly Portfolio Selection

## Why

The system currently generates per-stock signals daily but has no mechanism to synthesise those signals into an optimal weekly portfolio. A manual selection process across 50 Vietnamese stocks is time-consuming and misses diversification and correlation constraints. This feature automates that process — filtering, scoring, and optimising the best 5-stock basket every Monday morning before market open.

## What Changes

- **New DB tables**: `stock_universe` (50 curated stocks with sector mapping) and `weekly_portfolio_selection` (weekly recommendation history).
- **New Python module** `ml-service/portfolio/`: batch predictor, correlation matrix builder, composite scorer, brute-force combination optimiser, comparison engine, report generator, and weekly runner script.
- **New Telegram commands**: `/scan` for ad-hoc manual trigger; weekly scheduled Monday morning message with the recommended portfolio report.
- **ML pipeline extension**: predictions must now run for all 50 universe stocks, not just watchlist stocks; the existing `PredictionGenerator.generate_daily_predictions()` is extended with a `run_universe_predictions()` wrapper.
- **Database migration** `000019_add_portfolio_selection_tables.up.sql` to create the two new tables.

## Impact

- Affected specs: new capabilities `stock-universe`, `portfolio-optimizer`, `weekly-portfolio-report`
- Affected code:
  - `ml-service/signals/generator.py` — reused for per-stock filtering logic
  - `ml-service/daily/prediction_generator.py` — extended for universe-wide batch predictions
  - `internal/service/telegram/bot_service.go` — new `/scan` command handler
  - `db/migrations/` — one new migration file
  - `ml-service/daily/` — new `run_weekly_portfolio.py` runner
