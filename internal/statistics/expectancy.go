package statistics

import (
	"fmt"
	"math"
)

// CalculateExpectancy calculates expectancy (expected value per trade)
func CalculateExpectancy(trades []Trade, inRMultiples bool) ExpectancyMetrics {
	if len(trades) == 0 {
		return ExpectancyMetrics{}
	}

	var winners, losers []Trade
	var totalProfit, totalLoss float64

	for _, trade := range trades {
		value := trade.PnL
		if inRMultiples && trade.InitialRisk != 0 {
			value = trade.PnL / trade.InitialRisk
		}

		if value > 0 {
			winners = append(winners, trade)
			totalProfit += value
		} else if value < 0 {
			losers = append(losers, trade)
			totalLoss += value
		}
	}

	if len(winners) == 0 || len(losers) == 0 {
		return ExpectancyMetrics{
			Unit: getUnit(inRMultiples),
		}
	}

	avgWin := totalProfit / float64(len(winners))
	avgLoss := math.Abs(totalLoss / float64(len(losers)))

	totalTrades := float64(len(trades))
	winRate := float64(len(winners)) / totalTrades
	lossRate := float64(len(losers)) / totalTrades

	expectancy := (winRate * avgWin) - (lossRate * avgLoss)
	expectancyRatio := 0.0
	if avgLoss > 0 {
		expectancyRatio = expectancy / avgLoss
	}

	profitFactor := calculateProfitFactorValue(trades, inRMultiples)
	payoffRatio := avgWin / avgLoss

	return ExpectancyMetrics{
		Expectancy:      expectancy,
		ExpectancyRatio: expectancyRatio,
		WinRate:         winRate,
		LossRate:        lossRate,
		AvgWin:          avgWin,
		AvgLoss:         avgLoss,
		TotalProfit:     totalProfit,
		TotalLoss:       math.Abs(totalLoss),
		NetProfit:       totalProfit + totalLoss,
		Unit:            getUnit(inRMultiples),
		Interpretation:  interpretExpectancy(expectancy, inRMultiples),
		ProfitFactor:    profitFactor,
		PayoffRatio:     payoffRatio,
		PayoffFormatted: fmt.Sprintf("%.2f:1", payoffRatio),
	}
}

// CalculateProfitFactor calculates the ratio of gross profit to gross loss
func CalculateProfitFactor(trades []Trade) (float64, string) {
	factor := calculateProfitFactorValue(trades, false)
	interpretation := interpretProfitFactor(factor)
	return factor, interpretation
}

func calculateProfitFactorValue(trades []Trade, inRMultiples bool) float64 {
	if len(trades) == 0 {
		return 0
	}

	var grossProfit, grossLoss float64

	for _, trade := range trades {
		value := trade.PnL
		if inRMultiples && trade.InitialRisk != 0 {
			value = trade.PnL / trade.InitialRisk
		}

		if value > 0 {
			grossProfit += value
		} else if value < 0 {
			grossLoss += math.Abs(value)
		}
	}

	if grossProfit == 0 {
		return 0
	}
	if grossLoss == 0 {
		return math.Inf(1)
	}

	return grossProfit / grossLoss
}

// CalculatePayoffRatio calculates average win / average loss
func CalculatePayoffRatio(trades []Trade) (float64, string) {
	if len(trades) == 0 {
		return 0, ""
	}

	var totalWin, totalLoss float64
	var winCount, lossCount int

	for _, trade := range trades {
		if trade.PnL > 0 {
			totalWin += trade.PnL
			winCount++
		} else if trade.PnL < 0 {
			totalLoss += math.Abs(trade.PnL)
			lossCount++
		}
	}

	if winCount == 0 || lossCount == 0 {
		return 0, ""
	}

	avgWin := totalWin / float64(winCount)
	avgLoss := totalLoss / float64(lossCount)

	if avgLoss == 0 {
		return math.Inf(1), "Infinite (no losses)"
	}

	ratio := avgWin / avgLoss
	interpretation := interpretPayoffRatio(ratio)
	
	return ratio, interpretation
}

func getUnit(inRMultiples bool) string {
	if inRMultiples {
		return "R"
	}
	return "VND"
}

func interpretExpectancy(expectancy float64, inR bool) string {
	if inR {
		switch {
		case expectancy >= 1.0:
			return "Exceptional system (≥1.0R per trade)"
		case expectancy >= 0.5:
			return "Excellent system (≥0.5R per trade)"
		case expectancy >= 0.3:
			return "Good system (≥0.3R per trade)"
		case expectancy >= 0.2:
			return "Viable system (≥0.2R per trade)"
		case expectancy > 0:
			return "Marginal system - needs improvement"
		default:
			return "Losing system - major revision needed"
		}
	}

	if expectancy > 0 {
		return fmt.Sprintf("Profitable system (+%.0f VND per trade)", expectancy)
	}
	return fmt.Sprintf("Losing system (%.0f VND per trade)", expectancy)
}

func interpretProfitFactor(factor float64) string {
	if math.IsInf(factor, 1) {
		return "No losing trades (perfect!)"
	}

	switch {
	case factor >= 2.0:
		return "Excellent (≥2.0)"
	case factor >= 1.5:
		return "Good (≥1.5)"
	case factor >= 1.25:
		return "Acceptable (≥1.25)"
	case factor > 1.0:
		return "Marginal (>1.0)"
	case factor == 1.0:
		return "Breakeven (1.0)"
	default:
		return "Losing (<1.0)"
	}
}

func interpretPayoffRatio(ratio float64) string {
	if math.IsInf(ratio, 1) {
		return "Infinite - no losses"
	}

	switch {
	case ratio >= 3.0:
		return "Excellent (≥3:1)"
	case ratio >= 2.0:
		return "Good (≥2:1)"
	case ratio >= 1.5:
		return "Acceptable (≥1.5:1)"
	case ratio >= 1.0:
		return "Marginal (≥1:1)"
	default:
		return "Poor (<1:1) - winners smaller than losers"
	}
}
