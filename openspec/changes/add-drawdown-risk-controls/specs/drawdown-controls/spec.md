# Spec: Drawdown-Based Position Size Controls

Portfolio-level risk management through automated drawdown-based position sizing adjustments that preserve capital during losing streaks.

## ADDED Requirements

### Requirement: Portfolio Equity Tracking

The system MUST track daily portfolio equity to calculate drawdowns from peak performance.

**Equity Calculation**:

```
total_equity = open_positions_market_value + closed_positions_total_pnl + cash_balance
```

#### Scenario: Calculate equity with open and closed positions

**Given** user has initial capital = 100,000,000 VND  
**And** 2 open positions:

- VCI: 100 shares @ 45,000 VND (current price: 47,000 VND)
- HPG: 200 shares @ 30,000 VND (current price: 29,500 VND)  
  **And** 1 closed position with pnl = +500,000 VND  
  **When** equity is calculated  
  **Then** open_positions_value = (100 × 47,000) + (200 × 29,500) = 10,600,000 VND  
  **And** invested_capital = (100 × 45,000) + (200 × 30,000) = 10,500,000 VND  
  **And** cash_balance = 100,000,000 - 10,500,000 = 89,500,000 VND  
  **And** total_equity = 10,600,000 + 500,000 + 89,500,000 = 100,600,000 VND

#### Scenario: Store daily equity snapshot

**Given** total_equity calculated = 98,500,000 VND  
**And** peak_equity from history = 102,000,000 VND  
**When** daily snapshot is created  
**Then** system stores:

- snapshot_date = current_date
- total_equity = 98,500,000 VND
- peak_equity = 102,000,000 VND (not updated, current < peak)
- current_drawdown = (98,500,000 - 102,000,000) / 102,000,000 = -3.43%

---

### Requirement: Drawdown Calculation from Peak

The system MUST calculate current drawdown as percentage decline from historical peak equity.

**Formula**:

```
peak_equity = max(all historical equity values)
current_drawdown = (current_equity - peak_equity) / peak_equity
```

#### Scenario: Calculate drawdown when below peak

**Given** current_equity = 88,000,000 VND  
**And** peak_equity = 100,000,000 VND  
**When** drawdown is calculated  
**Then** current_drawdown = (88M - 100M) / 100M = -12.0%  
**And** drawdown status = "WARNING" (crossed -10% threshold)

#### Scenario: Update peak when equity reaches new high

**Given** current_equity = 105,000,000 VND  
**And** previous peak_equity = 102,000,000 VND  
**When** equity snapshot is created  
**Then** new peak_equity = 105,000,000 VND  
**And** current_drawdown = 0.0% (at new peak)

#### Scenario: Handle new user with no history

**Given** user account created today  
**And** no historical equity snapshots exist  
**When** peak equity is queried  
**Then** peak_equity = user.initial_capital  
**And** current_drawdown = 0.0%

---

### Requirement: Drawdown-Based Position Size Multiplier

The system MUST apply position size multipliers based on current drawdown thresholds.

**Multiplier Rules**:

- Drawdown > -5%: multiplier = 1.0 (normal trading)
- Drawdown -5% to -10%: multiplier = 1.0 (acceptable fluctuation)
- Drawdown -10% to -15%: multiplier = 0.5 (half position sizes)
- Drawdown < -15%: multiplier = 0.0 (stop all new trades)

#### Scenario: Normal trading when drawdown is acceptable

**Given** current_drawdown = -3.5%  
**When** drawdown multiplier is calculated  
**Then** multiplier = 1.0  
**And** position sizing proceeds normally

#### Scenario: Reduce position sizes when drawdown at -12%

**Given** current_drawdown = -12.0%  
**And** signal generated for VCI with expected allocation = 15%  
**When** position size is calculated  
**Then** drawdown_multiplier = 0.5  
**And** original_size = 0.10 × 1.5 × 1.0 = 0.15 (15%)  
**And** adjusted_size = 0.15 × 0.5 = 0.075 (7.5%)  
**And** position is opened with 7.5% allocation

#### Scenario: Stop trading when drawdown at -16%

**Given** current_drawdown = -16.0%  
**When** daily signal generation runs  
**Then** drawdown_multiplier = 0.0  
**And** all BUY signal generation is skipped  
**And** log message = "Trading stopped: portfolio drawdown at -16.0%"  
**And** CRITICAL alert sent to user

#### Scenario: Resume normal trading when drawdown recovers

**Given** drawdown was -13% yesterday (half sizing)  
**And** profitable trades bring drawdown to - 4% today  
**When** position sizing is calculated  
**Then** drawdown_multiplier = 1.0  
**And** normal position sizes resume

---

### Requirement: Drawdown Threshold Alerts

The system MUST send alerts when drawdown crosses predefined risk thresholds.

#### Scenario: Send WARNING alert at -8% drawdown

**Given** drawdown crosses -8.0% for first time  
**When** equity snapshot is created  
**Then** WARNING alert sent to user  
**And** alert message includes:

- Current equity value
- Peak equity value
- Current drawdown percentage
- Next threshold (-10%) distance

#### Scenario: Send CRITICAL alert when half-sizing triggered

**Given** drawdown crosses -10.0%  
**When** equity snapshot is created  
**Then** CRITICAL alert sent to user  
**And** alert message = "Position sizes reduced to 50%"  
**And** recommendation = "Review open positions, consider closing weak performers"

#### Scenario: Send EMERGENCY alert when trading stopped

**Given** drawdown crosses -15.0%  
**When** daily signal generation runs  
**Then** EMERGENCY alert sent to user  
**And** alert message = "Trading stopped to preserve capital"  
**And** recommendation = "No new positions will be opened until drawdown recovers above -10%"

---

### Requirement: Integration with Position Sizing

The system MUST integrate drawdown multiplier into existing position sizing calculations.

#### Scenario: Apply drawdown multiplier to position size

**Given** prediction for VCI: p10=-0.01, p50=0.03, p90=0.08  
**And** prediction_range = 0.09 (confidence_multiplier = 1.0)  
**And** horizon = 5 days (horizon_multiplier = 1.0)  
**And** current_drawdown = -11% (drawdown_multiplier = 0.5)  
**When** position size is calculated  
**Then** base_calculation = 0.10 × 1.0 × 1.0 = 0.10  
**And** with_drawdown = 0.10 × 0.5 = 0.05  
**And** final_allocation = 5% (half of normal 10%)

#### Scenario: Backward compatibility when no drawdown

**Given** position sizing called without drawdown_multiplier parameter  
**When** position size is calculated  
**Then** drawdown_multiplier defaults to 1.0  
**And** calculation proceeds as before (backward compatible)
