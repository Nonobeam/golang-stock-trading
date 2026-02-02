package backtest

import (
	"fmt"

	"github.com/nonobeam/golang-stock-trading/internal/data"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/internal/regime"
	"github.com/nonobeam/golang-stock-trading/internal/risk"
	"github.com/nonobeam/golang-stock-trading/internal/signals"
)

// BacktestEngine orchestrates the entire backtest process
type BacktestEngine struct {
	config *BacktestConfig

	// Data components
	dataLoader *HistoricalDataLoader

	// Trading components
	simulator      *TradeSimulator
	signalScanner  *signals.SignalScanner
	positionSizer  *risk.PositionSizer
	regimeDetector *regime.RegimeDetector

	// State tracking
	posTracker     *PositionTracker
	capitalTracker *CapitalTracker

	// Statistics
	signalsGenerated int
	signalsSkipped   int
	skipReasons      map[string]int
}

// NewBacktestEngine creates a new backtest engine
func NewBacktestEngine(config *BacktestConfig) (*BacktestEngine, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Validate config
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create components
	dataLoader := NewHistoricalDataLoader(config.DataPath)
	simulator := NewTradeSimulator(config.Commission, config.Slippage)
	capitalTracker := NewCapitalTracker(config.InitialCapital)
	posTracker := NewPositionTracker(capitalTracker)

	// Create signal scanner with default config
	signalScanner := signals.NewDefaultSignalScanner()

	// Create position sizer
	positionSizer := risk.NewPositionSizer(config.InitialCapital)

	// Create regime detector with default config
	regimeDetector := regime.NewRegimeDetector(regime.DefaultRegimeConfig())

	return &BacktestEngine{
		config:           config,
		dataLoader:       dataLoader,
		simulator:        simulator,
		signalScanner:    signalScanner,
		positionSizer:    positionSizer,
		regimeDetector:   regimeDetector,
		posTracker:       posTracker,
		capitalTracker:   capitalTracker,
		signalsGenerated: 0,
		signalsSkipped:   0,
		skipReasons:      make(map[string]int),
	}, nil
}

// Run executes the backtest and returns results
func (e *BacktestEngine) Run() (*BacktestResult, error) {
	logger.Info().Msg("Starting backtest...")
	logger.Info().
		Str("symbol", e.config.Symbol).
		Str("start", e.config.StartDate.Format("2006-01-02")).
		Str("end", e.config.EndDate.Format("2006-01-02")).
		Float64("initialCapital", e.config.InitialCapital).
		Msg("Backtest configuration")

	// 1. Load historical data
	logger.Info().Msg("Loading historical data...")
	allBars, err := e.dataLoader.LoadOHLCV(e.config.Symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}
	logger.Info().Int("bars", len(allBars)).Msg("Data loaded successfully")

	// Filter by date range
	filteredBars := e.filterByDateRange(allBars)
	if len(filteredBars) == 0 {
		return nil, fmt.Errorf("no data found in date range %s to %s",
			e.config.StartDate.Format("2006-01-02"),
			e.config.EndDate.Format("2006-01-02"))
	}
	logger.Info().Int("tradingDays", len(filteredBars)).Msg("Date range filtered")

	// Dry run mode - just validate and exit
	if e.config.DryRun {
		return e.performDryRun(filteredBars)
	}

	// 2. Process each day chronologically
	logger.Info().Msg("Processing historical data chronologically...")
	equityCurve := []EquityPoint{}

	for i, bar := range filteredBars {
		// Process this day
		if err := e.processDay(bar, filteredBars, i); err != nil {
			logger.Error().Err(err).
				Time("date", bar.Timestamp).
				Msg("Error processing day, continuing...")
		}

		// Record equity point
		currentEquity := e.getCurrentEquity(bar.Close)
		equityCurve = append(equityCurve, EquityPoint{
			Date:   bar.Timestamp,
			Equity: currentEquity,
		})

		// Log progress
		if e.config.Verbose && (i+1)%50 == 0 {
			logger.Info().
				Int("day", i+1).
				Int("total", len(filteredBars)).
				Int("percent", (i+1)*100/len(filteredBars)).
				Int("openPositions", e.posTracker.GetOpenPositionCount()).
				Int("closedTrades", len(e.posTracker.closedTrades)).
				Msg("Progress")
		}
	}

	// 3. Close all open positions at end
	logger.Info().Msg("Closing all open positions at backtest end...")
	lastBar := filteredBars[len(filteredBars)-1]
	if err := e.posTracker.CloseAllOpenPositions(lastBar.Timestamp, lastBar.Close); err != nil {
		return nil, fmt.Errorf("failed to close positions: %w", err)
	}

	// 4. Calculate metrics
	logger.Info().Msg("Calculating performance metrics...")
	metrics, err := e.calculateMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate metrics: %w", err)
	}

	// 5. Generate result
	finalReturn := ((e.capitalTracker.GetAvailableCash() - e.config.InitialCapital) / e.config.InitialCapital) * 100

	result := &BacktestResult{
		Config:         e.config,
		StartDate:      filteredBars[0].Timestamp,
		EndDate:        filteredBars[len(filteredBars)-1].Timestamp,
		TradingDays:    len(filteredBars),
		InitialCapital: e.config.InitialCapital,
		FinalCapital:   e.capitalTracker.GetAvailableCash(),
		ClosedTrades:   e.posTracker.closedTrades,
		EquityCurve:    equityCurve,
		Metrics:        metrics,
	}

	logger.Info().
		Float64("initialCapital", result.InitialCapital).
		Float64("finalCapital", result.FinalCapital).
		Float64("return", finalReturn).
		Int("trades", len(result.ClosedTrades)).
		Msg("Backtest complete!")

	return result, nil
}

// processDay processes one day of trading (SIMPLIFIED - no actual signal scanning yet)
func (e *BacktestEngine) processDay(bar data.OHLCV, allBars []data.OHLCV, currentIdx int) error {
	// Step 1: Update existing positions (check stops/targets)
	if err := e.posTracker.UpdatePositions(bar); err != nil {
		return fmt.Errorf("failed to update positions: %w", err)
	}

	// Step 2: Check if we can open new positions
	openCount := e.posTracker.GetOpenPositionCount()
	if openCount >= e.config.MaxPositions {
		// At position limit, skip signal scanning
		return nil
	}

	// TODO: Implement actual signal scanning
	// For now, we skip signal detection until we properly implement DataProvider
	// This will be completed in subsequent iterations

	return nil
}

// Helper functions

func (e *BacktestEngine) filterByDateRange(bars []data.OHLCV) []data.OHLCV {
	var filtered []data.OHLCV
	for _, bar := range bars {
		if (bar.Timestamp.Equal(e.config.StartDate) || bar.Timestamp.After(e.config.StartDate)) &&
			(bar.Timestamp.Equal(e.config.EndDate) || bar.Timestamp.Before(e.config.EndDate)) {
			filtered = append(filtered, bar)
		}
	}
	return filtered
}

func (e *BacktestEngine) getCurrentEquity(currentPrice float64) float64 {
	cash := e.capitalTracker.GetAvailableCash()
	positionValue := e.posTracker.GetTotalPositionValue(currentPrice)
	return cash + positionValue
}

func (e *BacktestEngine) recordSkip(reason string) {
	e.signalsSkipped++
	e.skipReasons[reason]++
}

func (e *BacktestEngine) performDryRun(bars []data.OHLCV) (*BacktestResult, error) {
	logger.Info().Msg("Dry run mode - validating setup only...")

	logger.Info().
		Int("barsLoaded", len(bars)).
		Msg("Dry run successful - data loaded and validated")

	return &BacktestResult{
		Config:         e.config,
		TradingDays:    len(bars),
		InitialCapital: e.config.InitialCapital,
		FinalCapital:   e.config.InitialCapital,
	}, nil
}

func (e *BacktestEngine) calculateMetrics() (*BacktestMetrics, error) {
	// Get equity curve from result (will be populated during Run)
	// For now, build a simple equity curve from closed trades
	equityCurve := e.buildEquityCurve()

	// Use comprehensive metrics calculator
	metrics, err := CalculateComprehensiveMetrics(e.posTracker.closedTrades, equityCurve, e.config.InitialCapital)
	if err != nil {
		return nil, err
	}

	// Calculate total P&L percentage
	metrics.TotalPnLPercent = (metrics.TotalPnL / e.config.InitialCapital) * 100

	return metrics, nil
}

// buildEquityCurve constructs equity curve from closed trades (chronological).
func (e *BacktestEngine) buildEquityCurve() []EquityPoint {
	if len(e.posTracker.closedTrades) == 0 {
		return []EquityPoint{}
	}

	equity := e.config.InitialCapital
	curve := []EquityPoint{{
		Date:   e.posTracker.closedTrades[0].EntryDate,
		Equity: equity,
	}}

	for _, trade := range e.posTracker.closedTrades {
		equity += trade.PnL
		curve = append(curve, EquityPoint{
			Date:   trade.ExitDate,
			Equity: equity,
		})
	}

	return curve
}

func validateConfig(config *BacktestConfig) error {
	if config.Symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	if config.InitialCapital <= 0 {
		return fmt.Errorf("initial capital must be positive")
	}
	if config.StartDate.IsZero() || config.EndDate.IsZero() {
		return fmt.Errorf("start and end dates must be set")
	}
	if config.EndDate.Before(config.StartDate) {
		return fmt.Errorf("end date must be after start date")
	}
	if config.MinSignalScore < 0 || config.MinSignalScore > 13 {
		return fmt.Errorf("min signal score must be between 0 and 13")
	}
	if config.MaxPositions < 1 {
		return fmt.Errorf("max positions must be at least 1")
	}
	if config.RiskPerTrade <= 0 || config.RiskPerTrade > 1 {
		return fmt.Errorf("risk per trade must be between 0 and 1")
	}
	return nil
}
