package regime

import "fmt"

// ClassifyBasicRegime performs basic regime classification using price and MA positions.
func ClassifyBasicRegime(
	currentPrice float64,
	ma50 float64,
	ma200 float64,
	highs []float64,
	lows []float64,
	config RegimeConfig,
) (RegimeType, int, []string) {

	factors := []string{}
	regimeScore := 0

	// Factor 1: Price position relative to 200 MA
	priceToMA200Ratio := currentPrice / ma200

	if priceToMA200Ratio >= config.StrongBullThreshold {
		regimeScore += 3
		percentAbove := (priceToMA200Ratio - 1) * 100
		factors = append(factors, fmt.Sprintf("Price %.1f%% above 200 MA (strong)", percentAbove))
	} else if priceToMA200Ratio >= config.MildBullLower {
		regimeScore += 1
		percentAbove := (priceToMA200Ratio - 1) * 100
		factors = append(factors, fmt.Sprintf("Price %.1f%% above 200 MA (mild)", percentAbove))
	} else if priceToMA200Ratio >= config.MildBearLower {
		regimeScore += 0
		percentBelow := (1 - priceToMA200Ratio) * 100
		factors = append(factors, fmt.Sprintf("Price %.1f%% below 200 MA (mild bear)", percentBelow))
	} else {
		regimeScore -= 2
		percentBelow := (1 - priceToMA200Ratio) * 100
		factors = append(factors, fmt.Sprintf("Price %.1f%% below 200 MA (strong bear)", percentBelow))
	}

	// Factor 2: MA relationship (50 vs 200)
	maGoldenCross := ma50 > ma200
	if maGoldenCross {
		regimeScore += 1
		factors = append(factors, "50 MA above 200 MA (bullish)")
	} else {
		regimeScore -= 1
		factors = append(factors, "50 MA below 200 MA (bearish)")
	}

	// Factor 3: Trend structure
	if len(highs) >= 60 && len(lows) >= 60 {
		structure := AnalyzeTrendStructure(highs, lows, 60)

		if structure.Type == "higher_highs_lows" {
			regimeScore += 2
			factors = append(factors, "Making higher highs and lows")
		} else if structure.Type == "lower_highs_lows" {
			regimeScore -= 2
			factors = append(factors, "Making lower highs and lows")
		} else {
			factors = append(factors, "Choppy structure")
		}
	}

	// Determine regime from score
	var regime RegimeType
	if regimeScore >= 5 {
		regime = RegimeStrongBull
	} else if regimeScore >= 2 {
		regime = RegimeMildBull
	} else if regimeScore >= -1 {
		regime = RegimeRangeBound
	} else if regimeScore >= -3 {
		regime = RegimeMildBear
	} else {
		regime = RegimeStrongBear
	}

	return regime, regimeScore, factors
}

// AnalyzeTrendStructure determines if price is making higher/lower highs and lows.
func AnalyzeTrendStructure(highs, lows []float64, lookback int) TrendStructure {
	if len(highs) < lookback || len(lows) < lookback {
		return TrendStructure{
			Type:     "unknown",
			Strength: "insufficient_data",
		}
	}

	// Get recent data
	recentHighs := highs[len(highs)-lookback:]
	recentLows := lows[len(lows)-lookback:]

	// Divide into three periods
	periodLength := lookback / 3

	period1Highs := recentHighs[:periodLength]
	period2Highs := recentHighs[periodLength : periodLength*2]
	period3Highs := recentHighs[periodLength*2:]

	period1Lows := recentLows[:periodLength]
	period2Lows := recentLows[periodLength : periodLength*2]
	period3Lows := recentLows[periodLength*2:]

	// Find max/min for each period
	period1High := max(period1Highs)
	period2High := max(period2Highs)
	period3High := max(period3Highs)

	period1Low := min(period1Lows)
	period2Low := min(period2Lows)
	period3Low := min(period3Lows)

	periodHighs := []float64{period1High, period2High, period3High}
	periodLows := []float64{period1Low, period2Low, period3Low}

	// Check for higher highs and higher lows
	higherHighs := period2High > period1High && period3High > period2High
	higherLows := period2Low > period1Low && period3Low > period2Low

	// Check for lower highs and lower lows
	lowerHighs := period2High < period1High && period3High < period2High
	lowerLows := period2Low < period1Low && period3Low < period2Low

	// Classify structure
	var structureType, strength string

	if higherHighs && higherLows {
		structureType = "higher_highs_lows"
		strength = "strong_uptrend"
	} else if lowerHighs && lowerLows {
		structureType = "lower_highs_lows"
		strength = "strong_downtrend"
	} else if higherHighs && !higherLows {
		structureType = "higher_highs_only"
		strength = "weak_uptrend"
	} else if lowerHighs && !lowerLows {
		structureType = "lower_highs_only"
		strength = "weak_downtrend"
	} else {
		structureType = "choppy"
		strength = "no_trend"
	}

	return TrendStructure{
		Type:        structureType,
		Strength:    strength,
		PeriodHighs: periodHighs,
		PeriodLows:  periodLows,
	}
}

// ClassifyWithHysteresis applies hysteresis to prevent rapid regime switching.
func ClassifyWithHysteresis(
	score int,
	previousRegime RegimeType,
	config RegimeConfig,
) (RegimeType, ConfidenceLevel) {

	margin := config.HysteresisMargin

	// If no previous regime, classify without hysteresis
	if previousRegime == "" {
		return classifyWithoutHysteresis(score), assignConfidence(score)
	}

	// Apply hysteresis based on previous regime
	switch previousRegime {
	case RegimeStrongBull:
		if float64(score) >= 5.0-margin {
			return RegimeStrongBull, ConfidenceHigh
		} else if score >= 2 {
			return RegimeMildBull, ConfidenceModerate
		}

	case RegimeMildBull:
		if float64(score) >= 7.0 {
			return RegimeStrongBull, ConfidenceModerate
		} else if float64(score) >= 2.0-margin {
			return RegimeMildBull, ConfidenceModerate
		} else if score >= -1 {
			return RegimeRangeBound, ConfidenceModerate
		}

	case RegimeRangeBound:
		if score >= 4 {
			return RegimeMildBull, ConfidenceModerate
		} else if float64(score) >= -3.0-margin {
			return RegimeRangeBound, ConfidenceModerate
		} else {
			return RegimeMildBear, ConfidenceModerate
		}

	case RegimeMildBear:
		if score >= -1 {
			return RegimeRangeBound, ConfidenceModerate
		} else if float64(score) >= -6.0+margin {
			return RegimeMildBear, ConfidenceModerate
		} else {
			return RegimeStrongBear, ConfidenceModerate
		}

	case RegimeStrongBear:
		if score >= -4 {
			return RegimeMildBear, ConfidenceModerate
		} else {
			return RegimeStrongBear, ConfidenceHigh
		}
	}

	// Default: classify without hysteresis
	return classifyWithoutHysteresis(score), ConfidenceLow
}

// classifyWithoutHysteresis performs simple classification without hysteresis.
func classifyWithoutHysteresis(score int) RegimeType {
	if score >= 6 {
		return RegimeStrongBull
	} else if score >= 3 {
		return RegimeMildBull
	} else if score >= -2 {
		return RegimeRangeBound
	} else if score >= -5 {
		return RegimeMildBear
	}
	return RegimeStrongBear
}

// assignConfidence assigns confidence level based on score extremity.
func assignConfidence(score int) ConfidenceLevel {
	if score >= 7 || score <= -7 {
		return ConfidenceHigh
	} else if (score >= 4 && score <= 6) || (score >= -6 && score <= -4) {
		return ConfidenceModerate
	}
	return ConfidenceLow
}

// Helper functions

func max(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	maxVal := values[0]
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

func min(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	minVal := values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}
