package scanner

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/api/dnse"
	"github.com/nonobeam/golang-stock-trading/internal/data"
	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/internal/regime/ftd"
	"github.com/nonobeam/golang-stock-trading/internal/risk"
	"github.com/nonobeam/golang-stock-trading/internal/signals"
	"github.com/nonobeam/golang-stock-trading/internal/websocket"
)

// DetectedSignal represents a signal with calculated position sizing
type DetectedSignal struct {
	Symbol       string
	SignalType   string
	Score        int
	EntryPrice   float64
	StopLoss     float64
	Targets      []float64
	PositionSize int
	RiskAmount   float64
	Regime       string
	DetectedAt   time.Time
}

// LiveScanner monitors stocks in real-time and detects trading signals
type LiveScanner struct {
	wsClient      *websocket.Client
	barCache      *BarCache
	signalScanner *signals.SignalScanner
	positionSizer *risk.PositionSizer


	// Database repositories
	signalRepo     *repository.SignalHistoryRepository
	watchlistRepo  *repository.WatchlistRepository
	userConfigRepo *repository.UserConfigRepository
	regimeRepo     *ftd.Repository

	// Telegram notifications
	botService BotService

	// Configuration
	minScore         int
	minBars          int
	minScoreForAlert int // Minimum score to send Telegram alert (default: 9)

	// Signal notification callback
	onSignalDetected func(signal *DetectedSignal)

	// Internal state
	mu       sync.RWMutex
	watching map[string]bool // symbol -> isWatching
	running  bool
	ctx      context.Context
	cancel   context.CancelFunc
}

// Config holds scanner configuration
type Config struct {
	DB               *sql.DB
	WSClient         *websocket.Client

	SignalScanner    *signals.SignalScanner
	PositionSizer    *risk.PositionSizer
	RegimeRepo       *ftd.Repository
	BotService       BotService // Optional Telegram bot for notifications
	MinScore         int
	BarCacheSize     int
	MinBars          int
	MinScoreForAlert int // Minimum score for Telegram alerts (default: 9)
}

// BotService interface for Telegram notifications
type BotService interface {
	NotifySignalDetected(symbol, signalType string, score int, entryPrice, stopLoss float64,
		targets []float64, positionSize int, riskAmount float64, regime string, detectedAt time.Time) error
}

// NewLiveScanner creates a new live scanner
func NewLiveScanner(cfg *Config) *LiveScanner {
	ctx, cancel := context.WithCancel(context.Background())

	// Set default min score for alerts if not configured
	minScoreForAlert := cfg.MinScoreForAlert
	if minScoreForAlert == 0 {
		minScoreForAlert = 9 // Default: only alert for very good signals
	}

	return &LiveScanner{
		wsClient:         cfg.WSClient,
		barCache:         NewBarCache(cfg.BarCacheSize),
		signalScanner:    cfg.SignalScanner,
		positionSizer:    cfg.PositionSizer,
		botService:       cfg.BotService,

		signalRepo:       repository.NewSignalHistoryRepository(cfg.DB),
		watchlistRepo:    repository.NewWatchlistRepository(cfg.DB),
		userConfigRepo:   repository.NewUserConfigRepository(cfg.DB),
		regimeRepo:       cfg.RegimeRepo,
		minScore:         cfg.MinScore,
		minBars:          cfg.MinBars,
		minScoreForAlert: minScoreForAlert,
		watching:         make(map[string]bool),
		ctx:              ctx,
		cancel:           cancel,
	}
}

// SetSignalCallback sets the callback function for signal notifications
func (s *LiveScanner) SetSignalCallback(callback func(signal *DetectedSignal)) {
	s.onSignalDetected = callback
}

// Start begins scanning watchlist symbols
func (s *LiveScanner) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("scanner already running")
	}
	s.running = true
	s.mu.Unlock()

	logger.Info().Msg("Live scanner starting...")

	// Load watchlist symbols from database
	symbols, err := s.watchlistRepo.GetSymbols(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to load watchlist: %w", err)
	}

	logger.Info().Int("count", len(symbols)).Msg("Loaded watchlist symbols")

	// Subscribe to WebSocket streams for each symbol
	for _, symbol := range symbols {
		if err := s.WatchSymbol(symbol); err != nil {
			logger.Error().Err(err).Str("symbol", symbol).Msg("Failed to watch symbol")
		}
	}

	logger.Info().Msg("Live scanner started successfully")
	return nil
}

// Stop stops the scanner
func (s *LiveScanner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	logger.Info().Msg("Stopping live scanner...")
	s.cancel()
	s.running = false

	// Unsubscribe from all symbols
	for symbol := range s.watching {
		s.unwatchSymbol(symbol)
	}

	logger.Info().Msg("Live scanner stopped")
}

// WatchSymbol starts monitoring a symbol
func (s *LiveScanner) WatchSymbol(symbol string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.watching[symbol] {
		return nil // Already watching
	}

	// Subscribe to 1-minute OHLC stream
	topic := fmt.Sprintf("quotes/krx/mdds/v2/ohlc/intraday/1m/%s", symbol)

	// Subscribe to topic
	if err := s.wsClient.Subscribe(topic); err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", symbol, err)
	}

	// Register topic handler
	s.wsClient.RegisterTopicHandler(topic, func(data []byte) {
		s.onBarUpdate(symbol, data)
	})

	s.watching[symbol] = true
	logger.Info().Str("symbol", symbol).Msg("Now watching symbol")

	return nil
}

// unwatchSymbol stops monitoring a symbol (internal, requires lock)
func (s *LiveScanner) unwatchSymbol(symbol string) {
	topic := fmt.Sprintf("quotes/krx/mdds/v2/ohlc/intraday/1m/%s", symbol)
	s.wsClient.Unsubscribe(topic)
	s.wsClient.UnregisterTopicHandler(topic)
	delete(s.watching, symbol)
	s.barCache.Clear(symbol)

	logger.Info().Str("symbol", symbol).Msg("Stopped watching symbol")
}

// onBarUpdate handles new bar data from WebSocket
func (s *LiveScanner) onBarUpdate(symbol string, data []byte) {
	// Parse OHLCV bar from WebSocket data
	bar, err := s.parseBar(data)
	if err != nil {
		logger.Error().Err(err).Str("symbol", symbol).Msg("Failed to parse bar")
		return
	}

	// Check/update daily constraints (ceiling/floor)
	// Note: We'd typically get this from StockInfo stream, but here we estimate or wait for separate update
	// Implementation note: StockInfo updates usually routed to a separate handler

	// Add to cache
	s.barCache.Add(symbol, bar)

	// Check if we have enough bars for analysis
	if !s.barCache.HasEnoughBars(symbol, s.minBars) {
		return // Not enough data yet
	}

	// Run signal detection
	go s.detectSignals(symbol)
}

// detectSignals analyzes bars and detects trading signals
func (s *LiveScanner) detectSignals(symbol string) {
	// Check if we have enough bars
	if !s.barCache.HasEnoughBars(symbol, s.minBars) {
		return
	}

	// Create data provider adapter
	dataProvider := NewBarCacheDataProvider(s.barCache)

	// Scan for signals using existing SignalScanner
	signals, err := s.signalScanner.ScanForSignals([]string{symbol}, dataProvider)
	if err != nil {
		logger.Error().Err(err).Str("symbol", symbol).Msg("Failed to scan for signals")
		return
	}

	// No signals found
	if len(signals) == 0 {
		return
	}

	// Get market regime for scoring
	regimeData, _ := dataProvider.GetMarketRegime(symbol)
	regimeType := "unknown"
	if regime, ok := regimeData["regime"].(string); ok {
		regimeType = regime
	}

	// Score each signal using scorecard
	for _, sig := range signals {
		// Use SignalScanner's built-in scoring based on confidence
		// Map confidence to score: VeryHigh=10, High=9, Moderate=7, Low=6
		score := confidenceToScore(sig.Confidence)

		// Filter by minimum score
		if score < s.minScore {
			logger.Debug().
				Str("symbol", symbol).
				Int("score", score).
				Msg("Signal filtered - score too low")
			continue
		}

		// Calculate position size
		capital := 100000000.0 // TODO: Get from user config

		
		// Determine risk per trade based on FTD Status
		riskPerTrade := 0.01 // Default 1%
		if s.regimeRepo != nil {
			m, err := s.regimeRepo.GetLatestMarketRegime(s.ctx)
			if err == nil && m != nil {
				if m.IsFTD {
					riskPerTrade = 0.02 // Aggressive 2% (FTD Confirmed)
					logger.Debug().Msg("Risk increased to 2% (FTD Confirmed)")
				} else if m.RallyAttemptDay != nil {
					riskPerTrade = 0.015 // Moderate 1.5% (Rally Attempt)
				} else {
					riskPerTrade = 0.005 // Defensive 0.5% (Downtrend)
				}
			}
		}

		posResult, err := s.positionSizer.CalculateSimple(
			sig.EntryPrice,
			sig.StopLoss,
			capital,
			riskPerTrade,
		)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to calculate position size")
			continue
		}

		// Create detected signal
		// Extract targets from map
		targets := []float64{}
		if t1, ok := sig.Targets["target1"]; ok && t1 > 0 {
			targets = append(targets, t1)
		}
		if t2, ok := sig.Targets["target2"]; ok && t2 > 0 {
			targets = append(targets, t2)
		}
		if t3, ok := sig.Targets["target3"]; ok && t3 > 0 {
			targets = append(targets, t3)
		}

		detectedSignal := &DetectedSignal{
			Symbol:       symbol,
			SignalType:   string(sig.Type),
			Score:        score,
			EntryPrice:   sig.EntryPrice,
			StopLoss:     sig.StopLoss,
			Targets:      targets,
			PositionSize: posResult.PositionSize,
			RiskAmount:   posResult.RiskAmount,
			Regime:       regimeType,
			DetectedAt:   sig.Timestamp,
		}

		// Save to database
		if err := s.saveSignal(detectedSignal); err != nil {
			logger.Error().Err(err).Str("symbol", symbol).Msg("Failed to save signal")
			continue
		}

		// Send Telegram notification if bot service configured and score meets threshold
		if s.botService != nil && score >= s.minScoreForAlert {
			if err := s.botService.NotifySignalDetected(
				detectedSignal.Symbol,
				detectedSignal.SignalType,
				detectedSignal.Score,
				detectedSignal.EntryPrice,
				detectedSignal.StopLoss,
				detectedSignal.Targets,
				detectedSignal.PositionSize,
				detectedSignal.RiskAmount,
				detectedSignal.Regime,
				detectedSignal.DetectedAt,
			); err != nil {
				logger.Error().Err(err).Str("symbol", symbol).Msg("Failed to send Telegram notification")
			}
		}

		// Notify via callback
		if s.onSignalDetected != nil {
			s.onSignalDetected(detectedSignal)
		}

		logger.Info().
			Str("symbol", symbol).
			Str("type", detectedSignal.SignalType).
			Int("score", score).
			Float64("entry", sig.EntryPrice).
			Msg("Signal detected and saved")
	}
}

// saveSignal saves a detected signal to database
func (s *LiveScanner) saveSignal(signal *DetectedSignal) error {
	regimeStr := signal.Regime
	historySignal := &repository.SignalHistory{
		Symbol:       signal.Symbol,
		SignalType:   signal.SignalType,
		Score:        signal.Score,
		EntryPrice:   signal.EntryPrice,
		StopLoss:     signal.StopLoss,
		Targets:      signal.Targets,
		PositionSize: &signal.PositionSize,
		RiskAmount:   &signal.RiskAmount,
		DetectedAt:   signal.DetectedAt,
		Regime:       &regimeStr,
		SentToUser:   false,
	}

	return s.signalRepo.Create(s.ctx, historySignal)
}

// parseBar parses DNSE WebSocket OHLC message into data.OHLCV format.
func (s *LiveScanner) parseBar(payload []byte) (data.OHLCV, error) {
	var msg dnse.OHLCMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return data.OHLCV{}, fmt.Errorf("failed to unmarshal OHLC message: %w", err)
	}

	return data.OHLCV{
		Timestamp: msg.Timestamp,
		Open:      msg.Open,
		High:      msg.High,
		Low:       msg.Low,
		Close:     msg.Close,
		Volume:    float64(msg.Volume),
	}, nil
}

// RefreshWatchlist reloads watchlist from database
func (s *LiveScanner) RefreshWatchlist() error {
	symbols, err := s.watchlistRepo.GetSymbols(s.ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Build set of new symbols
	newSymbols := make(map[string]bool)
	for _, sym := range symbols {
		newSymbols[sym] = true
	}

	// Remove symbols no longer in watchlist
	for sym := range s.watching {
		if !newSymbols[sym] {
			s.unwatchSymbol(sym)
		}
	}

	// Add new symbols
	for sym := range newSymbols {
		if !s.watching[sym] {
			topic := fmt.Sprintf("quotes/krx/mdds/v2/ohlc/intraday/1m/%s", sym)
			if err := s.wsClient.Subscribe(topic); err != nil {
				logger.Error().Err(err).Str("symbol", sym).Msg("Failed to subscribe")
				continue
			}
			// Register topic handler
			s.wsClient.RegisterTopicHandler(topic, func(data []byte) {
				s.onBarUpdate(sym, data)
			})
			s.watching[sym] = true
			logger.Info().Str("symbol", sym).Msg("Added symbol to scanner")
		}
	}

	return nil
}
