package test

import (
	"testing"

	"github.com/nonobeam/golang-stock-trading/internal/statistics"
)

func TestExpectancy_RMultiples(t *testing.T) {
	trades := []statistics.Trade{
		{Symbol: "VNM", PnL: 5000000, InitialRisk: 2000000, PnLPercent: 5.0},
		{Symbol: "HPG", PnL: -2000000, InitialRisk: 2000000, PnLPercent: -2.0},
		{Symbol: "FPT", PnL: 6400000, InitialRisk: 2000000, PnLPercent: 6.4},
		{Symbol: "VNM", PnL: -1600000, InitialRisk: 2000000, PnLPercent: -1.6},
		{Symbol: "HPG", PnL: 3000000, InitialRisk: 2000000, PnLPercent: 3.0},
		{Symbol: "FPT", PnL: -2000000, InitialRisk: 2000000, PnLPercent: -2.0},
		{Symbol: "VNM", PnL: 4000000, InitialRisk: 2000000, PnLPercent: 4.0},
		{Symbol: "HPG", PnL: -1000000, InitialRisk: 2000000, PnLPercent: -1.0},
		{Symbol: "FPT", PnL: 3600000, InitialRisk: 2000000, PnLPercent: 3.6},
		{Symbol: "VNM", PnL: -2400000, InitialRisk: 2000000, PnLPercent: -2.4},
	}

	metrics := statistics.CalculateExpectancy(trades, true)

	if metrics.WinRate != 0.5 {
		t.Errorf("Expected win rate 0.5, got %.2f", metrics.WinRate)
	}

	if !almostEqual(metrics.Expectancy, 0.65, 0.05) {
		t.Errorf("Expected expectancy ~0.65R, got %.2fR", metrics.Expectancy)
	}

	if metrics.Unit != "R" {
		t.Errorf("Expected unit 'R', got '%s'", metrics.Unit)
	}

	t.Logf("Expectancy: %.2fR, Interpretation: %s",
		metrics.Expectancy, metrics.Interpretation)
	t.Logf("Win Rate: %.1f%%, Avg Win: %.2fR, Avg Loss: %.2fR",
		metrics.WinRate*100, metrics.AvgWin, metrics.AvgLoss)
}

func TestExpectancy_LowWinRateHighPayoff(t *testing.T) {
	trades := []statistics.Trade{
		{Symbol: "VNM", PnL: 10000000, PnLPercent: 10.0},
		{Symbol: "HPG", PnL: -2000000, PnLPercent: -2.0},
		{Symbol: "FPT", PnL: 12000000, PnLPercent: 12.0},
		{Symbol: "VNM", PnL: -2000000, PnLPercent: -2.0},
		{Symbol: "HPG", PnL: -2000000, PnLPercent: -2.0},
		{Symbol: "FPT", PnL: -2000000, PnLPercent: -2.0},
		{Symbol: "VNM", PnL: -2000000, PnLPercent: -2.0},
		{Symbol: "HPG", PnL: -2000000, PnLPercent: -2.0},
		{Symbol: "FPT", PnL: -2000000, PnLPercent: -2.0},
		{Symbol: "VNM", PnL: -2000000, PnLPercent: -2.0},
	}

	metrics := statistics.CalculateExpectancy(trades, false)

	if metrics.WinRate != 0.2 {
		t.Errorf("Expected win rate 0.2 (20%%), got %.2f", metrics.WinRate)
	}

	if metrics.Expectancy <= 0 {
		t.Errorf("Expected positive expectancy, got %.0f", metrics.Expectancy)
	}

	t.Logf("Low win rate (%.1f%%) but positive expectancy: %.0f VND",
		metrics.WinRate*100, metrics.Expectancy)
}

func TestProfitFactor(t *testing.T) {
	trades := []statistics.Trade{
		{PnL: 15000000},
		{PnL: -8000000},
	}

	factor, interpretation := statistics.CalculateProfitFactor(trades)

	expected := 15000000.0 / 8000000.0
	if !almostEqual(factor, expected, 0.01) {
		t.Errorf("Expected profit factor ~%.3f, got %.3f", expected, factor)
	}

	t.Logf("Profit Factor: %.3f, Interpretation: %s", factor, interpretation)
}

func TestPayoffRatio(t *testing.T) {
	trades := []statistics.Trade{
		{PnL: 6500000},
		{PnL: 6500000},
		{PnL: -2800000},
		{PnL: -2800000},
	}

	ratio, interpretation := statistics.CalculatePayoffRatio(trades)

	expected := 6500000.0 / 2800000.0
	if !almostEqual(ratio, expected, 0.01) {
		t.Errorf("Expected payoff ratio ~%.2f, got %.2f", expected, ratio)
	}

	if ratio < 2.0 {
		t.Errorf("Expected ratio >= 2.0 for 'Good' rating, got %.2f", ratio)
	}

	t.Logf("Payoff Ratio: %.2f:1, Interpretation: %s", ratio, interpretation)
}

func TestExpectancy_EdgeCases(t *testing.T) {
	t.Run("NoTrades", func(t *testing.T) {
		metrics := statistics.CalculateExpectancy([]statistics.Trade{}, false)
		if metrics.Expectancy != 0 {
			t.Errorf("Expected 0 expectancy for no trades, got %.2f", metrics.Expectancy)
		}
	})

	t.Run("OnlyWinners", func(t *testing.T) {
		trades := []statistics.Trade{
			{PnL: 1000000},
			{PnL: 2000000},
		}
		metrics := statistics.CalculateExpectancy(trades, false)
		if metrics.AvgLoss != 0 {
			t.Errorf("Expected 0 avg loss for only winners, got %.0f", metrics.AvgLoss)
		}
	})

	t.Run("OnlyLosers", func(t *testing.T) {
		trades := []statistics.Trade{
			{PnL: -1000000},
			{PnL: -2000000},
		}
		metrics := statistics.CalculateExpectancy(trades, false)
		if metrics.AvgWin != 0 {
			t.Errorf("Expected 0 avg win for only losers, got %.0f", metrics.AvgWin)
		}
	})
}
