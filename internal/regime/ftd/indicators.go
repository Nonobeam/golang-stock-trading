package ftd

import (
	"math"
)

// CalculateRSI calculates the Relative Strength Index for a slice of prices.
// Returns 0 if insufficient data.
func CalculateRSI(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 0
	}

	// Calculate changes
	gains := make([]float64, 0, len(prices)-1)
	losses := make([]float64, 0, len(prices)-1)

	for i := 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains = append(gains, change)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -change)
		}
	}

	// Calculate initial average gain/loss
	var avgGain, avgLoss float64
	for i := 0; i < period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	// Calculate smoothed averages
	// Note: We need to iterate through the rest of the data if available
	// But usually RSI is calculated on the *latest* window.
	// Standard RSI uses Wilder's smoothing.
	// For simplicity and to match standard TA libs, we'll iterate through all remaining data points.
	
	for i := period; i < len(gains); i++ {
		avgGain = (avgGain*float64(period-1) + gains[i]) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + losses[i]) / float64(period)
	}

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

// CalculateSMA calculates the Simple Moving Average.
func CalculateSMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}

	sum := 0.0
	// Use the last 'period' prices
	start := len(prices) - period
	for i := start; i < len(prices); i++ {
		sum += prices[i]
	}

	return sum / float64(period)
}

// IsNewLow checks if the latest price is the lowest in the last N periods.
func IsNewLow(prices []float64, period int) bool {
	if len(prices) < period {
		return false
	}

	latest := prices[len(prices)-1]
	start := len(prices) - period
	
	// Check against previous prices (excluding latest)
	for i := start; i < len(prices)-1; i++ {
		if prices[i] <= latest {
			return false // Found a strictly lower or equal price previously
		}
	}
	return true
}

// IsAtSupport checks if current price is within tolerance% of a previous low support level.
// supportWindow: how far back to look for support (e.g., 60 days)
// tolerancePct: percentage proximity (e.g., 2.0 for 2%)
func IsAtSupport(prices []float64, supportWindow int, tolerancePct float64) (bool, float64) {
	if len(prices) < supportWindow {
		return false, 0
	}

	currentPrice := prices[len(prices)-1]
	tolerance := currentPrice * tolerancePct / 100.0
	
	// Find significant lows in the window (simple approach: lowest low)
	// Improved approach: Local minima
	minPrice := math.MaxFloat64
	
	// Look back, excluding recent 5 days to find *previous* support, not current dip
	lookbackStart := len(prices) - supportWindow
	lookbackEnd := len(prices) - 5
	
	if lookbackEnd <= lookbackStart {
		return false, 0
	}

	for i := lookbackStart; i < lookbackEnd; i++ {
		if prices[i] < minPrice {
			minPrice = prices[i]
		}
	}

	if minPrice == math.MaxFloat64 {
		return false, 0
	}

	// Check if current price is near this support
	if math.Abs(currentPrice-minPrice) <= tolerance {
		return true, minPrice
	}

	return false, 0
}
