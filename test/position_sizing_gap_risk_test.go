package test

import (
	"testing"

	"github.com/nonobeam/golang-stock-trading/internal/risk"
)

// TestGapRiskDivisor verifies Criterion 6.2: Fixed 3.25 divisor for Vietnam gap risk.
func TestGapRiskDivisor(t *testing.T) {
	sizer := risk.NewPositionSizer(100_000_000) // 100M VND capital

	tests := []struct {
		name               string
		entryPrice         float64
		stopPrice          float64
		tradeScore         int
		marketRegime       string
		expectedDivisor    float64
		expectedHasWarning bool
	}{
		{
			name:               "Standard trade with 3.25 divisor",
			entryPrice:         50000,
			stopPrice:          48000,
			tradeScore:         9,
			marketRegime:       "bull",
			expectedDivisor:    risk.VN_GAP_RISK_DIVISOR,
			expectedHasWarning: true,
		},
		{
			name:               "High score trade still uses 3.25",
			entryPrice:         60000,
			stopPrice:          57000,
			tradeScore:         11,
			marketRegime:       "bull",
			expectedDivisor:    risk.VN_GAP_RISK_DIVISOR,
			expectedHasWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sizer.Calculate(
				tt.entryPrice,
				tt.stopPrice,
				tt.tradeScore,
				tt.marketRegime,
				2500, // ATR
				0.3,  // correlation
				"SPY",
				2, 1, // gap count, max consecutive
			)

			if err != nil {
				t.Fatalf("Calculate() error = %v", err)
			}

			// Verify gap risk multiplier is always 3.25
			if result.GapRiskMultiplier != tt.expectedDivisor {
				t.Errorf("GapRiskMultiplier = %v, want %v", result.GapRiskMultiplier, tt.expectedDivisor)
			}

			// Verify worst-case loss warning exists
			if tt.expectedHasWarning && result.GapRiskWarning == "" {
				t.Error("Expected GapRiskWarning to be populated")
			}

			// Verify worst-case loss is calculated as 19.5% of position value
			expectedWorstCaseLoss := result.PositionValue * 0.195
			if diff := abs(result.WorstCaseLossVND - expectedWorstCaseLoss); diff > 1 {
				t.Errorf("WorstCaseLossVND = %v, want ~%v (diff: %v)",
					result.WorstCaseLossVND, expectedWorstCaseLoss, diff)
			}

			// Verify position size was adjusted correctly
			// Expected ratio = (1 / gapRiskDivisor) * volatilityFactor * correlationFactor
			// For test 1: ATR% = 2500/50000 = 5% → volatility factor = 0.8 (high)
			// For test 2: ATR% = 2500/60000 = 4.17% → volatility factor = 0.8 (high)
			// Correlation = 0.3 → correlation factor = 1.0 (low)
			// Expected: (1/3.25) * 0.8 * 1.0 = 0.246
			
			if result.BasePositionSize == 0 {
				t.Error("BasePositionSize should not be zero")
			}

			actualRatio := float64(result.PositionSize) / float64(result.BasePositionSize)
			
			// Calculate expected ratio from actual factors returned by the calculation
			expectedRatio := (1.0 / risk.VN_GAP_RISK_DIVISOR) * result.VolatilityFactor * result.CorrelationFactor

			// Only verify ratio if max position limit was NOT applied
			// If WasCapitalAdjusted is true, the position was further reduced by max position %
			if !result.WasCapitalAdjusted {
				// Allow for lot size rounding (15% tolerance)
				// Vietnam lot size is 100 shares, so rounding can cause significant deviations
				// Example: 172 shares → 100 shares after rounding (42% diff)
				if diff := abs(actualRatio - expectedRatio); diff > 0.15 {
					t.Errorf("Position size ratio = %.4f, want ~%.4f (1/%.2f * %.2f * %.2f) - difference %.4f",
						actualRatio, expectedRatio, risk.VN_GAP_RISK_DIVISOR, 
						result.VolatilityFactor, result.CorrelationFactor, diff)
				}
			}

			t.Logf("✓ Base position: %d shares", result.BasePositionSize)
			t.Logf("✓ Actual position: %d shares (%.2f%% of base)",
				result.PositionSize, actualRatio*100)
			t.Logf("✓ Gap risk divisor: %.2f, Volatility factor: %.2f, Correlation factor: %.2f",
				result.GapRiskMultiplier, result.VolatilityFactor, result.CorrelationFactor)
			if result.WasCapitalAdjusted {
				t.Logf("✓ Position was further reduced by max position limit (20%% of capital)")
			}
			t.Logf("✓ Worst-case loss: %.0f VND (%.2f%% of capital)",
				result.WorstCaseLossVND, result.WorstCaseLossPercent)
			t.Logf("✓ Warning: %s", result.GapRiskWarning)
		})
	}
}

// Helper functions
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
