# Spec: Transaction Costs

Transaction cost integration ensures all validations and signals account for Vietnamese market fees (0.15% brokerage + 0.1% selling tax = 0.4% round-trip).

## ADDED Requirements

### Requirement: Vietnamese Fee Calculation

The system MUST correctly calculate transaction costs using Vietnamese market rates.

**Rates:**

- Brokerage: 0.15% (both buy and sell)
- Tax: 0.1% (sell only)
- Round-trip: 0.40%

#### Scenario: Calculate fees for BUY trade

**Given** buying 100 shares of VCI at 37,000 VND  
**When** fees are calculated  
**Then** transaction_value = 100 × 37,000 = 3,700,000 VND  
**And** brokerage_fee = 3,700,000 × 0.0015 = 5,550 VND  
**And** total_cost = 3,700,000 + 5,550 = 3,705,550 VND

#### Scenario: Calculate fees for SELL trade

**Given** selling 100 shares of VCI at 38,500 VND  
**When** fees are calculated  
**Then** transaction_value = 100 × 38,500 = 3,850,000 VND  
**And** brokerage_fee = 3,850,000 × 0.0015 = 5,775 VND  
**And** tax = 3,850,000 × 0.001 = 3,850 VND  
**And** total_fee = 5,775 + 3,850 = 9,625 VND  
**And** net_proceeds = 3,850,000 - 9,625 = 3,840,375 VND

---

### Requirement: Minimum Profit Thresholds

The system MUST enforce minimum expected returns to ensure profitability after fees.

#### Scenario: Reject unprofitable 1-day signal

**Given** 1-day prediction shows p50 return = 0.8%  
**And** minimum threshold for 1-day = 1.0%  
**When** signal is generated  
**Then** signal should be HOLD  
**And** rationale = "Expected return 0.8% below 1.0% threshold after fees"

#### Scenario: Accept profitable 5-day signal

**Given** 5-day prediction shows p50 return = 2.3%  
**And** minimum threshold for 5-day = 1.5%  
**When** signal is generated  
**Then** signal can be BUY  
**And** net expected return after 0.4% fee = 1.9%

---

### Requirement: Fee-Adjusted Performance Metrics

The system MUST calculate fee-adjusted performance metrics including net profit factor.

#### Scenario: Calculate net profit factor

**Given** walk-forward test period with 20 trades  
**And** winning trades: [+2.5%, +1.8%, +3.1%, ...] (12 trades)  
**And** losing trades: [-1.2%, -0.8%, ...] (8 trades)  
**When** fee-adjusted profit factor calculated  
**Then** system should:

- Subtract 0.4% from all trades
- Net wins: [+2.1%, +1.4%, +2.7%, ...]
- Net losses: [-1.6%, -1.2%, ...]
- Sum net wins = 18.4%
- Sum net losses absolute = -11.2%
- Profit factor = 18.4 / 11.2 = 1.64

**And** profit factor > 1.5 required for tradeable strategy
