## ADDED Requirements

### Requirement: Stock Universe Registry

The system SHALL maintain a static registry of up to 50 curated Vietnamese stocks in the `stock_universe` database table. Each entry MUST include: ticker symbol, sector classification, exchange (HOSE/HNX/UPCOM), approximate average daily volume (in thousands of shares), and an `is_active` flag.

The universe is the single authoritative list of candidate stocks eligible for weekly portfolio selection. Only `is_active = TRUE` stocks are considered during scanning. Updates to the universe MUST be made via SQL (no code change required for additions or deactivations).

#### Scenario: Active universe loaded for scanning

- **WHEN** the weekly portfolio scanner runs
- **THEN** it fetches all rows from `stock_universe` WHERE `is_active = TRUE`
- **AND** the fetched list contains exactly the tickers to be evaluated

#### Scenario: Inactive ticker excluded

- **WHEN** a ticker has `is_active = FALSE` in `stock_universe`
- **THEN** that ticker is excluded from the candidate pool before any filtering occurs
- **AND** the exclusion is noted in the run log

#### Scenario: Sector classification present

- **WHEN** loading the universe
- **THEN** each ticker has a non-null `sector` value (e.g., "Banking", "Real Estate", "Technology", "Steel", "Energy", "Consumer", "Logistics", "Insurance")
- **AND** the sector value is used downstream by the optimiser to enforce the per-sector cap

### Requirement: Sector Classification Seed

The system SHALL seed the `stock_universe` table with all 50 Vietnamese tickers and their sector assignments as part of database migration `000019`. The initial sector assignments SHALL be based on the canonical list specified in the change proposal.

#### Scenario: Database migration populates universe

- **WHEN** migration `000019_add_portfolio_selection_tables.up.sql` is applied
- **THEN** the `stock_universe` table contains exactly 50 rows, each with a valid ticker, sector, exchange, and `is_active = TRUE`
