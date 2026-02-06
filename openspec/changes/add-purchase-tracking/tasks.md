# Implementation Tasks

## 1. Add Repository Method for Multiple Positions Query

**File:** `internal/db/repository/position_repository.go`

- [x] Add `GetAllOpenBySymbol(ctx context.Context, userID int64, symbol string) ([]*Position, error)` method
- [x] Query should return ALL open positions for a symbol (remove LIMIT 1)
- [x] Order by `entry_date DESC` to show most recent first
- [x] Reuse existing Position struct
- [x] Handle empty results gracefully

**Validation:** Query database with test data containing multiple positions for same symbol

---

## 2. Implement /buy Command Handler

**File:** `internal/service/telegram/bot_service.go`

- [x] Add case for "buy" in `handleCommand()` switch statement
- [x] Parse command arguments: symbol, quantity, price, optional date
- [x] Validate inputs (positive numbers, valid symbol format, valid date if provided)
- [x] Default to current date if not specified
- [x] Get user context from `getUserContext(chatID)`
- [x] Create Position struct with purchase details
  - Set `is_closed = false` (open position)
  - Set `stop_loss` to 0 or reasonable default (required field)
  - Leave targets NULL
  - Set entry_date, entry_price, quantity
- [x] Call `positionRepo.Create(ctx, position)` to save
- [x] Send confirmation message with purchase summary
- [x] Handle errors with user-friendly messages

**Validation:**

- Test `/buy VNM 100 85000` (uses today's date)
- Test `/buy VNM 200 84000 2026-01-25` (with specific date)
- Test error cases (invalid arguments, negative numbers)

---

## 3. Enhance /status Command Display

**File:** `internal/service/telegram/bot_service_status.go`

- [x] Replace `positionRepo.GetBySymbol()` with `positionRepo.GetAllOpenBySymbol()` (new method using the GetAllOpenBySymbol method)
- [x] For each watchlist symbol, fetch ALL positions
- [x] If no positions, show just current price
- [x] If positions exist:
  - [x] Calculate total quantity (sum of all position.Quantity)
  - [x] Calculate weighted average price: `sum(price × quantity) / total_quantity`
  - [x] Build formatted message showing:
    - Stock symbol and current price
    - Total holdings and average price
    - List of individual purchases (date, quantity, price)
    - Unrealized P&L if market data available
- [x] Format numbers with thousand separators
- [x] Keep existing formatting for stocks without positions

**Output Example:**

```
Stock Status (2 stocks)

VNM
  Price: 85,000 VND (+2.5%)

  Holdings:
  • Total: 500 shares | Avg: 82,400 VND

  Purchases:
  1. Feb 01: 200 @ 80,000 VND
  2. Jan 25: 300 @ 84,000 VND

  Unrealized P&L: +1,300,000 VND (+3.2%)

HPG
  Price: 25,500 VND (-1.2%)
  (No positions)
```

**Validation:**

- Test with single purchase per stock
- Test with multiple purchases (verify average calculation)
- Test with no purchases (verify graceful handling)
- Test P&L calculation

---

## 4. Update Help Documentation

**File:** `internal/service/telegram/bot_service.go`

- [x] Add `/buy` command to help text in `handleCommand()` help case
- [x] Include usage examples
- [x] Position in appropriate section (Portfolio commands)

**Text to add:**

```
/buy <symbol> <qty> <price> [date] - Record purchase
  Example: /buy VNM 100 85000
  With date: /buy VNM 100 85000 2026-01-25
```

**Validation:** Run `/help` and verify new command appears

---

## 5. Integration Testing

- [x] Code compiles successfully
- [ ] Manual testing: Create test scenario with watchlist containing 2 stocks
- [ ] Manual testing: Record multiple purchases for first stock using `/buy`
- [ ] Manual testing: Record single purchase for second stock
- [ ] Manual testing: Run `/status` and verify:
  - Both stocks appear
  - First stock shows multiple transactions
  - Second stock shows single transaction
  - Average price calculation is correct
  - Total quantities are correct
  - Formatting is clean and readable
- [ ] Manual testing: Test edge cases:
  - Empty watchlist
  - Watchlist with only positions (no market data)
  - Invalid `/buy` command syntax

---

## Dependencies

- Task 1 must complete before Task 3 (status display needs new repository method)
- Tasks 2 and 4 can be done in parallel
- Task 5 requires all previous tasks complete

## Estimated Effort

- Task 1: 30 minutes (simple query modification)
- Task 2: 1-2 hours (command parsing, validation, error handling)
- Task 3: 1-2 hours (aggregation logic, formatting)
- Task 4: 15 minutes (documentation update)
- Task 5: 30 minutes (testing)

**Total:** 3-5 hours
