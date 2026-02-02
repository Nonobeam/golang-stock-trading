package indicators

import (
	"github.com/nonobeam/golang-stock-trading/internal/data"
)

// RSI calculates Relative Strength Index using Wilder's smoothing.
type RSI struct {
	period int
}

// NewRSI creates a new RSI indicator (default period: 14).
func NewRSI(period int) *RSI {
	if period <= 0 {
		period = 14
	}
	return &RSI{period: period}
}

func (r *RSI) Name() string   { return "RSI" }
func (r *RSI) Period() int    { return r.period }

// Calculate computes RSI for the series.
func (r *RSI) Calculate(series *data.Series) (float64, error) {
	closes := series.Closes()
	return CalculateRSI(closes, r.period)
}

// CalculateRSI computes RSI using Wilder's smoothing method.
func CalculateRSI(prices []float64, period int) (float64, error) {
	if len(prices) < period+1 {
		return 0, data.ErrInsufficientData
	}

	gains := make([]float64, len(prices)-1)
	losses := make([]float64, len(prices)-1)

	// Calculate price changes
	for i := 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains[i-1] = change
		} else {
			losses[i-1] = -change
		}
	}

	// First average (SMA)
	avgGain := 0.0
	avgLoss := 0.0
	for i := 0; i < period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	// Apply Wilder's smoothing for remaining values
	for i := period; i < len(gains); i++ {
		avgGain = (avgGain*float64(period-1) + gains[i]) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + losses[i]) / float64(period)
	}

	// Calculate RSI
	if avgLoss == 0 {
		return 100, nil
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs)), nil
}

// MACD calculates Moving Average Convergence Divergence.
type MACD struct {
	fastPeriod   int
	slowPeriod   int
	signalPeriod int
}

// MACDResult holds MACD calculation results.
type MACDResult struct {
	MACD      float64
	Signal    float64
	Histogram float64
}

// NewMACD creates a new MACD indicator (default: 12, 26, 9).
func NewMACD(fast, slow, signal int) *MACD {
	if fast <= 0 {
		fast = 12
	}
	if slow <= 0 {
		slow = 26
	}
	if signal <= 0 {
		signal = 9
	}
	return &MACD{
		fastPeriod:   fast,
		slowPeriod:   slow,
		signalPeriod: signal,
	}
}

func (m *MACD) Name() string   { return "MACD" }
func (m *MACD) Period() int    { return m.slowPeriod }

// Calculate computes MACD line for the series.
func (m *MACD) Calculate(series *data.Series) (float64, error) {
	result, err := m.CalculateFull(series)
	if err != nil {
		return 0, err
	}
	return result.MACD, nil
}

// CalculateFull computes MACD, Signal, and Histogram.
func (m *MACD) CalculateFull(series *data.Series) (MACDResult, error) {
	closes := series.Closes()
	return CalculateMACD(closes, m.fastPeriod, m.slowPeriod, m.signalPeriod)
}

// CalculateMACD computes MACD, Signal, and Histogram.
func CalculateMACD(prices []float64, fast, slow, signal int) (MACDResult, error) {
	if len(prices) < slow+signal-1 {
		return MACDResult{}, data.ErrInsufficientData
	}

	// Calculate EMAs
	fastEMA, err := CalculateEMAArray(prices, fast)
	if err != nil {
		return MACDResult{}, err
	}

	slowEMA, err := CalculateEMAArray(prices, slow)
	if err != nil {
		return MACDResult{}, err
	}

	// Calculate MACD line (fast EMA - slow EMA)
	macdLine := make([]float64, len(prices))
	for i := slow - 1; i < len(prices); i++ {
		macdLine[i] = fastEMA[i] - slowEMA[i]
	}

	// Calculate Signal line (EMA of MACD)
	validMACD := macdLine[slow-1:]
	signalLine, err := CalculateEMAArray(validMACD, signal)
	if err != nil {
		return MACDResult{}, err
	}

	// Get last values
	lastMACD := macdLine[len(macdLine)-1]
	lastSignal := signalLine[len(signalLine)-1]
	histogram := lastMACD - lastSignal

	return MACDResult{
		MACD:      lastMACD,
		Signal:    lastSignal,
		Histogram: histogram,
	}, nil
}

// Stochastic calculates the Stochastic Oscillator.
type Stochastic struct {
	kPeriod  int
	kSmooth  int
	dPeriod  int
}

// StochasticResult holds Stochastic calculation results.
type StochasticResult struct {
	K float64
	D float64
}

// NewStochastic creates a new Stochastic indicator (default: 14, 3, 3).
func NewStochastic(kPeriod, kSmooth, dPeriod int) *Stochastic {
	if kPeriod <= 0 {
		kPeriod = 14
	}
	if kSmooth <= 0 {
		kSmooth = 3
	}
	if dPeriod <= 0 {
		dPeriod = 3
	}
	return &Stochastic{
		kPeriod: kPeriod,
		kSmooth: kSmooth,
		dPeriod: dPeriod,
	}
}

func (s *Stochastic) Name() string   { return "Stochastic" }
func (s *Stochastic) Period() int    { return s.kPeriod }

// Calculate computes %K for the series.
func (s *Stochastic) Calculate(series *data.Series) (float64, error) {
	result, err := s.CalculateFull(series)
	if err != nil {
		return 0, err
	}
	return result.K, nil
}

// CalculateFull computes %K and %D.
func (s *Stochastic) CalculateFull(series *data.Series) (StochasticResult, error) {
	highs := series.Highs()
	lows := series.Lows()
	closes := series.Closes()

	return CalculateStochastic(highs, lows, closes, s.kPeriod, s.kSmooth, s.dPeriod)
}

// CalculateStochastic computes %K and %D.
func CalculateStochastic(highs, lows, closes []float64, kPeriod, kSmooth, dPeriod int) (StochasticResult, error) {
	n := len(closes)
	needed := kPeriod + kSmooth + dPeriod - 2
	if n < needed {
		return StochasticResult{}, data.ErrInsufficientData
	}

	// Calculate raw %K values
	rawK := make([]float64, n)
	for i := kPeriod - 1; i < n; i++ {
		highestHigh := highs[i-kPeriod+1]
		lowestLow := lows[i-kPeriod+1]
		for j := i - kPeriod + 2; j <= i; j++ {
			if highs[j] > highestHigh {
				highestHigh = highs[j]
			}
			if lows[j] < lowestLow {
				lowestLow = lows[j]
			}
		}

		rangeHL := highestHigh - lowestLow
		if rangeHL == 0 {
			rawK[i] = 50 // No range, neutral
		} else {
			rawK[i] = ((closes[i] - lowestLow) / rangeHL) * 100
		}
	}

	// Smooth %K
	validRawK := rawK[kPeriod-1:]
	smoothK, err := CalculateSMAArray(validRawK, kSmooth)
	if err != nil {
		return StochasticResult{}, err
	}

	// Calculate %D (SMA of smoothed %K)
	validSmoothK := smoothK[kSmooth-1:]
	dLine, err := CalculateSMAArray(validSmoothK, dPeriod)
	if err != nil {
		return StochasticResult{}, err
	}

	return StochasticResult{
		K: smoothK[len(smoothK)-1],
		D: dLine[len(dLine)-1],
	}, nil
}
