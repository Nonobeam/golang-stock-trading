# Spec: R-Multiple Analytics and Reporting

Enhanced portfolio-level R-multiple tracking and analytics for strategy performance evaluation.

## ADDED Requirements

### Requirement: Daily R-Multiple Aggregation

The system MUST calculate and store daily portfolio-level R-multiple statistics for performance tracking.

**Aggregation Metrics**:

- Average R-multiple
- Median R-multiple
- Standard deviation
- Best/Worst R-multiple
- Win rate (% of R > 0)

#### Scenario: Calculate daily R-multiple statistics

**Given** user has 10 closed positions in last 30 days with R-multiples:

- [+2.5R, +1.2R, -0.8R, +3.1R, -1.5R, +0.9R, +1.8R, -0.3R, +2.2R, +1.5R]  
  **When** daily R-multiple statistics are calculated  
  **Then** avg_r_multiple = sum/10 = +1.06R  
  **And** median_r_multiple = 1.35R (middle value)  
  **And** best_r_multiple = +3.1R  
  **And** worst_r_multiple = -1.5R  
  **And** win_rate = 7/10 = 0.70 (70%)  
  **And** profitable_trades = 7  
  **And** total_trades = 10

#### Scenario: Store R-multiple statistics in database

**Given** daily statistics calculated  
**When** statistics are stored  
**Then** record inserted into `r_multiple_statistics` table with:

- user_id
- calculation_date = current_date
- avg_r_multiple, median_r_multiple, stddev_r_multiple
- best_r_multiple, worst_r_multiple
- win_rate, total_trades, profitable_trades

---

### Requirement: R-Multiple by Signal Type

The system MUST track R-multiple performance breakdown by signal type to identify best-performing patterns.

#### Scenario: Calculate R-multiple by signal type

**Given** closed positions with signal types:

- BUY_NEW: [+2.5R, +1.2R, +1.8R] → avg = +1.83R (3 trades)
- BUY_MORE: [+3.1R, +2.2R] → avg = +2.65R (2 trades)
- SELL (stop loss): [-0.8R, -1.5R, -0.3R] → avg = -0.87R (3 trades)  
  **When** R-multiple breakdown calculated  
  **Then** results grouped by signal_type:
- BUY_NEW: avg_r = +1.83R, trades = 3, win_rate = 100%
- BUY_MORE: avg_r = +2.65R, trades = 2, win_rate = 100%
- SELL: avg_r = -0.87R, trades = 3, win_rate = 0%

#### Scenario: Store signal type breakdown in metadata

**Given** R-multiple calculated per signal type  
**When** daily statistics stored  
**Then** r_by_signal_type JSON field contains:

```json
{
  "BUY_NEW": { "avg_r": 1.83, "trades": 3, "win_rate": 1.0 },
  "BUY_MORE": { "avg_r": 2.65, "trades": 2, "win_rate": 1.0 },
  "SELL": { "avg_r": -0.87, "trades": 3, "win_rate": 0.0 }
}
```

---

### Requirement: R-Multiple Historical Trends

The system MUST provide queryable R-multiple historical trends for performance monitoring.

#### Scenario: Query 90-day average R-multiple

**Given** r_multiple_statistics table has 90 daily records  
**When** 90-day average is queried  
**Then** system calculates:

- `SELECT AVG(avg_r_multiple) FROM r_multiple_statistics WHERE user_id = ? AND calculation_date >= NOW() - INTERVAL '90 days'`
- Returns rolling 90-day average (e.g., +1.2R)

#### Scenario: Identify performance degradation

**Given** 30-day avg R-multiple = +0.5R  
**And** 90-day avg R-multiple = +1.3R  
**When** trend analysis is performed  
**Then** system flags degradation:

- Recent performance 62% below historical average
- Recommendation: Review recent signal changes or market conditions

---

### Requirement: R-Multiple Reporting in Daily Workflows

The system MUST include R-multiple statistics in daily validation reports and dashboards.

#### Scenario: Include R-stats in daily signal report

**Given** daily signal generation completes  
**And** R-multiple statistics calculated  
**When** daily report is generated  
**Then** report includes section:

```
R-Multiple Performance (Last 30 Days):
- Average R: +1.2R
- Median R: +1.4R
- Win Rate: 68%
- Best Trade: +3.5R (VCI BUY_NEW on 2026-01-15)
- Worst Trade: -1.8R (HPG stopped out on 2026-01-22)
```

#### Scenario: Alert when average R-multiple drops below threshold

**Given** 30-day avg R-multiple drops to +0.3R  
**And** threshold for concern = +0.5R  
**When** daily statistics calculated  
**Then** WARNING alert sent:

- "R-multiple performance degrading: 30-day avg = +0.3R (below +0.5R threshold)"
- Recommendation: "Consider pausing trading to review strategy"

---

### Requirement: R-Multiple Visualization

The system MUST generate R-multiple distribution charts and equity curve visualizations.

#### Scenario: Generate R-multiple histogram

**Given** user has 50 closed positions  
**When** R-multiple report script is run  
**Then** histogram chart generated showing:

- X-axis: R-multiple bins [-2R to -1R, -1R to 0R, 0R to +1R, +1R to +2R, +2R to +3R, +3R+]
- Y-axis: Count of trades in each bin
- Chart saved to `logs/r_multiple_distribution_{user_id}.png`

#### Scenario: Generate equity curve with R-annotations

**Given** portfolio equity snapshots exist  
**And** closed positions with R-multiples available  
**When** equity curve plotted  
**Then** chart shows:

- Equity line over time
- Markers at each trade exit with R-multiple label
- Color-coded: Green for R > 0, Red for R < 0
- Peak equity line
- Drawdown shaded regions

---

### Requirement: R-Multiple Integration with Position Manager

The R-multiple calculation already exists in position manager. The system MUST ensure it's correctly calculated and stored.

**Existing Formula** (verify correctness):

```
r_multiple = (exit_price - entry_price) / (entry_price - stop_loss)
```

#### Scenario: Verify R-multiple calculation for winning trade

**Given** position opened:

- Entry price = 45,000 VND
- Stop loss = 42,000 VND
- Exit price = 48,000 VND  
  **When** position is closed  
  **Then** risk_per_share = 45,000 - 42,000 = 3,000 VND  
  **And** profit_per_share = 48,000 - 45,000 = 3,000 VND  
  **And** r_multiple = 3,000 / 3,000 = +1.0R  
  **And** r_multiple stored in positions table

#### Scenario: Verify R-multiple for losing trade (stopped out)

**Given** position opened:

- Entry price = 45,000 VND
- Stop loss = 42,000 VND
- Exit price = 42,000 VND (stopped out)  
  **When** position is closed  
  **Then** risk_per_share = 45,000 -42,000 = 3,000 VND  
  **And** profit_per_share = 42,000 - 45,000 = -3,000 VND  
  **And** r_multiple = -3,000 / 3,000 = -1.0R  
  **And** r_multiple stored in positions table

#### Scenario: Handle edge case where stop loss not set

**Given** position opened without explicit stop loss  
**And** entry_price = 45,000 VND  
**And** stop_loss = NULL or entry_price  
**When** position is closed  
**Then** risk_per_share defaults to entry_price or 1.0  
**And** r_multiple calculated as: (exit_price - entry_price) / default_risk  
**Or** r_multiple = NULL if risk cannot be determined
