package telegram

import (
	"context"
	"fmt"
	"strings"

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

	// Use default user ID 1 (can be made configurable via env var later)
	const defaultUserID = int64(1)
	ctx := context.Background()

	// Fetch watchlist
	watchlistItems, err := s.watchlistRepo.GetByUserID(ctx, defaultUserID)
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
		var priceText string
		var changeText string
		if s.marketDataSvc != nil {
			stockInfo := s.marketDataSvc.GetLatestStockInfo(symbol)
			if stockInfo != nil {
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

		// Check if user owns this stock
		var avgPriceText string
		if s.positionRepo != nil {
			position, err := s.positionRepo.GetBySymbol(ctx, defaultUserID, symbol)
			if err != nil {
				logger.Error().Err(err).Str("symbol", symbol).Msg("Failed to check position")
			} else if position != nil {
				avgPriceText = fmt.Sprintf(" | Avg: %s VND", formatPrice(position.EntryPrice))
			}
		}

		// Format entry
		msgBuilder.WriteString(fmt.Sprintf("<b>%s</b>\n", symbol))
		msgBuilder.WriteString(fmt.Sprintf("  Price: %s VND%s%s\n", priceText, changeText, avgPriceText))

		// Add spacing between entries (but not after the last one)
		if i < len(watchlistItems)-1 {
			msgBuilder.WriteString("\n")
		}
	}

	s.SendMessage(chatID, msgBuilder.String())
}
