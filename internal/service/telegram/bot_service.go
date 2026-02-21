package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/nonobeam/golang-stock-trading/internal/config"
	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
	"github.com/nonobeam/golang-stock-trading/internal/errors"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	mlconfig "github.com/nonobeam/golang-stock-trading/internal/ml/config"
	"github.com/nonobeam/golang-stock-trading/internal/regime/ftd"
	"github.com/nonobeam/golang-stock-trading/internal/service"
	positionsvc "github.com/nonobeam/golang-stock-trading/internal/service/position"
	"github.com/nonobeam/golang-stock-trading/internal/vn"
	"github.com/nonobeam/golang-stock-trading/proto/ml"
)

// ImportState tracks the state of an ongoing file import
type ImportState struct {
	WaitingFile bool   // True when bot is waiting for user to upload a file
	Symbol      string // The stock symbol for this import
	Provider    string // Data provider ("simplize", "other")
}

type BotService struct {
	bot          *tgbotapi.BotAPI
	activeChatID int64
	otpChans     map[int64]chan string
	mu           sync.RWMutex
	waitingOTP   map[int64]bool
	isRestartOTP map[int64]bool

	// User management
	userConfigRepo *repository.UserConfigRepository

	// Optional dependencies for commands
	riskManager      RiskManager
	positionTracker  PositionTracker
	restartHandler   RestartHandler
	positionRepo     *repository.PositionRepository
	positionSvc      *positionsvc.Service
	marketDataSvc    *service.MarketDataService
	watchlistRepo    *repository.WatchlistRepository
	regimeRepo       *ftd.Repository // Add this
	
	// Import state and ML client
	importState ImportState
	mlClient    ml.MLPredictionServiceClient
	
	// Single channel for app-level OTP request (legacy)
	otpChan chan string
}

func NewBotService(cfg *config.Config, userConfigRepo *repository.UserConfigRepository) (*BotService, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create Telegram bot")
		return nil, errors.Wrap(err, errors.ErrTelegramConnection)
	}

	logger.Info().Str("username", bot.Self.UserName).Msg("Telegram bot authorized")

	return &BotService{
		bot:            bot,
		activeChatID:   cfg.TelegramChatID,
		userConfigRepo: userConfigRepo,
		otpChans:       make(map[int64]chan string),
		waitingOTP:     make(map[int64]bool),
		isRestartOTP:   make(map[int64]bool),
		otpChan:        make(chan string),
	}, nil
}

func (s *BotService) Start(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := s.bot.GetUpdatesChan(u)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case update := <-updates:
				// Handle regular messages
				if update.Message != nil {
					s.handleMessage(update.Message)
				}

				// Handle callback queries (inline keyboard buttons)
				if update.CallbackQuery != nil {
					s.handleCallbackQuery(update.CallbackQuery)
				}
			}
		}
	}()
}

func (s *BotService) handleMessage(msg *tgbotapi.Message) {
	text := msg.Text
	chatID := msg.Chat.ID
	logger.Debug().Str("text", text).Int64("chatID", chatID).Msg("Received Telegram message")

	// Commands always take priority - check if this is a command first
	if msg.IsCommand() {
		s.handleCommand(msg)
		return
	}
	
	// Check for file upload (document)
	if msg.Document != nil {
		s.mu.Lock()
		waitingFile := s.importState.WaitingFile
		s.mu.Unlock()
		
		if waitingFile {
			s.handleFileUpload(msg)
			return
		}
	}

	// Non-command messages: check if we're waiting for OTP
	s.mu.RLock()
	waiting := s.waitingOTP[chatID]
	// Also check general waiting flag for app startup
	appWaiting := s.waitingOTP[0] // Hack using 0 for app-level
	s.mu.RUnlock()

	if waiting || appWaiting {
		otp := extractOTP(text)
		if otp != "" {
			// Check if this is a restart OTP
			s.mu.RLock()
			isRestart := s.isRestartOTP[chatID]
			s.mu.RUnlock()

			if isRestart {
				s.handleRestartOTP(chatID, otp)
			} else {
				// Send OTP to the user's channel
				s.mu.RLock()
				ch, exists := s.otpChans[chatID]
				s.mu.RUnlock()
				if exists {
					ch <- otp
					s.SendMessage(chatID, "Smart OTP received! Exchanging for trading token...")
				}
				
				// Send to app-level channel if waiting
				select {
				case s.otpChan <- otp:
					s.SendMessage(chatID, "Smart OTP received by application!")
				default:
				}
			}
		} else {
			s.SendMessage(chatID, "Invalid OTP format. Please send exactly 6 digits.")
		}
		return
	}

	// Not waiting for OTP and not a command
	if text != "" {
		s.SendMessage(chatID, "I'm not waiting for Smart OTP right now. Type /help to see available commands.")
	}
}

// handleCommand processes bot commands
func (s *BotService) handleCommand(msg *tgbotapi.Message) {
	text := msg.Text
	chatID := msg.Chat.ID

	switch msg.Command() {
	case "start":
		s.SendMessage(chatID, "Welcome to Stock Trading Bot!\\n\\nI will notify you when Smart OTP is needed. Get OTP from EntradeX app.\\n\\nType /help to see available commands.")
	case "status":
		s.handleStatusCommand(msg)
	case "help":
		helpText := "<b>Available Commands:</b>\n\n" +
			"<b>Data & AI:</b>\n" +
			"/import &lt;symbol&gt; - Import historical data\n" +
			"/train &lt;symbol&gt; - Train AI model\n" +
			"/predict &lt;symbol&gt; - Get ML predictions\n\n" +
			"<b>Signals:</b>\n" +
			"/signals - View today's signals\n" +
			"/watch &lt;symbol&gt; - Add to watchlist\n" +
			"/unwatch &lt;symbol&gt; - Remove from watchlist\n\n" +
			"<b>Portfolio:</b>\n" +
			"/scan [date] - Weekly portfolio scan (50 stocks)\n" +

			"/buy &lt;symbol&gt; &lt;qty&gt; &lt;price&gt; [date] - Record purchase\n" +

			"/risk - Show current risk status\n" +

			"/limits - Show all risk limits\n" +

			"/positions - List active positions\n" +

			"/position &lt;symbol&gt; - Position details\n\n" +

			"<b>General:</b>\n" +
			"/restart - Re-authenticate with new OTP\n" +
			"/status - Check bot status\n" +
			"/help - Show this help"
		s.SendMessage(chatID, helpText)
	case "import":
		s.handleImportCommand(msg)
	case "train":
		s.handleTrainCommand(msg)
	case "predict":
		s.handlePredictCommand(msg)
	case "time":
		s.handleTimeCommand(msg)
	case "signals":
		s.SendMessage(chatID, "<b>Today's Signals</b>\n\n<i>Feature coming soon. Signals will be shown here.</i>")
	case "watch":
		args := strings.Fields(text)
		if len(args) < 2 {
			s.SendMessage(chatID, "Usage: /watch &lt;symbol&gt;\nExample: /watch VNM")
			return
		}
		symbol := strings.ToUpper(args[1])
		
		// Get user context
		user, err := s.getUserContext(chatID)
		if err != nil {
			logger.Error().Err(err).Int64("chatID", chatID).Msg("Failed to get user context")
			s.SendMessage(chatID, "Failed to add stock to watchlist. Please try again.")
			return
		}
		
		// Check if watchlist repository is available
		if s.watchlistRepo == nil {
			s.SendMessage(chatID, "Watchlist feature not available. Please contact administrator.")
			return
		}
		
		// Add to watchlist
		ctx := context.Background()
		err = s.watchlistRepo.Create(ctx, user.UserID, symbol)
		if err != nil {
			logger.Error().Err(err).Str("symbol", symbol).Int64("userID", user.UserID).Msg("Failed to add to watchlist")
			s.SendMessage(chatID, fmt.Sprintf("Failed to add <code>%s</code> to watchlist: %s", symbol, err.Error()))
			return
		}
		
		s.SendMessage(chatID, fmt.Sprintf("Added <code>%s</code> to your watchlist!\n\nUse /status to view all tracked stocks.", symbol))
	case "unwatch":
		args := strings.Fields(text)
		if len(args) < 2 {
			s.SendMessage(chatID, "Usage: /unwatch &lt;symbol&gt;\nExample: /unwatch VNM")
			return
		}
		symbol := strings.ToUpper(args[1])
		
		// Get user context
		user, err := s.getUserContext(chatID)
		if err != nil {
			logger.Error().Err(err).Int64("chatID", chatID).Msg("Failed to get user context")
			s.SendMessage(chatID, "Failed to remove stock from watchlist. Please try again.")
			return
		}
		
		// Check if watchlist repository is available
		if s.watchlistRepo == nil {
			s.SendMessage(chatID, "Watchlist feature not available. Please contact administrator.")
			return
		}
		
		// Remove from watchlist
		ctx := context.Background()
		err = s.watchlistRepo.Delete(ctx, user.UserID, symbol)
		if err != nil {
			logger.Error().Err(err).Str("symbol", symbol).Int64("userID", user.UserID).Msg("Failed to remove from watchlist")
			s.SendMessage(chatID, fmt.Sprintf("Failed to remove <code>%s</code> from watchlist: %s", symbol, err.Error()))
			return
		}
		
		s.SendMessage(chatID, fmt.Sprintf("🗑 Removed <code>%s</code> from your watchlist.", symbol))
	case "buy":
		s.handleBuyCommand(msg)
	case "risk":
		s.handleRiskCommand(msg)
	case "limits":
		s.handleLimitsCommand(msg)
	case "positions":
		s.handlePositionsCommand(msg)
	case "addposition":
		s.handleAddPositionCommand(msg)
	case "editposition":
		s.handleEditPositionCommand(msg)
	case "position":
		s.handlePositionDetailCommand(msg)
	case "settlement":
		// TODO: Implement settlement status command
		s.SendMessage(msg.Chat.ID, "Settlement status command not yet implemented")
	case "lockedrisk":
		// TODO: Implement locked risk command
		s.SendMessage(msg.Chat.ID, "Locked risk command not yet implemented")
	case "restart":
		s.handleRestartCommand(msg)
	case "ftd":
		s.handleFTDCommand(msg)
	case "scan":
		s.handleScanCommand(msg)
	}
}

// getUserContext retrieves or creates a user configuration based on their chat ID
func (s *BotService) getUserContext(chatID int64) (*repository.UserConfig, error) {
	ctx := context.Background()
	user, err := s.userConfigRepo.GetOrCreateUserByChatID(ctx, chatID)
	if err != nil {
		logger.Error().Err(err).Int64("chatID", chatID).Msg("Failed to get or create user")
		return nil, err
	}
	
	if user.UserID == 0 {
		logger.Warn().Int64("chatID", chatID).Msg("User created with zero ID")
	}
	
	return user, nil
}

// SendMessage sends a message to a specific chat ID
func (s *BotService) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	_, err := s.bot.Send(msg)
	if err != nil {
		logger.Error().Err(err).Int64("chatID", chatID).Msg("Failed to send Telegram message")
		return err
	}
	return nil
}

// Broadcast sends a message to the active (admin) chat ID
func (s *BotService) Broadcast(text string) error {
	if s.activeChatID == 0 {
		return fmt.Errorf("no active chat ID for broadcast")
	}
	return s.SendMessage(s.activeChatID, text)
}

func (s *BotService) RequestSmartOTP(timeout time.Duration) (string, error) {
	s.mu.Lock()
	s.waitingOTP[0] = true // 0 for app-level request
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.waitingOTP[0] = false
		s.mu.Unlock()
	}()

	select {
	case <-s.otpChan:
	default:
	}

	timeoutMsg := fmt.Sprintf("%.0f", timeout.Minutes())
	// Use activeChatID if available, else this will fail or broadcast?
	// Assuming activeChatID is set from config
	if s.activeChatID == 0 {
		return "", fmt.Errorf("active chat ID not configured")
	}
	
	err := s.SendMessage(s.activeChatID, "<b>Smart OTP Required</b>\n\nPlease open <b>EntradeX</b> app and send your 6-digit Smart OTP.\n\nTimeout: " + timeoutMsg + " minutes")
	if err != nil {
		return "", err
	}

	select {
	case otp := <-s.otpChan:
		logger.Info().Msg("Smart OTP received from Telegram")
		return otp, nil
	case <-time.After(timeout):
		s.SendMessage(s.activeChatID, "Smart OTP timeout. Please run the application again.")
		return "", errors.ErrTradingTokenTimeout
	}
}

func (s *BotService) NotifyStockAlert(symbol string, price float64, alertType string) error {
	msg := fmt.Sprintf("<b>Stock Alert</b>\n\nSymbol: <code>%s</code>\nPrice: <code>%.2f</code>\nType: %s", symbol, price, alertType)
	return s.Broadcast(msg)
}

func (s *BotService) NotifyTradeExecuted(symbol string, side string, quantity int, price float64) error {
	msg := fmt.Sprintf("<b>Trade Executed</b>\n\nSymbol: <code>%s</code>\nSide: %s\nQuantity: %d\nPrice: %.2f", symbol, side, quantity, price)
	return s.Broadcast(msg)
}

// NotifyPriceMonitorAlert sends a price change alert with advice
func (s *BotService) NotifyPriceMonitorAlert(symbol string, price, changePct float64, alertType, advice string) error {
	emoji := "⚠️"
	if changePct > 0 {
		emoji = "🚀"
	}

	msg := fmt.Sprintf(
		"<b>%s PRICE ALERT</b>\n\n"+
			"<b>Symbol:</b> <code>%s</code>\n"+
			"<b>Price:</b> %s VND\n"+
			"<b>Change:</b> %.2f%%\n\n"+
			"<i>%s</i>",
		emoji,
		symbol,
		formatPrice(price),
		changePct*100,
		advice,
	)
	return s.Broadcast(msg)
}

func extractOTP(text string) string {
	otp := strings.TrimSpace(text)
	if len(otp) == 6 {
		for _, c := range otp {
			if c < '0' || c > '9' {
				return ""
			}
		}
		return otp
	}
	return ""
}

func (s *BotService) GetBot() *tgbotapi.BotAPI {
	return s.bot
}

// NotifySignalDetected sends a formatted trading signal alert to Telegram
func (s *BotService) NotifySignalDetected(symbol, signalType string, score int, entryPrice, stopLoss float64, targets []float64, positionSize int, riskAmount float64, regime string, detectedAt time.Time) error {
	// Choose emoji based on score
	var scoreEmoji string
	switch {
	case score >= 10:
		scoreEmoji = "⭐⭐⭐" // Excellent
	case score >= 9:
		scoreEmoji = "⭐⭐" // Very Good
	case score >= 7:
		scoreEmoji = "⭐" // Good
	default:
		scoreEmoji = "(Fair)" // Fair
	}

	// Format targets
	var targetText strings.Builder
	for i, target := range targets {
		if target > 0 {
			rMultiple := (target - entryPrice) / (entryPrice - stopLoss)
			percentGain := ((target - entryPrice) / entryPrice) * 100
			targetText.WriteString(fmt.Sprintf("  T%d: %s (+%.1f%%, %.1fR)\n", i+1, formatPrice(target), percentGain, rMultiple))
		}
	}

	// Calculate risk percentage
	stopPercent := ((entryPrice - stopLoss) / entryPrice) * 100

	msg := fmt.Sprintf(
		"<b>NEW SIGNAL DETECTED</b>\n\n"+
			"<b>Symbol:</b> <code>%s</code>\n"+
			"<b>Type:</b> %s\n"+
			"<b>Score:</b> %d/13 <i>(%s) %s</i>\n\n"+
			"<b>Entry:</b> %s VND\n"+
			"<b>Stop:</b> %s VND <i>(%.1f%%)</i>\n"+
			"<b>Targets:</b>\n%s\n"+
			"<b>Position Size:</b> %s shares\n"+
			"<b>Risk:</b> %s VND\n\n"+
			"<b>Regime:</b> %s\n"+
			"<b>Detected:</b> %s",
		symbol,
		signalType,
		score,
		getScoreQuality(score),
		scoreEmoji,
		formatPrice(entryPrice),
		formatPrice(stopLoss),
		stopPercent,
		targetText.String(),
		formatNumber(positionSize),
		formatPrice(riskAmount),
		regime,
		detectedAt.Format("2006-01-02 15:04:05"),
	)

	return s.Broadcast(msg)
}

// NotifyBatchSignals sends a daily summary of detected signals
func (s *BotService) NotifyBatchSignals(signals []SignalSummary) error {
	if len(signals) == 0 {
		return s.Broadcast("<b>Daily Signal Summary</b>\n\nNo signals detected today.")
	}

	// Group by score tier
	excellent := []SignalSummary{} // 10-13
	veryGood := []SignalSummary{}  // 9
	good := []SignalSummary{}      // 7-8
	fair := []SignalSummary{}      // <7

	for _, sig := range signals {
		switch {
		case sig.Score >= 10:
			excellent = append(excellent, sig)
		case sig.Score == 9:
			veryGood = append(veryGood, sig)
		case sig.Score >= 7:
			good = append(good, sig)
		default:
			fair = append(fair, sig)
		}
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("<b>Daily Signal Summary</b>\n\n<b>Total Signals:</b> %d\n\n", len(signals)))

	if len(excellent) > 0 {
		msg.WriteString(fmt.Sprintf("⭐⭐⭐ <b>Excellent (10-13):</b> %d\n", len(excellent)))
		for _, sig := range excellent {
			msg.WriteString(fmt.Sprintf("  • %s (%s) - Score: %d\n", sig.Symbol, sig.Type, sig.Score))
		}
		msg.WriteString("\n")
	}

	if len(veryGood) > 0 {
		msg.WriteString(fmt.Sprintf("⭐⭐ <b>Very Good (9):</b> %d\n", len(veryGood)))
		for _, sig := range veryGood {
			msg.WriteString(fmt.Sprintf("  • %s (%s) - Score: %d\n", sig.Symbol, sig.Type, sig.Score))
		}
		msg.WriteString("\n")
	}

	if len(good) > 0 {
		for _, sig := range good {
			msg.WriteString(fmt.Sprintf("  • %s (%s) - Score: %d\n", sig.Symbol, sig.Type, sig.Score))
		}
		msg.WriteString("\n")
	}

	if len(fair) > 0 {
		msg.WriteString(fmt.Sprintf("<b>Fair (<7):</b> %d\n", len(fair)))
	}

	msg.WriteString("\n<i>Use /signals for detailed information</i>")

	return s.Broadcast(msg.String())
}

// SignalSummary represents a signal for batch notifications
type SignalSummary struct {
	Symbol string
	Type   string
	Score  int
}

// Helper functions for formatting

func formatPrice(price float64) string {
	return fmt.Sprintf("%s", formatNumber(int(price)))
}

func formatNumber(n int) string {
	// Add thousand separators
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteString(",")
		}
		result.WriteRune(c)
	}
	return result.String()
}

func getScoreQuality(score int) string {
	switch {
	case score >= 10:
		return "Excellent"
	case score >= 9:
		return "Very Good"
	case score >= 7:
		return "Good"
	default:
		return "Fair"
	}
}

// Command handlers

func (s *BotService) handleRiskCommand(msg *tgbotapi.Message) {
	if s.riskManager == nil {
	chatID := msg.Chat.ID
		s.SendMessage(chatID, "⚠️ Risk manager not configured")
		return
	}

	portfolioRisk := s.riskManager.GetPortfolioRisk() * 100
	maxRisk := s.riskManager.GetMaxPortfolioRisk() * 100
	dailyLoss := s.riskManager.GetDailyLoss() * 100
	dailyLimit := s.riskManager.GetDailyLossLimit() * 100
	capitalUtil := s.riskManager.GetCapitalUtilization() * 100

	// Determine status emoji
	// Determine status
	riskStatus := "Normal"
	if portfolioRisk >= maxRisk*0.9 {
		riskStatus = "CRITICAL"
	} else if portfolioRisk >= maxRisk*0.8 {
		riskStatus = "WARNING"
	}

	respText := fmt.Sprintf(
		"<b>Portfolio Risk Status</b> [%s]\n\n"+
			"<b>Portfolio Risk:</b> %.2f%% / %.2f%%\n"+
			"<b>Daily Loss:</b> %.2f%% / %.2f%%\n"+
			"<b>Capital Used:</b> %.1f%%\n\n"+
			"<i>Use /limits for detailed limits</i>",
		riskStatus,
		portfolioRisk,
		maxRisk,
		dailyLoss,
		dailyLimit,
		capitalUtil,
	)

	s.SendMessage(msg.Chat.ID, respText)
}

func (s *BotService) handleLimitsCommand(msg *tgbotapi.Message) {
	if s.riskManager == nil {
	chatID := msg.Chat.ID
		s.SendMessage(chatID, "Risk manager not configured")
		return
	}

	respText := "<b>Risk Limits</b>\n\n" +
		"<b>Portfolio Limits:</b>\n" +
		"  • Max Risk: 6.0%\n" +
		"  • Max Positions: 3\n" +
		"  • Max Per Position: 2%\n\n" +
		"<b>Daily Limits:</b>\n" +
		"  • Max Daily Loss: 2.0%\n" +
		"  • Max Weekly Loss: 5.0%\n" +
		"  • Max Monthly Loss: 10.0%\n\n" +
		"<b>Position Limits:</b>\n" +
		"  • Max Sector Exposure: 50%\n" +
		"  • Min Liquidity: 100M VND/day\n\n" +
		"<i>Use /risk for current status</i>"

	s.SendMessage(msg.Chat.ID, respText)
}

func (s *BotService) handlePositionsCommand(msg *tgbotapi.Message) {
	if s.positionTracker == nil {
		s.SendMessage(msg.Chat.ID, "Position tracker not configured")
		return
	}

	// Get user context for capital
	user, err := s.getUserContext(msg.Chat.ID)
	capital := 0.0
	if err == nil && user.InitialCapital > 0 {
		capital = user.InitialCapital
	}

	positions := s.positionTracker.GetActivePositions()

	if len(positions) == 0 {
		s.SendMessage(msg.Chat.ID, "<b>Active Positions</b>\n\n<i>No active positions</i>")
		return
	}

	var msgBuilder strings.Builder
	msgBuilder.WriteString(fmt.Sprintf("<b>Active Positions</b> (%d)\n\n", len(positions)))

	totalAllocated := 0.0

	for _, pos := range positions {
		pnlStatus := "(+)"
		if pos.RMultiple < 0 {
			pnlStatus = "(-)"
		}

		// Calculate Allocation
		marketVal := pos.CurrentPrice * float64(pos.PositionSize)
		allocPct := 0.0
		allocStr := ""
		if capital > 0 {
			allocPct = (marketVal / capital) * 100
			totalAllocated += allocPct
			allocWarning := ""
			if allocPct > 20.0 {
				allocWarning = " ⚠️"
			}
			allocStr = fmt.Sprintf(" | Alloc: %.1f%%%s", allocPct, allocWarning)
		}

		msgBuilder.WriteString(fmt.Sprintf(
			"<b>%s</b> %s\n"+
				"  Entry: %s | Current: %s\n"+
				"  P&L: %+.2fR %s | Progress: %.0f%%\n"+
				"  Size: %d shares%s\n\n",
			pos.Symbol,
			pnlStatus,
			formatPrice(pos.EntryPrice),
			formatPrice(pos.CurrentPrice),
			pos.RMultiple,
			pnlStatus,
			pos.TargetProgress,
			pos.PositionSize,
			allocStr,
		))
	}

	if capital > 0 {
		msgBuilder.WriteString(fmt.Sprintf("<b>Total Allocation:</b> %.1f%%\n", totalAllocated))
	}

	msgBuilder.WriteString("<i>Use /position <symbol> for details</i>")

	s.SendMessage(msg.Chat.ID, msgBuilder.String())
}

// SetRiskManager sets the risk manager for command handling
func (s *BotService) SetRiskManager(rm RiskManager) {
	s.riskManager = rm
}

// SetPositionTracker sets the position tracker for command handling
func (s *BotService) SetPositionTracker(pt PositionTracker) {
	s.positionTracker = pt
}

// SetRestartHandler sets the restart handler for re-authentication
func (s *BotService) SetRestartHandler(rh RestartHandler) {
	s.restartHandler = rh
}

// SetPositionService sets the position service
func (s *BotService) SetPositionService(svc *positionsvc.Service) {
	s.positionSvc = svc
}

// SetPositionRepository sets the position repository for querying positions
func (s *BotService) SetPositionRepository(repo *repository.PositionRepository) {
	s.positionRepo = repo
	if s.positionSvc == nil {
		s.positionSvc = positionsvc.NewService(repo)
	}
}

// SetMarketDataService sets the market data service for price fetching
func (s *BotService) SetMarketDataService(svc *service.MarketDataService) {
	s.marketDataSvc = svc
}

// SetWatchlistRepository sets the watchlist repository for tracking queries
func (s *BotService) SetWatchlistRepository(repo *repository.WatchlistRepository) {
	s.watchlistRepo = repo
}

// SetMLClient sets the ML service client for training commands
func (s *BotService) SetMLClient(client ml.MLPredictionServiceClient) {
	s.mlClient = client
}

// SetRegimeRepository sets the regime repository for querying FTD status
func (s *BotService) SetRegimeRepository(repo *ftd.Repository) {
	s.regimeRepo = repo
}

// handleFTDCommand handles the /ftd command
func (s *BotService) handleFTDCommand(msg *tgbotapi.Message) {
	if s.regimeRepo == nil {
		s.SendMessage(msg.Chat.ID, "Market Regime repository not configured")
		return
	}

	ctx := context.Background()
	regime, err := s.regimeRepo.GetLatestMarketRegime(ctx)
	if err != nil {
		s.SendMessage(msg.Chat.ID, "Failed to retrieve market regime: "+err.Error())
		return
	}
	
	if regime == nil {
		s.SendMessage(msg.Chat.ID, "No market regime data available.")
		return
	}

	// Format message
	statusEmoji := "⚪"
	if regime.IsFTD {
		statusEmoji = "🟢" // FTD Confirmed
	} else if regime.RallyAttemptDay != nil {
		statusEmoji = "🟡" // Rally Attempt
	} else {
		statusEmoji = "🔴" // Downtrend/Correction
	}

	ftdInfo := "N/A"
	if regime.IsFTD {
		ftdInfo = "CONFIRMED"
	} else if regime.RallyAttemptDay != nil {
		ftdInfo = fmt.Sprintf("Day %d", *regime.RallyAttemptDay)
	}

	resp := fmt.Sprintf(
		"<b>Market Regime Status</b> %s\n\n"+
			"<b>Date:</b> %s\n"+
			"<b>Index:</b> %.2f\n"+
			"<b>Status:</b> %s\n"+
			"<b>FTD Status:</b> %s\n"+
			"<b>Distribution Days:</b> %d\n\n"+
			"<b>Scores:</b>\n"+
			"• Leader Participation: %d\n"+
			"• Configured FTD Score: %d\n",
		statusEmoji,
		regime.Date.Format("2006-01-02"),
		regime.IndexValue,
		statusEmoji, // Simplified status visual
		ftdInfo,
		regime.DistributionDayCount,
		regime.LeaderParticipationScore,
		0, // TODO: Store computed score in DB or calc on fly?
	)
	
	s.SendMessage(msg.Chat.ID, resp)
}

// handleRestartCommand handles the /restart command
func (s *BotService) handleRestartCommand(msg *tgbotapi.Message) {
	if s.restartHandler == nil {
	chatID := msg.Chat.ID
		s.SendMessage(chatID, "Restart handler not configured. Please restart the application manually.")
		return
	}

	s.mu.Lock()
	// Check if this specific user is waiting
	if s.waitingOTP[msg.Chat.ID] {
		s.mu.Unlock() // CRITICAL: Unlock before returning!
		s.SendMessage(msg.Chat.ID, "<b>Already Waiting for OTP</b>\n\nI am already expecting an OTP (likely for startup). Please just send the 6-digit code directly.")
		return
	}

	// Set waiting state (mutex already locked above)
	s.waitingOTP[msg.Chat.ID] = true
	s.isRestartOTP[msg.Chat.ID] = true
	s.mu.Unlock()

	s.SendMessage(msg.Chat.ID, "🔄 <b>Restarting Authentication</b>\n\nPlease open <b>EntradeX</b> app and send your 6-digit Smart OTP.")
}

func (s *BotService) handleTimeCommand(msg *tgbotapi.Message) {
	now := time.Now()
	
	// Get Vietnam time using the correct location from vn package
	vnTime := now
	// We can't access vn.vnLocation directly, but we can verify session
	session, err := vn.GetSessionForTime(now)
	
	sessionStatus := "CLOSED"
	if err == nil && session != nil {
		sessionStatus = "OPEN (" + session.Name + ")"
	} else if err == vn.ErrLunchBreak {
		sessionStatus = "LUNCH BREAK"
	}

	// Try to get Vietnam time explicitly for display
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err == nil {
		vnTime = now.In(loc)
	} else {
		// Fallback
		loc = time.FixedZone("Asia/Ho_Chi_Minh", 7*60*60)
		vnTime = now.In(loc)
	}

	respText := fmt.Sprintf(
		"<b>System Time Status</b>\n\n"+
			"<b>Server Time:</b> %s\n"+
			"<b>Vietnam Time:</b> %s\n"+
			"<b>Market Status:</b> %s",
		now.Format("15:04:05 Mon 02/01"),
		vnTime.Format("15:04:05 Mon 02/01"),
		sessionStatus,
	)
	s.SendMessage(msg.Chat.ID, respText)
}

// handleRestartOTP handles OTP input for restart flow
func (s *BotService) handleRestartOTP(chatID int64, otp string) {
	// Reset OTP waiting state
	defer func() {
		s.mu.Lock()
		delete(s.waitingOTP, chatID)
		delete(s.isRestartOTP, chatID)
		s.mu.Unlock()
	}()

	s.SendMessage(chatID, "Smart OTP received! Refreshing trading token...")

	// Call restart handler
	ctx := context.Background()
	if err := s.restartHandler.OnRestart(ctx, otp); err != nil {
		logger.Error().Err(err).Msg("Restart authentication failed")
		s.SendMessage(chatID, fmt.Sprintf("<b>Restart Failed</b>\n\nError: %s\n\nPlease try /restart again or restart the application.", err.Error()))
		return
	}

	s.SendMessage(chatID, "<b>Trading token refreshed successfully!</b>\n\nBot is ready for trading operations.")
}

// handleTrainCommand handles /train <SYMBOL> and /train all commands
func (s *BotService) handleTrainCommand(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	args := strings.Fields(msg.Text)
	if len(args) < 2 {
		s.SendMessage(chatID,
			"<b>Usage:</b>\n"+
				"  /train &lt;SYMBOL&gt; — train one stock (e.g. /train VCB)\n"+
				"  /train all       — retrain all stocks in the universe\n\n"+
				"Make sure data is imported first with /import.")
		return
	}

	if s.mlClient == nil {
		s.SendMessage(chatID, "<b>Error:</b> ML service not configured. Please contact administrator.")
		logger.Error().Msg("ML client not set in BotService")
		return
	}

	symbol := strings.ToUpper(args[1])

	// ── /train all ───────────────────────────────────────────────────────────
	if symbol == "ALL" {
		s.SendMessage(chatID,
			"<b>Bulk Training Started</b>\n\n"+
				"Backfill + train + predict for all tickers in the universe.\n"+
				"<i>You will receive a status message after each stock is done.</i>")

		// Open the gRPC stream in a goroutine — it blocks until all tickers are done
		go func(cid int64) {
			// No deadline — training 50 tickers could take 30+ minutes
			ctx := context.Background()
			stream, err := s.mlClient.TriggerBulkRetrain(ctx, &ml.TriggerBulkRetrainRequest{Force: true})
			if err != nil {
				logger.Error().Err(err).Msg("TriggerBulkRetrain stream open failed")
				s.SendMessage(cid, "<b>Bulk Train Failed</b>\n\nCould not reach ML service:\n"+err.Error())
				return
			}

			// Read updates as they arrive and forward to Telegram
			for {
				update, err := stream.Recv()
				if err != nil {
					// Stream closed — either done or error
					if err.Error() != "EOF" {
						logger.Error().Err(err).Msg("BulkRetrain stream error")
						s.SendMessage(cid, "<b>Bulk Train stream interrupted:</b>\n"+err.Error())
					}
					return
				}
				s.SendMessage(cid, update.Message)
			}
		}(chatID)
		return
	}

	// ── /train <SYMBOL> ──────────────────────────────────────────────────────
	s.SendMessage(chatID, fmt.Sprintf(
		"<b>Training AI Model</b>\n\n"+
			"Symbol: <code>%s</code>\n\n"+
			"This may take 2–5 minutes...",
		symbol))

	go func(cid int64) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		resp, err := s.mlClient.TriggerTraining(ctx, &ml.TriggerTrainingRequest{
			Ticker: symbol,
		})

		if err != nil {
			logger.Error().Err(err).Str("symbol", symbol).Msg("TriggerTraining gRPC failed")
			s.SendMessage(cid, fmt.Sprintf(
				"<b>Training Failed</b>\n\n"+
					"Symbol: <code>%s</code>\n"+
					"Error: %s\n\n"+
					"Please check:\n"+
					"1. You have imported data for this symbol\n"+
					"2. The ML service is running",
				symbol, err.Error()))
			return
		}

		if !resp.Success {
			s.SendMessage(cid, fmt.Sprintf(
				"<b>Training Failed</b>\n\n"+
					"Symbol: <code>%s</code>\n"+
					"Error: %s",
				symbol, resp.ErrorMessage))
			return
		}

		s.SendMessage(cid, fmt.Sprintf(
			"<b>Training Complete!</b>\n\n"+
				"Symbol: <code>%s</code>\n"+
				"Model Version: <code>%s</code>\n\n"+
				"The AI model is now ready for predictions.",
			symbol, resp.ModelVersion))
	}(chatID)
}

// handleImportCommand handles /import <SYMBOL> command
// Initiates the file upload flow for importing historical data
func (s *BotService) handleImportCommand(msg *tgbotapi.Message) {
	args := strings.Fields(msg.Text)
	if len(args) < 2 {
		s.SendMessage(msg.Chat.ID,
			"<b>Usage:</b> /import &lt;SYMBOL&gt;\n\n" +
			"<b>Example:</b> /import VCB\n\n" +
			"After running this command, upload your Excel file from Simplize.")
		return
	}

	symbol := strings.ToUpper(args[1])
	
	// Validate symbol format (basic check for Vietnamese stock symbols)
	if len(symbol) < 2 || len(symbol) > 10 {
		s.SendMessage(msg.Chat.ID, "Invalid symbol format. Please use 2-10 uppercase letters.")
		return
	}

	// Set state to wait for file upload
	s.mu.Lock()
	s.importState = ImportState{
		WaitingFile: true,
		Symbol:      symbol,
	}
	s.mu.Unlock()

	s.SendMessage(msg.Chat.ID, fmt.Sprintf(
		"<b>Import Historical Data</b>\n\n" +
		"Symbol: <code>%s</code>\n\n" +
		"Please upload the Excel file (.xlsx) from Simplize.\n\n" +
		"<i>Note: The file should contain OHLC (Open, High, Low, Close) data with dates.</i>",
		symbol))
}

// handleFileUpload processes uploaded Excel files
func (s *BotService) handleFileUpload(msg *tgbotapi.Message) {
	s.mu.Lock()
	symbol := s.importState.Symbol
	s.mu.Unlock()

	logger.Info().Str("symbol", symbol).Str("fileName", msg.Document.FileName).Msg("File upload received")

	// Step 1: Get file from Telegram
	fileID := msg.Document.FileID
	chatID := msg.Chat.ID
	file, err := s.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get file info from Telegram")
		s.SendMessage(chatID, fmt.Sprintf("<b>Error:</b> Failed to get file information: %v", err))
		s.resetImportState()
		return
	}

	// Step 2: Download file from Telegram servers
	fileURL := file.Link(s.bot.Token)
	resp, err := http.Get(fileURL)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to download file")
		s.SendMessage(chatID, fmt.Sprintf("<b>Error:</b> Failed to download file: %v", err))
		s.resetImportState()
		return
	}
	defer resp.Body.Close()

	// Step 3: Save to /tmp with timestamp
	timestamp := time.Now().Unix()
	filePath := fmt.Sprintf("/tmp/%s_%d.xlsx", symbol, timestamp)
	
	outFile, err := os.Create(filePath)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create temp file")
		s.SendMessage(chatID, fmt.Sprintf("<b>Error:</b> Failed to save file: %v", err))
		s.resetImportState()
		return
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to write file contents")
		s.SendMessage(chatID, fmt.Sprintf("<b>Error:</b> Failed to write file: %v", err))
		os.Remove(filePath) // Clean up
		s.resetImportState()
		return
	}

	logger.Info().Str("path", filePath).Msg("File saved successfully")

	// Step 4: Show provider selection keyboard
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Simplize", "provider_simplize_"+filePath),
			tgbotapi.NewInlineKeyboardButtonData("Other", "provider_other_"+filePath),
		),
	)

	msgText := fmt.Sprintf(
		"<b>File Received!</b>\n\n" +
		"File: <code>%s</code>\n" +
		"Size: %d KB\n\n" +
		"Please select your data provider:",
		msg.Document.FileName,
		msg.Document.FileSize/1024)
	
	reply := tgbotapi.NewMessage(msg.Chat.ID, msgText)
	reply.ParseMode = "HTML"
	reply.ReplyMarkup = keyboard
	s.bot.Send(reply)
}

// resetImportState clears the import state
func (s *BotService) resetImportState() {
	s.mu.Lock()
	s.importState = ImportState{}
	s.mu.Unlock()
}

// handleCallbackQuery processes inline keyboard button clicks
func (s *BotService) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	data := query.Data
	logger.Info().Str("data", data).Msg("Callback query received")

	// Acknowledge the callback immediately (removes loading indicator)
	s.bot.Request(tgbotapi.NewCallback(query.ID, "Processing..."))

	// Parse callback data: "provider_<type>_<filepath>"
	if !strings.HasPrefix(data, "provider_") {
		logger.Warn().Str("data", data).Msg("Unknown callback query")
		return
	}

	chatID := query.Message.Chat.ID

	parts := strings.SplitN(data, "_", 3)
	if len(parts) < 3 {
		logger.Error().Str("data", data).Msg("Invalid callback data format")
		s.SendMessage(chatID, "<b>Error:</b> Invalid callback data")
		return
	}

	provider := parts[1]
	filePath := parts[2]

	// Get symbol from state
	s.mu.Lock()
	symbol := s.importState.Symbol
	s.mu.Unlock()

	if symbol == "" {
		logger.Error().Msg("Symbol not found in import state")
		s.SendMessage(chatID, "<b>Error:</b> Import session expired. Please start again with /import.")
		return
	}

	// Handle "Other" provider
	if provider == "other" {
		s.SendMessage(chatID,
			"<b>Unsupported Provider</b>\n\n" +
			"Sorry, we currently only support Simplize data files.\n\n" +
			"If you need support for other providers, please contact the administrator.")
		
		// Clean up temp file
		if err := os.Remove(filePath); err != nil {
			logger.Warn().Err(err).Str("path", filePath).Msg("Failed to delete temp file")
		}
		
		s.resetImportState()
		return
	}

	// Handle "Simplize" provider
	if provider == "simplize" {
		s.handleSimplizeImport(chatID, symbol, filePath)
	}
}

// handleSimplizeImport executes the XLSX import tool for Simplize files
func (s *BotService) handleSimplizeImport(chatID int64, symbol, filePath string) {
	logger.Info().Str("symbol", symbol).Str("path", filePath).Msg("Starting Simplize import")

	// Send "processing" message
	s.SendMessage(chatID, fmt.Sprintf(
		"<b>Importing Data</b>\n\n" +
		"Symbol: <code>%s</code>\n" +
		"Provider: Simplize\n\n" +
		"Please wait, this may take a minute...",
		symbol))

	// Execute import tool
	cmd := exec.Command(
		"go", "run", "cmd/tools/xlsx_importer/main.go",
		"-symbol", symbol,
		"-file", filePath,
	)

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	
	// Clean up temp file
	defer func() {
		if err := os.Remove(filePath); err != nil {
			logger.Warn().Err(err).Str("path", filePath).Msg("Failed to delete temp file")
		} else {
			logger.Info().Str("path", filePath).Msg("Deleted temp file")
		}
	}()
	
	s.resetImportState()

	if err != nil {
		logger.Error().Err(err).Str("output", string(output)).Msg("Import failed")
		s.SendMessage(chatID, fmt.Sprintf(
			"<b>Import Failed</b>\n\n" +
			"Symbol: <code>%s</code>\n" +
			"Error: %s\n\n" +
			"<b>Debug Output:</b>\n<pre>%s</pre>",
			symbol, err.Error(), string(output)))
		return
	}

	// Success!
	logger.Info().Str("symbol", symbol).Msg("Import completed successfully")
	s.SendMessage(chatID, fmt.Sprintf(
		"<b>Import Complete!</b>\n\n" +
		"Symbol: <code>%s</code>\n\n" +
		"Historical data has been imported successfully.\n\n" +
		"<b>Next Step:</b> Train the AI model\n" +
		"Run: <code>/train %s</code>",
		symbol, symbol))
}

// handlePredictCommand handles /predict <SYMBOL> command
// Requests ML predictions and displays buy recommendations with T+2 settlement
func (s *BotService) handlePredictCommand(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	args := strings.Fields(msg.Text)
	if len(args) < 2 {
		s.SendMessage(chatID,
			"<b>Usage:</b> /predict &lt;SYMBOL&gt;\n\n" +
			"<b>Example:</b> /predict VCI\n\n" +
			"Get ML predictions and buy recommendations for a stock.\n" +
			"Make sure you've trained the model first using /train.")
		return
	}

	symbol := strings.ToUpper(args[1])

	// Check if ML client is configured
	if s.mlClient == nil {
		s.SendMessage(chatID, "<b>Error:</b> ML service not configured. Please contact administrator.")
		logger.Error().Msg("ML client not set in BotService")
		return
	}

	// Get current date
	today := time.Now().Format("2006-01-02")

	// Call ML service for prediction
	ctx, cancel := context.WithTimeout(context.Background(), mlconfig.PredictTimeout)
	defer cancel()

	resp, err := s.mlClient.Predict(ctx, &ml.PredictRequest{
		Ticker: symbol,
		Date:   today,
	})

	if err != nil {
		logger.Error().Err(err).Str("symbol", symbol).Msg("Predict gRPC failed")
		s.SendMessage(chatID, fmt.Sprintf(
			"<b>Prediction Failed</b>\n\n" +
			"Symbol: <code>%s</code>\n" +
			"Error: %s\n\n" +
			"Please check if:\n" +
			"1. You have trained the model using /train\n" +
			"2. The ML service is running",
			symbol, err.Error()))
		return
	}

	if !resp.Success {
		s.SendMessage(chatID, fmt.Sprintf(
			"<b>Prediction Failed</b>\n\n" +
			"Symbol: <code>%s</code>\n" +
			"Error: %s\n\n" +
			"Please train the model first:\n" +
			"/train %s",
			symbol, resp.ErrorMessage, symbol))
		return
	}

	// Calculate confidence score (0-100)
	// Lower uncertainty (p90-p10 spread) = higher confidence
	uncertainty := resp.P90 - resp.P10
	confidence := mlconfig.MaxConfidence
	if uncertainty > mlconfig.HighUncertainty {
		confidence = mlconfig.LowConfidence // High uncertainty
	} else if uncertainty > mlconfig.MediumUncertainty {
		confidence = mlconfig.MediumConfidence // Medium uncertainty
	} else {
		confidence = mlconfig.HighConfidence // Low uncertainty
	}
	
	// Bounds check
	if confidence > mlconfig.MaxConfidence {
		confidence = mlconfig.MaxConfidence
	}
	if confidence < mlconfig.MinConfidence {
		confidence = mlconfig.MinConfidence
	}

	// Calculate settlement date (T+2)
	tradeDate := time.Now()
	settlement := vn.CalculateSettlement(tradeDate)

	// Determine recommendation based on p50 of the longest horizon available (or 5d/10d)
	var recommendation string
	var entryAdvice string
	
	// Default to legacy P50 if no multi-horizon data
	mainP50 := resp.P50
	
	// Check if we have multi-horizon predictions
	hasMultiHorizon := len(resp.Predictions) > 0
	
	if hasMultiHorizon {
		// Try to find 10d or 5d prediction for recommendation
		for _, p := range resp.Predictions {
			if p.Horizon == 10 || (p.Horizon == 5 && mainP50 == resp.P50) {
				mainP50 = p.P50
			}
		}
	}
	
	if mainP50 > mlconfig.BuyThreshold {
		recommendation = "BUY 🚀"
		entryAdvice = "<i>Strong positive outlook</i>\n"
	} else if mainP50 > 0 {
		recommendation = "HOLD ⏸"
		entryAdvice = "<i>Marginal upside, consider waiting</i>\n"
	} else {
		recommendation = "AVOID ⚠️"
		entryAdvice = "<i>Negative outlook detected</i>\n"
	}

	// Add confidence warning (legacy)
	// TODO: Use horizon-specific confidence
	confidenceText := fmt.Sprintf("<b>Model Confidence:</b> %.0f%%", confidence)
	if confidence < 60 {
		confidenceText += " ⚠️ <i>Low confidence - high uncertainty</i>"
	}

	// Build message
	var msgBuilder strings.Builder
	msgBuilder.WriteString(fmt.Sprintf("<b>🤖 ML Prediction: %s</b>\n\n", symbol))
	msgBuilder.WriteString(confidenceText + "\n\n")

	if hasMultiHorizon {
		msgBuilder.WriteString("<b>Multi-Horizon Forecasts:</b>\n")
		// Sort predictions by horizon
		// (Assume sorted or just print)
		for _, p := range resp.Predictions {
			msgBuilder.WriteString(fmt.Sprintf("<b>%d-Day Horizon:</b>\n", p.Horizon))
			msgBuilder.WriteString(fmt.Sprintf("  • Expected (p50): %+.1f%%\n", p.P50*100))
			msgBuilder.WriteString(fmt.Sprintf("  • Range: [%+.1f%%, %+.1f%%]\n", p.P10*100, p.P90*100))
			msgBuilder.WriteString(fmt.Sprintf("  • Conf: %.0f%%\n\n", p.Confidence*100))
		}
	} else {
		// Legacy display
		msgBuilder.WriteString("<b>Predicted Returns (5-day):</b>\n")
		msgBuilder.WriteString(fmt.Sprintf("  • Pessimistic (p10): %+.1f%%\n", resp.P10*100))
		msgBuilder.WriteString(fmt.Sprintf("  • Expected (p50): %+.1f%%\n", resp.P50*100))
		msgBuilder.WriteString(fmt.Sprintf("  • Optimistic (p90): %+.1f%%\n\n", resp.P90*100))
	}

	msgBuilder.WriteString(fmt.Sprintf("<b>💡 Recommendation:</b> %s\n", recommendation))
	msgBuilder.WriteString(entryAdvice)
	msgBuilder.WriteString(fmt.Sprintf("<b>Settlement Date:</b> %s (T+2)\n\n",
		settlement.SettlementDate.Format("2006-01-02 Mon")))

	msgBuilder.WriteString(fmt.Sprintf("<i>Model version: %s</i>", resp.ModelVersion))

	s.SendMessage(chatID, msgBuilder.String())
}

