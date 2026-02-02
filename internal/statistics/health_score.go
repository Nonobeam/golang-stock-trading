package statistics

// HealthScoreCalculator calculates system health score (0-100)
type HealthScoreCalculator struct {
	vnConfig VNConfig
}

// NewHealthScoreCalculator creates a new health score calculator
func NewHealthScoreCalculator(config VNConfig) *HealthScoreCalculator {
	return &HealthScoreCalculator{vnConfig: config}
}

// Calculate computes overall health score
func (h *HealthScoreCalculator) Calculate(stats StatisticsResult, sampleSize int) SystemHealth {
	profitScore := h.scoreProfitability(stats.Expectancy.ExpectancyRatio)
	riskScore := h.scoreRiskManagement(stats.Drawdown.MaxDrawdownPercent)
	consistencyScore := h.scoreConsistency(stats.RiskAdjusted.SharpeRatio)

	totalScore := profitScore + riskScore + consistencyScore
	rating := h.getRating(totalScore)
	shouldTrade := totalScore >= 40 // Minimum viable threshold
	sizeMultiplier := h.getSizeMultiplier(totalScore)

	// Add confidence flag for small samples
	confidence := ""
	if sampleSize < 10 {
		confidence = "UNRELIABLE"
	} else if sampleSize < 30 {
		confidence = "LOW_CONFIDENCE"
	}

	return SystemHealth{
		Score:          totalScore,
		Rating:         rating,
		ShouldTrade:    shouldTrade,
		SizeMultiplier: sizeMultiplier,
		Confidence:     confidence,
	}
}

// scoreProfitability scores expectancy (max 40 points)
func (h *HealthScoreCalculator) scoreProfitability(expectancy float64) int {
	// VN-adjusted thresholds
	if expectancy >= 1.0 {
		return 40
	} else if expectancy >= 0.5 {
		return 30
	} else if expectancy >= 0.3 {
		return 20
	} else if expectancy >= h.vnConfig.AcceptableExpectancy { // 0.25R for VN
		return 10
	}
	return 0
}

// scoreRiskManagement scores drawdown (max 30 points)
func (h *HealthScoreCalculator) scoreRiskManagement(maxDD float64) int {
	// VN-adjusted thresholds (more lenient)
	if maxDD < 0.10 {
		return 30
	} else if maxDD < 0.15 {
		return 25
	} else if maxDD < 0.20 { // VN acceptable limit
		return 20
	} else if maxDD < 0.25 {
		return 10
	}
	return 0
}

// scoreConsistency scores Sharpe ratio (max 30 points)
func (h *HealthScoreCalculator) scoreConsistency(sharpe float64) int {
	// VN-adjusted thresholds
	if sharpe >= 2.0 {
		return 30
	} else if sharpe >= 1.5 {
		return 25
	} else if sharpe >= 1.0 {
		return 20
	} else if sharpe >= h.vnConfig.AcceptableSharpe { // 0.7 for VN
		return 10
	}
	return 0
}

// getRating converts score to rating
func (h *HealthScoreCalculator) getRating(score int) string {
	if score >= 80 {
		return "EXCELLENT"
	} else if score >= 60 {
		return "GOOD"
	} else if score >= 40 {
		return "FAIR"
	} else if score >= 20 {
		return "POOR"
	}
	return "FAILING"
}

// getSizeMultiplier returns position size adjustment
func (h *HealthScoreCalculator) getSizeMultiplier(score int) float64 {
	if score >= 60 {
		return 1.0 // Normal size
	} else if score >= 40 {
		return 0.75 // Reduce to 75%
	} else if score >= 20 {
		return 0.5 // Reduce to 50%
	}
	return 0.0 // Stop trading
}
