package statistics

import "fmt"

// CalculateWinRate calculates win rate metrics from trade results
func CalculateWinRate(trades []Trade) WinRateMetrics {
	if len(trades) == 0 {
		return WinRateMetrics{}
	}

	var winners, losers, breakevens []Trade
	var totalWinPnL, totalLossPnL float64

	for _, trade := range trades {
		switch {
		case trade.PnL > 0:
			winners = append(winners, trade)
			totalWinPnL += trade.PnL
		case trade.PnL < 0:
			losers = append(losers, trade)
			totalLossPnL += trade.PnL
		default:
			breakevens = append(breakevens, trade)
		}
	}

	total := float64(len(trades))
	winRate := float64(len(winners)) / total * 100
	lossRate := float64(len(losers)) / total * 100
	breakevenRate := float64(len(breakevens)) / total * 100

	var avgWin, avgLoss float64
	if len(winners) > 0 {
		avgWin = totalWinPnL / float64(len(winners))
	}
	if len(losers) > 0 {
		avgLoss = totalLossPnL / float64(len(losers))
	}

	return WinRateMetrics{
		TotalTrades:      len(trades),
		Winners:          len(winners),
		Losers:           len(losers),
		Breakevens:       len(breakevens),
		WinRate:          winRate,
		LossRate:         lossRate,
		BreakevenRate:    breakevenRate,
		WinRateFormatted: fmt.Sprintf("%.1f%%", winRate),
		AverageWin:       avgWin,
		AverageLoss:      avgLoss,
	}
}
