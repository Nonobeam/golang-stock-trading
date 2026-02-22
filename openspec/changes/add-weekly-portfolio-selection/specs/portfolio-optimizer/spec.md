## ADDED Requirements

### Requirement: Candidate Filtering

Before optimization, the system SHALL apply hard-rule filters to the stock universe to eliminate candidates that fail minimum quality thresholds. The filter stage MUST remove any ticker where:

- Floor-hit probability exceeds 20% (hard liquidity/risk cutoff)
- Average daily volume is below 100,000 shares (liquidity floor)
- Weighted p50 expected return across horizons does not exceed the fee threshold of 0.4% (net-of-fee viability)
- Model prediction confidence is below 60%
- No predictions are available for the stock

The filter stage MUST record the specific rule that eliminated each stock in the run audit trail for reporting.

#### Scenario: Floor-hit violator removed

- **WHEN** a stock has floor_hit_probability = 0.25
- **THEN** it is eliminated at the filter stage
- **AND** the audit trail records "Eliminated: floor_hit_probability 25.0% > 20% threshold"

#### Scenario: Low volume stock removed

- **WHEN** a stock has avg_daily_volume = 80,000 shares
- **THEN** it is eliminated at the filter stage with reason "Eliminated: avg_volume 80k < 100k threshold"

#### Scenario: Insufficient return stock removed

- **WHEN** a stock's weighted p50 return = 0.002 (0.2%) and fee threshold = 0.4%
- **THEN** it is eliminated with reason "Eliminated: expected return 0.20% ≤ fee threshold 0.40%"

#### Scenario: Surviving candidate pool

- **WHEN** all hard filters are applied to 50 universe stocks
- **THEN** the surviving candidate pool is at least 1 stock
- **AND** if fewer than 5 stocks survive, the system logs a warning and returns the best available candidates (fewer than 5)

### Requirement: Composite Scoring

The system SHALL assign each surviving candidate a composite score computed as a weighted sum of five sub-scores:

| Component             | Weight | Metric                                                                         |
| --------------------- | ------ | ------------------------------------------------------------------------------ |
| Weighted return score | 0.30   | p50 weighted: 0.20×(1d) + 0.35×(5d) + 0.45×(10d)                               |
| Risk-adjusted score   | 0.25   | 1 - normalised((p90 - p10) / p50) — penalises uncertainty                      |
| Liquidity score       | 0.20   | Reuses existing tier-based liquidity scoring                                   |
| Floor-hit penalty     | 0.15   | 1 - (floor_prob / 0.20); ranges 0 → 1 for prob in 0 → 20%                      |
| Momentum quality      | 0.10   | 1.0 if p10(10d) > 0, else 0.0 — bonus when pessimistic scenario still positive |

The score breakdown per component MUST be stored alongside the composite score in the result record.

#### Scenario: High-return, low-uncertainty stock scores well

- **WHEN** a stock has weighted p50 return = 0.06, p90-p10 spread = 0.02, floor_prob = 0.05, liquidity tier = High, p10(10d) = 0.01
- **THEN** its composite score is > 0.60

#### Scenario: High-uncertainty stock penalised

- **WHEN** two stocks have identical p50 but stock B has p90-p10 = 3x that of stock A
- **THEN** stock A has a higher composite score than stock B

#### Scenario: Missing horizon falls back gracefully

- **WHEN** a stock has 5d and 1d predictions but no 10d prediction
- **THEN** the horizon weighting adjusts to use available horizons (5d weight rises proportionally)
- **AND** no error is raised

### Requirement: Brute-Force Combination Optimiser

The system SHALL select the optimal 5-stock portfolio from the scored candidate pool by evaluating all C(n, 5) combinations and returning the combination with the highest sum of composite scores that also satisfies:

1. **Sector cap**: No more than 2 stocks from the same sector within the chosen 5.
2. **Correlation cap**: No pairwise Pearson correlation > 0.7 between any two selected stocks (using 90-day daily return history from `daily_bars`).

If the candidate pool has fewer than 5 stocks, the system SHALL return all available candidates and log a warning.

#### Scenario: Best combination selected

- **WHEN** 20 candidates remain after filtering
- **THEN** the optimiser evaluates C(20,5) = 15,504 combinations
- **AND** returns the combination with the highest total composite score that passes both constraints

#### Scenario: Sector cap enforced

- **WHEN** the top-5 individually scored stocks all belong to the "Banking" sector
- **THEN** the optimiser returns a combination that includes at most 2 Banking stocks
- **AND** fills the remaining 3 slots with the highest-scoring stocks from other sectors that satisfy the correlation constraint

#### Scenario: Correlation cap enforced

- **WHEN** two stocks have a pairwise correlation of 0.82 (> 0.70)
- **THEN** they are not both included in the selected basket

#### Scenario: Unknown correlation treated conservatively

- **WHEN** a stock pair has fewer than 30 overlapping daily return observations
- **THEN** their correlation is assumed to be 0.50 (neutral) and a warning is logged

### Requirement: Current Holdings Comparison

The system SHALL compare the recommended 5-stock basket against the user's current open positions fetched from the `positions` table. The comparison MUST compute:

- **Overlap count**: Number of current holdings that appear in the recommended basket.
- **Exit cost estimate**: For each current holding NOT in the recommended basket, the estimated round-trip exit fee = current_value × 0.004 (0.4% fee ≈ 0.33% sell + taxes) as a VND amount.
- **Rotation flag**: A rotation is only flagged as actionable if the recommended replacement stock's composite score exceeds the current holding's score by ≥ 15%.

The comparison output MUST be included in the weekly report and stored in `weekly_portfolio_selection`.

#### Scenario: Full overlap — no rotation needed

- **WHEN** all 5 currently held stocks appear in the recommended basket
- **THEN** the comparison output notes "No rotation needed — current holdings align with recommendation"

#### Scenario: Partial overlap — rotation suggested

- **WHEN** 3 current holdings are in the recommended basket and 2 are not
- **THEN** the report shows the 2 non-matching holdings with their exit cost estimates
- **AND** for each, whether the replacement's score improvement justifies rotation (≥ 15% threshold)

#### Scenario: No positions held

- **WHEN** the positions table has no open rows
- **THEN** the comparison section notes "No current positions — entering fresh"
- **AND** all 5 recommended stocks are flagged as new entries

### Requirement: Weekly Selection History Storage

The system SHALL persist every weekly recommendation to the `weekly_portfolio_selection` table with the following fields per recommended ticker: `week_start` (Monday date), `ticker`, `composite_score`, `score_breakdown` (JSONB with per-component scores and predictions), `rank` (1–5 or higher for near-misses), `is_selected` (TRUE for the final 5, FALSE for near-misses), `selection_reason`, `created_at`.

Near-miss stocks (those that passed the filter but were not selected due to sector cap or correlation cap) SHALL also be stored with `is_selected = FALSE` and `rank > 5` so performance can be retrospectively tracked.

#### Scenario: Five selected tickers stored

- **WHEN** the optimiser picks the final 5
- **THEN** 5 rows are inserted into `weekly_portfolio_selection` with `is_selected = TRUE` and `rank` 1–5

#### Scenario: Near-misses stored

- **WHEN** stocks passed filtering but were excluded by sector or correlation constraints
- **THEN** they are stored with `is_selected = FALSE` and `rank > 5`
- **AND** their `selection_reason` records why they were excluded (e.g., "Excluded: sector cap Banking already at 2")
