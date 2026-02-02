package statistics

import (
	"sort"
)

// BuildEquityCurve creates an equity curve from trade history
func BuildEquityCurve(trades []Trade, initialBalance float64) []EquityPoint {
	if len(trades) == 0 {
		return []EquityPoint{{Equity: initialBalance}}
	}

	sortedTrades := make([]Trade, len(trades))
	copy(sortedTrades, trades)

	sort.Slice(sortedTrades, func(i, j int) bool {
		return sortedTrades[i].ExitTime.Before(sortedTrades[j].ExitTime)
	})

	equityCurve := make([]EquityPoint, 0, len(trades)+1)

	if len(sortedTrades) > 0 {
		equityCurve = append(equityCurve, EquityPoint{
			Time:   sortedTrades[0].EntryTime,
			Equity: initialBalance,
		})
	}

	currentEquity := initialBalance
	for _, trade := range sortedTrades {
		currentEquity += trade.PnL

		equityCurve = append(equityCurve, EquityPoint{
			Time:   trade.ExitTime,
			Equity: currentEquity,
		})
	}

	return equityCurve
}

// GetEquityReturns calculates period-over-period returns from equity curve
func GetEquityReturns(equityCurve []EquityPoint) []float64 {
	if len(equityCurve) < 2 {
		return nil
	}

	returns := make([]float64, 0, len(equityCurve)-1)

	for i := 1; i < len(equityCurve); i++ {
		prevEquity := equityCurve[i-1].Equity
		currentEquity := equityCurve[i].Equity

		if prevEquity > 0 {
			returnPct := (currentEquity - prevEquity) / prevEquity
			returns = append(returns, returnPct)
		}
	}

	return returns
}

// ExtractEquityValues returns just the equity values from the curve
func ExtractEquityValues(equityCurve []EquityPoint) []float64 {
	values := make([]float64, len(equityCurve))
	for i, point := range equityCurve {
		values[i] = point.Equity
	}
	return values
}
