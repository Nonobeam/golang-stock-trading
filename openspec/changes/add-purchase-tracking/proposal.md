# Change: Add Purchase Tracking with /buy Command

## Why

Currently, users cannot easily record their stock purchases through the Telegram bot. The `/watch` command only adds stocks to a watchlist for monitoring, but doesn't track actual ownership. The `/status` command shows watchlist stocks but doesn't display comprehensive purchase history when users buy the same stock multiple times at different prices and volumes. Users need:

1. **Easy purchase recording** - Simple command to log when they buy stocks
2. **Transaction history** - See all individual purchases with dates, prices, and volumes
3. **Portfolio overview** - View total holdings and average purchase price per stock

Without this, users must manually track purchases externally or use the complex `/addposition` command which is designed for trading signals rather than simple purchase logging.

## What Changes

### New Capabilities

1. **Purchase Recording** (`/buy` command)
   - Add new `/buy` Telegram command to record stock purchases
   - Syntax: `/buy <SYMBOL> <QUANTITY> <PRICE> [DATE]`
   - If date is omitted, uses current date
   - Creates entry in `positions` table with purchase details

2. **Enhanced Status Display** (Updated `/status` command)
   - Show all purchases grouped by stock symbol
   - Display individual transactions with date, quantity, and price
   - Calculate and show total holdings per stock
   - Calculate and show weighted average purchase price
   - Show unrealized P&L when market data is available

### Modified Components

**Telegram Bot Service:**

- `internal/service/telegram/bot_service.go` - Add `/buy` command handler
- `internal/service/telegram/bot_service_status.go` - Enhance status display logic

**Position Repository:**

- `internal/db/repository/position_repository.go` - Add `GetAllOpenBySymbol()` method to retrieve all positions for a symbol (not just first one)

**Help Text:**

- Update `/help` command to include `/buy` command documentation

## Impact

### Affected Components

- `internal/service/telegram/bot_service.go` - New command handler
- `internal/service/telegram/bot_service_status.go` - Updated aggregation logic
- `internal/db/repository/position_repository.go` - New query method

### Benefits

- ✅ Quick and easy purchase recording via Telegram
- ✅ Complete transaction history per stock
- ✅ Automatic average price calculation
- ✅ Better portfolio visibility
- ✅ Supports dollar-cost averaging strategy tracking
- ✅ No database schema changes required (reuses existing `positions` table)

### Non-Breaking Changes

- Fully backward compatible
- `/watch` remains for monitoring stocks without ownership
- Existing `/addposition` command still works for advanced users
- No changes to database schema
