// Package risk provides position sizing and risk management calculations.
package risk

import (
	"github.com/nonobeam/golang-stock-trading/internal/config"
)

// RiskParams holds parameters for position sizing calculations.
type RiskParams struct {
	AccountBalance float64 // Total account value
	RiskPercent    float64 // Risk per trade as decimal (0.01 = 1%)
	EntryPrice     float64 // Entry price
	StopPrice      float64 // Stop loss price
	TradeScore     int     // Trade quality score (0-13)
}

// StopParams holds parameters for stop loss calculations.
type StopParams struct {
	EntryPrice    float64 // Entry price
	ATR           float64 // Average True Range
	Multiplier    float64 // ATR multiplier
	Percentage    float64 // Percentage for percentage-based stops
	TechnicalStop float64 // Technical level (swing low, EMA, etc.)
	Buffer        float64 // Buffer percentage below technical level
	IsLong        bool    // Long or short position
	IsOvernight   bool    // Held overnight (for VN gap adjustment)

	// For swing low calculation
	Lows            []float64 // Historical low prices (most recent last)
	LookbackPeriods int       // How far back to look for swing low

	// For floor awareness (Vietnam)
	ReferencePrice    float64 // Today's reference price for floor/ceiling
	DailyLimitPercent float64 // Daily limit (0.07 for 7%)

	// For pre-emptive exit strategy
	EnablePreemptive  bool    // Calculate early exit alert levels
	PreemptiveTrigger float64 // At what % of stop to exit (0.7 = 70%)
}

// TargetParams holds parameters for target calculations.
type TargetParams struct {
	EntryPrice       float64   // Entry price
	StopPrice        float64   // Stop loss price
	ATR              float64   // Average True Range
	ResistanceLevels []float64 // Technical resistance levels
	IsLong           bool      // Long or short position
}

// FibonacciParams holds parameters for Fibonacci extension calculations.
type FibonacciParams struct {
	SwingLow    float64   // Point A - starting low of the move
	SwingHigh   float64   // Point B - peak high of the move
	PullbackLow float64   // Point C - retracement low (entry point)
	FibRatios   []float64 // Custom Fib ratios (default: [0.618, 1.0, 1.618, 2.618])
}

// MeasuredMoveParams holds parameters for measured move calculations.
type MeasuredMoveParams struct {
	ConsolidationLow  float64   // Bottom of consolidation range
	ConsolidationHigh float64   // Top of consolidation range
	BreakoutPrice     float64   // Actual breakout entry price
	Multiples         []float64 // Range multipliers (default: [1.0, 1.5, 2.0])
}

// TrailingStopParams holds parameters for trailing stop calculations.
type TrailingStopParams struct {
	EntryPrice          float64 // Original entry price
	CurrentPrice        float64 // Current market price
	HighestPriceReached float64 // Highest price since entry
	Method              string  // "atr", "ema", "percentage", "swing_low"

	// Method-specific parameters
	ATR            float64 // For ATR method
	ATRMultiplier  float64 // For ATR method (default: 1.5)
	EMAValue       float64 // For EMA method
	Percentage     float64 // For percentage method (default: 5.0)
	RecentSwingLow float64 // For swing low method
}

// ComprehensiveTargetParams holds all parameters for comprehensive target analysis.
type ComprehensiveTargetParams struct {
	// Required
	EntryPrice float64
	StopLoss   float64
	IsLong     bool

	// Optional - for specific methods
	ATR              float64
	ResistanceLevels []float64
	FibParams        *FibonacciParams
	MeasuredParams   *MeasuredMoveParams
}

// Direction represents trade direction.
type Direction int

const (
	Long  Direction = 1
	Short Direction = -1
)

// GetLotSize returns the configured lot size (default: 100).
func GetLotSize() int {
	cfg := config.Get()
	if cfg.Trading.VNLotSize > 0 {
		return cfg.Trading.VNLotSize
	}
	return 100
}

// GetMaxStopPercent returns the configured max stop percent (default: 0.07).
func GetMaxStopPercent() float64 {
	cfg := config.Get()
	if cfg.Trading.MaxStopPercent > 0 {
		return cfg.Trading.MaxStopPercent
	}
	return 0.07
}

// GetGapRiskFactor returns the configured gap risk factor (default: 1.5).
func GetGapRiskFactor() float64 {
	cfg := config.Get()
	if cfg.Trading.GapRiskFactor > 0 {
		return cfg.Trading.GapRiskFactor
	}
	return 1.5
}

// GetDefaultRiskPercent returns the default risk percent (default: 0.01).
func GetDefaultRiskPercent() float64 {
	cfg := config.Get()
	if cfg.Trading.DefaultRiskPct > 0 {
		return cfg.Trading.DefaultRiskPct
	}
	return 0.01
}

// GetDefaultATRMultiplier returns the default ATR multiplier (default: 2.0).
func GetDefaultATRMultiplier() float64 {
	cfg := config.Get()
	if cfg.Trading.DefaultATRMult > 0 {
		return cfg.Trading.DefaultATRMult
	}
	return 2.0
}

// GetMaxPositionPercent returns the max position % of capital (default: 20).
func GetMaxPositionPercent() float64 {
	cfg := config.Get()
	if cfg.Trading.MaxPositionPercent > 0 {
		return cfg.Trading.MaxPositionPercent
	}
	return 20.0
}

// GetMinStopPercent returns the minimum stop distance % (default: 0.01 = 1%).
func GetMinStopPercent() float64 {
	cfg := config.Get()
	if cfg.Trading.MinStopPercent > 0 {
		return cfg.Trading.MinStopPercent
	}
	return 0.01
}

// GetMinStopDistance returns the minimum stop distance in VND (default: 500).
func GetMinStopDistance() float64 {
	cfg := config.Get()
	if cfg.Trading.MinStopDistance > 0 {
		return cfg.Trading.MinStopDistance
	}
	return 500.0
}

// GetSwingLookback returns the swing low lookback period (default: 20).
func GetSwingLookback() int {
	cfg := config.Get()
	if cfg.Trading.SwingLookback > 0 {
		return cfg.Trading.SwingLookback
	}
	return 20
}

// IsPreemptiveEnabled returns whether pre-emptive exit alerts are enabled.
func IsPreemptiveEnabled() bool {
	cfg := config.Get()
	return cfg.Trading.PreemptiveEnabled
}
