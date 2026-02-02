package backtest

import (
	"math"
	"testing"
	"time"
)

func TestBasicStatistics(t *testing.T) {
	// Create 10 sample trades with known results
	trades := []*ClosedTrade{
		{PnL: 2000000, RMultiple: 2.0, HoldingDays: 5, SignalType: "pullback"},
		{PnL: -1000000, RMultiple: -0.5, HoldingDays: 3, SignalType: "pullback"},
		{PnL: 3000000, RMultiple: 3.0, HoldingDays: 8, SignalType: "breakout"},
		{PnL: 1500000, RMultiple: 1.5, HoldingDays: 7, SignalType: "pullback"},
		{PnL: -800000, RMultiple: -0.4, HoldingDays: 2, SignalType: "breakout"},
		{PnL: 2500000, RMultiple: 2.5, HoldingDays: 10, SignalType: "pullback"},
		{PnL: 1000000, RMultiple: 1.0, HoldingDays: 4, SignalType: "breakout"},
		{PnL: -500000, RMultiple: -0.25, HoldingDays: 1, SignalType: "pullback"},
		{PnL: 1800000, RMultiple: 1.8, HoldingDays: 6, SignalType: "breakout"},
		{PnL: 2200000, RMultiple: 2.2, HoldingDays: 9, SignalType: "pullback"},
	}

	initialCapital := 100000000.0

	metrics, err := CalculateComprehensiveMetrics(trades, []EquityPoint{}, initialCapital)
	if err != nil {
		t.Fatalf("Failed to calculate metrics: %v", err)
	}

	// Verify total trades
	if metrics.TotalTrades != 10 {
		t.Errorf("Expected 10 trades, got %d", metrics.TotalTrades)
	}

	// Verify win/loss counts
	expectedWins := 7
	expectedLosses := 3
	if metrics.WinningTrades != expectedWins {
		t.Errorf("Expected %d winning trades, got %d", expectedWins, metrics.WinningTrades)
	}
	if metrics.LosingTrades != expectedLosses {
		t.Errorf("Expected %d losing trades, got %d", expectedLosses, metrics.LosingTrades)
	}

	// Verify win rate
	expectedWinRate := 70.0 // 7/10 = 70%
	if math.Abs(metrics.WinRate-expectedWinRate) > 0.01 {
		t.Errorf("Expected win rate %.2f%%, got %.2f%%", expectedWinRate, metrics.WinRate)
	}

	// Verify total P&L
	expectedPnL := 2000000.0 - 1000000 + 3000000 + 1500000 - 800000 + 2500000 + 1000000 - 500000 + 1800000 + 2200000
	if math.Abs(metrics.TotalPnL-expectedPnL) > 1 {
		t.Errorf("Expected total P&L %.2f, got %.2f", expectedPnL, metrics.TotalPnL)
	}

	// Verify average R-multiple
	expectedAvgR := (2.0 - 0.5 + 3.0 + 1.5 - 0.4 + 2.5 + 1.0 - 0.25 + 1.8 + 2.2) / 10
	if math.Abs(metrics.AvgRMultiple-expectedAvgR) > 0.01 {
		t.Errorf("Expected avg R-multiple %.2f, got %.2f", expectedAvgR, metrics.AvgRMultiple)
	}

	// Verify profit factor
	totalWins := 2000000.0 + 3000000 + 1500000 + 2500000 + 1000000 + 1800000 + 2200000
	totalLosses := 1000000.0 + 800000 + 500000
	expectedPF := totalWins / totalLosses
	if math.Abs(metrics.ProfitFactor-expectedPF) > 0.01 {
		t.Errorf("Expected profit factor %.2f, got %.2f", expectedPF, metrics.ProfitFactor)
	}
}

func TestSharpeRatio(t *testing.T) {
	// Create mock equity curve with known volatility
	startDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	equityCurve := []EquityPoint{
		{Date: startDate, Equity: 100000000},
		{Date: startDate.AddDate(0, 0, 1), Equity: 101000000}, // +1%
		{Date: startDate.AddDate(0, 0, 2), Equity: 102000000}, // +1%
		{Date: startDate.AddDate(0, 0, 3), Equity: 101500000}, // -0.5%
		{Date: startDate.AddDate(0, 0, 4), Equity: 103000000}, // +1.5%
		{Date: startDate.AddDate(0, 0, 5), Equity: 104000000}, // +1%
		{Date: startDate.AddDate(0, 0, 6), Equity: 103500000}, // -0.5%
		{Date: startDate.AddDate(0, 0, 7), Equity: 105000000}, // +1.5%
	}

	trades := []*ClosedTrade{} // Empty trades, just testing Sharpe calculation

	metrics, err := CalculateComprehensiveMetrics(trades, equityCurve, 100000000)
	if err != nil {
		t.Fatalf("Failed to calculate metrics: %v", err)
	}

	// Sharpe ratio should be calculated (may be 0 or negative with small sample)
	// We're just testing it doesn't crash and is a valid number
	if math.IsNaN(metrics.SharpeRatio) || math.IsInf(metrics.SharpeRatio, 0) {
		t.Errorf("Expected valid Sharpe ratio, got %.2f", metrics.SharpeRatio)
	}

	t.Logf("Sharpe Ratio: %.2f", metrics.SharpeRatio)
}

func TestDrawdownCalculation(t *testing.T) {
	// Equity curve with known drawdown: peak at 110M, trough at 98M
	equityCurve := []EquityPoint{
		{Equity: 100000000},
		{Equity: 105000000},
		{Equity: 110000000}, // Peak
		{Equity: 102000000}, // Drawdown starts
		{Equity: 98000000},  // Trough (-10.91%)
		{Equity: 103000000},
		{Equity: 115000000}, // New peak
	}

	trades := []*ClosedTrade{}

	metrics, err := CalculateComprehensiveMetrics(trades, equityCurve, 100000000)
	if err != nil {
		t.Fatalf("Failed to calculate metrics: %v", err)
	}

	// Expected drawdown: 98M - 110M = -12M
	expectedDrawdown := -12000000.0
	if math.Abs(metrics.MaxDrawdown-expectedDrawdown) > 100 { // Allow small tolerance
		t.Errorf("Expected max drawdown %.2f, got %.2f", expectedDrawdown, metrics.MaxDrawdown)
	}

	// Expected drawdown percent: -12M / 110M = -10.909%
	expectedDrawdownPct := -10.909
	if math.Abs(metrics.MaxDrawdownPercent-expectedDrawdownPct) > 0.1 { // Allow tolerance
		t.Errorf("Expected max drawdown %% %.2f, got %.2f", expectedDrawdownPct, metrics.MaxDrawdownPercent)
	}

	// Verify it's negative
	if metrics.MaxDrawdown >= 0 {
		t.Error("Expected negative drawdown, got non-negative")
	}

	t.Logf("Max Drawdown: %.2f (%.2f%%)", metrics.MaxDrawdown, metrics.MaxDrawdownPercent)
}

func TestBreakdownAnalysis(t *testing.T) {
	// Trades from different signal types
	trades := []*ClosedTrade{
		{SignalType: "pullback", PnL: 2000000, RMultiple: 2.0},
		{SignalType: "pullback", PnL: 1500000, RMultiple: 1.5},
		{SignalType: "pullback", PnL: -500000, RMultiple: -0.5},
		{SignalType: "breakout", PnL: 3000000, RMultiple: 3.0},
		{SignalType: "breakout", PnL: -1000000, RMultiple: -1.0},
		{SignalType: "crossover", PnL: 1000000, RMultiple: 1.0},
	}

	metrics, err := CalculateComprehensiveMetrics(trades, []EquityPoint{}, 100000000)
	if err != nil {
		t.Fatalf("Failed to calculate metrics: %v", err)
	}

	// Verify breakdown by signal type
	if len(metrics.BySignalType) != 3 {
		t.Errorf("Expected 3 signal types, got %d", len(metrics.BySignalType))
	}

	// Check pullback stats (3 trades: 2 wins, 1 loss)
	pullbackMetrics := metrics.BySignalType["pullback"]
	if pullbackMetrics == nil {
		t.Fatal("Pullback metrics not found")
	}
	if pullbackMetrics.TotalTrades != 3 {
		t.Errorf("Expected 3 pullback trades, got %d", pullbackMetrics.TotalTrades)
	}
	expectedPullbackWinRate := 66.67 // 2/3
	if math.Abs(pullbackMetrics.WinRate-expectedPullbackWinRate) > 0.1 {
		t.Errorf("Expected pullback win rate %.2f%%, got %.2f%%", expectedPullbackWinRate, pullbackMetrics.WinRate)
	}

	// Check breakout stats (2 trades: 1 win, 1 loss)
	breakoutMetrics := metrics.BySignalType["breakout"]
	if breakoutMetrics == nil {
		t.Fatal("Breakout metrics not found")
	}
	if breakoutMetrics.TotalTrades != 2 {
		t.Errorf("Expected 2 breakout trades, got %d", breakoutMetrics.TotalTrades)
	}

	t.Logf("Pullback: %d trades, %.2f%% win rate, %.2fR avg",
		pullbackMetrics.TotalTrades, pullbackMetrics.WinRate, pullbackMetrics.AvgRMultiple)
	t.Logf("Breakout: %d trades, %.2f%% win rate, %.2fR avg",
		breakoutMetrics.TotalTrades, breakoutMetrics.WinRate, breakoutMetrics.AvgRMultiple)
}

func TestStreakTracking(t *testing.T) {
	// Trade sequence: W, W, W, W, L, W, L, W, W (max win streak = 4)
	trades := []*ClosedTrade{
		{PnL: 1000000, RMultiple: 1.0},  // W
		{PnL: 1500000, RMultiple: 1.5},  // W
		{PnL: 2000000, RMultiple: 2.0},  // W
		{PnL: 1200000, RMultiple: 1.2},  // W (streak of 4)
		{PnL: -500000, RMultiple: -0.5}, // L
		{PnL: 1000000, RMultiple: 1.0},  // W
		{PnL: -300000, RMultiple: -0.3}, // L
		{PnL: 800000, RMultiple: 0.8},   // W
		{PnL: 1100000, RMultiple: 1.1},  // W
	}

	metrics, err := CalculateComprehensiveMetrics(trades, []EquityPoint{}, 100000000)
	if err != nil {
		t.Fatalf("Failed to calculate metrics: %v", err)
	}

	// Verify longest win streak
	expectedWinStreak := 4
	if metrics.LongestWinStreak != expectedWinStreak {
		t.Errorf("Expected longest win streak %d, got %d", expectedWinStreak, metrics.LongestWinStreak)
	}

	// Verify longest loss streak
	expectedLossStreak := 1
	if metrics.LongestLossStreak != expectedLossStreak {
		t.Errorf("Expected longest loss streak %d, got %d", expectedLossStreak, metrics.LongestLossStreak)
	}

	t.Logf("Longest win streak: %d, Longest loss streak: %d",
		metrics.LongestWinStreak, metrics.LongestLossStreak)
}

func TestHoldingPeriodStats(t *testing.T) {
	trades := []*ClosedTrade{
		{HoldingDays: 5},
		{HoldingDays: 8},
		{HoldingDays: 12},
		{HoldingDays: 3},
		{HoldingDays: 15},
	}

	metrics, err := CalculateComprehensiveMetrics(trades, []EquityPoint{}, 100000000)
	if err != nil {
		t.Fatalf("Failed to calculate metrics: %v", err)
	}

	// Expected average: (5 + 8 + 12 + 3 + 15) / 5 = 8.6 days (integer division = 8)
	expectedAvgDays := 8
	if metrics.AvgHoldingDays != expectedAvgDays {
		t.Errorf("Expected avg holding days %d, got %d", expectedAvgDays, metrics.AvgHoldingDays)
	}

	t.Logf("Average holding period: %d days", metrics.AvgHoldingDays)
}

func TestNoTrades(t *testing.T) {
	// Handle edge case of no trades
	trades := []*ClosedTrade{}

	metrics, err := CalculateComprehensiveMetrics(trades, []EquityPoint{}, 100000000)
	if err != nil {
		t.Fatalf("Failed to calculate metrics: %v", err)
	}

	if metrics.TotalTrades != 0 {
		t.Errorf("Expected 0 trades, got %d", metrics.TotalTrades)
	}
	if metrics.WinRate != 0 {
		t.Errorf("Expected 0%% win rate, got %.2f%%", metrics.WinRate)
	}
}
