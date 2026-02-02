package statistics

import (
	"math"
	"sort"
)

// AnalyzeTimeMetrics performs comprehensive time-based performance analysis
func AnalyzeTimeMetrics(trades []Trade) TimeMetrics {
	if len(trades) == 0 {
		return TimeMetrics{}
	}

	holdingPeriods := analyzeHoldingPeriods(trades)
	streaks := calculateStreaks(trades)

	return TimeMetrics{
		HoldingPeriods: holdingPeriods,
		Streaks:        streaks,
	}
}

// analyzeHoldingPeriods analyzes trade duration and identifies optimal periods
func analyzeHoldingPeriods(trades []Trade) HoldingPeriodMetrics {
	if len(trades) == 0 {
		return HoldingPeriodMetrics{}
	}

	// Calculate holding days for all trades
	allDays := make([]int, 0, len(trades))
	var winnerDays, loserDays []int

	for _, t := range trades {
		days := int(t.ExitTime.Sub(t.EntryTime).Hours() / 24)
		allDays = append(allDays, days)

		if t.PnL > 0 {
			winnerDays = append(winnerDays, days)
		} else {
			loserDays = append(loserDays, days)
		}
	}

	// Calculate statistics
	allStats := calculateHoldingStats(allDays)
	winnerStats := calculateHoldingStats(winnerDays)
	loserStats := calculateHoldingStats(loserDays)

	// Find optimal holding period
	optimalPeriod := findOptimalHoldingPeriod(trades)

	// Analyze time decay
	timeDecay := analyzeTimeDecay(trades)

	// Interpretation
	interpretation := interpretHoldingPeriods(allStats, winnerStats, loserStats)

	return HoldingPeriodMetrics{
		AllTrades:      allStats,
		Winners:        winnerStats,
		Losers:         loserStats,
		OptimalPeriod:  optimalPeriod,
		TimeDecay:      timeDecay,
		Interpretation: interpretation,
	}
}

// calculateHoldingStats computes statistics for holding days
func calculateHoldingStats(days []int) HoldingStats {
	if len(days) == 0 {
		return HoldingStats{}
	}

	// Convert to float64 for calculations
	daysFloat := make([]float64, len(days))
	for i, d := range days {
		daysFloat[i] = float64(d)
	}

	avgDays := mean(daysFloat)
	medianDays := median(daysFloat)
	stdDev := standardDeviation(daysFloat)

	minDays := days[0]
	maxDays := days[0]
	for _, d := range days {
		if d < minDays {
			minDays = d
		}
		if d > maxDays {
			maxDays = d
		}
	}

	return HoldingStats{
		AvgDays:    avgDays,
		MedianDays: medianDays,
		MinDays:    minDays,
		MaxDays:    maxDays,
		StdDev:     stdDev,
	}
}

// findOptimalHoldingPeriod identifies which holding period ranges perform best
func findOptimalHoldingPeriod(trades []Trade) OptimalPeriodMetrics {
	// Create buckets
	buckets := make(map[string]*PeriodBucket)
	buckets["0-5 days"] = &PeriodBucket{}
	buckets["5-10 days"] = &PeriodBucket{}
	buckets["10-20 days"] = &PeriodBucket{}
	buckets["20-30 days"] = &PeriodBucket{}
	buckets[">30 days"] = &PeriodBucket{}

	// Categorize trades
	for _, t := range trades {
		days := int(t.ExitTime.Sub(t.EntryTime).Hours() / 24)

		var r float64
		if t.InitialRisk > 0 {
			r = t.PnL / t.InitialRisk
		} else {
			r = t.PnLPercent / 100.0
		}

		var bucketName string
		if days < 5 {
			bucketName = "0-5 days"
		} else if days < 10 {
			bucketName = "5-10 days"
		} else if days < 20 {
			bucketName = "10-20 days"
		} else if days < 30 {
			bucketName = "20-30 days"
		} else {
			bucketName = ">30 days"
		}

		bucket := buckets[bucketName]
		bucket.Count++

		// Track R values for averaging
		if bucket.AvgR == 0 {
			bucket.AvgR = r
		} else {
			// Running average
			bucket.AvgR = (bucket.AvgR*float64(bucket.Count-1) + r) / float64(bucket.Count)
		}

		if t.PnL > 0 {
			// Update win rate (running average)
			prevWins := bucket.WinRate * float64(bucket.Count-1) / 100.0
			bucket.WinRate = (prevWins + 1) / float64(bucket.Count) * 100
		} else {
			prevWins := bucket.WinRate * float64(bucket.Count-1) / 100.0
			bucket.WinRate = prevWins / float64(bucket.Count) * 100
		}
	}

	// Convert to map for result
	bucketPerformance := make(map[string]PeriodBucket)
	for name, bucket := range buckets {
		bucketPerformance[name] = *bucket
	}

	// Find best performing period
	bestPeriod := ""
	bestAvgR := -math.MaxFloat64
	for name, bucket := range bucketPerformance {
		if bucket.Count > 0 && bucket.AvgR > bestAvgR {
			bestAvgR = bucket.AvgR
			bestPeriod = name
		}
	}

	return OptimalPeriodMetrics{
		BucketPerformance: bucketPerformance,
		BestPeriod:        bestPeriod,
		BestAvgR:          bestAvgR,
	}
}

// analyzeTimeDecay checks if performance degrades over time
func analyzeTimeDecay(trades []Trade) TimeDecayMetrics {
	var longHolds, shortHolds []float64

	for _, t := range trades {
		days := int(t.ExitTime.Sub(t.EntryTime).Hours() / 24)

		var r float64
		if t.InitialRisk > 0 {
			r = t.PnL / t.InitialRisk
		} else {
			r = t.PnLPercent / 100.0
		}

		if days > 30 {
			longHolds = append(longHolds, r)
		} else if days <= 10 {
			shortHolds = append(shortHolds, r)
		}
	}

	longAvg := mean(longHolds)
	shortAvg := mean(shortHolds)
	decayPresent := longAvg < shortAvg

	recommendation := "Long holds performing well"
	if decayPresent {
		recommendation = "Consider time stops at 30 days - performance degrades with time"
	}

	return TimeDecayMetrics{
		LongHoldAvgR:   longAvg,
		ShortHoldAvgR:  shortAvg,
		DecayPresent:   decayPresent,
		Recommendation: recommendation,
	}
}

// interpretHoldingPeriods provides human-readable interpretation
func interpretHoldingPeriods(all, winners, losers HoldingStats) string {
	interpretations := make([]string, 0, 2)

	// Average holding period classification
	if all.AvgDays < 5 {
		interpretations = append(interpretations, "Very short holds - more like day trading")
	} else if all.AvgDays < 15 {
		interpretations = append(interpretations, "Short-term swing trading")
	} else if all.AvgDays < 30 {
		interpretations = append(interpretations, "Standard swing trading timeframe")
	} else {
		interpretations = append(interpretations, "Long-term position trading")
	}

	// Winner vs loser comparison
	if winners.AvgDays > losers.AvgDays*1.5 {
		interpretations = append(interpretations, "Winners held longer (good - letting winners run)")
	} else if losers.AvgDays > winners.AvgDays*1.5 {
		interpretations = append(interpretations, "Losers held longer (bad - holding losers too long)")
	} else {
		interpretations = append(interpretations, "Similar holding periods for winners and losers")
	}

	result := ""
	for i, interp := range interpretations {
		if i > 0 {
			result += " | "
		}
		result += interp
	}
	return result
}

// calculateStreaks analyzes consecutive win/loss streaks
func calculateStreaks(trades []Trade) StreakMetrics {
	if len(trades) == 0 {
		return StreakMetrics{}
	}

	// Sort trades by exit time to ensure chronological order
	sortedTrades := make([]Trade, len(trades))
	copy(sortedTrades, trades)
	sort.Slice(sortedTrades, func(i, j int) bool {
		return sortedTrades[i].ExitTime.Before(sortedTrades[j].ExitTime)
	})

	// Convert to win/loss sequence
	outcomes := make([]bool, len(sortedTrades))
	for i, t := range sortedTrades {
		outcomes[i] = t.PnL > 0 // true = win, false = loss
	}

	// Find streaks
	var winStreaks, lossStreaks []int
	currentStreak := 1
	currentType := outcomes[0]

	for i := 1; i < len(outcomes); i++ {
		if outcomes[i] == currentType {
			currentStreak++
		} else {
			// Streak ended
			if currentType {
				winStreaks = append(winStreaks, currentStreak)
			} else {
				lossStreaks = append(lossStreaks, currentStreak)
			}
			currentStreak = 1
			currentType = outcomes[i]
		}
	}

	// Don't forget the last streak
	if currentType {
		winStreaks = append(winStreaks, currentStreak)
	} else {
		lossStreaks = append(lossStreaks, currentStreak)
	}

	// Calculate statistics
	maxWinStreak := maxInt(winStreaks)
	maxLossStreak := maxInt(lossStreaks)
	avgWinStreak := meanInt(winStreaks)
	avgLossStreak := meanInt(lossStreaks)

	return StreakMetrics{
		MaxWinStreak:     maxWinStreak,
		MaxLossStreak:    maxLossStreak,
		AvgWinStreak:     avgWinStreak,
		AvgLossStreak:    avgLossStreak,
		TotalWinStreaks:  len(winStreaks),
		TotalLossStreaks: len(lossStreaks),
		Interpretation:   interpretStreaks(maxWinStreak, maxLossStreak),
	}
}

// interpretStreaks provides risk management insights
func interpretStreaks(maxWins, maxLosses int) string {
	if maxLosses >= 7 {
		return "WARNING: Max loss streak of " + intToString(maxLosses) + " - need position size rules for drawdowns"
	} else if maxLosses >= 5 {
		return "Moderate: Max loss streak of " + intToString(maxLosses) + " - prepare for 7+ streaks"
	}
	return "Normal: Max loss streak of " + intToString(maxLosses) + " - within expected range"
}

// Helper functions

func maxInt(data []int) int {
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

func meanInt(data []int) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := 0
	for _, v := range data {
		sum += v
	}
	return float64(sum) / float64(len(data))
}

func intToString(n int) string {
	if n == 0 {
		return "0"
	}

	// Simple int to string conversion
	negative := n < 0
	if negative {
		n = -n
	}

	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}

	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}
