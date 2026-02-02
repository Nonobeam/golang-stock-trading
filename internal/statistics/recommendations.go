package statistics

import (
	"fmt"
	"sort"
)

// RecommendationEngine generates actionable recommendations
type RecommendationEngine struct{}

// NewRecommendationEngine creates a new recommendation engine
func NewRecommendationEngine() *RecommendationEngine {
	return &RecommendationEngine{}
}

// Generate creates prioritized recommendations based on comprehensive report
func (r *RecommendationEngine) Generate(report *ComprehensiveReport) []Recommendation {
	var recommendations []Recommendation

	// Check emergency conditions
	if rec := r.checkEmergencyStop(report); rec != nil {
		recommendations = append(recommendations, *rec)
	}

	// Check size reduction
	if rec := r.checkReduceSize(report); rec != nil {
		recommendations = append(recommendations, *rec)
	}

	// Check regime-based trading
	if rec := r.checkRegimeAvoidance(report); rec != nil {
		recommendations = append(recommendations, *rec)
	}

	// Check score filter adjustment
	if rec := r.checkScoreFilter(report); rec != nil {
		recommendations = append(recommendations, *rec)
	}

	// Check signal type performance
	if rec := r.checkSignalTypeFilter(report); rec != nil {
		recommendations = append(recommendations, *rec)
	}

	// Check time stops
	if rec := r.checkTimeStops(report); rec != nil {
		recommendations = append(recommendations, *rec)
	}

	// If system is good, recommend continue
	if len(recommendations) == 0 && report.Health.Score >= 60 {
		recommendations = append(recommendations, Recommendation{
			Priority: 5,
			Category: "CONTINUE",
			Action:   "Continue trading normally",
			Reason:   "System performing well within acceptable parameters",
			Impact:   "Maintain current approach",
		})
	}

	// Sort by priority and return top 5
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Priority < recommendations[j].Priority
	})

	if len(recommendations) > 5 {
		return recommendations[:5]
	}
	return recommendations
}

// checkEmergencyStop checks for system failure conditions
func (r *RecommendationEngine) checkEmergencyStop(report *ComprehensiveReport) *Recommendation {
	// Negative expectancy
	if report.BaseStats.Expectancy.ExpectancyRatio < 0 {
		return &Recommendation{
			Priority: 1,
			Category: "STOP",
			Action:   "STOP TRADING - System losing money",
			Reason:   "Negative expectancy: system loses on average",
			Impact:   "Prevent further capital loss",
		}
	}

	// Excessive drawdown
	if report.BaseStats.Drawdown.MaxDrawdownPercent > 0.25 {
		return &Recommendation{
			Priority: 1,
			Category: "STOP",
			Action:   "URGENT: STOP TRADING - Excessive drawdown",
			Reason:   "Max drawdown exceeds 25% - critical risk level",
			Impact:   "Protect remaining capital",
		}
	}

	return nil
}

// checkReduceSize checks if position sizes should be reduced
func (r *RecommendationEngine) checkReduceSize(report *ComprehensiveReport) *Recommendation {
	if report.Health.Score < 60 && report.Health.Score >= 40 {
		return &Recommendation{
			Priority: 2,
			Category: "REDUCE_SIZE",
			Action:   "Reduce position sizes to " + fmt.Sprintf("%.0f%%", report.Health.SizeMultiplier*100),
			Reason:   "System health score: " + report.Health.Rating,
			Impact:   "Lower risk exposure while system stabilizes",
		}
	}

	if report.BaseStats.Drawdown.MaxDrawdownPercent > 0.20 {
		return &Recommendation{
			Priority: 2,
			Category: "REDUCE_SIZE",
			Action:   "Reduce all position sizes by 50%",
			Reason:   "Max drawdown exceeds 20% acceptable limit",
			Impact:   "Reduce risk of further drawdown",
		}
	}

	return nil
}

// checkRegimeAvoidance checks if certain regimes should be avoided
func (r *RecommendationEngine) checkRegimeAvoidance(report *ComprehensiveReport) *Recommendation {
	if report.RegimeBreakdown.BestRegime == "" || report.RegimeBreakdown.WorstRegime == "" {
		return nil
	}

	bestStats := report.RegimeBreakdown.RegimeStats[report.RegimeBreakdown.BestRegime]
	worstStats := report.RegimeBreakdown.RegimeStats[report.RegimeBreakdown.WorstRegime]

	// If worst regime has negative expectancy and best is significantly better
	if worstStats.Expectancy.ExpectancyRatio < 0 && bestStats.Expectancy.ExpectancyRatio > 0.3 {
		return &Recommendation{
			Priority: 3,
			Category: "REGIME",
			Action:   "Avoid trading in " + report.RegimeBreakdown.WorstRegime + " market",
			Reason: fmt.Sprintf("%s: %.2fR vs %s: %.2fR",
				report.RegimeBreakdown.WorstRegime, worstStats.Expectancy.ExpectancyRatio,
				report.RegimeBreakdown.BestRegime, bestStats.Expectancy.ExpectancyRatio),
			Impact: "Focus trading capital on favorable market conditions",
		}
	}

	return nil
}

// checkScoreFilter checks if minimum score should be raised
func (r *RecommendationEngine) checkScoreFilter(report *ComprehensiveReport) *Recommendation {
	scoreRanges := report.BaseStats.Distribution.ByScoreRange
	if len(scoreRanges) < 2 {
		return nil
	}

	// Check if low scores are losing money
	for scoreRange, metrics := range scoreRanges {
		if scoreRange == "7-8" || scoreRange == "7-9" {
			if metrics.ExpectancyRatio < 0 {
				return &Recommendation{
					Priority: 3,
					Category: "FILTER",
					Action:   "Raise minimum score threshold to 9",
					Reason:   fmt.Sprintf("Score %s has negative expectancy: %.2fR", scoreRange, metrics.ExpectancyRatio),
					Impact:   "Filter out low-quality setups",
				}
			}
		}
	}

	return nil
}

// checkSignalTypeFilter checks signal type performance
func (r *RecommendationEngine) checkSignalTypeFilter(report *ComprehensiveReport) *Recommendation {
	signalTypes := report.BaseStats.Distribution.BySignalType
	if len(signalTypes) < 2 {
		return nil
	}

	// Find worst performing signal type
	var worstType string
	worstExp := 999.0
	for signalType, metrics := range signalTypes {
		if metrics.ExpectancyRatio < worstExp {
			worstExp = metrics.ExpectancyRatio
			worstType = signalType
		}
	}

	if worstExp < 0 {
		return &Recommendation{
			Priority: 4,
			Category: "FILTER",
			Action:   "Avoid " + worstType + " signal entries",
			Reason:   fmt.Sprintf("%s signals have negative expectancy: %.2fR", worstType, worstExp),
			Impact:   "Focus on higher-probability signal types",
		}
	}

	return nil
}

// checkTimeStops checks if time-based stops needed
func (r *RecommendationEngine) checkTimeStops(report *ComprehensiveReport) *Recommendation {
	timeMetrics := report.BaseStats.TimeMetrics.HoldingPeriods

	// Check if time decay is present
	if timeMetrics.TimeDecay.DecayPresent {
		return &Recommendation{
			Priority: 4,
			Category: "TIMING",
			Action:   timeMetrics.TimeDecay.Recommendation,
			Reason:   "Performance degrades with longer holding periods",
			Impact:   "Exit stagnant positions earlier to preserve capital",
		}
	}

	return nil
}
