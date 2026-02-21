package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	positionsvc "github.com/nonobeam/golang-stock-trading/internal/service/position"
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

	// Use Service instead of Repo directly
	if s.positionSvc == nil {
		// Fallback to Repo if Svc not initialized (should be by SetPositionRepository)
		if s.positionRepo == nil {
             s.SendMessage(chatID, "Service not configured")
             return
        }
        s.positionSvc = positionsvc.NewService(s.positionRepo)
	}
    
    // Check if position exists
    ctx := context.Background()
    // For manual addposition, we want to allow override or force create? 
    // /addposition usually implies creating a new logical position, but internal logic prevents duplicates.
    // If it exists, we should probably warn or call AddEntry?
    // Let's stick to CreatePosition logic which mirrors handleBuyCommand updates now.
    
    req := positionsvc.CreatePositionRequest{
        UserID: 1, // Default user
        Symbol: symbol,
        Price: entryPrice,
        Shares: quantity,
        Date: time.Now(),
        StopLoss: stopLoss,
    }
    
    // Try to create. If duplicate, we might get error.
    // Ideally we check first.
    existing, _ := s.positionRepo.GetBySymbol(ctx, 1, symbol)
    if existing != nil {
         // Add Entry instead
         err = s.positionSvc.AddEntry(ctx, positionsvc.AddEntryRequest{
             UserID: 1,
             Symbol: symbol,
             Price: entryPrice,
             Shares: quantity,
             Type: "BUY_MORE",
             Date: time.Now(),
         })
         if err != nil {
             s.SendMessage(chatID, "Failed to add to existing position: " + err.Error())
             return
         }
         s.SendMessage(chatID, fmt.Sprintf("Added %d shares to existing %s position.", quantity, symbol))
         return
    }

    _, err = s.positionSvc.CreatePosition(ctx, req)
    if err != nil {
         s.SendMessage(chatID, "Failed to create position: " + err.Error())
         return
    }

	s.SendMessage(chatID, fmt.Sprintf("Position opened for %s", symbol))
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

// handleBuyCommand handles /buy <symbol> <quantity> <price> [date]
// Records a stock purchase in the positions table
func (s *BotService) handleBuyCommand(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	text := msg.Text

	// Check dependencies
	if s.positionRepo == nil {
		s.SendMessage(chatID, "Position management not configured")
		return
	}
	
	// Ensure service is ready (lazy init if missed)
	if s.positionSvc == nil {
		s.positionSvc = positionsvc.NewService(s.positionRepo)
	}

	// Parse arguments
	args := strings.Fields(text)
	if len(args) < 4 {
		usageText := "<b>Usage:</b> /buy &lt;symbol&gt; &lt;quantity&gt; &lt;price&gt; [date]\n\n" +
			"<b>Examples:</b>\n" +
			"  /buy VNM 100 85000\n" +
			"  /buy VNM 100 85000 2026-01-25\n\n" +
			"<b>Parameters:</b>\n" +
			"  symbol - Stock symbol (e.g., VNM, HPG)\n" +
			"  quantity - Number of shares (positive integer)\n" +
			"  price - Purchase price per share (positive number)\n" +
			"  date - Optional purchase date (YYYY-MM-DD format, defaults to today)"
		s.SendMessage(chatID, usageText)
		return
	}

	// Parse symbol
	symbol := strings.ToUpper(args[1])

	// Parse quantity
	quantity, err := strconv.Atoi(args[2])
	if err != nil {
		s.SendMessage(chatID, "Invalid quantity. Must be a number.")
		return
	}
	if quantity <= 0 {
		s.SendMessage(chatID, "Invalid quantity. Must be a positive number.")
		return
	}

	// Parse price
	price, err := strconv.ParseFloat(args[3], 64)
	if err != nil {
		s.SendMessage(chatID, "Invalid price. Must be a number.")
		return
	}
	if price <= 0 {
		s.SendMessage(chatID, "Invalid price. Must be a positive number.")
		return
	}

	// Parse optional date
	var purchaseDate time.Time
	if len(args) >= 5 {
		purchaseDate, err = time.Parse("2006-01-02", args[4])
		if err != nil {
			s.SendMessage(chatID, "Invalid date format. Please use YYYY-MM-DD (e.g., 2026-01-25)")
			return
		}
	} else {
		purchaseDate = time.Now()
	}

	// Get user context
	user, err := s.getUserContext(chatID)
	if err != nil {
		logger.Error().Err(err).Int64("chatID", chatID).Msg("Failed to get user context")
		s.SendMessage(chatID, "Failed to record purchase. Please try again.")
		return
	}

	ctx := context.Background()

	// Check if position exists
	existingPos, err := s.positionRepo.GetBySymbol(ctx, user.UserID, symbol)
	if err != nil {
		logger.Error().Err(err).Msg("DB Error checking position")
		s.SendMessage(chatID, "Internal error checking position")
		return
	}

	if existingPos != nil {
		// ADD ENTRY
		req := positionsvc.AddEntryRequest{
			UserID: user.UserID,
			Symbol: symbol,
			Shares: quantity,
			Price:  price,
			Date:   purchaseDate,
			Type:   "BUY_MORE",
		}

		if err := s.positionSvc.AddEntry(ctx, req); err != nil {
			logger.Error().Err(err).Msg("Failed to add entry")
			s.SendMessage(chatID, "Failed to add entry: "+err.Error())
			return
		}

		// Calculate new average
		// We could fetch updated details, but for speed just confirm
		confirmText := fmt.Sprintf(
			"<b>Added to %s</b>\n\n"+
				"<b>Quantity:</b> +%s shares\n"+
				"<b>Price:</b> %s VND\n"+
				"<b>Total Cost:</b> %s VND\n\n"+
				"Use /status to see updated average cost.",
			symbol,
			formatNumber(quantity),
			formatPrice(price),
			formatPrice(price*float64(quantity)),
		)
		s.SendMessage(chatID, confirmText)

	} else {
		// CREATE POSITION
		// We need StopLoss for CreatePosition. /buy command doesn't provide it strictly?
		// Logic: Default stop loss if not provided. Bot /buy usage says: symbol, qty, price, [date].
		// It does NOT have Stop Loss.
		// /addposition had stop loss.
		// For /buy, we can set default stop loss (e.g. 7% or 0). Or ask user?
		// Existing logic in /addposition used 3% automatic.
		// Let's use 7% automatic for /buy command simplifiction.
		defaultStopLoss := price * 0.93

		req := positionsvc.CreatePositionRequest{
			UserID:   user.UserID,
			Symbol:   symbol,
			Shares:   quantity,
			Price:    price,
			Date:     purchaseDate,
			StopLoss: defaultStopLoss, 
		}

		_, err := s.positionSvc.CreatePosition(ctx, req)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to create position")
			s.SendMessage(chatID, "Failed to create position: "+err.Error())
			return
		}

		confirmText := fmt.Sprintf(
			"<b>New Position: %s</b>\n\n"+
				"<b>Quantity:</b> %s shares\n"+
				"<b>Entry Price:</b> %s VND\n"+
				"<b>Stop Loss:</b> %s VND (Auto -7%%)\n"+
				"<b>Total Cost:</b> %s VND\n",
			symbol,
			formatNumber(quantity),
			formatPrice(price),
			formatPrice(defaultStopLoss),
			formatPrice(price*float64(quantity)),
		)
		s.SendMessage(chatID, confirmText)
	}
}

// handlePositionDetailCommand handles /position <symbol>
// Displays detailed information about a specific position
func (s *BotService) handlePositionDetailCommand(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	text := msg.Text

	// Verify dependencies
	if s.positionRepo == nil {
		s.SendMessage(chatID, "Position management not configured")
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
	msgBuilder.WriteString(fmt.Sprintf("<b>Position Details: %s</b>\n\n", symbol))
	
	// Entry Details
	msgBuilder.WriteString("<b>Entry Details</b>\n")
	msgBuilder.WriteString(fmt.Sprintf("  Entry Date: %s\n", position.EntryDate.Format("2006-01-02")))
	msgBuilder.WriteString(fmt.Sprintf("  Entry Price: %s VND\n", formatPrice(position.EntryPrice)))
	msgBuilder.WriteString(fmt.Sprintf("  Quantity: %s shares\n", formatNumber(position.Quantity)))
	msgBuilder.WriteString(fmt.Sprintf("  Position Value: %s VND\n\n", formatPrice(position.EntryPrice*float64(position.Quantity))))
	
	// Current Status
	msgBuilder.WriteString("<b>Current Status</b>\n")
	msgBuilder.WriteString(fmt.Sprintf("  Current Price: %s VND\n", formatPrice(currentPrice)))
	msgBuilder.WriteString(fmt.Sprintf("  Days in Trade: %d\n", daysInTrade))
	msgBuilder.WriteString(fmt.Sprintf("  Unrealized P&L: %s VND (<b>%+.2f%%</b>)\n", formatPrice(unrealizedPL), unrealizedPLPercent))
	msgBuilder.WriteString(fmt.Sprintf("  R-Multiple: <b>%+.2fR</b>\n\n", rMultiple))
	
	// Risk Management
	msgBuilder.WriteString("<b>Risk Management</b>\n")
	msgBuilder.WriteString(fmt.Sprintf("  Stop Loss: %s VND\n", formatPrice(position.StopLoss)))
	msgBuilder.WriteString(fmt.Sprintf("  Distance to Stop: %s VND (%.2f%%)\n", formatPrice(stopDistance), stopDistancePercent))
	msgBuilder.WriteString(fmt.Sprintf("  Risk/Share: %s VND\n", formatPrice(riskPerShare)))
	msgBuilder.WriteString(fmt.Sprintf("  Total Risk: %s VND\n", formatPrice(totalRisk)))
	
	// Targets (if any)
	if (position.Target1 != nil && *position.Target1 > 0) || 
	   (position.Target2 != nil && *position.Target2 > 0) || 
	   (position.Target3 != nil && *position.Target3 > 0) {
		msgBuilder.WriteString("\n<b>Targets</b>\n")
		
		if position.Target1 != nil && *position.Target1 > 0 {
			t1Distance := *position.Target1 - currentPrice
			t1Percent := (t1Distance / currentPrice) * 100
			t1R := (*position.Target1 - position.EntryPrice) / riskPerShare
			t1Hit := currentPrice >= *position.Target1
			hitStatus := ""
			if t1Hit {
				hitStatus = "[HIT]"
			}
			msgBuilder.WriteString(fmt.Sprintf("  T1: %s VND (+%.1f%%, %.1fR) %s\n", formatPrice(*position.Target1), t1Percent, t1R, hitStatus))
		}
		
		if position.Target2 != nil && *position.Target2 > 0 {
			t2Distance := *position.Target2 - currentPrice
			t2Percent := (t2Distance / currentPrice) * 100
			t2R := (*position.Target2 - position.EntryPrice) / riskPerShare
			t2Hit := currentPrice >= *position.Target2
			hitStatus := ""
			if t2Hit {
				hitStatus = "[HIT]"
			}
			msgBuilder.WriteString(fmt.Sprintf("  T2: %s VND (+%.1f%%, %.1fR) %s\n", formatPrice(*position.Target2), t2Percent, t2R, hitStatus))
		}
		
		if position.Target3 != nil && *position.Target3 > 0 {
			t3Distance := *position.Target3 - currentPrice
			t3Percent := (t3Distance / currentPrice) * 100
			t3R := (*position.Target3 - position.EntryPrice) / riskPerShare
			t3Hit := currentPrice >= *position.Target3
			hitStatus := ""
			if t3Hit {
				hitStatus = "[HIT]"
			}
			msgBuilder.WriteString(fmt.Sprintf("  T3: %s VND (+%.1f%%, %.1fR) %s\n", formatPrice(*position.Target3), t3Percent, t3R, hitStatus))
		}
	}
	
	// Additional Info
	if (position.SignalType != nil && *position.SignalType != "") || (position.Score != nil && *position.Score > 0) {
		msgBuilder.WriteString("\n<b>Additional Info</b>\n")
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
