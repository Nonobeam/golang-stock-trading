# Spec: Liquidity Management

Liquidity management prevents execution slippage by enforcing position size limits based on daily trading volume.

## ADDED Requirements

### Requirement: Average Volume Calculation

The system MUST query 20-day average volume from `daily_bars` table.

#### Scenario: Calculate VCI average volume

**Given** VCI daily_bars for last 20 trading days  
**And** volumes: [520000, 480000, 550000, ..., 490000]  
**When** average calculated  
**Then** avg_volume = sum(volumes) / 20  
**Example:** avg_volume = 505,000 shares/day

---

### Requirement: Position Cap Enforcement

The system MUST limit positions to 1% of daily volume.

**Formula:**
$$\text{Max Shares} = \text{Avg Volume}_{20d} \times 0.01$$

#### Scenario: Cap oversized VCI position

**Given** VCI avg volume = 505,000 shares/day  
**And** position sizer recommends 10,000 shares  
**When** liquidity cap applied  
**Then** max_shares = 505,000 × 0.01 = 5,050 shares  
**And** recommended_shares reduced from 10,000 to 5,050  
**And** warning = "Liquidity cap: reduced from 10,000 to 5,050 shares"

---

### Requirement: Liquidity Scoring

The system MUST assign liquidity scores from 1-10 based on average daily volume.

#### Scenario: Score stock liquidity

**Given** stock average volume  
**When** liquidity score calculated  
**Then** scoring:

- Score 10: ≥ 5,000,000 shares/day (VNM, HPG)
- Score 5: 500,000 - 5,000,000 shares/day
- Score 1: 100,000 - 500,000 shares/day
- Exclude: < 100,000 shares/day (untradeable)

---

### Requirement: Execution Strategy

The system MUST recommend execution strategies for large orders to minimize market impact.

#### Scenario: Split large order

**Given** VCI avg volume = 505,000 shares/day  
**And** capped position = 5,050 shares (1% of volume)  
**When** execution strategy recommended  
**Then** split into 10 orders of 505 shares each  
**And** spread over 2-3 hours  
**And** suggested times: [09:30, 10:00, 10:30, 11:00, ...]
