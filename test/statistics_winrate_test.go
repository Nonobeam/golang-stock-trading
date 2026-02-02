package test

import (
	"testing"

	"github.com/nonobeam/golang-stock-trading/internal/statistics"
)

func TestWinRate_BalancedTrades(t *testing.T) {
	trades := []statistics.Trade{
		{Symbol: "VNM", PnL: 2500000, PnLPercent: 5.0},
		{Symbol: "HPG", PnL: -1000000, PnLPercent: -2.0},
		{Symbol: "FPT", PnL: 3200000, PnLPercent: 6.4},
		{Symbol: "VNM", PnL: -800000, PnLPercent: -1.6},
		{Symbol: "HPG", PnL: 1500000, PnLPercent: 3.0},
		{Symbol: "FPT", PnL: -1000000, PnLPercent: -2.0},
		{Symbol: "VNM", PnL: 2000000, PnLPercent: 4.0},
		{Symbol: "HPG", PnL: -500000, PnLPercent: -1.0},
		{Symbol: "FPT", PnL: 1800000, PnLPercent: 3.6},
		{Symbol: "VNM", PnL: -1200000, PnLPercent: -2.4},
	}

	metrics := statistics.CalculateWinRate(trades)

	if metrics.TotalTrades != 10 {
		t.Errorf("Expected 10 total trades, got %d", metrics.TotalTrades)
	}

	if metrics.Winners != 5 {
		t.Errorf("Expected 5 winners, got %d", metrics.Winners)
	}

	if metrics.Losers != 5 {
		t.Errorf("Expected 5 losers, got %d", metrics.Losers)
	}

	if metrics.WinRate != 50.0 {
		t.Errorf("Expected win rate 50%%, got %.1f%%", metrics.WinRate)
	}

	if metrics.LossRate != 50.0 {
		t.Errorf("Expected loss rate 50%%, got %.1f%%", metrics.LossRate)
	}

	expectedAvgWin := 2200000.0
	if almostEqual(metrics.AverageWin, expectedAvgWin, 1000) == false {
		t.Errorf("Expected avg win ~%.0f, got %.0f", expectedAvgWin, metrics.AverageWin)
	}

	expectedAvgLoss := -900000.0
	if almostEqual(metrics.AverageLoss, expectedAvgLoss, 1000) == false {
		t.Errorf("Expected avg loss ~%.0f, got %.0f", expectedAvgLoss, metrics.AverageLoss)
	}

	t.Logf("Win Rate: %s, Avg Win: %.0f, Avg Loss: %.0f",
		metrics.WinRateFormatted, metrics.AverageWin, metrics.AverageLoss)
}

func TestWinRate_AllWinners(t *testing.T) {
	trades := []statistics.Trade{
		{Symbol: "VNM", PnL: 1000000, PnLPercent: 2.0},
		{Symbol: "HPG", PnL: 2000000, PnLPercent: 4.0},
		{Symbol: "FPT", PnL: 1500000, PnLPercent: 3.0},
	}

	metrics := statistics.CalculateWinRate(trades)

	if metrics.WinRate != 100.0 {
		t.Errorf("Expected win rate 100%%, got %.1f%%", metrics.WinRate)
	}

	if metrics.LossRate != 0.0 {
		t.Errorf("Expected loss rate 0%%, got %.1f%%", metrics.LossRate)
	}

	if metrics.Losers != 0 {
		t.Errorf("Expected 0 losers, got %d", metrics.Losers)
	}
}

func TestWinRate_WithBreakevens(t *testing.T) {
	trades := []statistics.Trade{
		{Symbol: "VNM", PnL: 1000000, PnLPercent: 2.0},
		{Symbol: "HPG", PnL: 0, PnLPercent: 0.0},
		{Symbol: "FPT", PnL: -500000, PnLPercent: -1.0},
		{Symbol: "VNM", PnL: 0, PnLPercent: 0.0},
	}

	metrics := statistics.CalculateWinRate(trades)

	if metrics.Breakevens != 2 {
		t.Errorf("Expected 2 breakevens, got %d", metrics.Breakevens)
	}

	if metrics.BreakevenRate != 50.0 {
		t.Errorf("Expected breakeven rate 50%%, got %.1f%%", metrics.BreakevenRate)
	}
}

func almostEqual(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}
