package backtest

import (
	"math"

	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

// CalculateComprehensiveMetrics computes all performance metrics from trades and equity curve.
func CalculateComprehensiveMetrics(trades []*ClosedTrade, equityCurve []EquityPoint, initialCapital float64) (*BacktestMetrics, error) {
	metrics := &BacktestMetrics{
		TotalTrades:  len(trades),
		BySignalType: make(map[string]*SignalTypeMetrics),
		ByRegime:     make(map[string]*RegimeMetrics),
		RMultiples:   []float64{},
	}

	// Calculate drawdown from equity curve (can be done without trades)
	if len(equityCurve) > 0 {
		calculateDrawdown(metrics, equityCurve)
		calculateSharpeRatio(metrics, equityCurve, 0.05) // 5% risk-free rate (VN govt bonds)
		calculateSortinoRatio(metrics, equityCurve, 0.05)
		calculateCalmarRatio(metrics, initialCapital)
	}

	// If no trades, return early with equity curve metrics only
	if len(trades) == 0 {
		logger.Warn().Msg("No trades to analyze")
		return metrics, nil
	}

	// Calculate basic trade statistics
	calculateTradeStatistics(metrics, trades)

	// Calculate R-multiple analysis
	calculateRMultipleAnalysis(metrics, trades)

	// Calculate holding period statistics
	calculateHoldingStats(metrics, trades)

	// Calculate consecutive streaks
	calculateStreaks(metrics, trades)

	// Calculate breakdowns
	analyzeBySignalType(metrics, trades)
	// Note: analyzeByRegime requires regime info in trades - implement when available

	return metrics, nil
}

// calculateTradeStatistics computes win rate, profit factor, avg win/loss.
func calculateTradeStatistics(metrics *BacktestMetrics, trades []*ClosedTrade) {
	var totalWins, totalLosses float64
	var largestWin, largestLoss float64

	for _, trade := range trades {
		if trade.PnL > 0 {
			metrics.WinningTrades++
			totalWins += trade.PnL
			if trade.PnL > largestWin {
				largestWin = trade.PnL
			}
		} else if trade.PnL < 0 {
			metrics.LosingTrades++
			totalLosses += -trade.PnL // Make positive
			if trade.PnL < largestLoss {
				largestLoss = trade.PnL
			}
		}
		metrics.TotalPnL += trade.PnL
	}

	// Calculate percentages and averages
	if metrics.TotalTrades > 0 {
		metrics.WinRate = float64(metrics.WinningTrades) / float64(metrics.TotalTrades) * 100
	}

	if metrics.WinningTrades > 0 {
		metrics.AvgWin = totalWins / float64(metrics.WinningTrades)
	}

	if metrics.LosingTrades > 0 {
		metrics.AvgLoss = -totalLosses / float64(metrics.LosingTrades) // Negative
	}

	metrics.LargestWin = largestWin
	metrics.LargestLoss = largestLoss

	// Profit factor
	if totalLosses > 0 {
		metrics.ProfitFactor = totalWins / totalLosses
	} else if totalWins > 0 {
		metrics.ProfitFactor = 999.99 // Infinite (no losses)
	}
}

// calculateRMultipleAnalysis computes R-multiple distribution and expectancy.
func calculateRMultipleAnalysis(metrics *BacktestMetrics, trades []*ClosedTrade) {
	var totalR float64

	for _, trade := range trades {
		metrics.RMultiples = append(metrics.RMultiples, trade.RMultiple)
		totalR += trade.RMultiple
	}

	if metrics.TotalTrades > 0 {
		metrics.AvgRMultiple = totalR / float64(metrics.TotalTrades)
		metrics.Expectancy = metrics.AvgRMultiple
	}
}

// calculateHoldingStats computes average and median holding period.
func calculateHoldingStats(metrics *BacktestMetrics, trades []*ClosedTrade) {
	if len(trades) == 0 {
		return
	}

	totalDays := 0
	for _, trade := range trades {
		totalDays += trade.HoldingDays
	}

	metrics.AvgHoldingDays = totalDays / len(trades)
}

// calculateStreaks finds longest consecutive win and loss streaks.
func calculateStreaks(metrics *BacktestMetrics, trades []*ClosedTrade) {
	if len(trades) == 0 {
		return
	}

	currentWinStreak := 0
	currentLossStreak := 0
	maxWinStreak := 0
	maxLossStreak := 0

	for _, trade := range trades {
		if trade.PnL > 0 {
			currentWinStreak++
			currentLossStreak = 0
			if currentWinStreak > maxWinStreak {
				maxWinStreak = currentWinStreak
			}
		} else if trade.PnL < 0 {
			currentLossStreak++
			currentWinStreak = 0
			if currentLossStreak > maxLossStreak {
				maxLossStreak = currentLossStreak
			}
		}
	}

	metrics.LongestWinStreak = maxWinStreak
	metrics.LongestLossStreak = maxLossStreak
}

// calculateDrawdown finds maximum peak-to-trough decline in equity curve.
func calculateDrawdown(metrics *BacktestMetrics, equityCurve []EquityPoint) {
	if len(equityCurve) == 0 {
		return
	}

	// Initialize with first point
	maxEquity := equityCurve[0].Equity
	var maxDrawdown float64
	var maxDrawdownPercent float64

	for _, point := range equityCurve {
		// Update running maximum
		if point.Equity > maxEquity {
			maxEquity = point.Equity
		}

		// Calculate current drawdown
		if maxEquity > 0 {
			drawdown := point.Equity - maxEquity
			drawdownPercent := (drawdown / maxEquity) * 100

			// Track maximum drawdown (most negative)
			if drawdown < maxDrawdown {
				maxDrawdown = drawdown
				maxDrawdownPercent = drawdownPercent
			}
		}
	}

	metrics.MaxDrawdown = maxDrawdown
	metrics.MaxDrawdownPercent = maxDrawdownPercent
}

// calculateSharpeRatio computes annual Sharpe ratio from equity curve.
func calculateSharpeRatio(metrics *BacktestMetrics, equityCurve []EquityPoint, riskFreeRate float64) {
	if len(equityCurve) < 2 {
		return
	}

	// Calculate daily returns
	returns := make([]float64, 0, len(equityCurve)-1)
	for i := 1; i < len(equityCurve); i++ {
		if equityCurve[i-1].Equity > 0 {
			dailyReturn := (equityCurve[i].Equity - equityCurve[i-1].Equity) / equityCurve[i-1].Equity
			returns = append(returns, dailyReturn)
		}
	}

	if len(returns) < 2 {
		return
	}

	// Calculate mean return
	var sumReturns float64
	for _, r := range returns {
		sumReturns += r
	}
	meanReturn := sumReturns / float64(len(returns))

	// Calculate standard deviation
	var sumSquaredDiff float64
	for _, r := range returns {
		diff := r - meanReturn
		sumSquaredDiff += diff * diff
	}
	variance := sumSquaredDiff / float64(len(returns))
	stdDev := math.Sqrt(variance)

	if stdDev == 0 {
		// No volatility - either constant equity or issue with data
		return
	}

	// Annualize (assuming ~252 trading days per year)
	annualReturn := meanReturn * 252
	annualStdDev := stdDev * math.Sqrt(252)

	// Calculate Sharpe ratio
	metrics.SharpeRatio = (annualReturn - riskFreeRate) / annualStdDev
}

// calculateSortinoRatio computes Sortino ratio (downside deviation only).
func calculateSortinoRatio(metrics *BacktestMetrics, equityCurve []EquityPoint, riskFreeRate float64) {
	if len(equityCurve) < 2 {
		return
	}

	// Calculate daily returns
	returns := make([]float64, 0, len(equityCurve)-1)
	for i := 1; i < len(equityCurve); i++ {
		if equityCurve[i-1].Equity > 0 {
			dailyReturn := (equityCurve[i].Equity - equityCurve[i-1].Equity) / equityCurve[i-1].Equity
			returns = append(returns, dailyReturn)
		}
	}

	if len(returns) == 0 {
		return
	}

	// Calculate mean return
	var sumReturns float64
	for _, r := range returns {
		sumReturns += r
	}
	meanReturn := sumReturns / float64(len(returns))

	// Calculate downside deviation (only negative returns)
	var sumSquaredDownside float64
	downsideCount := 0
	for _, r := range returns {
		if r < 0 {
			sumSquaredDownside += r * r
			downsideCount++
		}
	}

	if downsideCount == 0 {
		// No downside, Sortino is very high
		metrics.SortinoRatio = 999.99
		return
	}

	downsideStdDev := math.Sqrt(sumSquaredDownside / float64(downsideCount))

	if downsideStdDev == 0 {
		return
	}

	// Annualize
	annualReturn := meanReturn * 252
	annualDownsideStdDev := downsideStdDev * math.Sqrt(252)

	// Calculate Sortino ratio
	metrics.SortinoRatio = (annualReturn - riskFreeRate) / annualDownsideStdDev
}

// calculateCalmarRatio computes Calmar ratio (return / max drawdown).
func calculateCalmarRatio(metrics *BacktestMetrics, initialCapital float64) {
	if metrics.MaxDrawdownPercent >= 0 {
		// No drawdown or positive (shouldn't happen)
		return
	}

	if initialCapital <= 0 {
		return
	}

	// Calculate annualized return (assuming metrics.TotalPnLPercent is already calculated)
	// For now, use absolute max drawdown
	absDrawdown := math.Abs(metrics.MaxDrawdownPercent)
	if absDrawdown > 0 {
		metrics.CalmarRatio = metrics.TotalPnLPercent / absDrawdown
	}
}

// analyzeBySignalType groups trades by signal type and calculates performance.
func analyzeBySignalType(metrics *BacktestMetrics, trades []*ClosedTrade) {
	grouped := make(map[string][]*ClosedTrade)

	// Group trades by signal type
	for _, trade := range trades {
		signalType := trade.SignalType
		if signalType == "" {
			signalType = "unknown"
		}
		grouped[signalType] = append(grouped[signalType], trade)
	}

	// Calculate metrics for each signal type
	for signalType, typeTrades := range grouped {
		typeMetrics := &SignalTypeMetrics{
			SignalType:  signalType,
			TotalTrades: len(typeTrades),
		}

		var wins, totalWins, totalLosses, totalR float64
		for _, trade := range typeTrades {
			if trade.PnL > 0 {
				wins++
				totalWins += trade.PnL
			} else if trade.PnL < 0 {
				totalLosses += -trade.PnL
			}
			totalR += trade.RMultiple
			typeMetrics.TotalPnL += trade.PnL
		}

		if typeMetrics.TotalTrades > 0 {
			typeMetrics.WinRate = (wins / float64(typeMetrics.TotalTrades)) * 100
			typeMetrics.AvgRMultiple = totalR / float64(typeMetrics.TotalTrades)
		}

		if totalLosses > 0 {
			typeMetrics.ProfitFactor = totalWins / totalLosses
		} else if totalWins > 0 {
			typeMetrics.ProfitFactor = 999.99
		}

		metrics.BySignalType[signalType] = typeMetrics
	}
}

// analyzeByRegime groups trades by market regime and calculates performance.
// Note: This requires regime information in ClosedTrade - implement when available.
func analyzeByRegime(metrics *BacktestMetrics, trades []*ClosedTrade) {
	// TODO: Implement when regime info is added to ClosedTrade
	// For now, this is a placeholder
	logger.Debug().Msg("Regime analysis not yet implemented - requires regime field in ClosedTrade")
}
