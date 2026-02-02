package test

import (
	"testing"

	"github.com/nonobeam/golang-stock-trading/internal/risk"
)

func TestComprehensiveTargets(t *testing.T) {
	t.Run("Multiple methods with consensus", func(t *testing.T) {
		params := risk.ComprehensiveTargetParams{
			EntryPrice:       52000,
			StopLoss:         49000,
			IsLong:           true,
			ATR:              2500,
			ResistanceLevels: []float64{55000, 58000, 61000, 65000},
		}

		result := risk.CalculateComprehensiveTargets(params)

		if result.EntryPrice != 52000 {
			t.Errorf("Expected entry 52000, got %.0f", result.EntryPrice)
		}

		if len(result.AllMethods) < 2 {
			t.Errorf("Expected at least 2 methods, got %d", len(result.AllMethods))
		}

		if _, ok := result.AllMethods["r_multiple"]; !ok {
			t.Error("Expected r_multiple method in results")
		}

		if _, ok := result.AllMethods["atr"]; !ok {
			t.Error("Expected atr method in results")
		}

		if len(result.ConsensusTargets) == 0 {
			t.Log("Warning: No consensus targets found (may be valid)")
		}

		if len(result.RecommendedStrategy.Targets) == 0 {
			t.Error("Expected recommended strategy to have targets")
		}
	})

	t.Run("Consensus finding with 3% tolerance", func(t *testing.T) {
		params := risk.ComprehensiveTargetParams{
			EntryPrice:       52000,
			StopLoss:         49000,
			IsLong:           true,
			ATR:              2500,
			ResistanceLevels: []float64{58000, 58500, 61000},
			FibParams: &risk.FibonacciParams{
				SwingLow:    45000,
				SwingHigh:   60000,
				PullbackLow: 52000,
			},
		}

		result := risk.CalculateComprehensiveTargets(params)

		if len(result.ConsensusTargets) > 0 {
			for _, ct := range result.ConsensusTargets {
				if ct.NumMethodsAgree < 2 {
					t.Errorf("Consensus target should have at least 2 methods, got %d", ct.NumMethodsAgree)
				}

				if ct.Confidence == "High" && ct.NumMethodsAgree < 3 {
					t.Errorf("High confidence requires 3+ methods, got %d", ct.NumMethodsAgree)
				}
			}
		}
	})

	t.Run("Strategy generation", func(t *testing.T) {
		params := risk.ComprehensiveTargetParams{
			EntryPrice: 52000,
			StopLoss:   49000,
			IsLong:     true,
		}

		result := risk.CalculateComprehensiveTargets(params)

		totalPercent := 0
		for _, target := range result.RecommendedStrategy.Targets {
			totalPercent += target.SellPercent

			if target.Price > 0 && target.Price < params.EntryPrice {
				t.Errorf("Target price %.0f should be above entry %.0f", target.Price, params.EntryPrice)
			}
		}

		if totalPercent != 100 {
			t.Errorf("Strategy should total 100%%, got %d%%", totalPercent)
		}
	})
}
