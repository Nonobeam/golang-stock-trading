package test

import (
	"testing"

	"github.com/nonobeam/golang-stock-trading/internal/risk"
)

func TestRMultipleTargets(t *testing.T) {
	t.Run("Long position with default R-multiples", func(t *testing.T) {
		result := risk.CalculateRMultipleTargets(52000, 49000, nil, true)

		if result.Method != "R-Multiple" {
			t.Errorf("Expected method R-Multiple, got %s", result.Method)
		}

		if result.RiskPerShare != 3000 {
			t.Errorf("Expected risk 3000, got %.0f", result.RiskPerShare)
		}

		if len(result.Targets) != 3 {
			t.Fatalf("Expected 3 targets, got %d", len(result.Targets))
		}

		expected := []float64{58000, 61000, 64000}
		for i, exp := range expected {
			if result.Targets[i].TargetPrice != exp {
				t.Errorf("Target %d: expected %.0f, got %.0f",
					i+1, exp, result.Targets[i].TargetPrice)
			}
		}

		if result.Targets[0].RMultiple != 2.0 {
			t.Errorf("Target 1 R-multiple: expected 2.0, got %.1f", result.Targets[0].RMultiple)
		}
	})

	t.Run("Custom R-multiples", func(t *testing.T) {
		result := risk.CalculateRMultipleTargets(52000, 49000, []float64{1.5, 2.5, 3.5}, true)

		if len(result.Targets) != 3 {
			t.Fatalf("Expected 3 targets, got %d", len(result.Targets))
		}

		expected := []float64{56500, 59500, 62500}
		for i, exp := range expected {
			if result.Targets[i].TargetPrice != exp {
				t.Errorf("Target %d: expected %.0f, got %.0f",
					i+1, exp, result.Targets[i].TargetPrice)
			}
		}
	})

	t.Run("Short position", func(t *testing.T) {
		result := risk.CalculateRMultipleTargets(52000, 55000, []float64{2.0}, false)

		if result.RiskPerShare != 3000 {
			t.Errorf("Expected risk 3000, got %.0f", result.RiskPerShare)
		}

		if result.Targets[0].TargetPrice != 46000 {
			t.Errorf("Expected target 46000, got %.0f", result.Targets[0].TargetPrice)
		}
	})

	t.Run("Rounding to nearest 100", func(t *testing.T) {
		result := risk.CalculateRMultipleTargets(50123, 48500, []float64{2.0}, true)

		target := result.Targets[0].TargetPrice
		if int(target)%100 != 0 {
			t.Errorf("Target not rounded to 100: %.0f", target)
		}
	})
}
