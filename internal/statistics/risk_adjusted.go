package statistics

import "math"

// CalculateSharpeRatio calculates Sharpe Ratio from periodic returns
func CalculateSharpeRatio(returns []float64, riskFreeRate float64, periodsPerYear int) RiskAdjustedMetrics {
	if len(returns) < 2 {
		return RiskAdjustedMetrics{}
	}

	avgReturn := mean(returns)
	stdDev := standardDeviation(returns)

	annualReturn := avgReturn * float64(periodsPerYear)
	annualStdDev := stdDev * math.Sqrt(float64(periodsPerYear))

	sharpe := 0.0
	if annualStdDev > 0 {
		sharpe = (annualReturn - riskFreeRate) / annualStdDev
	}

	return RiskAdjustedMetrics{
		SharpeRatio:          sharpe,
		AnnualReturn:         annualReturn,
		AnnualStdDev:         annualStdDev,
		RiskFreeRate:         riskFreeRate,
		ExcessReturn:         annualReturn - riskFreeRate,
		SharpeInterpretation: interpretSharpe(sharpe),
	}
}

// CalculateSortinoRatio calculates Sortino Ratio (downside deviation only)
func CalculateSortinoRatio(returns []float64, riskFreeRate float64, periodsPerYear int, targetReturn float64) RiskAdjustedMetrics {
	if len(returns) < 2 {
		return RiskAdjustedMetrics{}
	}

	avgReturn := mean(returns)
	annualReturn := avgReturn * float64(periodsPerYear)

	downsideReturns := make([]float64, 0)
	for _, r := range returns {
		if r < targetReturn {
			downsideReturns = append(downsideReturns, r)
		}
	}

	if len(downsideReturns) == 0 {
		return RiskAdjustedMetrics{
			SortinoRatio:          math.Inf(1),
			AnnualReturn:          annualReturn,
			AnnualDownsideDev:     0,
			RiskFreeRate:          riskFreeRate,
			ExcessReturn:          annualReturn - riskFreeRate,
			SortinoInterpretation: "No downside periods - exceptional",
		}
	}

	var sumSquares float64
	for _, r := range downsideReturns {
		diff := r - targetReturn
		sumSquares += diff * diff
	}

	downsideVariance := sumSquares / float64(len(returns))
	downsideDeviation := math.Sqrt(downsideVariance)
	annualDownsideDev := downsideDeviation * math.Sqrt(float64(periodsPerYear))

	sortino := 0.0
	if annualDownsideDev > 0 {
		sortino = (annualReturn - riskFreeRate) / annualDownsideDev
	} else {
		sortino = math.Inf(1)
	}

	return RiskAdjustedMetrics{
		SortinoRatio:          sortino,
		AnnualReturn:          annualReturn,
		AnnualDownsideDev:     annualDownsideDev,
		RiskFreeRate:          riskFreeRate,
		ExcessReturn:          annualReturn - riskFreeRate,
		SortinoInterpretation: "Higher is better - focuses on harmful volatility",
	}
}

// CalculateCalmarRatio calculates Calmar Ratio from equity curve
func CalculateCalmarRatio(equityCurve []float64, periodsPerYear int) RiskAdjustedMetrics {
	if len(equityCurve) < 2 {
		return RiskAdjustedMetrics{}
	}

	totalReturn := (equityCurve[len(equityCurve)-1] / equityCurve[0]) - 1
	numPeriods := len(equityCurve) - 1
	years := float64(numPeriods) / float64(periodsPerYear)

	if years <= 0 {
		return RiskAdjustedMetrics{}
	}

	annualReturn := math.Pow(1+totalReturn, 1/years) - 1

	maxDD := calculateMaxDrawdownPercent(equityCurve)

	calmar := 0.0
	if maxDD > 0 {
		calmar = annualReturn / (maxDD / 100)
	} else {
		calmar = math.Inf(1)
	}

	return RiskAdjustedMetrics{
		CalmarRatio:          calmar,
		AnnualReturn:         annualReturn,
		CalmarInterpretation: "Higher is better - return per unit of max loss",
	}
}

// Helper functions moved to helpers.go

func calculateMaxDrawdownPercent(equityCurve []float64) float64 {
	if len(equityCurve) == 0 {
		return 0
	}

	runningMax := equityCurve[0]
	maxDD := 0.0

	for _, equity := range equityCurve {
		if equity > runningMax {
			runningMax = equity
		}

		drawdown := (runningMax - equity) / runningMax
		if drawdown > maxDD {
			maxDD = drawdown
		}
	}

	return maxDD * 100
}

func interpretSharpe(sharpe float64) string {
	switch {
	case sharpe >= 3.0:
		return "Exceptional (≥3.0)"
	case sharpe >= 2.0:
		return "Excellent (≥2.0)"
	case sharpe >= 1.0:
		return "Good (≥1.0)"
	case sharpe >= 0.5:
		return "Acceptable (≥0.5)"
	case sharpe > 0:
		return "Poor but positive"
	default:
		return "Negative - losing money"
	}
}
