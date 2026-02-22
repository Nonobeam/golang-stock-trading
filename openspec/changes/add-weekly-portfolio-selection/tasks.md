## 1. Database

- [ ] 1.1 Create migration `000019_add_portfolio_selection_tables.up.sql` with:
  - `stock_universe` table: `ticker TEXT PK, sector TEXT, exchange TEXT (HOSE/HNX/UPCOM), avg_daily_volume_k INT, is_active BOOL DEFAULT TRUE, notes TEXT`
  - `weekly_portfolio_selection` table: `id SERIAL PK, week_start DATE, ticker TEXT, composite_score NUMERIC, score_breakdown JSONB, rank INT, is_selected BOOL, selection_reason TEXT, created_at TIMESTAMPTZ`
  - Seed INSERT for all 50 tickers with sector and exchange values
- [ ] 1.2 Create `000019_add_portfolio_selection_tables.down.sql` (DROP both tables)

## 2. ML Service — Core Portfolio Module

- [ ] 2.1 Create `ml-service/portfolio/__init__.py`
- [ ] 2.2 Create `ml-service/portfolio/universe.py` — loads `stock_universe` from DB (ticker list, sector map, volume map)
- [ ] 2.3 Create `ml-service/portfolio/correlation.py` — fetches 90d daily returns from `daily_bars`, builds pairwise Pearson correlation matrix using pandas
- [ ] 2.4 Create `ml-service/portfolio/filter.py` — applies hard filter rules (floor-hit > 0.20, volume < 100k, p50 < fee threshold 0.004, confidence < 0.60); returns reduced candidate list with filter-stage audit trail
- [ ] 2.5 Create `ml-service/portfolio/scorer.py` — computes composite score per candidate (return_score 0.30, risk_adjusted 0.25, liquidity 0.20, floor_penalty 0.15, momentum_quality 0.10); returns sorted list with score breakdown
- [ ] 2.6 Create `ml-service/portfolio/optimizer.py` — brute-force C(n,5) combination search; enforces max-2-per-sector constraint and max-pairwise-correlation-0.7 constraint; returns best basket and iteration metadata
- [ ] 2.7 Create `ml-service/portfolio/comparator.py` — compares recommended 5 with current `positions` table holdings; calculates overlap, exit cost estimates (fee \* value), rotation suggestions with 15% score-improvement threshold
- [ ] 2.8 Create `ml-service/portfolio/report.py` — assembles Telegram-formatted text report (recommended 5 + scores, current holdings comparison, rotation suggestions, diversification summary, warnings/near-misses)
- [ ] 2.9 Create `ml-service/portfolio/selector.py` — top-level orchestrator: loads universe → get predictions for all 50 → filter → score → optimise → compare → report → save to DB → send Telegram

## 3. ML Service — Predictions for Full Universe

- [ ] 3.1 Extend `ml-service/daily/prediction_generator.py` — add `run_universe_predictions(date: str)` that loads the active universe from DB and calls existing `generate_daily_predictions()` for all 50 tickers

## 4. ML Service — Weekly Runner

- [ ] 4.1 Create `ml-service/daily/run_weekly_portfolio.py` — standalone entry point that calls `selector.run()` with date = today; logs output; sends Telegram message directly via bot token (same pattern as `monitoring/drawdown_alerts.py`)

## 5. Telegram Bot — `/scan` Command

- [ ] 5.1 Add `/scan` handler in `internal/service/telegram/bot_service.go` that triggers the Python weekly runner via HTTP call to `ML_SERVICE_URL/portfolio/scan` endpoint and relays the response text to the user
- [ ] 5.2 Add `/portfolio/scan` route to ml-service's HTTP server (`ml-service/server/`) that calls `selector.run()` and returns the formatted report as plain text
- [ ] 5.3 Update `/help` command text to include `/scan` description

## 6. Tests

- [ ] 6.1 Create `ml-service/tests/test_portfolio_filter.py` — unit tests for filter logic (floor-hit boundary, volume boundary, fee-threshold boundary, confidence boundary)
- [ ] 6.2 Create `ml-service/tests/test_portfolio_scorer.py` — unit tests for composite score formula (verify weights sum to 1.0, edge cases for missing horizons)
- [ ] 6.3 Create `ml-service/tests/test_portfolio_optimizer.py` — unit tests for brute-force optimiser (sector cap enforcement, correlation cap enforcement, known-best-basket fixture)
- [ ] 6.4 Create `ml-service/tests/test_portfolio_report.py` — snapshot test for report text output given a fixed input

## 7. Documentation

- [ ] 7.1 Update `PROJECT_STRUCTURE.md` to document `ml-service/portfolio/` module and new DB tables
- [ ] 7.2 Update `algorithm_documentation.md` to describe the weekly portfolio selection algorithm, scoring weights, and optimiser constraints
