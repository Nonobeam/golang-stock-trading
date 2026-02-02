package statistics

import "fmt"

// StatisticsCalculator orchestrates all statistics calculations
type StatisticsCalculator struct {
	config StatisticsConfig
}

// NewStatisticsCalculator creates a new calculator with the given config
func NewStatisticsCalculator(config StatisticsConfig) *StatisticsCalculator {
	return &StatisticsCalculator{
		config: config,
	}
}

// NewDefaultStatisticsCalculator creates a calculator with default config
func NewDefaultStatisticsCalculator() *StatisticsCalculator {
	return &StatisticsCalculator{
		config: DefaultConfig(),
	}
}

// Calculate computes all statistics for the given trades
func (calc *StatisticsCalculator) Calculate(trades []Trade, initialBalance float64) StatisticsResult {
	result := StatisticsResult{
		TotalTrades:    len(trades),
		InitialBalance: initialBalance,
	}

	sampleSizeValid, warning := calc.ValidateSampleSize(trades)
	result.SampleSizeAdequate = sampleSizeValid
	result.SampleSizeWarning = warning

	if len(trades) == 0 {
		result.FinalBalance = initialBalance
		return result
	}

	result.WinRate = CalculateWinRate(trades)

	result.Expectancy = CalculateExpectancy(trades, false)

	equityCurve := BuildEquityCurve(trades, initialBalance)
	result.EquityCurve = equityCurve
	result.FinalBalance = equityCurve[len(equityCurve)-1].Equity

	equityValues := ExtractEquityValues(equityCurve)

	result.Drawdown = AnalyzeDrawdowns(equityValues, nil)

	netProfit := result.FinalBalance - initialBalance
	recoveryFactor, recoveryInterp := CalculateRecoveryFactor(
		netProfit,
		result.Drawdown.MaxDrawdownPercent,
	)
	result.Drawdown.RecoveryFactor = recoveryFactor
	result.Drawdown.RecoveryInterpretation = recoveryInterp

	returns := GetEquityReturns(equityCurve)
	if len(returns) >= 2 {
		sharpeMetrics := CalculateSharpeRatio(
			returns,
			calc.config.RiskFreeRate,
			calc.config.PeriodsPerYear,
		)

		sortinoMetrics := CalculateSortinoRatio(
			returns,
			calc.config.RiskFreeRate,
			calc.config.PeriodsPerYear,
			0.0,
		)

		calmarMetrics := CalculateCalmarRatio(
			equityValues,
			calc.config.PeriodsPerYear,
		)

		result.RiskAdjusted = RiskAdjustedMetrics{
			SharpeRatio:           sharpeMetrics.SharpeRatio,
			SortinoRatio:          sortinoMetrics.SortinoRatio,
			CalmarRatio:           calmarMetrics.CalmarRatio,
			AnnualReturn:          sharpeMetrics.AnnualReturn,
			AnnualStdDev:          sharpeMetrics.AnnualStdDev,
			AnnualDownsideDev:     sortinoMetrics.AnnualDownsideDev,
			RiskFreeRate:          calc.config.RiskFreeRate,
			ExcessReturn:          sharpeMetrics.ExcessReturn,
			SharpeInterpretation:  sharpeMetrics.SharpeInterpretation,
			SortinoInterpretation: sortinoMetrics.SortinoInterpretation,
			CalmarInterpretation:  calmarMetrics.CalmarInterpretation,
		}
	}

	result.Distribution = AnalyzeDistribution(trades, 5)

	// NEW: R-Distribution Analysis
	result.RDistribution = AnalyzeRDistribution(trades)

	// NEW: MAE/MFE Analysis
	result.MAEMFE = AnalyzeMAEMFE(trades)

	// NEW: Time-Based Metrics
	result.TimeMetrics = AnalyzeTimeMetrics(trades)

	return result
}

// ValidateSampleSize checks if the sample size is adequate
func (calc *StatisticsCalculator) ValidateSampleSize(trades []Trade) (bool, string) {
	if len(trades) < calc.config.MinimumSampleSize {
		warning := fmt.Sprintf(
			"Sample size (%d) below minimum (%d) - statistics may not be reliable",
			len(trades),
			calc.config.MinimumSampleSize,
		)
		return false, warning
	}
	return true, ""
}
