## ADDED Requirements

### Requirement: Average Cost Calculation

The system SHALL provide a function to calculate weighted average cost from all position entries for a ticker.

#### Scenario: Calculate average from multiple entries

- **WHEN** `calculate_average_cost(ticker)` is called
- **THEN** the system SHALL:
  - Retrieve all entries for the ticker from `position_entries`
  - Calculate `total_cost = SUM(entry.shares × entry.price)` across all entries
  - Calculate `total_shares = SUM(entry.shares)` across all entries
  - Return `avg_cost = total_cost / total_shares`

#### Scenario: Single entry returns entry price

- **WHEN** a ticker has only one position entry
- **THEN** `calculate_average_cost(ticker)` SHALL return the entry price of that single entry
- **AND** the result SHALL equal the original purchase price

#### Scenario: No entries returns None

- **WHEN** a ticker has no position entries
- **THEN** `calculate_average_cost(ticker)` SHALL return `None` or `null`

### Requirement: Position Update with Average Recalculation

The system SHALL update position aggregates when new shares are purchased, recalculating the weighted average cost.

#### Scenario: Adding shares recalculates average

- **WHEN** `update_position_after_buy(ticker, new_shares, new_price)` is called
- **THEN** the system SHALL:
  - Retrieve current position: `old_shares`, `old_avg_cost`
  - Calculate `new_avg_cost = (old_shares × old_avg_cost + new_shares × new_price) / (old_shares + new_shares)`
  - Calculate `entry_fee = new_shares × new_price × 0.0015`
  - Update `positions` table with:
    - `entry_price = new_avg_cost` (semantic: now average, not first entry)
    - `quantity = old_shares + new_shares`
    - `total_fees_paid = total_fees_paid + entry_fee`
    - `total_entries = total_entries + 1`
    - `last_entry_date = current_date`

#### Scenario: First purchase initializes position

- **WHEN** buying a ticker with no existing position
- **THEN** the system SHALL:
  - Set `avg_cost = new_price`
  - Set `total_shares = new_shares`
  - Set `total_entries = 1`
  - Set `first_entry_date = entry_date`
  - Set `last_entry_date = entry_date`

### Requirement: Position Capacity Checking

The system SHALL enforce position size limits based on portfolio allocation (20% max) and liquidity constraints (1% of average daily volume).

#### Scenario: Check remaining value capacity

- **WHEN** `check_buying_capacity(ticker, account_value)` is called
- **THEN** the system SHALL:
  - Calculate `current_position_value = current_shares × current_price`
  - Calculate `max_position_value = account_value × 0.20`
  - Calculate `remaining_value_capacity = max_position_value - current_position_value`
  - Return capacity information including `remaining_value_capacity`

#### Scenario: Check remaining share capacity

- **WHEN** checking buying capacity
- **THEN** the system SHALL:
  - Query `avg_daily_volume = average(volume)` over last 20 trading days
  - Calculate `max_shares_liquidity = avg_daily_volume × 0.01`
  - Calculate `remaining_share_capacity = max_shares_liquidity - current_shares`
  - Return capacity information including `remaining_share_capacity`

#### Scenario: At position limit

- **WHEN** checking capacity AND (`remaining_value_capacity <= 0` OR `remaining_share_capacity <= 0`)
- **THEN** the system SHALL return:
  - `at_limit = True`
  - `max_buyable = 0`

#### Scenario: Calculate maximum buyable

- **WHEN** position is not at limit
- **THEN** the system SHALL:
  - Calculate `max_buyable_value = min(remaining_value_capacity, remaining_share_capacity × current_price)`
  - Return `max_buyable = max_buyable_value / current_price` (shares)

#### Scenario: Total risk validation blocks BUY_MORE

- **WHEN** checking capacity for adding shares to existing position with stop-loss
- **THEN** the system SHALL validate total portfolio risk:
  - Calculate `current_risk = (current_shares) × (avg_cost - stop_loss)`
  - Calculate `additional_risk = (new_shares) × (new_price - stop_loss)`
  - Calculate `total_risk = current_risk + additional_risk`
  - If `total_risk > account_value × 0.02`: set `at_limit = True`
- **AND** reject BUY_MORE even if allocation capacity remains
- **WHERE** `stop_loss` is based on first entry price, not average cost

## MODIFIED Requirements

### Requirement: Position Partial Exit

The system SHALL support selling a portion of a multi-entry position with proportional fee allocation and average cost preservation.

#### Scenario: Partial sale reduces quantity

- **WHEN** selling `shares_sold` from a position with `total_shares`
- **THEN** the system SHALL:
  - Calculate `proportional_fees = total_fees_paid × (shares_sold / total_shares)`
  - Calculate `exit_fee = (exit_price × shares_sold) × 0.0025`
  - Calculate `cost_basis = avg_cost × shares_sold`
  - Calculate `gross_proceeds = exit_price × shares_sold`
  - Calculate `realized_pnl = gross_proceeds - cost_basis - proportional_fees - exit_fee`

#### Scenario: Remaining position updates

- **WHEN** partial exit completes
- **THEN** the system SHALL update the position:
  - `quantity = total_shares - shares_sold`
  - `total_fees_paid = total_fees_paid - proportional_fees`
  - `entry_price` SHALL remain unchanged (average cost stays the same)
- **AND** position entries SHALL remain unchanged (immutable log)

#### Scenario: Full exit includes all fees

- **WHEN** selling entire position (`shares_sold = total_shares`)
- **THEN** the system SHALL:
  - Use `total_fees_paid + exit_fee` for complete fee accounting
  - Calculate `realized_pnl = (exit_price × total_shares) - (avg_cost × total_shares) - total_fees_paid - exit_fee`
- **AND** mark position as closed

## ADDED Requirements

### Requirement: Entry Quality Analysis

The system SHALL provide metrics to analyze the quality of individual entries relative to the current average cost.

#### Scenario: Calculate entry quality score

- **WHEN** analyzing a position entry
- **THEN** the system SHALL calculate:
  - `entry_quality = (avg_cost - entry.price) / entry.price`
- **WHERE**:
  - Positive `entry_quality` indicates purchase below current average (good entry)
  - Negative `entry_quality` indicates purchase above current average (poor entry)

#### Scenario: Aggregate entry quality metrics

- **WHEN** generating position analysis
- **THEN** the system SHALL report:
  - Count of entries above average price
  - Count of entries below average price
  - Weighted average quality score across all entries

### Requirement: Total Return Calculation

The system SHALL calculate total return accounting for all entry costs and accumulated fees.

#### Scenario: Calculate total invested amount

- **WHEN** calculating total return for a position
- **THEN** the system SHALL:
  - Calculate `total_invested = SUM(entry.shares × entry.price) + total_fees_paid`
  - Calculate `current_value = total_shares × current_price`
  - Calculate `total_return_percent = (current_value - total_invested) / total_invested × 100`

#### Scenario: Holding period weighted return

- **WHEN** analyzing return over time
- **THEN** the system SHALL:
  - For each entry, calculate `days_held = current_date - entry.entry_date`
  - Calculate `share_days = entry.shares × days_held`
  - Calculate `weighted_return = SUM(entry.shares × days_held × return) / SUM(share_days)`
