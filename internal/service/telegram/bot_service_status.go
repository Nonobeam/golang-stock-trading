package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

func (s *BotService) handleStatusCommand(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	
	// Check dependencies
	if s.watchlistRepo == nil {
		s.SendMessage(chatID, "Watchlist not configured")
		return
	}

	// Get user context
	user, err := s.getUserContext(chatID)
	if err != nil {
		logger.Error().Err(err).Int64("chatID", chatID).Msg("Failed to get user context")
		s.SendMessage(chatID, "Failed to retrieve user information. Please try again.")
		return
	}

	ctx := context.Background()

	// Fetch watchlist
	watchlistItems, err := s.watchlistRepo.GetByUserID(ctx, user.UserID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to fetch watchlist")
		s.SendMessage(chatID, "Failed to retrieve watchlist. Please try again later.")
		return
	}

	// Handle empty watchlist
	if len(watchlistItems) == 0 {
		s.SendMessage(chatID, "<b>Stock Status</b>\n\nNo stocks are currently being tracked.\n\nUse /watch &lt;symbol&gt; to add stocks to your watchlist.")
		return
	}

	// Build status message
	var msgBuilder strings.Builder
	msgBuilder.WriteString(fmt.Sprintf("<b>Stock Status</b> (%d stocks)\n\n", len(watchlistItems)))

	for i, item := range watchlistItems {
		symbol := item.Symbol

		// Get current price from market data service (if available)
		var currentPrice float64
		var priceText string
		var changeText string
		if s.marketDataSvc != nil {
			stockInfo := s.marketDataSvc.GetLatestStockInfo(symbol)
			if stockInfo != nil {
				currentPrice = stockInfo.LastPrice
				priceText = formatPrice(stockInfo.LastPrice)
				changePct := ((stockInfo.LastPrice - stockInfo.Reference) / stockInfo.Reference) * 100
				if changePct > 0 {
					changeText = fmt.Sprintf(" (+%.2f%%)", changePct)
				} else if changePct < 0 {
					changeText = fmt.Sprintf(" (%.2f%%)", changePct)
				}
			} else {
				priceText = "N/A"
			}
		} else {
			priceText = "N/A"
		}

		// Display stock header
		msgBuilder.WriteString(fmt.Sprintf("<b>%s</b>\n", symbol))
		msgBuilder.WriteString(fmt.Sprintf("  Price: %s VND%s\n", priceText, changeText))

		// Get all positions for this stock
		if s.positionRepo != nil {
			positions, err := s.positionRepo.GetAllOpenBySymbol(ctx, user.UserID, symbol)
			if err != nil {
				logger.Error().Err(err).Str("symbol", symbol).Msg("Failed to fetch positions")
			} else if len(positions) > 0 {
				// Calculate aggregated metrics
				totalQty := 0
				totalCost := 0.0
				for _, pos := range positions {
					totalQty += pos.Quantity
					totalCost += pos.EntryPrice * float64(pos.Quantity)
				}
				avgPrice := totalCost / float64(totalQty)

				// Display holdings summary
				msgBuilder.WriteString("\n  <b>Holdings:</b>\n")
				msgBuilder.WriteString(fmt.Sprintf("  • Total: %s shares | Avg: %s VND\n", formatNumber(totalQty), formatPrice(avgPrice)))

				// Display transaction history (last 5 entries)
				msgBuilder.WriteString("\n  <b>Transaction History:</b>\n")
				entries, err := s.positionRepo.GetEntries(ctx, user.UserID, symbol)
				if err != nil {
					logger.Error().Err(err).Str("symbol", symbol).Msg("Failed to fetch entries")
					msgBuilder.WriteString("  (Failed to load history)\n")
				} else {
					count := 0
					for _, entry := range entries {
						// Simple filter: show recent 5 entries. 
						// Since GetEntries sorts by date DESC, we take top 5.
						if count >= 5 {
							break
						}
						
						typeEmoji := "🔹" // Buy
						if entry.TransactionType != "BUY_NEW" && entry.TransactionType != "BUY_MORE" {
							typeEmoji = "🔸" // Other
						}

						dateStr := formatPurchaseDate(entry.EntryDate)
						msgBuilder.WriteString(fmt.Sprintf("  %s %s: %s @ %s\n", 
							typeEmoji,
							dateStr,
							formatNumber(entry.SharesPurchased), 
							formatPrice(entry.EntryPrice)))
						count++
					}
					if len(entries) > 5 {
						msgBuilder.WriteString(fmt.Sprintf("  ...and %d more\n", len(entries)-5))
					}
				}

				// Calculate and display unrealized P&L if we have current price
				if currentPrice > 0 {
					unrealizedPL := (currentPrice - avgPrice) * float64(totalQty)
					unrealizedPLPct := ((currentPrice - avgPrice) / avgPrice) * 100
					plSign := ""
					if unrealizedPL > 0 {
						plSign = "+"
					}
					msgBuilder.WriteString(fmt.Sprintf("\n  <b>Unrealized P&L:</b> %s%s VND (%s%.2f%%)\n", 
						plSign, 
						formatPrice(unrealizedPL), 
						plSign, 
						unrealizedPLPct))
				}
			} else {
				msgBuilder.WriteString("  (No positions)\n")
			}
		}

		// Add spacing between stocks (but not after the last one)
		if i < len(watchlistItems)-1 {
			msgBuilder.WriteString("\n")
		}
	}

	s.SendMessage(chatID, msgBuilder.String())
}

// formatPurchaseDate formats a date for purchase list display
func formatPurchaseDate(t time.Time) string {
	now := time.Now()
	// If same year, show just month and day
	if t.Year() == now.Year() {
		return t.Format("Jan 02")
	}
	// If different year, include year
	return t.Format("Jan 02, 2006")
}
