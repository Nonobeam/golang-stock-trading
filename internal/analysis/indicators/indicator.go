// Package indicators provides technical indicator calculations for stock analysis.
package indicators

import (
	"github.com/nonobeam/golang-stock-trading/internal/data"
)

// Indicator defines the interface for all technical indicators.
type Indicator interface {
	// Calculate computes the indicator value for the series
	Calculate(series *data.Series) (float64, error)
	// Name returns the indicator name
	Name() string
	// Period returns the lookback period
	Period() int
}

// Result holds indicator calculation results with optional additional values.
type Result struct {
	Value  float64
	Signal float64 // For indicators with signal lines (MACD, Stochastic)
	Upper  float64 // For bands (Bollinger)
	Lower  float64 // For bands (Bollinger)
	Extra  map[string]float64
}

// NewResult creates a simple result with just the main value.
func NewResult(value float64) Result {
	return Result{Value: value}
}
