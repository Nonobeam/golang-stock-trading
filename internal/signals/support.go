package signals

import (
	"math"

	"github.com/nonobeam/golang-stock-trading/internal/data"
)

// FindSupportLevels identifies support price levels using swing lows, Fibonacci retracements, and round numbers.
func FindSupportLevels(series *data.Series, lookback int) []SupportLevel {
	if series.Len() < lookback {
		return []SupportLevel{}
	}
	
	supports := []SupportLevel{}
	
	// Add swing low supports
	swingLows := findSwingLows(series, lookback)
	supports = append(supports, swingLows...)
	
	// Add Fibonacci supports
	fibLevels := findFibonacciSupports(series, lookback)
	supports = append(supports, fibLevels...)
	
	// Add round number supports
	lastClose, _ := series.Last()
	roundNumbers := findRoundNumbers(lastClose.Close, 0.10)
	supports = append(supports, roundNumbers...)
	
	return supports
}

// FindResistanceLevels identifies resistance price levels using swing highs.
func FindResistanceLevels(series *data.Series, lookback int) []SupportLevel {
	if series.Len() < lookback {
		return []SupportLevel{}
	}
	
	resistances := []SupportLevel{}
	
	// Add swing high resistances
	swingHighs := findSwingHighs(series, lookback)
	resistances = append(resistances, swingHighs...)
	
	// Add Fibonacci resistances (from retracements)
	fibLevels := findFibonacciResistances(series, lookback)
	resistances = append(resistances, fibLevels...)
	
	return resistances
}

// CheckSupportConfluence checks if price is near multiple support levels.
func CheckSupportConfluence(price float64, supports []SupportLevel, tolerancePercent float64) bool {
	confluenceCount := 0
	tolerance := price * (tolerancePercent / 100.0)
	
	for _, support := range supports {
		if math.Abs(price-support.Price) <= tolerance {
			confluenceCount++
		}
	}
	
	return confluenceCount >= 2 // At least 2 supports align
}

// findSwingLows identifies local minima as support levels.
func findSwingLows(series *data.Series, lookback int) []SupportLevel {
	supports := []SupportLevel{}
	lows := series.Lows()
	
	if len(lows) < lookback {
		return supports
	}
	
	recentLows := lows[len(lows)-lookback:]
	
	// Find local minima (window = 5 bars)
	window := 5
	for i := window; i < len(recentLows)-window; i++ {
		isLocalMin := true
		centerLow := recentLows[i]
		
		// Check if center is lowest in window
		for j := i - window; j <= i+window; j++ {
			if j != i && recentLows[j] < centerLow {
				isLocalMin = false
				break
			}
		}
		
		if isLocalMin {
			supports = append(supports, SupportLevel{
				Price:      centerLow,
				Type:       "Swing Low",
				Confidence: 0.8, // High confidence for swing lows
			})
		}
	}
	
	return supports
}

// findSwingHighs identifies local maxima as resistance levels.
func findSwingHighs(series *data.Series, lookback int) []SupportLevel {
	resistances := []SupportLevel{}
	highs := series.Highs()
	
	if len(highs) < lookback {
		return resistances
	}
	
	recentHighs := highs[len(highs)-lookback:]
	
	// Find local maxima (window = 5 bars)
	window := 5
	for i := window; i < len(recentHighs)-window; i++ {
		isLocalMax := true
		centerHigh := recentHighs[i]
		
		// Check if center is highest in window
		for j := i - window; j <= i+window; j++ {
			if j != i && recentHighs[j] > centerHigh {
				isLocalMax = false
				break
			}
		}
		
		if isLocalMax {
			resistances = append(resistances, SupportLevel{
				Price:      centerHigh,
				Type:       "Swing High",
				Confidence: 0.8,
			})
		}
	}
	
	return resistances
}

// findFibonacciSupports calculates Fibonacci retracement levels as supports.
func findFibonacciSupports(series *data.Series, lookback int) []SupportLevel {
	highs := series.Highs()
	lows := series.Lows()
	
	if len(highs) < lookback {
		return []SupportLevel{}
	}
	
	recentHighs := highs[len(highs)-lookback:]
	recentLows := lows[len(lows)-lookback:]
	
	// Find recent swing high and low
	swingHigh := recentHighs[0]
	swingLow := recentLows[0]
	
	for _, high := range recentHighs {
		if high > swingHigh {
			swingHigh = high
		}
	}
	
	for _, low := range recentLows {
		if low < swingLow {
			swingLow = low
		}
	}
	
	rangeSize := swingHigh - swingLow
	
	// Calculate Fibonacci retracement levels (supports during downtrend/pullback)
	fibRatios := map[string]float64{
		"23.6%": 0.236,
		"38.2%": 0.382,
		"50.0%": 0.500,
		"61.8%": 0.618,
	}
	
	supports := []SupportLevel{}
	for name, ratio := range fibRatios {
		level := swingHigh - (rangeSize * ratio)
		supports = append(supports, SupportLevel{
			Price:      level,
			Type:       "Fibonacci " + name,
			Confidence: 0.6, // Moderate confidence
		})
	}
	
	return supports
}

// findFibonacciResistances calculates Fibonacci extension levels as resistances.
func findFibonacciResistances(series *data.Series, lookback int) []SupportLevel {
	highs := series.Highs()
	lows := series.Lows()
	
	if len(highs) < lookback {
		return []SupportLevel{}
	}
	
	recentHighs := highs[len(highs)-lookback:]
	recentLows := lows[len(lows)-lookback:]
	
	swingHigh := recentHighs[0]
	swingLow := recentLows[0]
	
	for _, high := range recentHighs {
		if high > swingHigh {
			swingHigh = high
		}
	}
	
	for _, low := range recentLows {
		if low < swingLow {
			swingLow = low
		}
	}
	
	rangeSize := swingHigh - swingLow
	
	// Fibonacci extensions (resistances during uptrend)
	fibExtensions := map[string]float64{
		"127.2%": 1.272,
		"161.8%": 1.618,
	}
	
	resistances := []SupportLevel{}
	for name, ratio := range fibExtensions {
		level := swingHigh + (rangeSize * (ratio - 1.0))
		resistances = append(resistances, SupportLevel{
			Price:      level,
			Type:       "Fibonacci Extension " + name,
			Confidence: 0.5, // Lower confidence for extensions
		})
	}
	
	return resistances
}

// findRoundNumbers identifies psychological round number levels near current price.
func findRoundNumbers(currentPrice float64, tolerancePercent float64) []SupportLevel {
	roundNumbers := []SupportLevel{}
	
	// Determine magnitude (1000, 5000, 10000, 50000, 100000)
	magnitudes := []float64{1000, 5000, 10000, 50000, 100000}
	
	for _, mag := range magnitudes {
		// Find nearest round number at this magnitude
		roundedDown := math.Floor(currentPrice/mag) * mag
		roundedUp := math.Ceil(currentPrice/mag) * mag
		
		// Check if within tolerance
		tolerance := currentPrice * tolerancePercent
		
		if math.Abs(currentPrice-roundedDown) <= tolerance {
			roundNumbers = append(roundNumbers, SupportLevel{
				Price:      roundedDown,
				Type:       "Round Number",
				Confidence: 0.5, // Psychological levels have moderate confidence
			})
		}
		
		if math.Abs(currentPrice-roundedUp) <= tolerance && roundedUp != roundedDown {
			roundNumbers = append(roundNumbers, SupportLevel{
				Price:      roundedUp,
				Type:       "Round Number",
				Confidence: 0.5,
			})
		}
	}
	
	return roundNumbers
}
