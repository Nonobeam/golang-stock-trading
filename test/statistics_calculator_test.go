package test

import (
	"testing"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/statistics"
)

func TestStatisticsCalculator_FullCalculation(t *testing.T) {
	now := time.Now()

	trades := make([]statistics.Trade, 50)
	for i := 0; i < 50; i++ {
		pnl := float64((i%3)-1) * 1000000.0
		if i%2 == 0 {
			pnl = float64(i%5+1) * 500000.0
		}

		trades[i] = statistics.Trade{
			Symbol:      "VNM",
			EntryTime:   now.Add(time.Duration(i) * 24 * time.Hour),
			ExitTime:    now.Add(time.Duration(i+1) * 24 * time.Hour),
			EntryPrice:  50000 + float64(i)*100,
			ExitPrice:   50000 + float64(i)*100 + pnl/1000,
			Quantity:    1000,
			PnL:         pnl,
			PnLPercent:  pnl / 50000000.0 * 100,
			InitialRisk: 2000000,
			SignalType:  []string{"Pullback", "Breakout"}[i%2],
			Score:       7 + (i % 7),
		}
	}

	calc := statistics.NewDefaultStatisticsCalculator()
	result := calc.Calculate(trades, 100000000)

	if result.TotalTrades != 50 {
		t.Errorf("Expected 50 trades, got %d", result.TotalTrades)
	}

	if !result.SampleSizeAdequate {
		t.Errorf("Expected sample size to be adequate for 50 trades")
	}

	if result.SampleSizeWarning != "" {
		t.Errorf("Expected no warning for 50 trades, got: %s", result.SampleSizeWarning)
	}

	if result.WinRate.TotalTrades != 50 {
		t.Error("Win rate not calculated")
	}

	if result.Expectancy.Expectancy == 0 {
		t.Error("Expectancy not calculated")
	}

	if len(result.EquityCurve) == 0 {
		t.Error("Equity curve not built")
	}

	if result.Distribution.BySignalType == nil {
		t.Error("Signal type distribution not analyzed")
	}

	t.Logf("=== Statistics Results ===")
	t.Logf("Win Rate: %.1f%%", result.WinRate.WinRate)
	t.Logf("Expectancy: %.0f %s (%s)",
		result.Expectancy.Expectancy,
		result.Expectancy.Unit,
		result.Expectancy.Interpretation)
	t.Logf("Profit Factor: %.2f", result.Expectancy.ProfitFactor)
	t.Logf("Sharpe Ratio: %.2f (%s)",
		result.RiskAdjusted.SharpeRatio,
		result.RiskAdjusted.SharpeInterpretation)
	t.Logf("Max Drawdown: %.2f%%", result.Drawdown.MaxDrawdownPercent)
	t.Logf("Recovery Factor: %.2f (%s)",
		result.Drawdown.RecoveryFactor,
		result.Drawdown.RecoveryInterpretation)
	t.Logf("Final Balance: %.0f (from %.0f)",
		result.FinalBalance, result.InitialBalance)
}

func TestStatisticsCalculator_SmallSample(t *testing.T) {
	trades := make([]statistics.Trade, 15)
	now := time.Now()

	for i := 0; i < 15; i++ {
		trades[i] = statistics.Trade{
			Symbol:     "VNM",
			EntryTime:  now.Add(time.Duration(i) * 24 * time.Hour),
			ExitTime:   now.Add(time.Duration(i+1) * 24 * time.Hour),
			PnL:        float64(i%2*2-1) * 1000000.0,
			SignalType: "Pullback",
		}
	}

	calc := statistics.NewDefaultStatisticsCalculator()
	result := calc.Calculate(trades, 100000000)

	if result.SampleSizeAdequate {
		t.Error("Expected sample size to be inadequate for 15 trades")
	}

	if result.SampleSizeWarning == "" {
		t.Error("Expected warning for small sample size")
	}

	t.Logf("Sample size warning: %s", result.SampleSizeWarning)
}

func TestStatisticsCalculator_NoTrades(t *testing.T) {
	calc := statistics.NewDefaultStatisticsCalculator()
	initialBalance := 100000000.0
	metrics := calc.Calculate([]statistics.Trade{}, initialBalance)

	if metrics.TotalTrades != 0 {
		t.Errorf("Expected 0 total trades, got %d", metrics.TotalTrades)
	}

	if metrics.FinalBalance != initialBalance {
		t.Errorf("Expected final balance %.0f, got %.0f", initialBalance, metrics.FinalBalance)
	}

	// With no trades, equity curve should be empty or have only initial point
	if len(metrics.EquityCurve) > 0 {
		t.Logf("Equity curve has %d points for zero trades (acceptable)", len(metrics.EquityCurve))
	}

	// Check that new metrics are zero/empty
	if metrics.RDistribution.MeanR != 0 {
		t.Errorf("Expected RDistribution.MeanR to be 0 for no trades")
	}
}
