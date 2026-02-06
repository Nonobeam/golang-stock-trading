# Capability: Enhanced Status Display

## MODIFIED Requirements

### Requirement: /status command shows aggregated purchase history per stock

**Priority:** High  
**Rationale:** Users need comprehensive view of all purchases for each stock to understand their complete position

#### Scenario: Display single purchase

**Given** a user has one purchase of HPG  
**And** the watchlist contains HPG  
**When** the user sends `/status`  
**Then** the bot displays:

```
Stock Status (1 stock)

HPG
  Price: 25,500 VND (+2.1%)

  Holdings:
  • Total: 500 shares | Avg: 24,800 VND

  Purchases:
  1. Feb 02: 500 @ 24,800 VND
```

#### Scenario: Display multiple purchases with calculated average

**Given** a user has three purchases of VNM:

- 100 shares @ 80,000 on 2026-01-20
- 200 shares @ 85,000 on 2026-02-01
- 150 shares @ 82,000 on 2026-02-03

**And** the watchlist contains VNM  
**When** the user sends `/status`  
**Then** the bot calculates:

- Total quantity: 100 + 200 + 150 = 450 shares
- Weighted average: (100×80,000 + 200×85,000 + 150×82,000) / 450 = 82,889 VND

**And** displays:

```
VNM
  Price: 85,000 VND (+2.5%)

  Holdings:
  • Total: 450 shares | Avg: 82,889 VND

  Purchases:
  1. Feb 03: 150 @ 82,000 VND
  2. Feb 01: 200 @ 85,000 VND
  3. Jan 20: 100 @ 80,000 VND
```

**And** purchases are ordered by date descending (most recent first)

#### Scenario: Display unrealized P&L when market data available

**Given** a user owns 300 shares with average price 82,000 VND  
**And** the current market price is 85,000 VND  
**And** market data service is available  
**When** the user sends `/status`  
**Then** the display includes:

```
Holdings:
• Total: 300 shares | Avg: 82,000 VND

Unrealized P&L: +900,000 VND (+3.7%)
```

**Where**:

- Absolute P&L = (85,000 - 82,000) × 300 = 900,000
- Percentage P&L = (85,000 - 82,000) / 82,000 = 3.7%

#### Scenario: Handle stock with no purchases

**Given** a user has VNM in watchlist  
**And** has no recorded purchases of VNM  
**And** market data shows VNM @ 85,000 VND  
**When** the user sends `/status`  
**Then** the display shows:

```
VNM
  Price: 85,000 VND (+2.5%)
  (No positions)
```

#### Scenario: Display multiple stocks with mixed purchase counts

**Given** a user's watchlist has 3 stocks:

- VNM: 2 purchases
- HPG: 1 purchase
- FPT: 0 purchases (monitoring only)

**When** the user sends `/status`  
**Then** each stock displays its purchase summary:

- VNM shows aggregated total and 2 transactions listed
- HPG shows total and 1 transaction listed
- FPT shows "(No positions)"

**And** stocks are shown in watchlist order

### Requirement: Average price calculation uses volume weighting

**Priority:** High  
**Rationale:** Ensures accurate cost basis calculation for P&L tracking

#### Scenario: Calculate weighted average correctly

**Given** a user has purchases:

- 100 shares @ 100,000 VND
- 400 shares @ 80,000 VND

**When** the system calculates average price  
**Then** the formula is: (100×100,000 + 400×80,000) / (100+400)  
**And** the result is: 84,000 VND (not the simple average of 90,000)

### Requirement: Purchase dates are formatted for readability

**Priority:** Medium  
**Rationale:** Improves user experience by showing concise, readable dates

#### Scenario: Format recent purchase dates

**Given** today is 2026-02-03  
**And** a purchase was made on 2026-02-01  
**When** displaying the purchase list  
**Then** the date shows as "Feb 01" (month abbreviation + day)

#### Scenario: Format older purchase dates with year

**Given** today is 2026-02-03  
**And** a purchase was made on 2025-12-15  
**When** displaying the purchase list  
**Then** the date shows as "Dec 15, 2025" (includes year for clarity)

## ADDED Requirements

### Requirement: Position repository supports querying all positions for a symbol

**Priority:** High  
**Rationale:** Enables retrieving complete transaction history for display

#### Scenario: Query multiple open positions for same symbol

**Given** the database contains 3 open positions for VNM for user ID 1  
**When** the repository calls `GetAllOpenBySymbol(ctx, 1, "VNM")`  
**Then** all 3 positions are returned  
**And** they are ordered by `entry_date` descending (newest first)

#### Scenario: Query returns empty for symbol with no positions

**Given** the user has no positions for FPT  
**When** the repository calls `GetAllOpenBySymbol(ctx, 1, "FPT")`  
**Then** an empty slice is returned (not an error)

#### Scenario: Query only returns open positions

**Given** the user has 2 open and 1 closed position for VNM  
**When** the repository calls `GetAllOpenBySymbol(ctx, 1, "VNM")`  
**Then** only the 2 open positions are returned  
**And** the closed position is excluded
