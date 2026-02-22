# Design: Purchase Tracking System

## Overview

This change introduces a simple, user-friendly way to track stock purchases through Telegram while leveraging the existing `positions` table. The design prioritizes ease of use and reuses existing database schema.

## Architecture Decisions

### Decision 1: Reuse `positions` Table for Purchase Records

**Rationale:**

- The `positions` table already has all needed fields: `symbol`, `entry_date`, `entry_price`, `quantity`, `user_id`
- Avoids database migration complexity
- Maintains consistency with existing position tracking
- Purchase records are just "open positions without stop-loss/target strategies"

**Trade-offs:**

- ✅ **Pro**: No schema changes required
- ✅ **Pro**: Existing infrastructure for queries and display
- ✅ **Pro**: Compatible with trading strategy workflows
- ⚠️ **Con**: `stop_loss` field is required but meaningless for simple purchases (will set to 0 or entry_price)

### Decision 2: Separate `/buy` Command from `/addposition`

**Rationale:**

- `/addposition` is designed for trading strategies with stop-loss and targets
- `/buy` is optimized for simple purchase logging (fewer parameters)
- Different user intents: "I bought stock" vs "I'm opening a trade"

**Trade-offs:**

- ✅ **Pro**: Better UX - simpler command for common use case
- ✅ **Pro**: Clearer mental model for users
- ⚠️ **Con**: Two commands that create similar records

### Decision 3: Show All Positions in `/status`, Not Just Latest

**Current Behavior:**

- `GetBySymbol()` uses `LIMIT 1`, only returns one position per symbol
- Users with multiple purchases see incomplete data

**New Behavior:**

- New method `GetAllOpenBySymbol()` returns all open positions for a symbol
- `/status` aggregates them to show complete picture

**Why:**

- Users need to see full transaction history
- Accurate average price requires all purchases
- Supports dollar-cost averaging tracking

## Data Flow

### Purchase Recording Flow

```
User sends: /buy VNM 100 85000

→ Telegram Bot receives message
→ Parse command arguments
  - symbol: VNM
  - quantity: 100
  - price: 85000
  - date: <current date> (default)

→ Validate inputs
  - Check quantity > 0
  - Check price > 0
  - Validate date format if provided

→ Get user context from chat_id

→ Create Position record:
  - user_id: from context
  - symbol: VNM
  - entry_date: parsed or current
  - entry_price: 85000
  - quantity: 100
  - stop_loss: 0 (required field, not used)
  - is_closed: false
  - targets: NULL

→ Save to database via PositionRepository

→ Send confirmation to user
```

### Status Display Flow

```
User sends: /status

→ Get user's watchlist symbols

For each symbol in watchlist:

  → Query ALL open positions for symbol
     using GetAllOpenBySymbol(user_id, symbol)

  → If positions exist:
      - Calculate total_qty = sum(position.quantity)
      - Calculate avg_price = sum(price × qty) / total_qty
      - Get current price from market data
      - Calculate unrealized P&L if price available
      - Format purchase list (date, qty, price)

  → Build formatted message section

→ Combine all symbols into final message

→ Send to user
```

## Component Interactions

```
┌─────────────┐
│   Telegram  │
│     Bot     │
└──────┬──────┘
       │ /buy command
       ▼
┌─────────────────┐
│  BotService     │
│  handleCommand()│
└──────┬──────────┘
       │ Create position
       ▼
┌──────────────────┐
│ PositionRepo     │
│   Create()       │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│   PostgreSQL     │
│  positions table │
└──────────────────┘

---

┌─────────────┐
│   Telegram  │
│     Bot     │
└──────┬──────┘
       │ /status command
       ▼
┌──────────────────────┐
│ BotServiceStatus     │
│ handleStatusCommand()│
└──────┬───────────────┘
       │
       ├─→ Get watchlist
       │   (WatchlistRepo)
       │
       ├─→ For each symbol:
       │     GetAllOpenBySymbol()
       │     (PositionRepo)
       │
       ├─→ Get current prices
       │     (MarketDataService)
       │
       └─→ Format & aggregate
           Send message
```

## Error Handling

### Purchase Recording Errors

| Error Condition     | User Message                                        |
| ------------------- | --------------------------------------------------- |
| Missing arguments   | "Usage: /buy <symbol> <quantity> <price> [date]..." |
| Invalid quantity    | "❌ Invalid quantity. Must be a positive number."   |
| Invalid price       | "❌ Invalid price. Must be a positive number."      |
| Invalid date format | "❌ Invalid date format. Please use YYYY-MM-DD"     |
| Database error      | "Failed to record purchase. Please try again."      |

### Display Errors

| Error Condition            | Behavior                                     |
| -------------------------- | -------------------------------------------- |
| No market data available   | Show "(Price unavailable)"                   |
| Database query fails       | Show error message, don't crash              |
| Empty watchlist            | Show "No stocks are currently being tracked" |
| Watchlist repo unavailable | Show "Watchlist not configured"              |

## Example Scenarios

### Scenario: Single Purchase

```
User: /watch VNM
User: /buy VNM 100 85000
User: /status

Output:
━━━━━━━━━━━━━━━━
Stock Status (1 stock)

VNM
  Price: 87,000 VND (+2.4%)

  Holdings:
  • Total: 100 shares | Avg: 85,000 VND

  Purchases:
  1. Feb 03: 100 @ 85,000 VND

  Unrealized P&L: +200,000 VND (+2.4%)
```

### Scenario: Multiple Purchases (Dollar-Cost Averaging)

```
User: /buy VNM 100 80000 2026-01-20
User: /buy VNM 150 85000 2026-02-01
User: /buy VNM 50 82000 2026-02-03
User: /status

Output:
━━━━━━━━━━━━━━━━
VNM
  Price: 86,000 VND (+1.2%)

  Holdings:
  • Total: 300 shares | Avg: 82,500 VND

  Purchases:
  1. Feb 03: 50 @ 82,000 VND
  2. Feb 01: 150 @ 85,000 VND
  3. Jan 20: 100 @ 80,000 VND

  Unrealized P&L: +1,050,000 VND (+4.2%)

Calculation:
  Avg = (100×80k + 150×85k + 50×82k) / 300
      = (8M + 12.75M + 4.1M) / 300
      = 24.85M / 300
      = 82,833 VND

  P&L = (86,000 - 82,833) × 300
      = 950,100 VND
```

## Testing Strategy

1. **Unit Tests** (if feasible):
   - Position repository `GetAllOpenBySymbol()` method
   - Average price calculation logic

2. **Integration Tests**:
   - End-to-end `/buy` command flow
   - End-to-end `/status` display with various scenarios

3. **Manual Tests**:
   - Multiple purchases of same stock
   - Mixed watchlist (some with purchases, some without)
   - Edge cases (invalid inputs, empty states)
   - Formatting and readability in Telegram

## Implementation Notes

- Command parsing should be lenient with date formats (consider supporting DD/MM/YYYY in future)
- Number formatting should use thousand separators for VND amounts (already exists in codebase)
- Consider adding date validation to prevent future dates
- Keep confirmation messages concise but informative
- Preserve existing `/addposition` functionality unchanged
