# Spec: Exit Decision Engine

## ADDED Requirements

### Requirement: Exit Evaluation

The system MUST evaluate all open positions daily against exit thresholds to determine if partial or full exits should be executed.

#### Scenario: Evaluate position for Target 1 exit

```
GIVEN a position with:
  - entry_price: 30,000 VND
  - current_price: 34,500 VND
  - target1: 34,500 VND (+15%)
  - target1_filled: false
WHEN the exit evaluator checks the position
THEN it MUST return an exit decision for SELL_TARGET1 with 30% of shares
```

#### Scenario: Evaluate position for Target 2 exit

```
GIVEN a position with:
  - entry_price: 30,000 VND
  - current_price: 37,500 VND
  - target1_filled: true
  - target2: 37,500 VND (+25%)
  - target2_filled: false
  - resistance1: 35,000 VND (cleared)
WHEN the exit evaluator checks the position
THEN it MUST return an exit decision for SELL_TARGET2 with 30% of remaining shares
AND require price > resistance1
```

#### Scenario: Trailing stop exit for remaining position

```
GIVEN a position with:
  - target1_filled: true
  - target2_filled: true
  - current_shares: 40
  - trailing_stop: 39,000 VND
  - current_price: 38,500 VND
WHEN the exit evaluator checks the position
THEN it MUST return an exit decision for SELL_TARGET3 with 100% of remaining shares (40 shares)
```

#### Scenario: Emergency exit overrides all targets

```
GIVEN a position with:
  - target1_filled: false
  - floor_hit_probability: 35%
  - current_shares: 100
WHEN the exit evaluator checks the position
THEN it MUST return an exit decision for SELL_EMERGENCY with 100% of shares
AND ignore all target logic
```

### Requirement: Exit Execution

The system MUST execute sell orders via DNSE API for positions that meet exit criteria.

#### Scenario: Successful exit execution

```
GIVEN an exit decision for SELL_TARGET1 with 30 shares at symbol VCI
WHEN the exit executor places the order
THEN it MUST call DNSE sell order API
AND receive order confirmation
AND update position database with target1_filled=true
AND send Telegram notification to user
```

#### Scenario: Partial fill handling

```
GIVEN an exit decision for 30 shares
WHEN DNSE API returns partial fill of 20 shares
THEN the system MUST update position with actual filled quantity (20)
AND recompute remaining target percentages proportionally
AND notify user of partial execution status
```

#### Scenario: API failure with retry

```
GIVEN an exit decision for 30 shares
WHEN the first API call times out
THEN the system MUST retry with exponential backoff (1s, 2s, 4s)
AND attempt up to 3 total tries
AND if all retries fail, log error and send alert WITHOUT updating database
```

### Requirement: Signal Type Extension

The system MUST support new exit signal types for graduated profit-taking.

#### Scenario: Generate SELL_TARGET1 signal

```
GIVEN a position meeting Target 1 exit criteria (+15% profit)
WHEN the signal generator evaluates the position
THEN it MUST create a signal with type=SELL_TARGET1
AND include exit_percentage=30%, target_level=1, exit_reason="15% profit target"
```

#### Scenario: Generate SELL_TARGET2 signal

```
GIVEN a position meeting Target 2 exit criteria (+25% and cleared resistance)
WHEN the signal generator evaluates the position
THEN it MUST create a signal with type=SELL_TARGET2
AND include exit_percentage=30%, target_level=2, exit_reason="25% profit + cleared resistance"
```

#### Scenario: Generate SELL_EMERGENCY signal

```
GIVEN a position with floor_hit_probability > 30%
WHEN the signal generator evaluates the position
THEN it MUST create a signal with type=SELL_EMERGENCY
AND include exit_percentage=100%, exit_reason="Floor hit risk >30%"
```

## MODIFIED Requirements

### Requirement: Position Tracking

The system MUST track target fill status for each position to prevent duplicate exits.

#### Scenario: Track Target 1 fill

```
GIVEN a successful Target 1 exit execution
WHEN the position is updated in the database
THEN it MUST set target1_filled=true
AND record target1_exit_price
AND record target1_exit_date
AND reduce current_shares by exit quantity
```

#### Scenario: Prevent duplicate Target 1 exit

```
GIVEN a position with target1_filled=true
WHEN the exit evaluator checks the position again
THEN it MUST NOT generate another SELL_TARGET1 signal
AND skip to evaluating Target 2
```

### Requirement: Daily Signal Workflow

The system MUST evaluate exits before generating new entry signals.

#### Scenario: Daily workflow with exit evaluation

```
GIVEN the daily signal workflow starts
WHEN the workflow executes
THEN it MUST:
  1. Load all open positions
  2. Evaluate each position for exit criteria
  3. Execute any pending exits
  4. Update position database
  5. Generate new entry signals
  6. Output all signals (exits + entries)
```

## ADDED Requirements (Market Constraints)

### Requirement: T+2 Settlement Risk

The system MUST adjust exit sizes for positions entered on Thursday/Friday to account for weekend settlement lock.

#### Scenario: Thursday entry with weekend settlement

```
GIVEN a position entered on Thursday
AND today is Friday
AND Target 1 exit is triggered
WHEN the exit executor calculates exit quantity
THEN it MUST reduce the planned 30% exit to 15% (50% reduction)
AND include reason "T+2 weekend settlement adjustment"
```

### Requirement: Liquidity Constraints

The system MUST validate exit sizes do not exceed 0.5% of daily trading volume.

#### Scenario: Large exit exceeds liquidity limit

```
GIVEN a position with 1000 shares to exit
AND the stock's daily volume is 50,000 shares (0.5% = 250 shares)
WHEN the exit executor validates the order
THEN it MUST reject the exit with error "Exceeds 0.5% volume limit"
AND split the exit across multiple days
```

## Related Capabilities

- **position-scaling**: Graduated exit percentages (30%/30%/40%)
- **trailing-stops**: Existing trailing stop calculation (already implemented)
