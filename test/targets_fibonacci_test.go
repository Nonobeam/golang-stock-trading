package test

import (
	"testing"

	"github.com/nonobeam/golang-stock-trading/internal/risk"
)

func TestFibonacciExtensions(t *testing.T) {
	t.Run("Standard Fibonacci ratios", func(t *testing.T) {
		params := risk.FibonacciParams{
			SwingLow:    45000,
			SwingHigh:   60000,
			PullbackLow: 52000,
		}

		result := risk.CalculateFibonacciExtensions(params)

		if result.Method != "Fibonacci Extensions" {
			t.Errorf("Expected method Fibonacci Extensions, got %s", result.Method)
		}

		if result.WaveSize != 15000 {
			t.Errorf("Expected wave size 15000, got %.0f", result.WaveSize)
		}

		if len(result.Targets) != 4 {
			t.Fatalf("Expected 4 targets, got %d", len(result.Targets))
		}

		expected := []float64{61300, 67000, 76300, 91300}
		tolerance := 100.0

		for i, exp := range expected {
			actual := result.Targets[i].TargetPrice
			if actual < exp-tolerance || actual > exp+tolerance {
				t.Errorf("Target %d: expected ~%.0f, got %.0f", i+1, exp, actual)
			}
		}
	})

	t.Run("Custom Fib ratios", func(t *testing.T) {
		params := risk.FibonacciParams{
			SwingLow:    45000,
			SwingHigh:   60000,
			PullbackLow: 52000,
			FibRatios:   []float64{1.0, 1.618},
		}

		result := risk.CalculateFibonacciExtensions(params)

		if len(result.Targets) != 2 {
			t.Fatalf("Expected 2 targets, got %d", len(result.Targets))
		}
	})

	t.Run("Invalid swing data", func(t *testing.T) {
		params := risk.FibonacciParams{
			SwingLow:    60000,
			SwingHigh:   45000,
			PullbackLow: 52000,
		}

		result := risk.CalculateFibonacciExtensions(params)

		if len(result.Targets) != 0 {
			t.Errorf("Expected 0 targets for invalid swing, got %d", len(result.Targets))
		}
	})
}
