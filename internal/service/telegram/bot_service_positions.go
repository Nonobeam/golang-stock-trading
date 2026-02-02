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

