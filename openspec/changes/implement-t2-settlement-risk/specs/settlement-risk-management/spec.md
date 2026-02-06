# Settlement Risk Management

## ADDED Requirements

### Requirement: Settlement Status Tracking

The system SHALL track the settlement status of each position to distinguish between locked (unsellable) and liquid (sellable) shares.

#### Scenario: New position created

- **WHEN** a user purchases shares
- **THEN** the system SHALL set settlement_status to LOCKED_T0
- **AND** calculate settlement_date as purchase_date + 2 trading days (excluding weekends and holidays)
- **AND** calculate can_sell_date as settlement_date + 1 trading day

#### Scenario: Settlement status transitions

- **WHEN** the daily settlement update job runs at 16:30
- **THEN** the system SHALL update positions based on current date:
  - If current_date = purchase_date: settlement_status = LOCKED_T0
  - If current_date = purchase_date + 1 trading day: settlement_status = LOCKED_T1
  - If current_date = purchase_date + 2 trading days: settlement_status = LOCKED_T2
  - If current_date >= settlement_date + 1 trading day: settlement_status = LIQUID

#### Scenario: Settlement date calculation with holidays

- **WHEN** calculating settlement_date
- **THEN** the system SHALL skip weekends (Saturday, Sunday)
- **AND** skip Vietnam public holidays (New Year, Reunification Day, Labour Day, National Day, Tết, Hung Kings' Day)
- **AND** count only trading days toward the T+2 settlement period

### Requirement: Locked Capital Risk Calculation

The system SHALL calculate locked capital risk for positions in settlement using worst-case floor-hit scenarios.

#### Scenario: Calculate locked risk for HOSE exchange

- **WHEN** a position has settlement_status of LOCKED_T0, LOCKED_T1, or LOCKED_T2
- **AND** the position is on HOSE exchange
- **THEN** locked_risk SHALL be calculated as: shares * entry_price * 0.20 (20% of capital at risk)
- **AND** this reflects worst-case 7% floor hit with margin of safety

#### Scenario: Calculate locked risk for HNX exchange

- **WHEN** a position has settlement_status of LOCKED_T0, LOCKED_T1, or LOCKED_T2
- **AND** the position is on HNX exchange
- **THEN** locked_risk SHALL be calculated as: shares * entry_price * 0.30 (30% of capital at risk)
- **AND** this reflects worst-case 10% floor hit with margin of safety

#### Scenario: Calculate locked risk for UPCOM exchange

- **WHEN** a position has settlement_status of LOCKED_T0, LOCKED_T1, or LOCKED_T2
- **AND** the position is on UPCOM exchange
- **THEN** locked_risk SHALL be calculated as: shares * entry_price * 0.40 (40% of capital at risk)
- **AND** this reflects worst-case 15% floor hit with margin of safety

#### Scenario: Calculate liquid risk for settled positions

- **WHEN** a position has settlement_status of LIQUID
- **THEN** liquid_risk SHALL be calculated as: shares * (entry_price - stop_loss)
- **AND** this represents controllable risk with executable stop loss

### Requirement: Locked Risk Budget Enforcement

The system SHALL enforce a maximum locked risk threshold to prevent excessive capital being trapped in unsellable positions.

#### Scenario: Check locked risk before new purchase

- **WHEN** evaluating a BUY_NEW or BUY_MORE signal
- **THEN** the system SHALL calculate total_locked_risk = sum(all locked_risk for positions in LOCKED_T0/T1/T2)
- **AND** calculate new_locked_risk for the proposed purchase
- **AND** if (total_locked_risk + new_locked_risk) > (account_value * 0.10), reject the signal
- **AND** generate message: "Cannot buy: would create excessive locked capital risk"

#### Scenario: Locked risk threshold configurable

- **WHEN** checking locked risk budget
- **THEN** the system SHALL use the configured locked_risk_threshold from user_config
- **AND** default to 0.10 (10% of account value) if not configured
- **AND** allow values between 0.05 (5%) and 0.20 (20%)

#### Scenario: Portfolio risk composition report

- **WHEN** generating risk reports
- **THEN** the system SHALL separately report:
  - Total locked capital value
  - Total locked risk amount (worst-case loss)
  - Total liquid capital value
  - Total liquid risk amount (controlled risk)
  - Locked risk as percentage of account value
  - Risk budget remaining

### Requirement: Stop Loss Execution Validation

The system SHALL validate settlement status before attempting to execute stop loss orders.

#### Scenario: Stop loss triggered on locked position

- **WHEN** current_price <= stop_loss
- **AND** settlement_status is LOCKED_T0, LOCKED_T1, or LOCKED_T2
- **THEN** the system SHALL NOT execute the stop loss order
- **AND** generate warning: "Stop loss cannot be executed, shares in settlement until [can_sell_date]"
- **AND** log the theoretical stop loss breach for tracking

#### Scenario: Stop loss triggered on liquid position

- **WHEN** current_price <= stop_loss
- **AND** settlement_status is LIQUID
- **THEN** the system SHALL execute the stop loss order
- **AND** proceed with normal stop loss execution flow

#### Scenario: Track theoretical vs executable stops

- **WHEN** a stop loss is breached but cannot be executed
- **THEN** the system SHALL record:
  - theoretical_stop_breach_date
  - stop_price
  - actual_price at time of breach
  - settlement_status preventing execution
  - days_until_executable
- **AND** this data SHALL be used for performance analysis and validation

### Requirement: Entry Day Restrictions

The system SHALL apply position sizing adjustments based on day of week to account for settlement lock duration.

#### Scenario: Full position size Monday through Wednesday

- **WHEN** generating BUY_NEW or BUY_MORE signal
- **AND** current day is Monday, Tuesday, or Wednesday
- **THEN** the system SHALL allow 100% of calculated position size
- **AND** settlement will complete within same trading week

#### Scenario: Reduced position size Thursday and Friday

- **WHEN** generating BUY_NEW or BUY_MORE signal
- **AND** current day is Thursday or Friday
- **THEN** the system SHALL reduce calculated position size to 50%
- **AND** include message: "Late week entry: position size reduced due to extended settlement lock over weekend"

#### Scenario: Entry day restriction override

- **WHEN** user explicitly overrides entry day restriction
- **THEN** the system SHALL allow full position size
- **AND** log the override action with reason
- **AND** still enforce locked risk budget limits

### Requirement: Settlement State Monitoring

The system SHALL provide real-time visibility into settlement status and locked capital.

#### Scenario: Position settlement dashboard

- **WHEN** viewing active positions
- **THEN** the system SHALL display for each position:
  - Settlement status (LOCKED_T0, LOCKED_T1, LOCKED_T2, LIQUID)
  - Days until liquid (if locked)
  - Locked capital value
  - Locked risk amount
  - Can sell date

#### Scenario: Daily settlement alerts

- **WHEN** positions transition to LIQUID status
- **THEN** the system SHALL send notification: "Position [ticker] now liquid: [quantity] shares can be sold, stop loss now executable"

#### Scenario: Approaching locked risk limit

- **WHEN** total locked risk exceeds 80% of configured threshold
- **THEN** the system SHALL send warning: "Locked risk at [percentage]% of limit, only [remaining] VND available for new purchases"
