package test

import (
	"testing"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/regime"
	"github.com/nonobeam/golang-stock-trading/internal/statistics"
)

func TestComprehensiveStatistics_Integration(t *testing.T) {
	now := time.Now()

	// Create realistic trade set with regime data
	trades := make([]statistics.Trade, 50)
	for i := 0; i < 50; i++ {
		pnl := float64((i%3)-1) * 1000000.0
		if i%2 == 0 {
			pnl = float64(i%5+1) * 500000.0
		}

		var regimeType regime.RegimeType
		switch i % 3 {
		case 0:
			regimeType = regime.RegimeStrongBull
		case 1:
			regimeType = regime.RegimeRangeBound
		default:
			regimeType = regime.RegimeStrongBear
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
			Regime:      regimeType,
		}
	}

	// Create comprehensive calculator
	calc := statistics.NewVNComprehensiveCalculator()

	// Generate comprehensive report
	report, err := calc.Generate(trades, 100000000)
	if err != nil {
		t.Fatalf("Failed to generate comprehensive report: %v", err)
	}

	// Validate report structure
	if report == nil {
		t.Fatal("Report is nil")
	}

	// Check basic stats calculated
	if report.TotalTrades != 50 {
		t.Errorf("Expected 50 trades, got %d", report.TotalTrades)
	}

	// Check VN metrics calculated
	if report.VNMetrics.CapitalEfficiency == 0 {
		t.Error("Capital efficiency not calculated")
	}

	// Check health score
	if report.Health.Score < 0 || report.Health.Score > 100 {
		t.Errorf("Health score out of bounds: %d", report.Health.Score)
	}

	// Check recommendations generated
	if len(report.Recommendations) == 0 {
		t.Error("No recommendations generated")
	}

	t.Logf("System Health: %s (%d/100)", report.Health.Rating, report.Health.Score)
	t.Logf("Should Trade: %v", report.Health.ShouldTrade)
	t.Logf("Recommendations: %d", len(report.Recommendations))
}
