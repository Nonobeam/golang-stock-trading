package backtest

import (
	"fmt"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/data"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/internal/signals"
)

// PositionTracker tracks open positions and closed trades during backtest
type PositionTracker struct {
	openPositions  map[string]*BacktestPosition
	closedTrades   []*ClosedTrade
	capitalTracker *CapitalTracker
}

// NewPositionTracker creates a new position tracker
func NewPositionTracker(capitalTracker *CapitalTracker) *PositionTracker {
	return &PositionTracker{
		openPositions:  make(map[string]*BacktestPosition),
		closedTrades:   []*ClosedTrade{},
		capitalTracker: capitalTracker,
	}
}

// AddPosition opens a new position (SIMPLIFIED VERSION)
func (tracker *PositionTracker) AddPosition(
	symbol string,
	signalType signals.SignalType,
	fill *SimulatedFill,
	stopPrice float64,
	targets []float64,
	score int,
) error {
	// Create backtest position with tracking
	btPos := &BacktestPosition{
		Position:         nil, // Will use embedded fields directly for now
		SignalID:         fmt.Sprintf("%s-%d", symbol, time.Now().Unix()),
		SignalType:       string(signalType),
		SignalScore:      score,
		EntryReason:      "backtest_signal",
		InitialStop:      stopPrice,
		CurrentStop:      stopPrice,
		Targets:          targets,
		TargetsHit:       make([]bool, len(targets)),
		InitialRisk:      (fill.FilledPrice - stopPrice) * float64(fill.FilledShares),
		CapitalAllocated: fill.TotalCost,
		MAE:              0,
		MFE:              0,
		EntryDate:        time.Now(),
		EntryPrice:       fill.FilledPrice,
		Shares:           fill.FilledShares,
	}

	// Store position
	tracker.openPositions[symbol] = btPos

	// Deduct capital
	tracker.capitalTracker.DeductCash(fill.TotalCost)

	logger.Debug().
		Str("symbol", symbol).
		Str("type", string(signalType)).
		Int("score", score).
		Float64("entry", fill.FilledPrice).
		Int("shares", fill.FilledShares).
		Float64("stop", stopPrice).
		Msg("Position added to tracker")

	return nil
}

// UpdatePositions updates all open positions with current bar data
func (tracker *PositionTracker) UpdatePositions(bar data.OHLCV) error {
	// Create dummy symbol from bar if not present
	// In real backtest, symbol will be passed from config
	symbol := "VCB" // TODO: Get from config

	pos, exists := tracker.openPositions[symbol]
	if !exists {
		return nil // No position for this symbol
	}

	// Update MAE (Maximum Adverse Excursion)
	drawdown := pos.EntryPrice - bar.Low
	if drawdown > pos.MAE {
		pos.MAE = drawdown
	}

	// Update MFE (Maximum Favorable Excursion)
	favorableMove := bar.High - pos.EntryPrice
	if favorableMove > pos.MFE {
		pos.MFE = favorableMove
	}

	// Check if stop was hit
	if bar.Low <= pos.CurrentStop {
		if err := tracker.closePosition(symbol, pos.CurrentStop, bar.Timestamp, "stop_loss"); err != nil {
			return fmt.Errorf("failed to close position on stop: %w", err)
		}
		return nil
	}

	// Check if any targets were hit
	for i, target := range pos.Targets {
		if !pos.TargetsHit[i] && bar.High >= target {
			pos.TargetsHit[i] = true
			// Close entire position at first target (simplified)
			if err := tracker.closePosition(symbol, target, bar.Timestamp, fmt.Sprintf("target_%d", i+1)); err != nil {
				return fmt.Errorf("failed to close position on target: %w", err)
			}
			return nil
		}
	}

	return nil
}

// closePosition closes a position and records the trade
func (tracker *PositionTracker) closePosition(
	symbol string,
	exitPrice float64,
	exitDate time.Time,
	exitReason string,
) error {
	pos, exists := tracker.openPositions[symbol]
	if !exists {
		return fmt.Errorf("position not found for symbol %s", symbol)
	}

	// Calculate P&L
	pnl := (exitPrice - pos.EntryPrice) * float64(pos.Shares)
	pnlPercent := (exitPrice - pos.EntryPrice) / pos.EntryPrice * 100

	// Calculate R-multiple
	initialRisk := pos.EntryPrice - pos.InitialStop
	rMultiple := 0.0
	if initialRisk > 0 {
		rMultiple = (exitPrice - pos.EntryPrice) / initialRisk
	}

	// Calculate holding days
	holdingDays := int(exitDate.Sub(pos.EntryDate).Hours() / 24)

	// Create closed trade record
	trade := &ClosedTrade{
		Symbol:      symbol,
		SignalType:  pos.SignalType,
		SignalScore: pos.SignalScore,
		EntryDate:   pos.EntryDate,
		EntryPrice:  pos.EntryPrice,
		ExitDate:    exitDate,
		ExitPrice:   exitPrice,
		ExitReason:  exitReason,
		Shares:      pos.Shares,
		PnL:         pnl,
		PnLPercent:  pnlPercent,
		RMultiple:   rMultiple,
		HoldingDays: holdingDays,
		InitialRisk: pos.InitialRisk,
		MAE:         pos.MAE,
		MFE:         pos.MFE,
		Commission:  0, // TODO: Add from fill data
		Slippage:    0, // TODO: Add from fill data
	}

	// Add to closed trades
	tracker.closedTrades = append(tracker.closedTrades, trade)

	// Remove from open positions
	delete(tracker.openPositions, symbol)

	// Return capital (with T+2 settlement)
	proceeds := exitPrice * float64(pos.Shares)
	tracker.capitalTracker.AddPendingSettlement(proceeds, exitDate)

	logger.Debug().
		Str("symbol", symbol).
		Str("reason", exitReason).
		Float64("pnl", pnl).
		Float64("rMultiple", rMultiple).
		Int("holdingDays", holdingDays).
		Msg("Position closed")

	return nil
}

// CloseAllOpenPositions closes all open positions (called at end of backtest)
func (tracker *PositionTracker) CloseAllOpenPositions(finalDate time.Time, finalPrice float64) error {
	logger.Info().Int("count", len(tracker.openPositions)).Msg("Force-closing all open positions")

	for symbol := range tracker.openPositions {
		logger.Warn().
			Str("symbol", symbol).
			Msg("Position still open at backtest end, force closing")

		if err := tracker.closePosition(symbol, finalPrice, finalDate, "end_of_backtest"); err != nil {
			return fmt.Errorf("failed to close position %s: %w", symbol, err)
		}
	}

	return nil
}

// GetOpenPositionCount returns the number of currently open positions
func (tracker *PositionTracker) GetOpenPositionCount() int {
	return len(tracker.openPositions)
}

// GetTotalPositionValue returns the current market value of all open positions
func (tracker *PositionTracker) GetTotalPositionValue(currentPrice float64) float64 {
	totalValue := 0.0
	for _, pos := range tracker.openPositions {
		posValue := currentPrice * float64(pos.Shares)
		totalValue += posValue
	}
	return totalValue
}

// GetClosedTrades returns all closed trades
func (tracker *PositionTracker) GetClosedTrades() []*ClosedTrade {
	return tracker.closedTrades
}
