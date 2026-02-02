package indicators

import (
	"math"

	"github.com/nonobeam/golang-stock-trading/internal/data"
)

// ATR calculates Average True Range using Wilder's smoothing.
type ATR struct {
	period int
}

// NewATR creates a new ATR indicator (default period: 14).
func NewATR(period int) *ATR {
	if period <= 0 {
		period = 14
	}
	return &ATR{period: period}
}

func (a *ATR) Name() string   { return "ATR" }
func (a *ATR) Period() int    { return a.period }

// Calculate computes ATR for the series.
func (a *ATR) Calculate(series *data.Series) (float64, error) {
	highs := series.Highs()
	lows := series.Lows()
	closes := series.Closes()
	return CalculateATR(highs, lows, closes, a.period)
}

// TrueRange calculates the True Range for a single bar.
func TrueRange(high, low, prevClose float64) float64 {
	tr1 := high - low
	tr2 := math.Abs(high - prevClose)
	tr3 := math.Abs(low - prevClose)

	return math.Max(tr1, math.Max(tr2, tr3))
}

// CalculateATR computes ATR using Wilder's smoothing.
func CalculateATR(highs, lows, closes []float64, period int) (float64, error) {
	n := len(closes)
	if n < period+1 {
		return 0, data.ErrInsufficientData
	}

	// Calculate True Ranges
	trueRanges := make([]float64, n-1)
	for i := 1; i < n; i++ {
		trueRanges[i-1] = TrueRange(highs[i], lows[i], closes[i-1])
	}

	// First ATR is SMA of first 'period' TRs
	atr := 0.0
	for i := 0; i < period; i++ {
		atr += trueRanges[i]
	}
	atr /= float64(period)

	// Apply Wilder's smoothing for subsequent values
	for i := period; i < len(trueRanges); i++ {
		atr = (atr*float64(period-1) + trueRanges[i]) / float64(period)
	}

	return atr, nil
}

// ATRPercent calculates ATR as a percentage of price.
func ATRPercent(atr, close float64) float64 {
	if close == 0 {
		return 0
	}
	return (atr / close) * 100
}

// BollingerBands calculates Bollinger Bands.
type BollingerBands struct {
	period int
	stdDev float64
}

// BollingerResult holds Bollinger Bands calculation results.
type BollingerResult struct {
	Upper     float64
	Middle    float64
	Lower     float64
	Bandwidth float64 // (Upper - Lower) / Middle
	PercentB  float64 // (Price - Lower) / (Upper - Lower)
}

// NewBollingerBands creates a new Bollinger Bands indicator (default: 20 period, 2 std dev).
func NewBollingerBands(period int, stdDev float64) *BollingerBands {
	if period <= 0 {
		period = 20
	}
	if stdDev <= 0 {
		stdDev = 2.0
	}
	return &BollingerBands{
		period: period,
		stdDev: stdDev,
	}
}

func (b *BollingerBands) Name() string   { return "BollingerBands" }
func (b *BollingerBands) Period() int    { return b.period }

// Calculate computes the middle band (SMA).
func (b *BollingerBands) Calculate(series *data.Series) (float64, error) {
	closes := series.Closes()
	result, err := CalculateBollingerBands(closes, b.period, b.stdDev)
	if err != nil {
		return 0, err
	}
	return result.Middle, nil
}

// CalculateFull computes all Bollinger Bands values.
func (b *BollingerBands) CalculateFull(series *data.Series) (BollingerResult, error) {
	closes := series.Closes()
	return CalculateBollingerBands(closes, b.period, b.stdDev)
}

// CalculateBollingerBands computes upper, middle, lower bands and derived metrics.
func CalculateBollingerBands(prices []float64, period int, stdDevMult float64) (BollingerResult, error) {
	n := len(prices)
	if n < period {
		return BollingerResult{}, data.ErrInsufficientData
	}

	// Calculate SMA (middle band)
	sum := 0.0
	for i := n - period; i < n; i++ {
		sum += prices[i]
	}
	middle := sum / float64(period)

	// Calculate standard deviation
	variance := 0.0
	for i := n - period; i < n; i++ {
		diff := prices[i] - middle
		variance += diff * diff
	}
	stdDev := math.Sqrt(variance / float64(period))

	// Calculate bands
	upper := middle + stdDevMult*stdDev
	lower := middle - stdDevMult*stdDev

	// Calculate derived metrics
	bandwidth := 0.0
	if middle != 0 {
		bandwidth = (upper - lower) / middle
	}

	percentB := 0.0
	rangeUL := upper - lower
	if rangeUL != 0 {
		currentPrice := prices[n-1]
		percentB = (currentPrice - lower) / rangeUL
	}

	return BollingerResult{
		Upper:     upper,
		Middle:    middle,
		Lower:     lower,
		Bandwidth: bandwidth,
		PercentB:  percentB,
	}, nil
}
