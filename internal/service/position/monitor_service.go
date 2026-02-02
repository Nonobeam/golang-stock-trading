package position

import (
	"context"
	"sync"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/analysis/indicators"
	"github.com/nonobeam/golang-stock-trading/internal/api"
	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/internal/position"
	"github.com/nonobeam/golang-stock-trading/internal/vn"
)

// MonitorService monitors active positions and evaluates stop-loss adjustments.
type MonitorService struct {
	positionRepo  *repository.PositionRepository
	dnseClient    *api.DNSEClient
	stopEngine    *position.StopManagementEngine
	alertCallback func(alert PositionAlert)
	
	mu            sync.RWMutex
	lastAlerts    map[string]time.Time // positionID -> last alert time
	running       bool
	cancelFunc    context.CancelFunc
}

// PositionAlert represents an alert to be sent.
type PositionAlert struct {
	PositionID   string
	Symbol       string
	AlertType    AlertType
	CurrentPrice float64
	StopPrice    float64
	RMultiple    float64
	Message      string
}

// AlertType specifies the type of position alert.
type AlertType string

const (
	AlertStopLoss   AlertType = "stop_loss"
	AlertBreakeven  AlertType = "breakeven"
	AlertTarget     AlertType = "target"
	AlertTimeExit   AlertType = "time_exit"
)

// NewMonitorService creates a new position monitoring service.
func NewMonitorService(
	positionRepo *repository.PositionRepository,
	dnseClient *api.DNSEClient,
	stopEngine *position.StopManagementEngine,
	alertCallback func(alert PositionAlert),
) *MonitorService {
	return &MonitorService{
		positionRepo:  positionRepo,
		dnseClient:    dnseClient,
		stopEngine:    stopEngine,
		alertCallback: alertCallback,
		lastAlerts:    make(map[string]time.Time),
	}
}

// Start begins monitoring positions.
func (s *MonitorService) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.mu.Unlock()

	monitorCtx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel

	logger.Info().Msg("Position monitor service starting")

	// Run monitoring loop
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// Initial check
	s.checkPositions(monitorCtx)

	for {
		select {
		case <-ticker.C:
			s.checkPositions(monitorCtx)
		case <-monitorCtx.Done():
			logger.Info().Msg("Position monitor service stopped")
			return nil
		}
	}
}

// Stop stops the monitoring service.
func (s *MonitorService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancelFunc != nil {
		s.cancelFunc()
	}
	s.running = false
}

// checkPositions evaluates all active positions.
func (s *MonitorService) checkPositions(ctx context.Context) {
	// Only run during market hours
	if !vn.IsMarketOpen() {
		logger.Debug().Msg("Market closed, skipping position check")
		return
	}

	logger.Debug().Msg("Checking active positions")

	// Get all open positions for user 1 (default user)
	const defaultUserID = int64(1)
	positions, err := s.positionRepo.GetOpenPositions(ctx, defaultUserID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to fetch open positions")
		return
	}

	logger.Info().Int("count", len(positions)).Msg("Evaluating positions")

	for _, pos := range positions {
		if err := s.evaluatePosition(ctx, pos); err != nil {
			logger.Error().
				Err(err).
				Str("symbol", pos.Symbol).
				Str("positionID", pos.ID).
				Msg("Failed to evaluate position")
		}
	}
}

// evaluatePosition evaluates a single position for stop adjustments.
func (s *MonitorService) evaluatePosition(ctx context.Context, pos *repository.Position) error {
	symbol := pos.Symbol

	// Fetch OHLC data (14 days for ATR calculation)
	candles, err := s.dnseClient.GetOHLC(symbol, "1") // "1" = daily
	if err != nil {
		return err
	}

	if len(candles) < 14 {
		logger.Warn().
			Str("symbol", symbol).
			Int("candles", len(candles)).
			Msg("Insufficient historical data for ATR")
		return nil
	}

	// Extract price arrays for ATR calculation
	highs := make([]float64, len(candles))
	lows := make([]float64, len(candles))
	closes := make([]float64, len(candles))

	for i, candle := range candles {
		highs[i] = candle.High
		lows[i] = candle.Low
		closes[i] = candle.Close
	}

	// Calculate ATR
	atr, err := indicators.CalculateATR(highs, lows, closes, 14)
	if err != nil {
		logger.Error().Err(err).Str("symbol", symbol).Msg("Failed to calculate ATR")
		return err
	}

	// Get current price (latest candle close)
	currentPrice := candles[len(candles)-1].Close

	// Convert repository.Position to position.Position for stop engine
	positionForEngine := s.convertToEnginePosition(pos, currentPrice)

	// Prepare indicators
	indicators := &position.Indicators{
		ATR: atr,
	}

	// Evaluate stop adjustment
	adjustment := s.stopEngine.EvaluateStopAdjustment(positionForEngine, currentPrice, indicators)
	
	if adjustment != nil && adjustment.ShouldAdjust {
		logger.Info().
			Str("symbol", symbol).
			Float64("oldStop", pos.StopLoss).
			Float64("newStop", adjustment.NewStop).
			Str("reason", string(adjustment.Reason)).
			Msg("Stop adjustment recommended")

		// Update position in database
		pos.StopLoss = adjustment.NewStop
		// Note: Repository update would happen here when we wire this up

		// Send alert
		s.sendAlert(pos, currentPrice, adjustment)
	}

	// Check if stop loss is hit
	if s.isStopHit(pos, currentPrice) {
		s.sendStopLossAlert(pos, currentPrice)
	}

	return nil
}

// convertToEnginePosition converts repository.Position to position.Position.
func (s *MonitorService) convertToEnginePosition(dbPos *repository.Position, currentPrice float64) *position.Position {
	return &position.Position{
		PositionID:          dbPos.ID,
		Ticker:              dbPos.Symbol,
		EntryDate:           dbPos.EntryDate,
		EntryPrice:          dbPos.EntryPrice,
		SharesRemaining:     dbPos.Quantity,
		StopLoss:            dbPos.StopLoss,
		HighestPriceReached: currentPrice, // Simplified - should track actual highest
		PositionType:        "long",       // Assuming long for now
		RiskPerShare:        dbPos.EntryPrice - dbPos.StopLoss,
	}
}

// isStopHit checks if current price has hit the stop-loss.
func (s *MonitorService) isStopHit(pos *repository.Position, currentPrice float64) bool {
	// For long positions
	return currentPrice <= pos.StopLoss
}

// sendAlert sends a stop adjustment alert if not rate-limited.
func (s *MonitorService) sendAlert(pos *repository.Position, currentPrice float64, adjustment *position.StopAdjustmentResult) {
	if s.alertCallback == nil {
		return
	}

	// Check rate limiting
	s.mu.Lock()
	lastAlert, exists := s.lastAlerts[pos.ID]
	if exists && time.Since(lastAlert) < 1*time.Hour {
		s.mu.Unlock()
		return
	}
	s.lastAlerts[pos.ID] = time.Now()
	s.mu.Unlock()

	// Calculate R-multiple
	rMultiple := (currentPrice - pos.EntryPrice) / (pos.EntryPrice - pos.StopLoss)

	alertType := AlertBreakeven
	if adjustment.Reason == position.ReasonTargetHit {
		alertType = AlertTarget
	}

	alert := PositionAlert{
		PositionID:   pos.ID,
		Symbol:       pos.Symbol,
		AlertType:    alertType,
		CurrentPrice: currentPrice,
		StopPrice:    adjustment.NewStop,
		RMultiple:    rMultiple,
		Message:      adjustment.Details,
	}

	s.alertCallback(alert)
}

// sendStopLossAlert sends an alert when stop-loss is hit.
func (s *MonitorService) sendStopLossAlert(pos *repository.Position, currentPrice float64) {
	if s.alertCallback == nil {
		return
	}

	// Check rate limiting
	s.mu.Lock()
	alertKey := pos.ID + "_stop"
	lastAlert, exists := s.lastAlerts[alertKey]
	if exists && time.Since(lastAlert) < 1*time.Hour {
		s.mu.Unlock()
		return
	}
	s.lastAlerts[alertKey] = time.Now()
	s.mu.Unlock()

	rMultiple := (currentPrice - pos.EntryPrice) / (pos.EntryPrice - pos.StopLoss)

	alert := PositionAlert{
		PositionID:   pos.ID,
		Symbol:       pos.Symbol,
		AlertType:    AlertStopLoss,
		CurrentPrice: currentPrice,
		StopPrice:    pos.StopLoss,
		RMultiple:    rMultiple,
		Message:      "Stop-loss level hit - consider selling",
	}

	s.alertCallback(alert)
}
