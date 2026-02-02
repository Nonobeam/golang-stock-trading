package indicators

import (
	"github.com/nonobeam/golang-stock-trading/internal/data"
)

// SMA calculates Simple Moving Average.
type SMA struct {
	period int
}

// NewSMA creates a new SMA indicator with the specified period.
func NewSMA(period int) *SMA {
	return &SMA{period: period}
}

func (s *SMA) Name() string   { return "SMA" }
func (s *SMA) Period() int    { return s.period }

// Calculate computes SMA for the series.
func (s *SMA) Calculate(series *data.Series) (float64, error) {
	closes := series.Closes()
	return CalculateSMA(closes, s.period)
}

// CalculateSMA computes SMA on a price slice, returning the last value.
func CalculateSMA(prices []float64, period int) (float64, error) {
	if len(prices) < period {
		return 0, data.ErrInsufficientData
	}

	sum := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
	}
	return sum / float64(period), nil
}

// CalculateSMAArray computes SMA for all valid positions.
// Returns array where first (period-1) values are 0.
func CalculateSMAArray(prices []float64, period int) ([]float64, error) {
	if len(prices) < period {
		return nil, data.ErrInsufficientData
	}

	result := make([]float64, len(prices))

	// Calculate first SMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	result[period-1] = sum / float64(period)

	// Use rolling window for subsequent values
	for i := period; i < len(prices); i++ {
		sum = sum - prices[i-period] + prices[i]
		result[i] = sum / float64(period)
	}

	return result, nil
}

// EMA calculates Exponential Moving Average.
type EMA struct {
	period int
}

// NewEMA creates a new EMA indicator with the specified period.
func NewEMA(period int) *EMA {
	return &EMA{period: period}
}

func (e *EMA) Name() string   { return "EMA" }
func (e *EMA) Period() int    { return e.period }

// Calculate computes EMA for the series.
func (e *EMA) Calculate(series *data.Series) (float64, error) {
	closes := series.Closes()
	return CalculateEMA(closes, e.period)
}

// CalculateEMA computes EMA on a price slice, returning the last value.
func CalculateEMA(prices []float64, period int) (float64, error) {
	array, err := CalculateEMAArray(prices, period)
	if err != nil {
		return 0, err
	}
	return array[len(array)-1], nil
}

// CalculateEMAArray computes EMA for all valid positions.
// First value is SMA bootstrap, subsequent use exponential smoothing.
func CalculateEMAArray(prices []float64, period int) ([]float64, error) {
	if len(prices) < period {
		return nil, data.ErrInsufficientData
	}

	result := make([]float64, len(prices))
	k := 2.0 / float64(period+1) // Smoothing factor

	// Bootstrap: first EMA is SMA of first 'period' prices
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	result[period-1] = sum / float64(period)

	// Calculate subsequent EMAs
	for i := period; i < len(prices); i++ {
		result[i] = result[i-1] + k*(prices[i]-result[i-1])
	}

	return result, nil
}
