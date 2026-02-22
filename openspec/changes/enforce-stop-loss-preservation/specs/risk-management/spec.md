## ADDED Requirements

### Requirement: Stop-Loss Preservation at First Entry

The system SHALL lock stop-loss levels at the first entry price to prevent risk drift from subsequent purchases or manual modifications.

#### Scenario: Stop-loss set at first entry

- **GIVEN** a new position is created with entry price 36,850 VND
- **WHEN** stop-loss is calculated at 4.75% below entry (35,100 VND)
- **THEN** `first_entry_stop_loss` SHALL be set to 35,100 VND
- **AND** `stop_loss_locked` SHALL be set to TRUE
- **AND** `stop_loss` SHALL equal `first_entry_stop_loss`

#### Scenario: Stop-loss modification blocked when locked

- **GIVEN** a position with `first_entry_stop_loss = 35,100` and `stop_loss_locked = TRUE`
- **WHEN** user attempts to update `stop_loss` to 36,000
- **THEN** the system SHALL reject the update with error "stop_loss is locked and cannot be modified"
- **AND** `stop_loss` SHALL remain 35,100
- **AND** `stop_loss_last_modified` SHALL NOT be updated

#### Scenario: Stop-loss modification allowed with override

- **GIVEN** a position with `stop_loss_locked = TRUE`
- **WHEN** user sets `stop_loss_locked = FALSE` with `stop_loss_override_reason = "trailing stop manual adjustment"`
- **AND** updates `stop_loss` to 36,500
- **THEN** the system SHALL accept the update
- **AND** `stop_loss` SHALL be set to 36,500
- **AND** `stop_loss_last_modified` SHALL be set to current timestamp

#### Scenario: Stop-loss preserved across multiple entries

- **GIVEN** a position with first entry at 36,850 and `first_entry_stop_loss = 35,100`
- **WHEN** a second entry is added at 38,000 (average cost now 37,233)
- **THEN** `first_entry_stop_loss` SHALL remain 35,100 (unchanged)
- **AND** `stop_loss` SHALL remain 35,100
- **AND** average cost updates SHALL NOT trigger stop-loss recalculation

### Requirement: Per-Entry Risk Aggregation

The system SHALL calculate total position risk by summing risk across individual entries rather than using average cost, to accurately account for multiple purchases at different prices against a single stop-loss level.

#### Scenario: Single entry risk calculation

- **GIVEN** a position with one entry: 100 shares at 36,850, stop at 35,100
- **WHEN** calculating total risk
- **THEN** total_risk SHALL equal 100 × (36,850 - 35,100) = 175,000 VND

#### Scenario: Multi-entry risk aggregation

- **GIVEN** a position with two entries:
  - Entry 1: 100 shares at 36,850
  - Entry 2: 50 shares at 38,000
  - Stop-loss: 35,100 (from first entry)
- **WHEN** calculating total risk
- **THEN** risk_entry_1 SHALL equal 100 × (36,850 - 35,100) = 175,000 VND
- **AND** risk_entry_2 SHALL equal 50 × (38,000 - 35,100) = 145,000 VND
- **AND** total_risk SHALL equal 175,000 + 145,000 = 320,000 VND

#### Scenario: Risk limit enforcement with multiple entries

- **GIVEN** account value of 10,000,000 VND (max risk 2% = 200,000 VND)
- **AND** existing position: 100 shares at 36,850, stop 35,100 (risk 175,000)
- **WHEN** checking capacity for additional purchase at 38,000
- **THEN** remaining_risk_capacity SHALL equal 200,000 - 175,000 = 25,000 VND
- **AND** max_shares_by_risk SHALL equal 25,000 / (38,000 - 35,100) = 8 shares
- **AND** attempting to buy 50 shares SHALL be blocked

#### Scenario: Risk calculation with modified stop (unlocked)

- **GIVEN** a position with `stop_loss_locked = FALSE` and `stop_loss` modified to 36,000
- **AND** `first_entry_stop_loss = 35,100`
- **WHEN** calculating risk before capacity check
- **THEN** the system SHALL validate stop_loss against first_entry_stop_loss
- **AND** SHALL log warning "stop_loss modified from first entry for {ticker}"
- **AND** SHALL use current `stop_loss = 36,000` for risk calculation (not first_entry)

### Requirement: Three-Tier Capacity Status

The system SHALL provide three distinct capacity status levels (NORMAL, APPROACHING_LIMIT, AT_HARD_LIMIT) with soft warnings at 18% portfolio value or 0.8% daily volume, and hard blocks at 20% or 1%.

#### Scenario: Normal capacity status

- **GIVEN** account value 20,000,000 VND
- **AND** position value 3,000,000 VND (15% of account)
- **AND** position shares 0.5% of daily volume
- **WHEN** checking buying capacity
- **THEN** capacity_status SHALL equal "NORMAL"
- **AND** at_hard_limit SHALL be FALSE
- **AND** at_soft_limit SHALL be FALSE
- **AND** warnings list SHALL be empty

#### Scenario: Soft limit warning by value

- **GIVEN** account value 20,000,000 VND
- **AND** position value 3,700,000 VND (18.5% of account)
- **AND** position shares 0.6% of daily volume
- **WHEN** checking buying capacity
- **THEN** capacity_status SHALL equal "APPROACHING_LIMIT"
- **AND** at_hard_limit SHALL be FALSE
- **AND** at_soft_limit SHALL be TRUE
- **AND** warnings SHALL include "position value approaching 20% limit (currently 18.5%)"
- **AND** distance_to_hard_limit_value SHALL equal 0.015 (1.5%)

#### Scenario: Soft limit warning by volume

- **GIVEN** average daily volume 1,000,000 shares
- **AND** current position 8,500 shares (0.85% of daily volume)
- **AND** position value 15% of account
- **WHEN** checking buying capacity
- **THEN** capacity_status SHALL equal "APPROACHING_LIMIT"
- **AND** at_soft_limit SHALL be TRUE
- **AND** warnings SHALL include "position shares approaching 1% of daily volume (currently 0.85%)"
- **AND** distance_to_hard_limit_volume SHALL equal 0.0015 (0.15%)

#### Scenario: Hard limit blocks BUY_MORE signal

- **GIVEN** position value exactly 20% of account value
- **WHEN** signal generator evaluates BUY_MORE
- **THEN** capacity_status SHALL equal "AT_HARD_LIMIT"
- **AND** at_hard_limit SHALL be TRUE
- **AND** signal SHALL be overridden to HOLD
- **AND** reason SHALL include "position_at_hard_capacity_limit"
- **AND** recommended_shares SHALL equal 0

#### Scenario: Soft limit constrains BUY_MORE recommendation

- **GIVEN** capacity_status = "APPROACHING_LIMIT"
- **AND** distance_to_hard_limit_value = 1.2% (can add 240,000 VND more)
- **AND** ideal BUY_MORE recommendation is 100 shares at 3,000 per share (300,000 VND)
- **WHEN** calculating position change
- **THEN** signal SHALL remain BUY_MORE
- **AND** recommended_shares SHALL be constrained to 240,000 / 3,000 = 80 shares
- **AND** reason SHALL include "limited by approaching capacity (1.2% remaining to hard limit)"

### Requirement: Daily Stop-Loss Integrity Validation

The system SHALL validate stop-loss integrity daily before generating signals, ensuring all position stop-loss levels match their first entry stops within a configurable tolerance.

#### Scenario: Clean positions pass validation

- **GIVEN** 5 open positions all with `stop_loss = first_entry_stop_loss`
- **WHEN** running daily stop-loss integrity check
- **THEN** validation SHALL return success
- **AND** no violations SHALL be reported
- **AND** signal generation SHALL proceed normally

#### Scenario: Modified stop-loss detected

- **GIVEN** a position with `first_entry_stop_loss = 35,100`
- **AND** `stop_loss = 36,000` (modified by 2.56%)
- **WHEN** running daily integrity check with 1% tolerance
- **THEN** validation SHALL detect violation
- **AND** SHALL log error "Stop loss integrity violation: {ticker} stop at 36,000 should be 35,100 (deviation 2.56%)"
- **AND** SHALL add position to violations list

#### Scenario: Tolerance allows rounding differences

- **GIVEN** a position with `first_entry_stop_loss = 35,100`
- **AND** `stop_loss = 35,150` (0.14% difference due to tick rounding)
- **AND** tolerance configured at 1%
- **WHEN** running integrity check
- **THEN** validation SHALL pass (within tolerance)
- **AND** no violation SHALL be reported

#### Scenario: Violations block signal generation

- **GIVEN** daily integrity check found 2 violations
- **WHEN** attempting to generate daily signals
- **THEN** signal generation SHALL be blocked
- **AND** error SHALL be logged "Cannot generate signals: 2 stop-loss integrity violations detected"
- **AND** alert notification SHALL be sent to user
- **AND** violation details SHALL be included in alert

#### Scenario: Override allows signal generation despite violations

- **GIVEN** daily integrity check found violations
- **AND** user sets validation_override = TRUE with reason
- **WHEN** attempting to generate signals
- **THEN** signal generation SHALL proceed
- **AND** warning SHALL be logged "Proceeding with signal generation despite violations: {reason}"

### Requirement: Stop-Loss Audit Trail

The system SHALL maintain an audit trail of stop-loss modifications including timestamp, old value, new value, and reason for override.

#### Scenario: Timestamp updated on stop-loss change

- **GIVEN** a position with `stop_loss = 35,100` and `stop_loss_locked = FALSE`
- **WHEN** `stop_loss` is updated to 36,500 at 2026-02-03 10:30:00
- **THEN** `stop_loss_last_modified` SHALL be set to 2026-02-03 10:30:00

#### Scenario: Override reason required for unlocked stop

- **GIVEN** a position with `stop_loss_locked = FALSE`
- **WHEN** attempting to update `stop_loss` without providing `stop_loss_override_reason`
- **THEN** the system SHALL reject the update
- **AND** error SHALL be "stop_loss_override_reason required when stop_loss_locked is FALSE"

#### Scenario: Override reason stored for audit

- **GIVEN** a position being unlocked for stop modification
- **WHEN** `stop_loss_locked` is set to FALSE with `stop_loss_override_reason = "manually trailing stop based on support level"`
- **THEN** `stop_loss_override_reason` SHALL be persisted
- **AND** reason SHALL be retrievable for audit queries

## MODIFIED Requirements

None - This is a new capability being added to the system.

## REMOVED Requirements

None - No existing requirements are being removed.
