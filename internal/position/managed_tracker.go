package position

import (
	"fmt"
	"strings"
	"time"
)

// ManagedPositionTracker combines PositionTracker with automatic stop management.
type ManagedPositionTracker struct {
	Tracker     *PositionTracker
	StopManager *StopManagementEngine
}

// NewManagedPositionTracker creates a new managed position tracker.
func NewManagedPositionTracker(rules *StopAdjustmentRule) *ManagedPositionTracker {
	return &ManagedPositionTracker{
		Tracker:     NewPositionTracker(),
		StopManager: NewStopManagementEngine(rules),
	}
}

// AddPosition adds a new position (delegates to underlying tracker).
func (m *ManagedPositionTracker) AddPosition(params AddPositionParams) *Position {
	return m.Tracker.AddPosition(params)
}

// UpdatePositionWithStopManagement updates position price and manages stops.
func (m *ManagedPositionTracker) UpdatePositionWithStopManagement(
	positionID string,
	currentPrice float64,
	indicators *Indicators,
	autoAdjust bool,
) StopUpdateResult {
	// First, update position price using base tracker
	priceResult := m.Tracker.UpdatePositionPrice(positionID, currentPrice, nil)

	result := StopUpdateResult{
		PriceUpdateResult: priceResult,
		StopAdjusted:      false,
	}

	if priceResult.Error != "" {
		return result
	}

	// Get position for stop evaluation
	position, err := m.Tracker.GetPosition(positionID)
	if err != nil {
		return result
	}

	// Evaluate stop adjustment
	adjustment := m.StopManager.EvaluateStopAdjustment(position, currentPrice, indicators)

	if adjustment != nil && adjustment.ShouldAdjust {
		if autoAdjust {
			// Apply the adjustment
			oldStop := position.StopLoss
			m.StopManager.ApplyAdjustment(position, currentPrice, adjustment)

			result.StopAdjusted = true
			result.StopAdjustment = adjustment
			result.StopAdjustmentDetails = &StopAdjustment{
				Timestamp:    time.Now(),
				OldStop:      oldStop,
				NewStop:      adjustment.NewStop,
				Reason:       adjustment.Reason,
				CurrentPrice: currentPrice,
				ProfitLocked: adjustment.ProfitLocked,
				Details:      adjustment.Details,
			}
		} else {
			// Return suggestion but don't apply
			result.StopAdjustment = adjustment
		}
	}

	return result
}

// PartialExit records a partial exit (delegates to underlying tracker).
func (m *ManagedPositionTracker) PartialExit(positionID string, exitPrice float64, sharesToSell int, reason string) ExitResult {
	return m.Tracker.PartialExit(positionID, exitPrice, sharesToSell, reason)
}

// ClosePosition closes entire position (delegates to underlying tracker).
func (m *ManagedPositionTracker) ClosePosition(positionID string, exitPrice float64, reason string) ExitResult {
	return m.Tracker.ClosePosition(positionID, exitPrice, reason)
}

// GetPosition returns a position by ID (delegates to underlying tracker).
func (m *ManagedPositionTracker) GetPosition(positionID string) (*Position, error) {
	return m.Tracker.GetPosition(positionID)
}

// GetPositionMetrics returns current metrics (delegates to underlying tracker).
func (m *ManagedPositionTracker) GetPositionMetrics(positionID string) (PositionMetrics, error) {
	return m.Tracker.GetPositionMetrics(positionID)
}

// GetAllPositionsSummary returns portfolio summary (delegates to underlying tracker).
func (m *ManagedPositionTracker) GetAllPositionsSummary() PortfolioSummary {
	return m.Tracker.GetAllPositionsSummary()
}

// GetStopAdjustmentHistory returns stop adjustment history for a position.
func (m *ManagedPositionTracker) GetStopAdjustmentHistory(positionID string) *StopAdjustmentHistory {
	return m.StopManager.GetAdjustmentHistory(positionID)
}

// GetStopAdjustmentSummary generates a text summary of all stop adjustments.
func (m *ManagedPositionTracker) GetStopAdjustmentSummary(positionID string) string {
	history := m.StopManager.GetAdjustmentHistory(positionID)

	if history == nil || len(history.Adjustments) == 0 {
		return fmt.Sprintf("No stop adjustments for position %s", positionID)
	}

	var output strings.Builder

	output.WriteString(fmt.Sprintf("STOP ADJUSTMENT HISTORY: %s\n", positionID))
	output.WriteString(strings.Repeat("=", 80))
	output.WriteString("\n\n")

	for _, adj := range history.Adjustments {
		output.WriteString(fmt.Sprintf("%s: %.0f → %.0f (%s)\n",
			adj.Timestamp.Format("2006-01-02 15:04"),
			adj.OldStop,
			adj.NewStop,
			adj.Reason))
		output.WriteString(fmt.Sprintf("  Current Price: %.0f\n", adj.CurrentPrice))
		output.WriteString(fmt.Sprintf("  Profit Locked: %.0f VND\n", adj.ProfitLocked))
		output.WriteString(fmt.Sprintf("  Details: %s\n\n", adj.Details))
	}

	return output.String()
}

// PositionCount returns number of open positions.
func (m *ManagedPositionTracker) PositionCount() int {
	return m.Tracker.PositionCount()
}
