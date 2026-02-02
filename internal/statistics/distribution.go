package statistics

import "sort"

// AnalyzeBySignalType groups trades by signal type and calculates metrics
func AnalyzeBySignalType(trades []Trade) map[string]ExpectancyMetrics {
	groups := make(map[string][]Trade)

	for _, trade := range trades {
		signalType := trade.SignalType
		if signalType == "" {
			signalType = "Unknown"
		}
		groups[signalType] = append(groups[signalType], trade)
	}

	results := make(map[string]ExpectancyMetrics)
	for signalType, groupTrades := range groups {
		results[signalType] = CalculateExpectancy(groupTrades, false)
	}

	return results
}

// AnalyzeByRegime groups trades by market regime and calculates metrics
func AnalyzeByRegime(trades []Trade) map[string]ExpectancyMetrics {
	groups := make(map[string][]Trade)

	for _, trade := range trades {
		regime := string(trade.Regime)
		if regime == "" {
			regime = "Unknown"
		}
		groups[regime] = append(groups[regime], trade)
	}

	results := make(map[string]ExpectancyMetrics)
	for regime, groupTrades := range groups {
		results[regime] = CalculateExpectancy(groupTrades, false)
	}

	return results
}

// AnalyzeByScoreRange groups trades by score ranges and calculates metrics
func AnalyzeByScoreRange(trades []Trade) map[string]ExpectancyMetrics {
	groups := make(map[string][]Trade)

	for _, trade := range trades {
		var scoreRange string
		switch {
		case trade.Score >= 13:
			scoreRange = "13+"
		case trade.Score >= 11:
			scoreRange = "11-12"
		case trade.Score >= 9:
			scoreRange = "9-10"
		case trade.Score >= 7:
			scoreRange = "7-8"
		default:
			scoreRange = "<7"
		}

		groups[scoreRange] = append(groups[scoreRange], trade)
	}

	results := make(map[string]ExpectancyMetrics)
	for scoreRange, groupTrades := range groups {
		results[scoreRange] = CalculateExpectancy(groupTrades, false)
	}

	return results
}

// FindBestWorstTrades returns top N best and worst trades by P&L percentage
func FindBestWorstTrades(trades []Trade, n int) (best, worst []Trade) {
	if len(trades) == 0 {
		return nil, nil
	}

	sortedTrades := make([]Trade, len(trades))
	copy(sortedTrades, trades)

	sort.Slice(sortedTrades, func(i, j int) bool {
		return sortedTrades[i].PnLPercent > sortedTrades[j].PnLPercent
	})

	bestCount := n
	if bestCount > len(sortedTrades) {
		bestCount = len(sortedTrades)
	}
	best = sortedTrades[:bestCount]

	worstCount := n
	if worstCount > len(sortedTrades) {
		worstCount = len(sortedTrades)
	}
	worst = sortedTrades[len(sortedTrades)-worstCount:]

	sort.Slice(worst, func(i, j int) bool {
		return worst[i].PnLPercent < worst[j].PnLPercent
	})

	return best, worst
}

// AnalyzeDistribution performs comprehensive distribution analysis
func AnalyzeDistribution(trades []Trade, topN int) DistributionMetrics {
	best, worst := FindBestWorstTrades(trades, topN)

	return DistributionMetrics{
		BySignalType: AnalyzeBySignalType(trades),
		ByRegime:     AnalyzeByRegime(trades),
		ByScoreRange: AnalyzeByScoreRange(trades),
		BestTrades:   best,
		WorstTrades:  worst,
	}
}
