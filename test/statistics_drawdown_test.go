package test

import (
	"testing"

	"github.com/nonobeam/golang-stock-trading/internal/statistics"
)

func TestDrawdown_MaximumDrawdown(t *testing.T) {
	equityCurve := []float64{
		100000000, 105000000, 110000000, 108000000,
		102000000, 98000000, 105000000, 112000000,
		108000000, 115000000,
	}

	metrics := statistics.AnalyzeDrawdowns(equityCurve, nil)

	peak := 110000000.0
	trough := 98000000.0
	expectedDD := (peak - trough) / peak * 100

	if !almostEqual(metrics.MaxDrawdownPercent, expectedDD, 0.1) {
		t.Errorf("Expected max DD ~%.2f%%, got %.2f%%",
			expectedDD, metrics.MaxDrawdownPercent)
	}

	if metrics.MaxDDPeakIdx != 2 {
		t.Errorf("Expected peak idx 2, got %d", metrics.MaxDDPeakIdx)
	}

	if metrics.MaxDDTroughIdx != 5 {
		t.Errorf("Expected trough idx 5, got %d", metrics.MaxDDTroughIdx)
	}

	if !metrics.Recovered {
		t.Error("Expected drawdown to have recovered")
	}

	if metrics.RecoveryIdx == nil || *metrics.RecoveryIdx != 7 {
		t.Errorf("Expected recovery at idx 7")
	}

	t.Logf("Max DD: %.2f%%, Peak: %.0f, Trough: %.0f, Recovered: %v",
		metrics.MaxDrawdownPercent,
		metrics.MaxDDPeakValue,
		metrics.MaxDDTroughValue,
		metrics.Recovered)
}

func TestDrawdown_CurrentlyInDrawdown(t *testing.T) {
	equityCurve := []float64{
		100000000, 110000000, 105000000, 95000000,
	}

	metrics := statistics.AnalyzeDrawdowns(equityCurve, nil)

	if metrics.Recovered {
		t.Error("Expected drawdown to not be recovered")
	}

	if metrics.RecoveryIdx != nil {
		t.Error("Expected recovery idx to be nil")
	}

	if metrics.CurrentDrawdownPercent == 0 {
		t.Error("Expected current drawdown to be non-zero")
	}

	t.Logf("Currently in drawdown: %.2f%%", metrics.CurrentDrawdownPercent)
}

func TestDrawdown_NoDrawdown(t *testing.T) {
	equityCurve := []float64{
		100000000, 105000000, 110000000, 115000000, 120000000,
	}

	metrics := statistics.AnalyzeDrawdowns(equityCurve, nil)

	if metrics.MaxDrawdownPercent > 0.01 {
		t.Errorf("Expected minimal drawdown, got %.2f%%", metrics.MaxDrawdownPercent)
	}

	t.Logf("No significant drawdown detected: %.4f%%", metrics.MaxDrawdownPercent)
}

func TestRecoveryFactor(t *testing.T) {
	netProfit := 30000000.0
	maxDrawdown := 12.0

	factor, interpretation := statistics.CalculateRecoveryFactor(netProfit, maxDrawdown)

	expected := netProfit / maxDrawdown
	if !almostEqual(factor, expected, 0.01) {
		t.Errorf("Expected recovery factor ~%.2f, got %.2f", expected, factor)
	}

	if factor < 2.0 {
		t.Errorf("Expected factor >= 2.0, got %.2f", factor)
	}

	t.Logf("Recovery Factor: %.2f, Interpretation: %s", factor, interpretation)
}

func TestRecoveryFactor_Exceptional(t *testing.T) {
	factor, interp := statistics.CalculateRecoveryFactor(100000000, 0)

	if factor <= 0 {
		t.Errorf("Expected infinite or very high factor, got %.2f", factor)
	}

	if interp != "No drawdown - exceptional" {
		t.Errorf("Expected exceptional interpretation, got '%s'", interp)
	}
}
