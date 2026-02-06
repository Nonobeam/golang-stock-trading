## MODIFIED Requirements

### Requirement: Position-Aware Signal Generation

The system SHALL integrate position capacity checking into signal generation logic to prevent BUY_MORE signals when position limits are reached.

#### Scenario: BUY_MORE blocked at capacity

- **WHEN** generating signals for a ticker with an existing position
- **AND** the position is at capacity limit (value or liquidity)
- **THEN** the system SHALL:
  - Check `capacity = check_buying_capacity(ticker, account_value)`
  - If `capacity.at_limit == True`: return signal `HOLD` with reason `already_at_position_limit`
  - NOT generate BUY_MORE signal even if predictions are favorable

#### Scenario: BUY_MORE allowed with capacity

- **WHEN** generating signals for a ticker with room to add
- **AND** predictions indicate strong upward movement
- **THEN** the system SHALL:
  - Check capacity: `capacity = check_buying_capacity(ticker, account_value)`
  - If `capacity.at_limit == False` AND prediction criteria met:
    - Generate signal `BUY_MORE`
    - Include `recommended_shares` constrained by `capacity.max_buyable`

#### Scenario: New position not affected by capacity

- **WHEN** generating signals for a ticker with no existing position
- **THEN** capacity checks SHALL NOT apply
- **AND** signal generation SHALL proceed normally based on predictions and risk parameters

### Requirement: Average Cost in Stop-Loss Checking

The system SHALL use average cost (not first entry price) when checking if stop-loss levels are triggered for existing positions.

#### Scenario: Stop-loss trigger uses average cost

- **WHEN** checking if a position has hit stop-loss
- **AND** position has multiple entries with average cost
- **THEN** the system SHALL:
  - Compare `current_price <= stop_loss`
  - WHERE `stop_loss` is calculated from **first entry price** (not average)
  - Generate `SELL` signal with reason `stop_loss_triggered`

#### Scenario: Unrealized loss calculation uses average

- **WHEN** calculating current unrealized P&L for reporting
- **THEN** the system SHALL:
  - Use `unrealized_pnl = (current_price - avg_cost) × total_shares`
  - Use `unrealized_percent = (current_price - avg_cost) / avg_cost × 100`
- **WHERE** `avg_cost` is the weighted average from all entries

### Requirement: Target Level Checking with Average Cost

The system SHALL use average cost when evaluating profit-taking targets for multi-entry positions.

#### Scenario: Target check uses average cost for profit

- **WHEN** checking if position has reached target levels
- **THEN** the system SHALL calculate:
  - `current_profit_percent = (current_price - avg_cost) / avg_cost × 100`
  - Compare against target levels (T1, T2, T3)

#### Scenario: Target reached with weak outlook triggers partial sell

- **WHEN** `current_price >= target_1` AND predictions show weak outlook (e.g., `p50_5d < 0.01`)
- **THEN** the system SHALL:
  - Generate `SELL_PARTIAL` signal
  - Set reason `target_1_reached_weak_outlook`
  - Calculate profit based on average cost

## ADDED Requirements

### Requirement: Capacity-Constrained Position Sizing

The system SHALL modify position sizing equations to respect capacity limits when calculating recommended share quantities.

#### Scenario: Ideal size constrained by capacity

- **WHEN** calculating position size for BUY_MORE signal
- **THEN** the system SHALL:
  - Calculate `ideal_size = account_value × 0.10 × m_confidence × m_horizon`
  - Check `capacity = check_buying_capacity(ticker, account_value)`
  - Calculate `constrained_size = min(ideal_size, capacity.max_buyable)`
  - Return `shares = floor(constrained_size / current_price)`

#### Scenario: Share capacity further constrains size

- **WHEN** constrained size is calculated
- **THEN** the system SHALL apply additional constraint:
  - `final_shares = min(shares, capacity.remaining_share_capacity)`
- **WHERE** liquidity limit (1% daily volume) is enforced

#### Scenario: Zero shares at limit

- **WHEN** position is at capacity limit
- **THEN** `calculate_constrained_size()` SHALL return 0 shares
- **AND** signal SHALL be `HOLD` instead of `BUY_MORE`

### Requirement: Daily Report Capacity Information

The system SHALL include capacity information in daily position monitoring reports.

#### Scenario: Report shows remaining capacity per position

- **WHEN** generating daily report
- **THEN** for each position, the system SHALL display:
  - Current shares and average cost
  - `remaining_value_capacity` in VND
  - `remaining_share_capacity` as integer shares
  - Boolean flag `at_limit` indicating if position is maxed out

#### Scenario: Capacity warning for near-limit positions

- **WHEN** a position has less than 10% remaining capacity
- **THEN** the report SHALL:
  - Highlight the position with warning indicator
  - Show exact remaining capacity
  - Note that BUY_MORE signals will soon be blocked

#### Scenario: Recommended action respects capacity

- **WHEN** report includes recommended actions
- **THEN** for each signal, the system SHALL:
  - Show recommended shares accounting for capacity limits
  - If signal is `BUY_MORE` but `at_limit == True`: override to `HOLD`
  - Display reason: `Position at 20% allocation limit` or `Position at 1% liquidity limit`

## MODIFIED Requirements

### Requirement: Signal Confidence Adjustment

The system SHALL incorporate capacity information into signal recommendation confidence scoring.

#### Scenario: Reduced confidence near capacity

- **WHEN** generating BUY_MORE signal AND remaining capacity is less than ideal position size
- **THEN** the system MAY:
  - Reduce signal confidence score proportionally
  - Note in signal metadata: `capacity_constrained = true`

#### Scenario: Full confidence when capacity available

- **WHEN** remaining capacity exceeds ideal position size
- **THEN** signal confidence SHALL be based solely on prediction quality
- **AND** capacity SHALL NOT reduce confidence
