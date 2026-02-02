package signals

import "math"

// DetectBullishCandle identifies bullish reversal candlestick patterns.
// Detects: Hammer, Bullish Engulfing, Doji, Long Lower Wick.
func DetectBullishCandle(
	currentOpen, currentHigh, currentLow, currentClose float64,
	prevOpen, prevHigh, prevLow, prevClose float64,
) *CandlePattern {
	
	body := math.Abs(currentClose - currentOpen)
	totalRange := currentHigh - currentLow
	upperWick := currentHigh - math.Max(currentOpen, currentClose)
	lowerWick := math.Min(currentOpen, currentClose) - currentLow
	
	isGreen := currentClose > currentOpen
	
	patternsFound := []string{}
	isBullish := false
	strength := "none"
	
	// Avoid division by zero
	if totalRange == 0 {
		return &CandlePattern{
			IsBullish:   false,
			PatternName: "No pattern (zero range)",
			Strength:    "none",
			Patterns:    []string{},
		}
	}
	
	bodyPercent := body / totalRange
	lowerWickPercent := lowerWick / totalRange
	upperWickPercent := upperWick / totalRange
	
	// Pattern 1: Hammer
	// Small body at top, long lower wick (2-3x body)
	isHammer := (
		bodyPercent < 0.3 &&
		lowerWickPercent > 0.5 &&
		upperWickPercent < 0.1)
	
	if isHammer {
		isBullish = true
		strength = "strong"
		patternsFound = append(patternsFound, "Hammer")
	}
	
	// Pattern 2: Bullish Engulfing
	// Current green candle engulfs previous red candle
	if prevOpen > 0 && prevClose > 0 {
		prevIsRed := prevClose < prevOpen
		currentEngulfs := currentClose > prevOpen && currentOpen < prevClose
		
		isBullishEngulfing := isGreen && prevIsRed && currentEngulfs
		
		if isBullishEngulfing {
			isBullish = true
			strength = "strong"
			patternsFound = append(patternsFound, "Bullish Engulfing")
		}
	}
	
	// Pattern 3: Doji
	// Open ≈ Close (indecision at support = potential reversal)
	bodyTiny := body / currentClose < 0.001 // Less than 0.1% of price
	
	if bodyTiny {
		isBullish = true
		if strength == "none" {
			strength = "moderate"
		}
		patternsFound = append(patternsFound, "Doji")
	}
	
	// Pattern 4: Long Lower Wick
	// Lower wick 2x body size (rejection of lower prices)
	if lowerWick > body*2 {
		isBullish = true
		if strength == "none" {
			strength = "moderate"
		}
		patternsFound = append(patternsFound, "Long Lower Wick")
	}
	
	patternName := "No bullish pattern"
	if len(patternsFound) > 0 {
		patternName = ""
		for i, p := range patternsFound {
			if i > 0 {
				patternName += ", "
			}
			patternName += p
		}
	}
	
	return &CandlePattern{
		IsBullish:   isBullish,
		PatternName: patternName,
		Strength:    strength,
		Patterns:    patternsFound,
	}
}
