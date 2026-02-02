package statistics

import (
	"sort"
)

// ComprehensiveCalculator orchestrates all comprehensive statistics
type ComprehensiveCalculator struct {
	vnConfig   VNConfig
	baseCalc   *StatisticsCalculator
	vnAdj      *VNAdjustments
	healthCalc *HealthScoreCalculator
	recEngine  *RecommendationEngine
}

// NewVNComprehensiveCalculator creates a new comprehensive calculator with VN defaults
func NewVNComprehensiveCalculator() *ComprehensiveCalculator {
	vnConfig := DefaultVNConfig()

	return &ComprehensiveCalculator{
		vnConfig:   vnConfig,
		baseCalc:   NewDefaultStatisticsCalculator(),
		vnAdj:      NewVNAdjustments(vnConfig),
		healthCalc: NewHealthScoreCalculator(vnConfig),
		recEngine:  NewRecommendationEngine(),
	}
}

// Generate creates a comprehensive statistics report
func (c *ComprehensiveCalculator) Generate(trades []Trade, initialBalance float64) (*ComprehensiveReport, error) {
	// Validate input
	if len(trades) == 0 {
		return &ComprehensiveReport{
			Health: SystemHealth{
				Score:       0,
				Rating:      "NO_DATA",
				ShouldTrade: false,
			},
			SampleSizeStatus: "INSUFFICIENT",
			TotalTrades:      0,
			InitialBalance:   initialBalance,
			FinalBalance:     initialBalance,
			Recommendations: []Recommendation{
				{
					Priority: 1,
					Category: "STOP",
					Action:   "No trades to analyze",
					Reason:   "Start trading to generate statistics",
					Impact:   "N/A",
				},
			},
		}, nil
	}

	// Sort trades by exit time
	sortedTrades := make([]Trade, len(trades))
	copy(sortedTrades, trades)
	sort.Slice(sortedTrades, func(i, j int) bool {
		return sortedTrades[i].ExitTime.Before(sortedTrades[j].ExitTime)
	})

	// Step 1: Calculate base statistics
	baseStats := c.baseCalc.Calculate(sortedTrades, initialBalance)

	// Step 2: Calculate VN-specific metrics
	vnMetrics := c.vnAdj.CalculateVNMetrics(sortedTrades, baseStats.Expectancy.ExpectancyRatio)

	// Step 3: Regime breakdown (if trades have regime data)
	regimeBreakdown := BreakdownByRegime(sortedTrades, c.baseCalc, initialBalance)

	// Step 4: Calculate health score
	health := c.healthCalc.Calculate(baseStats, len(sortedTrades))

	// Step 5: Determine sample size status
	sampleStatus, sampleWarning := c.getSampleSizeStatus(len(sortedTrades))

	// Build initial report
	report := &ComprehensiveReport{
		BaseStats:         baseStats,
		VNMetrics:         vnMetrics,
		RegimeBreakdown:   regimeBreakdown,
		Health:            health,
		SampleSizeStatus:  sampleStatus,
		SampleSizeWarning: sampleWarning,
		TotalTrades:       len(sortedTrades),
		InitialBalance:    initialBalance,
		FinalBalance:      baseStats.FinalBalance,
	}

	// Step 6: Generate recommendations
	recommendations := c.recEngine.Generate(report)
	report.Recommendations = recommendations

	return report, nil
}

// getSampleSizeStatus returns tiered sample size assessment
func (c *ComprehensiveCalculator) getSampleSizeStatus(sampleSize int) (string, string) {
	if sampleSize < 10 {
		return "INSUFFICIENT", "Insufficient data (" + string(rune(sampleSize+'0')) + " trades) - do NOT make decisions"
	} else if sampleSize < 30 {
		return "WARNING", "WARNING - Small sample (" + string(rune(sampleSize+'0')) + " trades), statistics unreliable"
	} else if sampleSize < 50 {
		return "ADEQUATE", "Basic analysis possible with " + string(rune(sampleSize+'0')) + " trades"
	} else if sampleSize < 100 {
		return "GOOD", "Reliable sample size (" + string(rune(sampleSize+'0')) + " trades)"
	}
	return "EXCELLENT", "Statistically significant sample (" + string(rune(sampleSize+'0')) + "+ trades)"
}

// GetDashboard returns a human-readable dashboard (simplified version)
func (c *ComprehensiveCalculator) GetDashboard(report *ComprehensiveReport) string {
	// Simplified dashboard - full version would be in formatter.go
	emoji := "🟢"
	if report.Health.Rating == "FAIR" {
		emoji = "🟡"
	} else if report.Health.Rating == "POOR" || report.Health.Rating == "FAILING" {
		emoji = "🔴"
	}

	dashboard := "════════════════════════════════════════════\n"
	dashboard += "      TRADING SYSTEM PERFORMANCE REPORT\n"
	dashboard += "════════════════════════════════════════════\n\n"
	dashboard += emoji + " SYSTEM STATUS: " + report.Health.Rating + " (Score: " + string(rune(report.Health.Score+'0')) + "/100)\n"
	dashboard += "📊 " + string(rune(report.TotalTrades+'0')) + " trades analyzed\n"
	dashboard += "⚠️  " + report.SampleSizeWarning + "\n\n"

	dashboard += "CORE METRICS:\n"
	dashboard += "─────────────────────────────────────────\n"
	// Add metrics here (simplified for now)

	dashboard += "\n🎯 TOP RECOMMENDATIONS:\n"
	dashboard += "─────────────────────────────────────────\n"
	for i, rec := range report.Recommendations {
		if i >= 3 {
			break
		}
		dashboard += string(rune(i+1+'0')) + ". " + rec.Action + "\n"
		dashboard += "   Reason: " + rec.Reason + "\n\n"
	}

	return dashboard
}
