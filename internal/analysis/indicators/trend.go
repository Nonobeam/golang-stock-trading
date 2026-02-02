package indicators

import (
	"math"

	"github.com/nonobeam/golang-stock-trading/internal/data"
)

// ADX calculates Average Directional Index.
type ADX struct {
	period int
}

// ADXResult holds ADX calculation results.
type ADXResult struct {
	ADX      float64
	PlusDI   float64
	MinusDI  float64
}

// NewADX creates a new ADX indicator (default period: 14).
func NewADX(period int) *ADX {
	if period <= 0 {
		period = 14
	}
	return &ADX{period: period}
}

func (a *ADX) Name() string   { return "ADX" }
func (a *ADX) Period() int    { return a.period }

// Calculate computes ADX for the series.
func (a *ADX) Calculate(series *data.Series) (float64, error) {
	result, err := a.CalculateFull(series)
	if err != nil {
		return 0, err
	}
	return result.ADX, nil
}

// CalculateFull computes ADX, +DI, and -DI.
func (a *ADX) CalculateFull(series *data.Series) (ADXResult, error) {
	highs := series.Highs()
	lows := series.Lows()
	closes := series.Closes()
	return CalculateADX(highs, lows, closes, a.period)
}

// CalculateADX computes ADX, +DI, and -DI.
func CalculateADX(highs, lows, closes []float64, period int) (ADXResult, error) {
	n := len(closes)
	// Need enough data for ADX calculation: 2*period for smoothing
	if n < 2*period {
		return ADXResult{}, data.ErrInsufficientData
	}

	// Calculate True Ranges, +DM, -DM
	tr := make([]float64, n-1)
	plusDM := make([]float64, n-1)
	minusDM := make([]float64, n-1)

	for i := 1; i < n; i++ {
		// True Range
		tr[i-1] = TrueRange(highs[i], lows[i], closes[i-1])

		// Directional Movement
		upMove := highs[i] - highs[i-1]
		downMove := lows[i-1] - lows[i]

		if upMove > downMove && upMove > 0 {
			plusDM[i-1] = upMove
		}
		if downMove > upMove && downMove > 0 {
			minusDM[i-1] = downMove
		}
	}

	// First smoothed values (SMA)
	smoothedTR := 0.0
	smoothedPlusDM := 0.0
	smoothedMinusDM := 0.0

	for i := 0; i < period; i++ {
		smoothedTR += tr[i]
		smoothedPlusDM += plusDM[i]
		smoothedMinusDM += minusDM[i]
	}

	// Apply Wilder's smoothing
	for i := period; i < len(tr); i++ {
		smoothedTR = smoothedTR - (smoothedTR / float64(period)) + tr[i]
		smoothedPlusDM = smoothedPlusDM - (smoothedPlusDM / float64(period)) + plusDM[i]
		smoothedMinusDM = smoothedMinusDM - (smoothedMinusDM / float64(period)) + minusDM[i]
	}

	// Calculate +DI and -DI
	plusDI := 0.0
	minusDI := 0.0
	if smoothedTR != 0 {
		plusDI = (smoothedPlusDM / smoothedTR) * 100
		minusDI = (smoothedMinusDM / smoothedTR) * 100
	}

	// Calculate DX values for ADX smoothing
	dxValues := make([]float64, 0)

	// Re-calculate to collect DX values
	sTR := 0.0
	sPlusDM := 0.0
	sMinusDM := 0.0

	for i := 0; i < period; i++ {
		sTR += tr[i]
		sPlusDM += plusDM[i]
		sMinusDM += minusDM[i]
	}

	for i := period - 1; i < len(tr); i++ {
		if i >= period {
			sTR = sTR - (sTR / float64(period)) + tr[i]
			sPlusDM = sPlusDM - (sPlusDM / float64(period)) + plusDM[i]
			sMinusDM = sMinusDM - (sMinusDM / float64(period)) + minusDM[i]
		}

		pDI := 0.0
		mDI := 0.0
		if sTR != 0 {
			pDI = (sPlusDM / sTR) * 100
			mDI = (sMinusDM / sTR) * 100
		}

		dx := 0.0
		diSum := pDI + mDI
		if diSum != 0 {
			dx = (math.Abs(pDI-mDI) / diSum) * 100
		}
		dxValues = append(dxValues, dx)
	}

	// Calculate ADX (smoothed DX)
	if len(dxValues) < period {
		return ADXResult{}, data.ErrInsufficientData
	}

	// First ADX is SMA of first 'period' DX values
	adx := 0.0
	for i := 0; i < period; i++ {
		adx += dxValues[i]
	}
	adx /= float64(period)

	// Apply Wilder's smoothing for subsequent values
	for i := period; i < len(dxValues); i++ {
		adx = (adx*float64(period-1) + dxValues[i]) / float64(period)
	}

	return ADXResult{
		ADX:     adx,
		PlusDI:  plusDI,
		MinusDI: minusDI,
	}, nil
}

// TrendStrength returns a description of trend strength based on ADX value.
func TrendStrength(adx float64) string {
	switch {
	case adx < 20:
		return "weak"
	case adx < 25:
		return "developing"
	case adx < 50:
		return "strong"
	default:
		return "very_strong"
	}
}
