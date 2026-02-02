package indicators

import (
	"github.com/nonobeam/golang-stock-trading/internal/data"
)

// OBV calculates On-Balance Volume.
type OBV struct{}

// NewOBV creates a new OBV indicator.
func NewOBV() *OBV {
	return &OBV{}
}

func (o *OBV) Name() string   { return "OBV" }
func (o *OBV) Period() int    { return 0 } // OBV doesn't have a period

// Calculate computes OBV for the series.
func (o *OBV) Calculate(series *data.Series) (float64, error) {
	closes := series.Closes()
	volumes := series.Volumes()
	return CalculateOBV(closes, volumes)
}

// CalculateOBV computes cumulative On-Balance Volume.
func CalculateOBV(closes, volumes []float64) (float64, error) {
	n := len(closes)
	if n < 2 || len(volumes) != n {
		return 0, data.ErrInsufficientData
	}

	obv := volumes[0] // Initial OBV

	for i := 1; i < n; i++ {
		if closes[i] > closes[i-1] {
			obv += volumes[i]
		} else if closes[i] < closes[i-1] {
			obv -= volumes[i]
		}
		// If close unchanged, OBV stays the same
	}

	return obv, nil
}

// CalculateOBVArray computes OBV for each position.
func CalculateOBVArray(closes, volumes []float64) ([]float64, error) {
	n := len(closes)
	if n < 2 || len(volumes) != n {
		return nil, data.ErrInsufficientData
	}

	result := make([]float64, n)
	result[0] = volumes[0]

	for i := 1; i < n; i++ {
		if closes[i] > closes[i-1] {
			result[i] = result[i-1] + volumes[i]
		} else if closes[i] < closes[i-1] {
			result[i] = result[i-1] - volumes[i]
		} else {
			result[i] = result[i-1]
		}
	}

	return result, nil
}

// VWAP calculates Volume-Weighted Average Price.
type VWAP struct{}

// NewVWAP creates a new VWAP indicator.
func NewVWAP() *VWAP {
	return &VWAP{}
}

func (v *VWAP) Name() string   { return "VWAP" }
func (v *VWAP) Period() int    { return 0 }

// Calculate computes VWAP for the series.
func (v *VWAP) Calculate(series *data.Series) (float64, error) {
	bars := series.All()
	return CalculateVWAP(bars)
}

// CalculateVWAP computes cumulative VWAP.
func CalculateVWAP(bars []data.OHLCV) (float64, error) {
	if len(bars) == 0 {
		return 0, data.ErrInsufficientData
	}

	cumulativePV := 0.0
	cumulativeV := 0.0

	for _, bar := range bars {
		tp := bar.TypicalPrice()
		cumulativePV += tp * bar.Volume
		cumulativeV += bar.Volume
	}

	if cumulativeV == 0 {
		return 0, nil
	}

	return cumulativePV / cumulativeV, nil
}

// VolumeMA calculates Volume Moving Average.
type VolumeMA struct {
	period int
}

// NewVolumeMA creates a new Volume MA indicator.
func NewVolumeMA(period int) *VolumeMA {
	if period <= 0 {
		period = 20
	}
	return &VolumeMA{period: period}
}

func (v *VolumeMA) Name() string   { return "VolumeMA" }
func (v *VolumeMA) Period() int    { return v.period }

// Calculate computes Volume MA for the series.
func (v *VolumeMA) Calculate(series *data.Series) (float64, error) {
	volumes := series.Volumes()
	return CalculateSMA(volumes, v.period)
}

// RelativeVolume calculates relative volume vs average.
func RelativeVolume(currentVolume, avgVolume float64) float64 {
	if avgVolume == 0 {
		return 0
	}
	return currentVolume / avgVolume
}

// VolumePercentile calculates the percentile rank of current volume.
func VolumePercentile(volumes []float64, period int) (float64, error) {
	n := len(volumes)
	if n < period {
		return 0, data.ErrInsufficientData
	}

	// Get last 'period' volumes for comparison
	window := volumes[n-period : n]
	currentVolume := volumes[n-1]

	// Count how many volumes are less than current
	countLess := 0
	for _, v := range window {
		if v < currentVolume {
			countLess++
		}
	}

	return (float64(countLess) / float64(period)) * 100, nil
}
