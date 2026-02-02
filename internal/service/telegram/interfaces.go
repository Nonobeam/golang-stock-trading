package telegram

import (
	"context"
	"time"
)

// RiskManager interface for portfolio risk data
type RiskManager interface {
	GetPortfolioRisk() float64
	GetDailyLoss() float64
	GetCapitalUtilization() float64
	GetMaxPortfolioRisk() float64
	GetDailyLossLimit() float64
}

// Position represents an active trading position
type Position struct {
	Symbol         string
	EntryPrice     float64
	CurrentPrice   float64
	StopLoss       float64
	Targets        []float64
	EntryDate      time.Time
	PositionSize   int
	RMultiple      float64
	TargetProgress float64 // 0-100%
	StopDistance   float64 // Percentage away from stop
}

// PositionTracker interface for position monitoring
type PositionTracker interface {
	GetActivePositions() []Position
}

// RestartHandler interface for re-authentication flow
type RestartHandler interface {
	// OnRestart handles the restart flow with a new OTP
	// Returns nil on success, error on failure
	OnRestart(ctx context.Context, otp string) error
}
