# Design: Weekly Portfolio Selection

## Context

The project already implements per-stock ML predictions (p10/p50/p90 across 1d/5d/10d horizons), floor-hit probability classification, fee-adjusted return checks, liquidity tier scoring, and a correlation-based rejection guard. The weekly portfolio selection feature is a new orchestration layer that reuses these primitives batch-wide rather than per-stock.

Key constraints from the domain:

- Vietnam T+2 settlement means a weekly hold is broadly 3 tradeable days (Mon–Wed entry; sellable Thu onward).
- Price limit ±7% per day; positions can hit the "floor" and become illiquid overnight.
- Trading fees are approximately 0.15–0.25% per leg, so a round-trip costs ~0.3–0.5%; threshold must exceed this.

## Goals / Non-Goals

**Goals:**

- Scan all 50 universe stocks every Monday morning and emit the best 5 to hold for the week.
- Enforce hard sector cap (≤2 per sector in the 5-stock portfolio) and correlation cap (max pairwise ≤ 0.7).
- Store weekly recommendations for future performance retrospective.
- Deliver the report via Telegram (scheduled + manual `/scan` command).

**Non-Goals:**

- Automatic order placement — this is advisory only.
- Intra-week rebalancing — runs once per week.
- Exact share quantity suggestion — report recommends tickers, not lot sizes (user decides sizing).
- Managing the 50-stock universe list automatically — it is hard-coded initially in the DB table.

## Decisions

### Brute-force Optimisation

**Decision:** Enumerate all C(n, 5) combinations of the filtered candidate pool and pick the maximum-score basket satisfying constraints.

**Rationale:** After hard-filter the pool is ≤35 stocks. C(35,5) = 324,632 combinations. Each combo evaluation is O(10) (5 scores + 10 pairwise correlations check). Total ≈ 3M operations — completes in under 2 seconds on a single CPU thread. No need for linear programming or evolutionary algorithms.

**Alternatives considered:** Mean-variance portfolio optimisation (scipy) — adds a dependency and is harder to explain; random search — non-deterministic; greedy selection — misses global optimum.

### Composite Score Formula

```
composite_score =
    0.30 * return_score_weighted      # weighted p50 across horizons
  + 0.25 * risk_adjusted_score        # penalise wide p90-p10 spread
  + 0.20 * liquidity_score            # reuse existing tier system
  + 0.15 * floor_penalty_score        # 1 - (floor_prob / 0.2)
  + 0.10 * momentum_quality_score     # p10 > 0 bonus
```

Weights are initial values; the report records them for later tuning.

### Correlation Matrix

Uses daily return series from `daily_bars` for the last 90 calendar days (≈63 trading days). Pearson correlation. Built once per run from a single SQL query returning all 50 tickers' close prices, pivoted into a matrix in pandas.

### Horizon Weighting for Return Score

```
return_score_weighted = 0.20 * p50(1d) + 0.35 * p50(5d) + 0.45 * p50(10d)
```

Weights 10d most heavily since the selection is for a weekly hold.

### Storage

- `stock_universe` table: static list of 50 tickers with sector and exchange columns. Managed via SQL insert, not code.
- `weekly_portfolio_selection` table: one row per week per ticker selected, with the full score breakdown stored as JSONB.

### Report Format (Telegram)

Plain text with emoji section dividers — avoids Telegram Markdown parsing issues. Delivered via existing `bot_service.go` message sender. The Python runner calls the Go Telegram HTTP endpoint via `ML_SERVICE_URL` callback, or directly via the bot token (simpler, Python calls Telegram API directly using python-telegram-bot already in requirements if present, otherwise via raw requests).

> **Decision**: Python weekly runner sends the Telegram message directly using the bot token from env, matching the pattern already used in `monitoring/drawdown_alerts.py`. No new Go endpoint needed.

### Scheduling

The weekly run is triggered by a new cron-style scheduler in the existing `main.py` ML service entry point, or as a standalone script invoked by an OS scheduler (cron / Windows Task Scheduler). Since the project uses Docker Compose, a separate container step or a cron job inside the ml-service container is preferred.

## Risks / Trade-offs

| Risk                                                                                | Mitigation                                                                               |
| ----------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| Predictions for all 50 stocks may not exist if model hasn't seen sufficient history | Filter stage skips tickers with missing predictions; warning logged                      |
| Correlation matrix incomplete if some tickers have <60d history                     | Fall back to 0.5 correlation for unknown pairs; log warning                              |
| Telegram message too long for single message                                        | Split into 2–3 messages (summary + details) if needed                                    |
| Sector classification wrong/stale                                                   | Sector stored in DB, maintainable via SQL UPDATE; initial values hard-coded in migration |

## Migration Plan

1. Run `000019_add_portfolio_selection_tables.up.sql` (no data migration needed — tables start empty).
2. Populate `stock_universe` with the 50 tickers via migration seed INSERT statements.
3. On next Monday run, the feature generates its first recommendation.
4. Rollback: `000019_add_portfolio_selection_tables.down.sql` drops both new tables — no existing data affected.

## Open Questions (resolved by design above)

1. **Who maintains the 50-stock universe?** → Initially seeded in the DB migration. Maintained via SQL UPDATE. No UI.
2. **Exact quantities or just tickers?** → Tickers only; sizing handled by existing Kelly/drawdown system.
3. **Report stored or Telegram only?** → Stored in `weekly_portfolio_selection` table AND sent to Telegram.
