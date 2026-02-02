package statistics

import (
	"fmt"
)

// BreakdownByRegime groups trades by regime and calculates stats for each
func BreakdownByRegime(trades []Trade, calc *StatisticsCalculator, initialBalance float64) RegimeBreakdown {
	regimeGroups := make(map[string][]Trade)

	// Group trades by regime
	for _, trade := range trades {
		regimeName := string(trade.Regime)
		if regimeName == "" {
			regimeName = "UNKNOWN"
		}
		regimeGroups[regimeName] = append(regimeGroups[regimeName], trade)
	}

	// Calculate stats for each regime
	regimeStats := make(map[string]StatisticsResult)
	var insufficientNotes []string

	for regimeName, regimeTrades := range regimeGroups {
		if len(regimeTrades) < 5 {
			insufficientNotes = append(insufficientNotes, regimeName+" ("+fmt.Sprintf("%d trades", len(regimeTrades))+")")
			continue
		}
		stats := calc.Calculate(regimeTrades, initialBalance)
		regimeStats[regimeName] = stats
	}

	// Identify best and worst regimes
	bestRegime := ""
	worstRegime := ""
	bestExpectancy := -999.0
	worstExpectancy := 999.0

	for regimeName, stats := range regimeStats {
		if stats.Expectancy.ExpectancyRatio > bestExpectancy {
			bestExpectancy = stats.Expectancy.ExpectancyRatio
			bestRegime = regimeName
		}
		if stats.Expectancy.ExpectancyRatio < worstExpectancy {
			worstExpectancy = stats.Expectancy.ExpectancyRatio
			worstRegime = regimeName
		}
	}

	// Calculate regime contributions to P&L
	contributions := make(map[string]float64)
	var totalPnL float64
	regimePnL := make(map[string]float64)

	for _, trade := range trades {
		regimeName := string(trade.Regime)
		if regimeName == "" {
			regimeName = "UNKNOWN"
		}
		regimePnL[regimeName] += trade.PnL
		totalPnL += trade.PnL
	}

	if totalPnL != 0 {
		for regimeName, pnl := range regimePnL {
			contributions[regimeName] = (pnl / totalPnL) * 100
		}
	}

	return RegimeBreakdown{
		RegimeStats:           regimeStats,
		BestRegime:            bestRegime,
		WorstRegime:           worstRegime,
		RegimeContributions:   contributions,
		InsufficientDataNotes: insufficientNotes,
	}
}
