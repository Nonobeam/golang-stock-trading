package scoring

import (
	"fmt"
	"math"
)

// ScoreTrendAlignment scores trend alignment (0-3 points).
// +1: Price above 20 EMA
// +1: Price above 50 EMA
// +1: Weekly uptrend (price > 200 SMA + higher highs/lows)
func ScoreTrendAlignment(
	currentPrice, ema20, ema50 float64,
	weeklyPrice, weeklySMA200 float64,
	weeklyStructure string,
) ComponentScore {
	score := ComponentScore{
		Score:    0,
		MaxScore: 3,
		MinScore: 0,
		Name:     "Trend Alignment",
		Details:  []string{},
	}

	// Check 1: Price above 20 EMA
	if currentPrice > ema20 {
		score.Score++
		score.Details = append(score.Details,
			fmt.Sprintf("✓ Price (%.0f) above 20 EMA (%.0f)", currentPrice, ema20))
	} else {
		score.Details = append(score.Details,
			fmt.Sprintf("✗ Price (%.0f) below 20 EMA (%.0f)", currentPrice, ema20))
	}

	// Check 2: Price above 50 EMA
	if currentPrice > ema50 {
		score.Score++
		score.Details = append(score.Details,
			fmt.Sprintf("✓ Price (%.0f) above 50 EMA (%.0f)", currentPrice, ema50))
	} else {
		score.Details = append(score.Details,
			fmt.Sprintf("✗ Price (%.0f) below 50 EMA (%.0f)", currentPrice, ema50))
	}

	// Check 3: Weekly uptrend
	weeklyUptrend := weeklyPrice > weeklySMA200 && weeklyStructure == "higher_highs_lows"
	if weeklyUptrend {
		score.Score++
		score.Details = append(score.Details,
			"✓ Weekly uptrend: Price above 200 SMA, higher highs/lows")
	} else {
		score.Details = append(score.Details,
			"✗ Weekly not in uptrend")
	}

	return score
}

// ScoreSetupQuality scores setup quality (0-3 points).
// +1: Clear support level present (within 3%)
// +1: Consolidation pattern (≥5 bars)
// +1: Volume confirms setup
func ScoreSetupQuality(
	currentPrice, supportLevel float64,
	supportType string,
	hasConsolidation bool,
	consolidationBars int,
	volumeConfirms bool,
) ComponentScore {
	score := ComponentScore{
		Score:    0,
		MaxScore: 3,
		MinScore: 0,
		Name:     "Setup Quality",
		Details:  []string{},
	}

	// Check 1: Clear support level
	if supportType != "none" && supportType != "" {
		distancePercent := 0.0
		if currentPrice > 0 {
			distancePercent = math.Abs(currentPrice-supportLevel) / currentPrice * 100
		}

		if distancePercent <= 3.0 {
			score.Score++
			score.Details = append(score.Details,
				fmt.Sprintf("✓ Support at %.0f (%s), %.1f%% away", supportLevel, supportType, distancePercent))
		} else {
			score.Details = append(score.Details,
				fmt.Sprintf("⚠ Support at %.0f but %.1f%% away (>3%%)", supportLevel, distancePercent))
		}
	} else {
		score.Details = append(score.Details, "✗ No clear support level identified")
	}

	// Check 2: Consolidation pattern
	if hasConsolidation {
		if consolidationBars >= 5 {
			score.Score++
			score.Details = append(score.Details,
				fmt.Sprintf("✓ Consolidation pattern (%d bars)", consolidationBars))
		} else {
			score.Details = append(score.Details,
				fmt.Sprintf("⚠ Consolidation too brief (%d bars, need 5+)", consolidationBars))
		}
	} else {
		score.Details = append(score.Details, "✗ No consolidation or pullback pattern")
	}

	// Check 3: Volume confirmation
	if volumeConfirms {
		score.Score++
		score.Details = append(score.Details, "✓ Volume pattern confirms setup")
	} else {
		score.Details = append(score.Details,
			"✗ Volume does not confirm (no decrease on pullback or no increase on bounce)")
	}

	return score
}

// ScoreMomentum scores momentum indicators (0-2 points).
// +1: RSI in favorable range (40-70 for uptrend)
// +1: MACD positive OR crossing bullishly with growing histogram
func ScoreMomentum(
	rsi float64,
	macd, macdSignal, macdHistogram, previousHistogram float64,
	trendDirection string,
) ComponentScore {
	score := ComponentScore{
		Score:    0,
		MaxScore: 2,
		MinScore: 0,
		Name:     "Momentum",
		Details:  []string{},
	}

	// Check 1: RSI in favorable range
	if trendDirection == "up" || trendDirection == "" {
		// For longs: RSI 40-70 is favorable
		if rsi >= 40 && rsi <= 70 {
			score.Score++
			if rsi <= 55 {
				score.Details = append(score.Details,
					fmt.Sprintf("✓ RSI at %.1f (healthy pullback range 40-55)", rsi))
			} else {
				score.Details = append(score.Details,
					fmt.Sprintf("✓ RSI at %.1f (strong but not overextended)", rsi))
			}
		} else if rsi > 70 {
			score.Details = append(score.Details,
				fmt.Sprintf("⚠ RSI at %.1f (>70, overextended risk)", rsi))
		} else if rsi >= 30 && rsi < 40 {
			score.Details = append(score.Details,
				fmt.Sprintf("⚠ RSI at %.1f (below 40, trend weakening)", rsi))
		} else {
			score.Details = append(score.Details,
				fmt.Sprintf("✗ RSI at %.1f (<30, too weak for uptrend trade)", rsi))
		}
	} else { // downtrend
		if rsi >= 30 && rsi <= 60 {
			score.Score++
			score.Details = append(score.Details,
				fmt.Sprintf("✓ RSI at %.1f (favorable for downtrend)", rsi))
		} else {
			score.Details = append(score.Details,
				fmt.Sprintf("✗ RSI at %.1f (unfavorable range for short)", rsi))
		}
	}

	// Check 2: MACD bullish
	macdAboveSignal := macd > macdSignal
	histogramGrowing := macdHistogram > previousHistogram

	if trendDirection == "up" || trendDirection == "" {
		if macd > 0 {
			score.Score++
			score.Details = append(score.Details,
				fmt.Sprintf("✓ MACD positive (%.2f)", macd))
		} else if macdAboveSignal && histogramGrowing {
			score.Score++
			score.Details = append(score.Details, "✓ MACD crossing up, histogram growing")
		} else if !macdAboveSignal {
			score.Details = append(score.Details, "✗ MACD below signal line")
		} else {
			score.Details = append(score.Details, "✗ MACD histogram not growing")
		}
	} else { // downtrend
		if macd < 0 {
			score.Score++
			score.Details = append(score.Details, "✓ MACD negative (bearish)")
		} else if !macdAboveSignal && !histogramGrowing {
			score.Score++
			score.Details = append(score.Details, "✓ MACD crossing down")
		} else {
			score.Details = append(score.Details, "✗ MACD not bearish")
		}
	}

	return score
}

// ScoreRiskReward scores risk/reward profile (0-2 points).
// +1: R:R ratio ≥ 2:1
// +1: Stop distance ≤ 7% or ≤ 2×ATR
func ScoreRiskReward(entryPrice, stopLoss, targetPrice, atr float64) ComponentScore {
	score := ComponentScore{
		Score:    0,
		MaxScore: 2,
		MinScore: 0,
		Name:     "Risk/Reward",
		Details:  []string{},
	}

	// Calculate risk and reward
	risk := math.Abs(entryPrice - stopLoss)
	reward := math.Abs(targetPrice - entryPrice)

	riskPercent := 0.0
	if entryPrice > 0 {
		riskPercent = (risk / entryPrice) * 100
	}

	// Calculate R:R ratio
	rrRatio := 0.0
	if risk > 0 {
		rrRatio = reward / risk
	}

	// Check 1: R:R ratio ≥ 2:1
	if rrRatio >= 2.0 {
		score.Score++
		if rrRatio >= 3.0 {
			score.Details = append(score.Details,
				fmt.Sprintf("✓ Excellent R:R ratio: %.2f:1", rrRatio))
		} else {
			score.Details = append(score.Details,
				fmt.Sprintf("✓ Good R:R ratio: %.2f:1", rrRatio))
		}
	} else if rrRatio >= 1.5 {
		score.Details = append(score.Details,
			fmt.Sprintf("⚠ Marginal R:R ratio: %.2f:1 (prefer 2:1+)", rrRatio))
	} else {
		score.Details = append(score.Details,
			fmt.Sprintf("✗ Poor R:R ratio: %.2f:1 (need 2:1 minimum)", rrRatio))
	}

	// Check 2: Stop distance reasonable
	stopAcceptable := false

	// Method 1: Percentage check (≤7%)
	if riskPercent <= 7.0 {
		stopAcceptable = true
		score.Details = append(score.Details,
			fmt.Sprintf("✓ Stop distance: %.1f%% (≤7%%)", riskPercent))
	}

	// Method 2: ATR check (≤2×ATR) if percentage check failed
	if !stopAcceptable && atr > 0 {
		atrMultiple := risk / atr
		if atrMultiple <= 2.0 {
			stopAcceptable = true
			score.Details = append(score.Details,
				fmt.Sprintf("✓ Stop distance: %.1f× ATR (≤2×)", atrMultiple))
		} else {
			score.Details = append(score.Details,
				fmt.Sprintf("⚠ Stop distance: %.1f× ATR (>2×)", atrMultiple))
		}
	}

	if !stopAcceptable && atr == 0 {
		score.Details = append(score.Details,
			fmt.Sprintf("⚠ Stop distance: %.1f%% (>7%%)", riskPercent))
	}

	if stopAcceptable {
		score.Score++
	}

	return score
}

// ScoreContext scores market context (-2 to +3 bonus points).
// +1: VN-Index above 50 MA
// +1: Sector relative strength > 1.05
// +1/-1/-2: News sentiment
func ScoreContext(
	vnIndexPrice, vnIndexMA50 float64,
	sectorRS float64,
	newsSentiment string,
) ComponentScore {
	score := ComponentScore{
		Score:    0,
		MaxScore: 3,
		MinScore: -2,
		Name:     "Context (Bonus)",
		Details:  []string{},
	}

	// Check 1: VN-Index trend
	if vnIndexPrice > vnIndexMA50 {
		score.Score++
		percentAbove := 0.0
		if vnIndexMA50 > 0 {
			percentAbove = ((vnIndexPrice / vnIndexMA50) - 1) * 100
		}
		score.Details = append(score.Details,
			fmt.Sprintf("✓ VN-Index in uptrend (+%.1f%% above 50 MA)", percentAbove))
	} else {
		percentBelow := 0.0
		if vnIndexPrice > 0 {
			percentBelow = ((vnIndexMA50 / vnIndexPrice) - 1) * 100
		}
		score.Details = append(score.Details,
			fmt.Sprintf("✗ VN-Index in downtrend (-%.1f%% below 50 MA)", percentBelow))
	}

	// Check 2: Sector relative strength
	if sectorRS > 1.05 {
		score.Score++
		outperformance := (sectorRS - 1) * 100
		score.Details = append(score.Details,
			fmt.Sprintf("✓ Sector outperforming market by %.1f%%", outperformance))
	} else if sectorRS > 0.98 {
		score.Details = append(score.Details, "○ Sector in-line with market")
	} else {
		underperformance := (1 - sectorRS) * 100
		score.Details = append(score.Details,
			fmt.Sprintf("✗ Sector underperforming by %.1f%%", underperformance))
	}

	// Check 3: News sentiment
	switch newsSentiment {
	case "positive":
		score.Score++
		score.Details = append(score.Details, "✓ Positive news catalyst")
	case "neutral", "":
		score.Details = append(score.Details, "○ No significant news")
	case "negative":
		score.Score--
		score.Details = append(score.Details, "✗ Negative news (-1 point)")
	case "very_negative":
		score.Score -= 2
		score.Details = append(score.Details, "✗✗ Very negative news (-2 points)")
	}

	return score
}
