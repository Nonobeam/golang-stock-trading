package statistics

import (
	"math"
)

// AnalyzeMAEMFE performs comprehensive MAE/MFE aggregate analysis
func AnalyzeMAEMFE(trades []Trade) MAEMFEMetrics {
	if len(trades) == 0 {
		return MAEMFEMetrics{}
	}

	// Separate winners and losers
	var winners, losers []Trade
	for _, t := range trades {
		if t.PnL > 0 {
			winners = append(winners, t)
		} else if t.PnL <= 0 {
			losers = append(losers, t)
		}
	}

	// Analyze each category
	winnerMAE := analyzeWinnerMAE(winners)
	loserMAE := analyzeLoserMAE(losers)
	winnerMFE := analyzeWinnerMFE(winners)
	loserMFE := analyzeLoserMFE(losers)

	// Generate actionable insights
	stopInsights := generateStopInsights(winnerMAE, loserMAE)
	targetInsights := generateTargetInsights(winnerMFE, loserMFE)

	return MAEMFEMetrics{
		WinnerMAE:      winnerMAE,
		LoserMAE:       loserMAE,
		WinnerMFE:      winnerMFE,
		LoserMFE:       loserMFE,
		StopInsights:   stopInsights,
		TargetInsights: targetInsights,
	}
}

// analyzeWinnerMAE analyzes how far winners went against you before recovering
func analyzeWinnerMAE(winners []Trade) WinnerMAEMetrics {
	if len(winners) == 0 {
		return WinnerMAEMetrics{
			Interpretation: "No winning trades to analyze",
		}
	}

	maes := make([]float64, 0, len(winners))
	for _, w := range winners {
		maes = append(maes, math.Abs(w.MAE_R))
	}

	avgMAE := mean(maes)
	medianMAE := median(maes)
	maxMAE := max(maes)

	// Distribution by ranges
	distribution := make(map[string]int)
	distribution["0-0.3R"] = 0
	distribution["0.3-0.5R"] = 0
	distribution["0.5-0.8R"] = 0
	distribution["0.8-1.0R"] = 0
	distribution[">1.0R"] = 0

	for _, mae := range maes {
		if mae < 0.3 {
			distribution["0-0.3R"]++
		} else if mae < 0.5 {
			distribution["0.3-0.5R"]++
		} else if mae < 0.8 {
			distribution["0.5-0.8R"]++
		} else if mae < 1.0 {
			distribution["0.8-1.0R"]++
		} else {
			distribution[">1.0R"]++
		}
	}

	return WinnerMAEMetrics{
		AvgMAE:          avgMAE,
		MedianMAE:       medianMAE,
		MaxMAE:          maxMAE,
		MAEDistribution: distribution,
		Interpretation:  interpretWinnerMAE(avgMAE, maxMAE),
	}
}

// interpretWinnerMAE provides actionable insights
func interpretWinnerMAE(avgMAE, maxMAE float64) string {
	if avgMAE < 0.3 {
		return "Excellent - winners rarely go against you significantly"
	} else if avgMAE < 0.5 {
		return "Good - normal adverse movement before winning"
	} else if avgMAE < 0.8 {
		return "Acceptable - consider wider stops to reduce false exits"
	}
	return "Poor - stops may be too tight, many winners go deep against you"
}

// analyzeLoserMAE analyzes how far losers went before stopping out
func analyzeLoserMAE(losers []Trade) LoserMAEMetrics {
	if len(losers) == 0 {
		return LoserMAEMetrics{
			Interpretation: "No losing trades to analyze",
		}
	}

	maes := make([]float64, 0, len(losers))
	finalLosses := make([]float64, 0, len(losers))

	for _, l := range losers {
		maes = append(maes, math.Abs(l.MAE_R))

		// Calculate final loss in R-multiples
		var finalR float64
		if l.InitialRisk > 0 {
			finalR = math.Abs(l.PnL / l.InitialRisk)
		} else {
			finalR = math.Abs(l.PnLPercent / 100.0)
		}
		finalLosses = append(finalLosses, finalR)
	}

	avgMAE := mean(maes)
	avgFinalLoss := mean(finalLosses)
	maxLoss := max(finalLosses)

	// Count excessive losses (> 1.2R)
	excessiveCount := 0
	for _, loss := range finalLosses {
		if loss > 1.2 {
			excessiveCount++
		}
	}
	excessivePct := 0.0
	if len(losers) > 0 {
		excessivePct = float64(excessiveCount) / float64(len(losers)) * 100
	}

	return LoserMAEMetrics{
		AvgMAE:             avgMAE,
		AvgFinalLoss:       avgFinalLoss,
		MaxLoss:            maxLoss,
		ExcessiveLossCount: excessiveCount,
		ExcessiveLossPct:   excessivePct,
		Interpretation:     interpretLoserMAE(avgFinalLoss, excessiveCount),
	}
}

// interpretLoserMAE provides stop effectiveness insights
func interpretLoserMAE(avgLoss float64, excessiveCount int) string {
	if avgLoss < 1.1 && excessiveCount == 0 {
		return "Excellent - stops are well respected"
	} else if avgLoss < 1.3 {
		return "Good - stops mostly working as intended"
	} else if avgLoss < 1.5 {
		return "Fair - some slippage on stops"
	}
	return "Poor - significant stop slippage or gaps"
}

// analyzeWinnerMFE analyzes profit capture efficiency
func analyzeWinnerMFE(winners []Trade) WinnerMFEMetrics {
	if len(winners) == 0 {
		return WinnerMFEMetrics{
			Interpretation: "No winning trades to analyze",
		}
	}

	mfes := make([]float64, 0, len(winners))
	finals := make([]float64, 0, len(winners))
	efficiencies := make([]float64, 0, len(winners))
	gaveBack50 := 0

	for _, w := range winners {
		mfe := w.MFE_R
		var finalR float64
		if w.InitialRisk > 0 {
			finalR = w.PnL / w.InitialRisk
		} else {
			finalR = w.PnLPercent / 100.0
		}

		mfes = append(mfes, mfe)
		finals = append(finals, finalR)

		if mfe > 0 {
			efficiency := (finalR / mfe) * 100
			efficiencies = append(efficiencies, efficiency)

			if efficiency < 50 {
				gaveBack50++
			}
		}
	}

	avgMFE := mean(mfes)
	avgFinal := mean(finals)
	avgEfficiency := mean(efficiencies)

	return WinnerMFEMetrics{
		AvgMFE:         avgMFE,
		AvgFinal:       avgFinal,
		AvgEfficiency:  avgEfficiency,
		GaveBack50Pct:  gaveBack50,
		Interpretation: interpretWinnerMFE(avgEfficiency, gaveBack50),
	}
}

// interpretWinnerMFE provides profit capture insights
func interpretWinnerMFE(avgEfficiency float64, gaveBack50 int) string {
	if avgEfficiency > 80 {
		return "Excellent - capturing most of the move"
	} else if avgEfficiency > 60 {
		return "Good - reasonable profit capture"
	} else if avgEfficiency > 40 {
		return "Fair - consider tighter trailing stops"
	}
	return "Poor - giving back too much profit, tighten exits"
}

// analyzeLoserMFE analyzes false signals (losers that were profitable)
func analyzeLoserMFE(losers []Trade) LoserMFEMetrics {
	if len(losers) == 0 {
		return LoserMFEMetrics{
			Interpretation: "No losing trades to analyze",
		}
	}

	wereProfitable := 0
	profitableMFEs := make([]float64, 0)

	for _, l := range losers {
		if l.MFE_R > 0 {
			wereProfitable++
			profitableMFEs = append(profitableMFEs, l.MFE_R)
		}
	}

	profitablePct := float64(wereProfitable) / float64(len(losers)) * 100
	avgMFE := mean(profitableMFEs)

	return LoserMFEMetrics{
		WereProfitable:     wereProfitable,
		ProfitablePct:      profitablePct,
		AvgMFEOfProfitable: avgMFE,
		Interpretation:     interpretLoserMFE(profitablePct, wereProfitable, len(losers)),
	}
}

// interpretLoserMFE provides false signal insights
func interpretLoserMFE(profitablePct float64, profitable, total int) string {
	if profitablePct < 10 {
		return "Good - losers never showed profit, stops appropriate"
	} else if profitablePct < 30 {
		return "Acceptable - some losers briefly profitable"
	}
	return "Poor - many losers were profitable, consider taking partial profits earlier"
}

// generateStopInsights creates actionable stop placement recommendations
func generateStopInsights(winnerMAE WinnerMAEMetrics, loserMAE LoserMAEMetrics) []string {
	insights := make([]string, 0)

	// Check if winners go too far against
	if winnerMAE.AvgMAE > 0.8 {
		insights = append(insights, "Winners frequently go 0.8R+ against you - consider widening stops to 1.5-2× current distance")
	} else if winnerMAE.AvgMAE < 0.3 {
		insights = append(insights, "Winners rarely go significantly against you - current stops may be wider than necessary")
	}

	// Check for excessive losses
	if loserMAE.ExcessiveLossCount > 0 {
		insights = append(insights, "Some losses exceeded 1.2R - review stop execution, consider limit orders or tighter stops")
	}

	if len(insights) == 0 {
		insights = append(insights, "Stop placement appears well-optimized")
	}

	return insights
}

// generateTargetInsights creates actionable target/exit recommendations
func generateTargetInsights(winnerMFE WinnerMFEMetrics, loserMFE LoserMFEMetrics) []string {
	insights := make([]string, 0)

	// Check profit capture efficiency
	if winnerMFE.AvgEfficiency < 50 {
		insights = append(insights, "Only capturing <50% of maximum move - implement tighter trailing stops or earlier targets")
	}

	if winnerMFE.GaveBack50Pct > 0 {
		insights = append(insights, "Many winners give back >50% of profits - take partial profits at 2R, trail remainder tighter")
	}

	// Check for false signals
	if loserMFE.ProfitablePct > 30 {
		insights = append(insights, "Many losers were briefly profitable - consider breakeven stops earlier or partial profit taking")
	}

	if len(insights) == 0 {
		insights = append(insights, "Exit strategy appears well-optimized")
	}

	return insights
}

// Helper functions

func max(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	maxVal := data[0]
	for _, v := range data[1:] {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}
