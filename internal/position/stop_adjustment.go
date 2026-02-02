package position

import (
	"encoding/json"
	"time"
)

// StopAdjustment records a single stop adjustment with all details.
type StopAdjustment struct {
	Timestamp    time.Time            `json:"timestamp"`
	OldStop      float64              `json:"oldStop"`
	NewStop      float64              `json:"newStop"`
	Reason       StopAdjustmentReason `json:"reason"`
	CurrentPrice float64              `json:"currentPrice"`
	ProfitLocked float64              `json:"profitLocked"` // Profit now protected
	Details      string               `json:"details"`
}

// StopAdjustmentResult contains the result of evaluating a stop adjustment.
type StopAdjustmentResult struct {
	ShouldAdjust bool                 `json:"shouldAdjust"`
	NewStop      float64              `json:"newStop"`
	Reason       StopAdjustmentReason `json:"reason"`
	ProfitLocked float64              `json:"profitLocked"`
	Details      string               `json:"details"`
}

// StopAdjustmentHistory stores all adjustments for a position.
type StopAdjustmentHistory struct {
	PositionID  string           `json:"positionId"`
	Adjustments []StopAdjustment `json:"adjustments"`
}

// AddAdjustment adds a new adjustment to the history.
func (h *StopAdjustmentHistory) AddAdjustment(adj StopAdjustment) {
	if h.Adjustments == nil {
		h.Adjustments = []StopAdjustment{}
	}
	h.Adjustments = append(h.Adjustments, adj)
}

// GetLastAdjustment returns the most recent adjustment.
func (h *StopAdjustmentHistory) GetLastAdjustment() *StopAdjustment {
	if len(h.Adjustments) == 0 {
		return nil
	}
	return &h.Adjustments[len(h.Adjustments)-1]
}

// ToJSON converts history to JSON bytes.
func (h *StopAdjustmentHistory) ToJSON() ([]byte, error) {
	return json.Marshal(h)
}

// StopUpdateResult contains the result of updating a position with stop management.
type StopUpdateResult struct {
	PriceUpdateResult // Embedded from position tracker

	// Stop management specific
	StopAdjusted          bool                  `json:"stopAdjusted"`
	StopAdjustment        *StopAdjustmentResult `json:"stopAdjustment,omitempty"`
	StopAdjustmentDetails *StopAdjustment       `json:"stopAdjustmentDetails,omitempty"`
}
