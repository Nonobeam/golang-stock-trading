package scoring

import "fmt"

// TradeScorer provides comprehensive trade scoring with configurable thresholds.
type TradeScorer struct {
	MinScoreToTrade int
	ScoreRiskMap    map[int]float64 // score -> risk percent
}

// NewTradeScorer creates a new TradeScorer with default configuration.
func NewTradeScorer() *TradeScorer {
	return &TradeScorer{
		MinScoreToTrade: 7,
		ScoreRiskMap: map[int]float64{
			7:  1.0,
			8:  1.0,
			9:  1.5,
			10: 1.5,
			11: 2.0,
			12: 2.0,
			13: 2.0,
		},
	}
}

// Score evaluates a trade setup and returns comprehensive scoring result.
func (ts *TradeScorer) Score(setup TradeSetup) ScoreResult {
	result := ScoreResult{
		TotalScore:  0,
		MaxScore:    13,
		ShouldTrade: false,
		RiskPercent: 0,
	}

	// Step 1: Check liquidity filters (must pass)
	liquidityResult := CheckLiquidity(
		setup.AvgDailyVolume,
		setup.AvgDailyTurnover,
		setup.ZeroVolumeDays,
	)
	result.Liquidity = liquidityResult

	if !liquidityResult.Passes {
		result.Recommendation = "REJECT - Failed liquidity filters"
		result.Summary = ScoreSummary{
			Strengths:      []string{},
			Weaknesses:     liquidityResult.Issues,
			OverallQuality: "Poor",
		}
		return result
	}

	// Step 2: Score each component
	trendScore := ScoreTrendAlignment(
		setup.CurrentPrice,
		setup.EMA20,
		setup.EMA50,
		setup.WeeklyPrice,
		setup.WeeklySMA200,
		setup.WeeklyStructure,
	)
	result.ComponentScores.TrendAlignment = trendScore

	setupScore := ScoreSetupQuality(
		setup.CurrentPrice,
		setup.SupportLevel,
		setup.SupportType,
		setup.HasConsolidation,
		setup.ConsolidationBars,
		setup.VolumeConfirms,
	)
	result.ComponentScores.SetupQuality = setupScore

	momentumScore := ScoreMomentum(
		setup.RSI,
		setup.MACD,
		setup.MACDSignal,
		setup.MACDHistogram,
		setup.PreviousHistogram,
		"up", // Default to uptrend
	)
	result.ComponentScores.Momentum = momentumScore

	rrScore := ScoreRiskReward(
		setup.EntryPrice,
		setup.StopLoss,
		setup.Target,
		setup.ATR,
	)
	result.ComponentScores.RiskReward = rrScore

	contextScore := ScoreContext(
		setup.VNIndexPrice,
		setup.VNIndexMA50,
		setup.SectorRS,
		setup.NewsSentiment,
	)
	result.ComponentScores.Context = contextScore

	// Step 3: Calculate total score
	result.TotalScore = trendScore.Score +
		setupScore.Score +
		momentumScore.Score +
		rrScore.Score +
		contextScore.Score

	// Ensure score doesn't go below 0
	if result.TotalScore < 0 {
		result.TotalScore = 0
	}

	// Step 4: Determine recommendation
	result.ShouldTrade = result.TotalScore >= ts.MinScoreToTrade

	if !result.ShouldTrade {
		result.Recommendation = fmt.Sprintf("REJECT - Score %d below minimum %d",
			result.TotalScore, ts.MinScoreToTrade)
		result.RiskPercent = 0
	} else {
		result.RiskPercent = ts.getRiskPercent(result.TotalScore)
		if result.TotalScore >= 11 {
			result.Recommendation = fmt.Sprintf("STRONG BUY - Score %d/13 (Risk %.1f%%)",
				result.TotalScore, result.RiskPercent)
		} else if result.TotalScore >= 9 {
			result.Recommendation = fmt.Sprintf("BUY - Score %d/13 (Risk %.1f%%)",
				result.TotalScore, result.RiskPercent)
		} else {
			result.Recommendation = fmt.Sprintf("CAUTIOUS BUY - Score %d/13 (Risk %.1f%%)",
				result.TotalScore, result.RiskPercent)
		}
	}

	// Step 5: Generate summary
	result.Summary = ts.generateSummary(
		result.TotalScore,
		trendScore,
		setupScore,
		momentumScore,
		rrScore,
		contextScore,
	)

	return result
}

// getRiskPercent returns risk percentage based on score.
func (ts *TradeScorer) getRiskPercent(score int) float64 {
	if riskPct, ok := ts.ScoreRiskMap[score]; ok {
		return riskPct
	}
	if score < 7 {
		return 0
	}
	return 1.0 // Default
}

// generateSummary creates a summary with strengths and weaknesses.
func (ts *TradeScorer) generateSummary(
	totalScore int,
	trend, setup, momentum, rr, context ComponentScore,
) ScoreSummary {
	summary := ScoreSummary{
		Strengths:  []string{},
		Weaknesses: []string{},
	}

	// Analyze each component
	if trend.Score == 3 {
		summary.Strengths = append(summary.Strengths, "Perfect trend alignment")
	} else if trend.Score <= 1 {
		summary.Weaknesses = append(summary.Weaknesses, "Weak trend alignment")
	}

	if setup.Score == 3 {
		summary.Strengths = append(summary.Strengths, "Excellent setup quality")
	} else if setup.Score <= 1 {
		summary.Weaknesses = append(summary.Weaknesses, "Poor setup structure")
	}

	if momentum.Score == 2 {
		summary.Strengths = append(summary.Strengths, "Strong momentum")
	} else if momentum.Score == 0 {
		summary.Weaknesses = append(summary.Weaknesses, "Weak or negative momentum")
	}

	if rr.Score == 2 {
		summary.Strengths = append(summary.Strengths, "Favorable risk/reward")
	} else if rr.Score == 0 {
		summary.Weaknesses = append(summary.Weaknesses, "Poor risk/reward profile")
	}

	if context.Score >= 2 {
		summary.Strengths = append(summary.Strengths, "Supportive market context")
	} else if context.Score < 0 {
		summary.Weaknesses = append(summary.Weaknesses, "Negative market context")
	}

	// Assess overall quality
	summary.OverallQuality = ts.assessQuality(totalScore)

	return summary
}

// assessQuality returns quality label based on score.
func (ts *TradeScorer) assessQuality(score int) string {
	switch {
	case score >= 11:
		return "Excellent"
	case score >= 9:
		return "Good"
	case score >= 7:
		return "Acceptable"
	default:
		return "Poor"
	}
}
