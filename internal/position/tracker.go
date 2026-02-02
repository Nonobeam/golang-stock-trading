package position

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	// ErrPositionNotFound is returned when a position ID doesn't exist.
	ErrPositionNotFound = errors.New("position not found")
	// ErrInsufficientShares is returned when trying to sell more shares than available.
	ErrInsufficientShares = errors.New("insufficient shares remaining")
)

// PositionTracker manages all open and closed positions.
type PositionTracker struct {
	mu                sync.RWMutex
	positions         map[string]*Position
	closedPositions   []*Position
	metricsCalculator *PositionMetricsCalculator
}

// NewPositionTracker creates a new position tracker.
func NewPositionTracker() *PositionTracker {
	return &PositionTracker{
		positions:         make(map[string]*Position),
		closedPositions:   make([]*Position, 0),
		metricsCalculator: NewMetricsCalculator(),
	}
}

// AddPositionParams contains parameters for adding a new position.
type AddPositionParams struct {
	Ticker       string
	EntryPrice   float64
	Shares       int
	StopLoss     float64
	Targets      []Target
	RiskPercent  float64
	PositionType string
	SetupType    string
	TradeScore   int
	Notes        string
}

// AddPosition adds a new position to tracking.
func (t *PositionTracker) AddPosition(params AddPositionParams) *Position {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Generate unique position ID
	positionID := fmt.Sprintf("%s_%s", params.Ticker, time.Now().Format("20060102_150405"))

	// Default position type
	posType := params.PositionType
	if posType == "" {
		posType = "long"
	}

	// Create position
	position := &Position{
		PositionID:          positionID,
		Ticker:              params.Ticker,
		EntryDate:           time.Now(),
		EntryPrice:          params.EntryPrice,
		Shares:              params.Shares,
		StopLoss:            params.StopLoss,
		Targets:             params.Targets,
		RiskPercent:         params.RiskPercent,
		PositionType:        posType,
		SetupType:           params.SetupType,
		TradeScore:          params.TradeScore,
		Notes:               params.Notes,
		CurrentPrice:        params.EntryPrice,
		LastUpdated:         time.Now(),
		HighestPriceReached: params.EntryPrice,
		LowestPriceReached:  params.EntryPrice,
		SharesRemaining:     params.Shares,
		Exits:               []Exit{},
	}

	// Initialize calculated fields
	position.Initialize()

	// Store position
	t.positions[positionID] = position

	return position
}

// UpdatePositionPrice updates a position with the current market price.
func (t *PositionTracker) UpdatePositionPrice(positionID string, currentPrice float64, timestamp *time.Time) PriceUpdateResult {
	t.mu.Lock()
	defer t.mu.Unlock()

	position, exists := t.positions[positionID]
	if !exists {
		return PriceUpdateResult{
			PositionID: positionID,
			Error:      ErrPositionNotFound.Error(),
		}
	}

	// Update current price
	position.CurrentPrice = currentPrice
	if timestamp != nil {
		position.LastUpdated = *timestamp
	} else {
		position.LastUpdated = time.Now()
	}

	// Update extremes
	if currentPrice > position.HighestPriceReached {
		position.HighestPriceReached = currentPrice
		t := position.LastUpdated
		position.HighestDate = &t
	}

	if currentPrice < position.LowestPriceReached {
		position.LowestPriceReached = currentPrice
		t := position.LastUpdated
		position.LowestDate = &t
	}

	// Calculate updated metrics
	metrics := t.metricsCalculator.CalculateMetrics(position)

	// Check for alerts
	alerts := CheckAlerts(position, &metrics)

	return PriceUpdateResult{
		PositionID: positionID,
		Metrics:    metrics,
		Alerts:     alerts,
		Timestamp:  position.LastUpdated,
	}
}

// PartialExit records a partial exit from a position.
func (t *PositionTracker) PartialExit(positionID string, exitPrice float64, sharesToSell int, reason string) ExitResult {
	t.mu.Lock()
	defer t.mu.Unlock()

	position, exists := t.positions[positionID]
	if !exists {
		return ExitResult{
			PositionID: positionID,
			Error:      ErrPositionNotFound.Error(),
		}
	}

	if sharesToSell > position.SharesRemaining {
		return ExitResult{
			PositionID: positionID,
			Error:      fmt.Sprintf("%s: cannot sell %d shares, only %d remaining", ErrInsufficientShares.Error(), sharesToSell, position.SharesRemaining),
		}
	}

	// Record exit
	exitRecord := Exit{
		Date:   time.Now(),
		Price:  exitPrice,
		Shares: sharesToSell,
		Reason: reason,
	}

	position.Exits = append(position.Exits, exitRecord)
	position.SharesRemaining -= sharesToSell

	// Calculate P&L on this exit
	var exitPL float64
	if position.PositionType == "long" {
		exitPL = (exitPrice - position.EntryPrice) * float64(sharesToSell)
	} else {
		exitPL = (position.EntryPrice - exitPrice) * float64(sharesToSell)
	}

	exitPLPercent := ((exitPrice - position.EntryPrice) / position.EntryPrice) * 100
	var exitR float64
	if position.RiskPerShare > 0 {
		exitR = exitPL / (position.RiskPerShare * float64(sharesToSell))
	}

	// If fully exited, move to closed positions
	fullyClosed := position.SharesRemaining == 0
	if fullyClosed {
		t.closedPositions = append(t.closedPositions, position)
		delete(t.positions, positionID)
	}

	return ExitResult{
		PositionID:      positionID,
		ExitPrice:       exitPrice,
		SharesSold:      sharesToSell,
		SharesRemaining: position.SharesRemaining,
		ExitPL:          exitPL,
		ExitPLPercent:   exitPLPercent,
		ExitR:           exitR,
		Reason:          reason,
		FullyClosed:     fullyClosed,
		Timestamp:       time.Now(),
	}
}

// ClosePosition closes the entire position.
func (t *PositionTracker) ClosePosition(positionID string, exitPrice float64, reason string) ExitResult {
	t.mu.RLock()
	position, exists := t.positions[positionID]
	t.mu.RUnlock()

	if !exists {
		return ExitResult{
			PositionID: positionID,
			Error:      ErrPositionNotFound.Error(),
		}
	}

	return t.PartialExit(positionID, exitPrice, position.SharesRemaining, reason)
}

// GetPosition returns a position by ID.
func (t *PositionTracker) GetPosition(positionID string) (*Position, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	position, exists := t.positions[positionID]
	if !exists {
		return nil, ErrPositionNotFound
	}

	return position, nil
}

// GetPositionMetrics returns current metrics for a specific position.
func (t *PositionTracker) GetPositionMetrics(positionID string) (PositionMetrics, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	position, exists := t.positions[positionID]
	if !exists {
		return PositionMetrics{}, ErrPositionNotFound
	}

	return t.metricsCalculator.CalculateMetrics(position), nil
}

// GetAllPositionsSummary returns a summary of all open positions.
func (t *PositionTracker) GetAllPositionsSummary() PortfolioSummary {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.positions) == 0 {
		return PortfolioSummary{
			NumPositions: 0,
			TotalValue:   0,
			Positions:    []PositionSummary{},
		}
	}

	var totalValue, totalUnrealizedPL, totalRealizedPL, totalRisk float64
	positionSummaries := make([]PositionSummary, 0, len(t.positions))

	for positionID, position := range t.positions {
		metrics := t.metricsCalculator.CalculateMetrics(position)

		positionSummaries = append(positionSummaries, PositionSummary{
			PositionID:          positionID,
			Ticker:              position.Ticker,
			EntryPrice:          position.EntryPrice,
			CurrentPrice:        position.CurrentPrice,
			SharesRemaining:     position.SharesRemaining,
			UnrealizedPL:        metrics.UnrealizedPL,
			UnrealizedPLPercent: metrics.UnrealizedPLPercent,
			RMultiple:           metrics.RMultiple,
			DaysInTrade:         metrics.DaysInTrade,
			StopDistancePercent: metrics.StopDistancePercent,
		})

		totalValue += metrics.PositionRemainingValue
		totalUnrealizedPL += metrics.UnrealizedPL
		totalRealizedPL += metrics.RealizedPL
		totalRisk += metrics.RiskRemaining
	}

	return PortfolioSummary{
		NumPositions:      len(t.positions),
		TotalValue:        totalValue,
		TotalUnrealizedPL: totalUnrealizedPL,
		TotalRealizedPL:   totalRealizedPL,
		TotalPL:           totalUnrealizedPL + totalRealizedPL,
		TotalRisk:         totalRisk,
		Positions:         positionSummaries,
	}
}

// GetClosedPositions returns all closed positions.
func (t *PositionTracker) GetClosedPositions() []*Position {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.closedPositions
}

// SavePositions saves all positions to a JSON file.
func (t *PositionTracker) SavePositions(filepath string) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	data := SavedPositions{
		Positions:       t.positions,
		ClosedPositions: t.closedPositions,
		SavedAt:         time.Now(),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal positions: %w", err)
	}

	err = os.WriteFile(filepath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write positions file: %w", err)
	}

	return nil
}

// LoadPositions loads positions from a JSON file.
func (t *PositionTracker) LoadPositions(filepath string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	jsonData, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read positions file: %w", err)
	}

	var data SavedPositions
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal positions: %w", err)
	}

	t.positions = data.Positions
	t.closedPositions = data.ClosedPositions

	// Ensure maps are initialized
	if t.positions == nil {
		t.positions = make(map[string]*Position)
	}
	if t.closedPositions == nil {
		t.closedPositions = make([]*Position, 0)
	}

	return nil
}

// PositionCount returns the number of open positions.
func (t *PositionTracker) PositionCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.positions)
}
