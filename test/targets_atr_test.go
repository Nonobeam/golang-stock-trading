package test

import (
	"testing"

	"github.com/nonobeam/golang-stock-trading/internal/risk"
)

func TestATRTargets(t *testing.T) {
	t.Run("Standard ATR multipliers", func(t *testing.T) {
		result := risk.CalculateATRTargets(52000, 2500, nil, true)

		if result.Method != "ATR-Based" {
			t.Errorf("Expected method ATR-Based, got %s", result.Method)
		}

		if len(result.Targets) != 3 {
			t.Fatalf("Expected 3 targets, got %d", len(result.Targets))
		}

		expected := []float64{57000, 59500, 62000}
		for i, exp := range expected {
			if result.Targets[i].TargetPrice != exp {
				t.Errorf("Target %d: expected %.0f, got %.0f",
					i+1, exp, result.Targets[i].TargetPrice)
			}
		}
	})

	t.Run("Custom ATR multipliers", func(t *testing.T) {
		result := risk.CalculateATRTargets(52000, 2500, []float64{1.5, 2.5}, true)

		if len(result.Targets) != 2 {
			t.Fatalf("Expected 2 targets, got %d", len(result.Targets))
		}

		expected := []float64{55750, 58250}
		for i, exp := range expected {
			actual := result.Targets[i].TargetPrice
			if actual < exp-50 || actual > exp+50 {
				t.Errorf("Target %d: expected ~%.0f, got %.0f", i+1, exp, actual)
			}
		}
	})

	t.Run("Invalid ATR", func(t *testing.T) {
		result := risk.CalculateATRTargets(52000, 0, nil, true)

		if len(result.Targets) != 0 {
			t.Errorf("Expected 0 targets for invalid ATR, got %d", len(result.Targets))
		}
	})
}
