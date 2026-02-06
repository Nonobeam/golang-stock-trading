## ADDED Requirements

### Requirement: Position Entries Transaction Log

The system SHALL maintain a complete transaction log of all stock purchase entries in a separate `position_entries` table to enable average cost calculation and audit trail.

#### Scenario: New purchase creates entry record

- **WHEN** a user purchases shares of a stock
- **THEN** the system SHALL create a new record in `position_entries` with:
  - `entry_id` (UUID primary key)
  - `ticker` (stock symbol, VARCHAR(10))
  - `entry_date` (purchase date, TIMESTAMP)
  - `entry_price` (price per share, DECIMAL(15,2))
  - `shares_purchased` (quantity, INTEGER)
  - `entry_fee_paid` (calculated as shares × price × 0.0015, DECIMAL(15,2))
  - `transaction_type` (either 'BUY_NEW' or 'BUY_MORE', VARCHAR(20))

#### Scenario: Multiple entries for same ticker

- **WHEN** a user purchases the same stock multiple times
- **THEN** each purchase SHALL create a separate entry record
- **AND** all entries for a ticker SHALL be queryable by ticker symbol

#### Scenario: Entry records are immutable

- **WHEN** an entry record is created
- **THEN** the record SHALL NOT be modified (append-only log)
- **AND** position exits SHALL NOT delete entry records
- **AND** historical entries SHALL remain for audit purposes

### Requirement: Aggregate Position Tracking

The system SHALL extend the `positions` table with aggregate fields that summarize data from all position entries.

#### Scenario: Position tracks total entries count

- **WHEN** a new position entry is added
- **THEN** the `positions.total_entries` field SHALL increment by 1
- **AND** the field SHALL accurately reflect the count of entries in `position_entries` for that ticker

#### Scenario: Position tracks accumulated fees

- **WHEN** a new position entry is added
- **THEN** the `positions.total_fees_paid` SHALL increase by the entry's `entry_fee_paid`
- **AND** on position exit, the exit fee (exit_value × 0.0025) SHALL be added to `total_fees_paid`

#### Scenario: Position tracks entry date range

- **WHEN** the first purchase of a ticker is made
- **THEN** `positions.first_entry_date` SHALL be set to the entry date
- **WHEN** subsequent purchases are made
- **THEN** `positions.last_entry_date` SHALL be updated to the most recent entry date
- **AND** `first_entry_date` SHALL remain unchanged

#### Scenario: Position maintains weighted average cost

- **WHEN** a new position entry is added
- **THEN** the system SHALL calculate weighted average cost as:
  - `avg_cost = SUM(shares_i × price_i) / SUM(shares_i)` across all entries
- **AND** store the result in `positions.entry_price`
- **AND** update `positions.quantity` to total shares across all entries

## MODIFIED Requirements

### Requirement: Position P&L Calculation

The system SHALL calculate profit/loss metrics using weighted average cost from all position entries rather than a single entry price.

#### Scenario: Unrealized P&L uses average cost

- **WHEN** calculating unrealized P&L for an open position
- **THEN** the system SHALL use formula:
  - `unrealized_pnl = (current_price - avg_cost) × total_shares`
  - `unrealized_percent = ((current_price - avg_cost) / avg_cost) × 100`
- **WHERE** `avg_cost` is calculated from all position entries

#### Scenario: Realized P&L includes all fees

- **WHEN** closing an entire position
- **THEN** the system SHALL calculate:
  - `total_cost = (avg_cost × total_shares) + total_fees_paid + exit_fee`
  - `total_proceeds = exit_price × total_shares`
  - `realized_pnl = total_proceeds - total_cost`

#### Scenario: Partial exit allocates fees proportionally

- **WHEN** selling a portion of a position
- **THEN** the system SHALL:
  - Calculate `proportional_fees = total_fees_paid × (shares_sold / total_shares)`
  - Calculate `exit_fee = (exit_price × shares_sold) × 0.0025`
  - Calculate `realized_pnl = (exit_price × shares_sold) - (avg_cost × shares_sold) - proportional_fees - exit_fee`
- **AND** update `total_fees_paid = total_fees_paid - proportional_fees` for remaining position

#### Scenario: R-multiple calculation uses average cost

- **WHEN** calculating risk-reward ratio
- **THEN** the system SHALL use:
  - `r_multiple = (exit_price - avg_cost) / (avg_cost - stop_loss)`
- **WHERE** `stop_loss` is based on first entry price (not average)

## ADDED Requirements

### Requirement: Position Entry Migration

The system SHALL provide a migration to backfill existing positions into the `position_entries` table.

#### Scenario: Existing positions create synthetic entries

- **WHEN** running the migration
- **THEN** for each existing position in `positions` table:
  - Create one entry in `position_entries` with:
    - `ticker = position.symbol`
    - `entry_date = position.entry_date`
    - `entry_price = position.entry_price`
    - `shares_purchased = position.quantity`
    - `entry_fee_paid = position.quantity × position.entry_price × 0.0015`
    - `transaction_type = 'BUY_NEW'`
  - Set `total_entries = 1`
  - Set `first_entry_date = position.entry_date`
  - Set `last_entry_date = position.entry_date`
  - Set `total_fees_paid = calculated entry fee`

#### Scenario: Migration preserves existing P&L

- **WHEN** migration completes
- **THEN** for all existing positions:
  - `positions.entry_price` SHALL remain unchanged (equals first and only entry)
  - Existing `pnl` and `pnl_percent` calculations SHALL produce identical results

#### Scenario: Migration is idempotent

- **WHEN** migration is run multiple times
- **THEN** existing entries SHALL NOT be duplicated
- **AND** only positions without corresponding entries SHALL be backfilled

### Requirement: Database Indexes for Entry Queries

The system SHALL create indexes to support efficient querying of position entries.

#### Scenario: Ticker-based entry lookup

- **WHEN** querying all entries for a ticker
- **THEN** an index on `position_entries(ticker, entry_date DESC)` SHALL optimize the query

#### Scenario: Date-range entry queries

- **WHEN** analyzing entries within a date range
- **THEN** an index on `position_entries(entry_date)` SHALL support efficient filtering
