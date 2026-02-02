package regime

import "fmt"

// AdjustScoreForRegime adjusts trade score based on regime alignment.
func AdjustScoreForRegime(
	baseScore int,
	stockRegime *RegimeResult,
	marketRegime *RegimeResult,
) (adjustedScore int, alignmentAdjustment int, alignmentNote string, shouldTrade bool) {

	alignmentAdjustment = 0

	// Perfect alignment: Both strong bull
	if stockRegime.Regime == RegimeStrongBull && marketRegime.Regime == RegimeStrongBull {
		alignmentAdjustment = 2
		alignmentNote = "Perfect alignment: Stock and market both strong"

		// Good alignment: Both bullish
	} else if isBullish(stockRegime.Regime) && isBullish(marketRegime.Regime) {
		alignmentAdjustment = 1
		alignmentNote = "Good alignment: Both bullish"

		// Neutral: Mixed signals
	} else if (stockRegime.Regime == RegimeMildBull || stockRegime.Regime == RegimeRangeBound) &&
		(marketRegime.Regime == RegimeMildBull || marketRegime.Regime == RegimeRangeBound) {
		alignmentAdjustment = 0
		alignmentNote = "Neutral alignment"

		// Warning: Stock bullish but market bearish
	} else if isBullish(stockRegime.Regime) && isBearish(marketRegime.Regime) {
		alignmentAdjustment = -2
		alignmentNote = "Warning: Stock bullish but market bearish"

		// Poor: Any bear market involvement
	} else {
		alignmentAdjustment = -1
		alignmentNote = "Poor alignment: Bearish conditions"
	}

	adjustedScore = baseScore + alignmentAdjustment

	// Determine minimum score required by market regime
	minRequired := GetMinScoreByRegime(marketRegime.Regime)

	shouldTrade = adjustedScore >= minRequired

	return adjustedScore, alignmentAdjustment, alignmentNote, shouldTrade
}

// GetMinScoreByRegime returns minimum trade score required for each regime.
func GetMinScoreByRegime(regime RegimeType) int {
	minScores := map[RegimeType]int{
		RegimeStrongBull: 7,
		RegimeMildBull:   7,
		RegimeRangeBound: 8,
		RegimeMildBear:   9,
		RegimeStrongBear: 10,
	}

	if score, ok := minScores[regime]; ok {
		return score
	}
	return 8 // Default
}

// ShouldTradeInRegime determines if a trade should be taken given score and regime.
func ShouldTradeInRegime(score int, regime RegimeType) (bool, string) {
	minRequired := GetMinScoreByRegime(regime)

	if score >= minRequired {
		if regime == RegimeStrongBull && score >= 10 {
			return true, "STRONG BUY - Excellent setup in strong bull market"
		} else if (regime == RegimeStrongBull || regime == RegimeMildBull) && score >= 8 {
			return true, "BUY - Good setup in bullish market"
		} else if regime == RegimeRangeBound {
			return true, "CAUTIOUS BUY - Reduce size, tight stops in ranging market"
		} else {
			return true, "SELECTIVE BUY - Only if exceptional conviction"
		}
	}

	return false, fmt.Sprintf("SKIP - Score %d insufficient for %s market (need %d)", score, regime, minRequired)
}

// Helper functions

func isBullish(regime RegimeType) bool {
	return regime == RegimeStrongBull || regime == RegimeMildBull
}

func isBearish(regime RegimeType) bool {
	return regime == RegimeMildBear || regime == RegimeStrongBear
}
