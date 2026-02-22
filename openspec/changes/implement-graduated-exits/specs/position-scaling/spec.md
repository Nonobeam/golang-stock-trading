# Spec: Position Scaling

## ADDED Requirements

### Requirement: Graduated Exit Percentages

The system MUST scale position exits according to the 30%/30%/40% allocation strategy.

#### Scenario: Calculate Target 1 exit shares (30%)

```
GIVEN a position with initial_shares=100
AND target1_filled=false
WHEN calculating exit shares for Target 1
THEN it MUST return 30 shares (30% of 100)
```

#### Scenario: Calculate Target 2 exit shares (30% of remaining)

```
GIVEN a position with:
  - initial_shares=100
  - target1_filled=true
  - current_shares=70 (after Target 1 exit of 30)
  - target2_filled=false
WHEN calculating exit shares for Target 2
THEN it MUST return 30 shares (30% of initial 100, NOT 30% of 70)
```

#### Scenario: Calculate Target 3 exit shares (remaining 40%)

```
GIVEN a position with:
  - initial_shares=100
  - target1_filled=true
  - target2_filled=true
  - current_shares=40
WHEN calculating exit shares for Target 3
THEN it MUST return 40 shares (all remaining, which is 40% of initial)
```

### Requirement: Share Rounding

The system MUST handle share quantity rounding for fractional calculations.

#### Scenario: Round down fractional shares

```
GIVEN a position with initial_shares=97
WHEN calculating 30% for Target 1
THEN it MUST return 29 shares (97 * 0.30 = 29.1, rounded down)
AND ensure total exits never exceed initial shares
```

#### Scenario: Ensure minimum exit quantity

```
GIVEN a position with initial_shares=5
WHEN calculating 30% for Target 1
THEN it MUST return 1 share (minimum, as 5 * 0.30 = 1.5 rounds to 1)
AND adjust remaining targets proportionally
```

### Requirement: Exit Validation

The system MUST validate exit quantities do not exceed available shares.

#### Scenario: Prevent over-exit

```
GIVEN a position with current_shares=40
WHEN an exit decision requests 50 shares
THEN it MUST reject the exit with error "Exit quantity exceeds available shares"
```

#### Scenario: Validate target sequence

```
GIVEN a position with target1_filled=false
WHEN attempting to execute Target 2 exit
THEN it MUST reject with error "Cannot exit Target 2 before Target 1"
```

## MODIFIED Requirements

### Requirement: Position State Management

The system MUST maintain accurate share counts through multiple partial exits.

#### Scenario: Multi-stage exit tracking

```
GIVEN a position lifecycle:
  - Initial: current_shares=100
  - After T1: current_shares=70, target1_filled=true
  - After T2: current_shares=40, target2_filled=true
  - After T3: current_shares=0, status=CLOSED
WHEN each exit is executed
THEN the database MUST reflect accurate current_shares at each stage
AND prevent negative share counts
```

## Database Schema Extensions

### Positions Table

```sql
ALTER TABLE positions ADD COLUMN IF NOT EXISTS target1_filled BOOLEAN DEFAULT FALSE;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS target2_filled BOOLEAN DEFAULT FALSE;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS trailing_stop_active BOOLEAN DEFAULT FALSE;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS target1_exit_price DECIMAL(10,2);
ALTER TABLE positions ADD COLUMN IF NOT EXISTS target2_exit_price DECIMAL(10,2);
ALTER TABLE positions ADD COLUMN IF NOT EXISTS target1_exit_date TIMESTAMP;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS target2_exit_date TIMESTAMP;
```

## Related Capabilities

- **exit-decision-engine**: Determines WHEN to exit (this spec defines HOW MUCH)
