package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

// handleAddPositionCommand handles /addposition <symbol> <price> <qty> [stop]
func (s *BotService) handleAddPositionCommand(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	text := msg.Text
	if s.positionRepo == nil {
		s.SendMessage(chatID, "Position management not configured")
		return
	}

	args := strings.Fields(text)
	if len(args) < 4 {
		s.SendMessage(chatID, "Usage: /addposition &lt;symbol&gt; &lt;price&gt; &lt;quantity&gt; [stop]\nExample: /addposition VNM 75000 100 71000")
		return
	}

	symbol := strings.ToUpper(args[1])
	entryPrice, err := strconv.ParseFloat(args[2], 64)
	if err != nil {
		s.SendMessage(chatID, "Invalid entry price. Please use a number.")
		return
	}

	quantity, err := strconv.Atoi(args[3])
	if err != nil {
		s.SendMessage(chatID, "Invalid quantity. Please use a whole number.")
		return
	}

	var stopLoss float64
	if len(args) >= 5 {
		stopLoss, err = strconv.ParseFloat(args[4], 64)
		if err != nil {
			s.SendMessage(chatID, "Invalid stop-loss. Please use a number.")
			return
		}
	} else {
		// Calculate automatic stop-loss (3% below entry for simplicity)
		stopLoss = entryPrice * 0.97
	}

	// Validate inputs
	if entryPrice <= 0 || quantity <= 0 || stopLoss <= 0 {
		s.SendMessage(chatID, "All values must be positive numbers")
		return
	}

	if stopLoss >= entryPrice {
		s.SendMessage(chatID, "Stop-loss must be below entry price for long positions")
		return
	}

	// Create position in database
	const defaultUserID = int64(1)
	position := &repository.Position{
		ID:         uuid.New().String(),
		UserID:     defaultUserID,
		Symbol:     symbol,
		EntryDate:  time.Now(),
		EntryPrice: entryPrice,
		Quantity:   quantity,
		StopLoss:   stopLoss,
		IsClosed:   false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	ctx := context.Background()
	if err := s.positionRepo.Create(ctx, position); err != nil {
		logger.Error().Err(err).Msg("Failed to create position")
		s.SendMessage(chatID, "Failed to add position. Please try again.")
		return
	}

	// Calculate risk metrics
	riskPerShare := entryPrice - stopLoss
	totalRisk := riskPerShare * float64(quantity)
	riskPercent := (riskPerShare / entryPrice) * 100

	respText := fmt.Sprintf(
		"<b>Position Added</b>\n\n"+
			"Symbol: <b>%s</b>\n"+
			"Entry: %s VND\n"+
			"Quantity: %d shares\n"+
			"Stop-Loss: %s VND\n\n"+
			"<b>Risk Analysis:</b>\n"+
			"Risk/Share: %s VND (%.2f%%)\n"+
			"Total Risk: %s VND\n\n"+
			"Position is now being monitored!",
		symbol,
		formatPrice(entryPrice),
		quantity,
		formatPrice(stopLoss),
		formatPrice(riskPerShare),
		riskPercent,
		formatPrice(totalRisk),
	)

	s.SendMessage(chatID, respText)
}

// handleEditPositionCommand handles /editposition <symbol>
func (s *BotService) handleEditPositionCommand(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	text := msg.Text
	if s.positionRepo == nil {
		s.SendMessage(chatID, "Position management not configured")
		return
	}

	args := strings.Fields(text)
	if len(args) < 2 {
		s.SendMessage(chatID, "Usage: /editposition &lt;symbol&gt;\nExample: /editposition VNM")
		return
	}

	symbol := strings.ToUpper(args[1])

	// Get position from database
	const defaultUserID = int64(1)
	ctx := context.Background()
	position, err := s.positionRepo.GetBySymbol(ctx, defaultUserID, symbol)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to fetch position")
		s.SendMessage(chatID, "Failed to retrieve position")
		return
	}

	if position == nil {
		s.SendMessage(chatID, fmt.Sprintf("No active position found for %s", symbol))
		return
	}

	// Show current position details and ask for new stop
	respText := fmt.Sprintf(
		"<b>Edit Position: %s</b>\n\n"+
			"Current Entry: %s VND\n"+
			"Current Stop: %s VND\n"+
			"Quantity: %d shares\n\n"+
			"Reply with new stop-loss price:",
		symbol,
		formatPrice(position.EntryPrice),
		formatPrice(position.StopLoss),
		position.Quantity,
	)

	s.SendMessage(chatID, respText)
	// Note: Full implementation would need state management to handle the reply
	// For now, this is a simplified version
}

// handlePositionDetailCommand handles /position <symbol>
// Displays detailed information about a specific position
func (s *BotService) handlePositionDetailCommand(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	text := msg.Text

	// Verify dependencies
	if s.positionRepo == nil {
		s.SendMessage(chatID, "⚠️ Position management not configured")
		return
	}

	// Parse arguments
	args := strings.Fields(text)
	if len(args) < 2 {
		s.SendMessage(chatID, "<b>Usage:</b> /position &lt;symbol&gt;\n\n<b>Example:</b> /position HPG")
		return
	}

	symbol := strings.ToUpper(args[1])

	// Get user context
	user, err := s.getUserContext(chatID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get user context")
		s.SendMessage(chatID, "Failed to retrieve user information")
		return
	}

	// Fetch position from database
	ctx := context.Background()
	position, err := s.positionRepo.GetBySymbol(ctx, user.UserID, symbol)
	if err != nil {
		logger.Error().Err(err).Str("symbol", symbol).Msg("Failed to fetch position")
		s.SendMessage(chatID, "Failed to retrieve position details")
		return
	}

	if position == nil {
		s.SendMessage(chatID, fmt.Sprintf("<b>No Active Position</b>\n\nNo open position found for <code>%s</code>", symbol))
		return
	}

	// Get current market price
	currentPrice := position.EntryPrice // Default to entry price
	if s.marketDataSvc != nil {
		stockInfo := s.marketDataSvc.GetLatestStockInfo(symbol)
		if stockInfo != nil && stockInfo.LastPrice > 0 {
			currentPrice = stockInfo.LastPrice
		}
	}

	// Calculate metrics
	daysInTrade := int(time.Since(position.EntryDate).Hours() / 24)
	unrealizedPL := (currentPrice - position.EntryPrice) * float64(position.Quantity)
	unrealizedPLPercent := ((currentPrice - position.EntryPrice) / position.EntryPrice) * 100
	
	riskPerShare := position.EntryPrice - position.StopLoss
	rMultiple := 0.0
	if riskPerShare > 0 {
		rMultiple = (currentPrice - position.EntryPrice) / riskPerShare
	}
	
	stopDistance := currentPrice - position.StopLoss
	stopDistancePercent := (stopDistance / currentPrice) * 100
	totalRisk := riskPerShare * float64(position.Quantity)

	// Build response message
	var msgBuilder strings.Builder
	
	// Header
	plEmoji := "📊"
	if unrealizedPL > 0 {
		plEmoji = "🟢"
	} else if unrealizedPL < 0 {
		plEmoji = "🔴"
	}
	
	msgBuilder.WriteString(fmt.Sprintf("<b>%s Position Details: %s</b>\n\n", plEmoji, symbol))
	
	// Entry Details
	msgBuilder.WriteString("<b>📥 Entry Details</b>\n")
	msgBuilder.WriteString(fmt.Sprintf("  Entry Date: %s\n", position.EntryDate.Format("2006-01-02")))
	msgBuilder.WriteString(fmt.Sprintf("  Entry Price: %s VND\n", formatPrice(position.EntryPrice)))
	msgBuilder.WriteString(fmt.Sprintf("  Quantity: %s shares\n", formatNumber(position.Quantity)))
	msgBuilder.WriteString(fmt.Sprintf("  Position Value: %s VND\n\n", formatPrice(position.EntryPrice*float64(position.Quantity))))
	
	// Current Status
	msgBuilder.WriteString("<b>📈 Current Status</b>\n")
	msgBuilder.WriteString(fmt.Sprintf("  Current Price: %s VND\n", formatPrice(currentPrice)))
	msgBuilder.WriteString(fmt.Sprintf("  Days in Trade: %d\n", daysInTrade))
	msgBuilder.WriteString(fmt.Sprintf("  Unrealized P&L: %s VND (<b>%+.2f%%</b>)\n", formatPrice(unrealizedPL), unrealizedPLPercent))
	msgBuilder.WriteString(fmt.Sprintf("  R-Multiple: <b>%+.2fR</b>\n\n", rMultiple))
	
	// Risk Management
	msgBuilder.WriteString("<b>🛡️ Risk Management</b>\n")
	msgBuilder.WriteString(fmt.Sprintf("  Stop Loss: %s VND\n", formatPrice(position.StopLoss)))
	msgBuilder.WriteString(fmt.Sprintf("  Distance to Stop: %s VND (%.2f%%)\n", formatPrice(stopDistance), stopDistancePercent))
	msgBuilder.WriteString(fmt.Sprintf("  Risk/Share: %s VND\n", formatPrice(riskPerShare)))
	msgBuilder.WriteString(fmt.Sprintf("  Total Risk: %s VND\n", formatPrice(totalRisk)))
	
	// Targets (if any)
	if (position.Target1 != nil && *position.Target1 > 0) || 
	   (position.Target2 != nil && *position.Target2 > 0) || 
	   (position.Target3 != nil && *position.Target3 > 0) {
		msgBuilder.WriteString("\n<b>🎯 Targets</b>\n")
		
		if position.Target1 != nil && *position.Target1 > 0 {
			t1Distance := *position.Target1 - currentPrice
			t1Percent := (t1Distance / currentPrice) * 100
			t1R := (*position.Target1 - position.EntryPrice) / riskPerShare
			t1Hit := currentPrice >= *position.Target1
			hitEmoji := "⏳"
			if t1Hit {
				hitEmoji = "✅"
			}
			msgBuilder.WriteString(fmt.Sprintf("  %s T1: %s VND (+%.1f%%, %.1fR)\n", hitEmoji, formatPrice(*position.Target1), t1Percent, t1R))
		}
		
		if position.Target2 != nil && *position.Target2 > 0 {
			t2Distance := *position.Target2 - currentPrice
			t2Percent := (t2Distance / currentPrice) * 100
			t2R := (*position.Target2 - position.EntryPrice) / riskPerShare
			t2Hit := currentPrice >= *position.Target2
			hitEmoji := "⏳"
			if t2Hit {
				hitEmoji = "✅"
			}
			msgBuilder.WriteString(fmt.Sprintf("  %s T2: %s VND (+%.1f%%, %.1fR)\n", hitEmoji, formatPrice(*position.Target2), t2Percent, t2R))
		}
		
		if position.Target3 != nil && *position.Target3 > 0 {
			t3Distance := *position.Target3 - currentPrice
			t3Percent := (t3Distance / currentPrice) * 100
			t3R := (*position.Target3 - position.EntryPrice) / riskPerShare
			t3Hit := currentPrice >= *position.Target3
			hitEmoji := "⏳"
			if t3Hit {
				hitEmoji = "✅"
			}
			msgBuilder.WriteString(fmt.Sprintf("  %s T3: %s VND (+%.1f%%, %.1fR)\n", hitEmoji, formatPrice(*position.Target3), t3Percent, t3R))
		}
	}
	
	// Additional Info
	if (position.SignalType != nil && *position.SignalType != "") || (position.Score != nil && *position.Score > 0) {
		msgBuilder.WriteString("\n<b>ℹ️ Additional Info</b>\n")
		if position.SignalType != nil && *position.SignalType != "" {
			msgBuilder.WriteString(fmt.Sprintf("  Signal Type: %s\n", *position.SignalType))
		}
		if position.Score != nil && *position.Score > 0 {
			msgBuilder.WriteString(fmt.Sprintf("  Entry Score: %d/13\n", *position.Score))
		}
	}
	
	if position.Notes != nil && *position.Notes != "" {
		msgBuilder.WriteString(fmt.Sprintf("\n<i>Notes: %s</i>\n", *position.Notes))
	}

	s.SendMessage(chatID, msgBuilder.String())
}
