package test

import (
	"testing"

	"github.com/nonobeam/golang-stock-trading/internal/risk"
)

// --- Basic Position Sizing Tests ---

func TestFixedRisk_PositionSize(t *testing.T) {
	params := risk.RiskParams{
		AccountBalance: 100000000, // 100M VND
		RiskPercent:    0.01,      // 1%
		EntryPrice:     50000,
		StopPrice:      48000, // 2000 VND stop distance
	}

	size, err := risk.FixedRisk(params)
	if err != nil {
		t.Fatalf("FixedRisk error: %v", err)
	}

	// (100M * 0.01) / 2000 = 500 shares
	expected := 500
	if size != expected {
		t.Errorf("FixedRisk = %d, want %d", size, expected)
	}
}

func TestFixedRiskDetailed(t *testing.T) {
	params := risk.RiskParams{
		AccountBalance: 100000000,
		RiskPercent:    0.015, // 1.5%
		EntryPrice:     52000,
		StopPrice:      49000,
	}

	result, err := risk.FixedRiskDetailed(params)
	if err != nil {
		t.Fatalf("FixedRiskDetailed error: %v", err)
	}

	if result.PositionSize != 500 {
		t.Errorf("PositionSize = %d, want 500", result.PositionSize)
	}
	if result.RiskPerShare != 3000 {
		t.Errorf("RiskPerShare = %.0f, want 3000", result.RiskPerShare)
	}
	if result.PositionValue != 26000000 {
		t.Errorf("PositionValue = %.0f, want 26000000", result.PositionValue)
	}
	t.Logf("Position: %d shares, Value: %.0f, Risk: %.2f%%",
		result.PositionSize, result.PositionValue, result.RiskPercent)
}

func TestScoreBased_Scaling(t *testing.T) {
	baseSize := 500

	tests := []struct {
		score    int
		expected int
	}{
		{5, 200},  // 0.5x = 250, rounded to 200
		{8, 500},  // 1.0x
		{10, 600}, // 1.25x = 625, rounded to 600
		{12, 700}, // 1.5x = 750, rounded to 700
	}

	for _, tt := range tests {
		result := risk.ScoreBased(baseSize, tt.score)
		if result != tt.expected {
			t.Errorf("ScoreBased(%d, score=%d) = %d, want %d", baseSize, tt.score, result, tt.expected)
		}
	}
}

// --- Volatility Adjustment Tests ---

func TestVolatilityFactor(t *testing.T) {
	tests := []struct {
		atr            float64
		price          float64
		expectedFactor float64
		expectedClass  string
	}{
		{1200, 50000, 1.2, "low"},     // 2.4% ATR (< 3%)
		{2000, 50000, 1.0, "normal"},  // 4% ATR (3-5%)
		{3500, 50000, 0.8, "high"},    // 7% ATR (5-8%)
		{5000, 50000, 0.6, "extreme"}, // 10% ATR (> 8%)
	}

	for _, tt := range tests {
		result := risk.CalculateVolatilityFactor(tt.atr, tt.price)
		if result.Factor != tt.expectedFactor {
			t.Errorf("VolatilityFactor(atr=%.0f, price=%.0f) = %.1f, want %.1f",
				tt.atr, tt.price, result.Factor, tt.expectedFactor)
		}
		if result.Classification != tt.expectedClass {
			t.Errorf("Classification = %s, want %s", result.Classification, tt.expectedClass)
		}
	}
}

// --- Capital Constraint Tests ---

func TestApplyMaxPositionLimit(t *testing.T) {
	// Position exceeding 20% limit
	positionSize := 1000
	entryPrice := 50000.0
	totalCapital := 100000000.0
	maxPositionPct := 20.0

	// Position value = 50M (50% of capital) - should be reduced
	adjusted, wasAdjusted := risk.ApplyMaxPositionLimit(positionSize, entryPrice, totalCapital, maxPositionPct)

	if !wasAdjusted {
		t.Error("Expected position to be adjusted")
	}

	// Max 20% = 20M / 50000 = 400 shares
	if adjusted != 400 {
		t.Errorf("Adjusted position = %d, want 400", adjusted)
	}
}

// --- Score-Based Risk Tests ---

func TestGetRiskPercentByScore(t *testing.T) {
	tests := []struct {
		score        int
		regime       string
		expectedRisk float64
	}{
		{6, "bull", 0.0},            // Score too low
		{7, "bull", 0.01},           // 1%
		{9, "bull", 0.015},          // 1.5%
		{11, "bull", 0.02},          // 2%
		{11, "bear", 0.01},          // 2% * 0.5 = 1%
		{9, "range", 0.01125},       // 1.5% * 0.75
	}

	for _, tt := range tests {
		result := risk.GetRiskPercentByScore(tt.score, tt.regime)
		if result != tt.expectedRisk {
			t.Errorf("GetRiskPercentByScore(score=%d, %s) = %.4f, want %.4f",
				tt.score, tt.regime, result, tt.expectedRisk)
		}
	}
}

// --- Correlation Factor Tests ---

func TestGetCorrelationFactor(t *testing.T) {
	tests := []struct {
		correlation    float64
		expectedFactor float64
	}{
		{0.9, 0.0},   // Too high, skip
		{0.75, 0.5},  // High, reduce 50%
		{0.6, 0.8},   // Moderate, reduce 20%
		{0.3, 1.0},   // Low, no adjustment
	}

	for _, tt := range tests {
		factor, _ := risk.GetCorrelationFactor(tt.correlation)
		if factor != tt.expectedFactor {
			t.Errorf("GetCorrelationFactor(%.2f) = %.1f, want %.1f",
				tt.correlation, factor, tt.expectedFactor)
		}
	}
}

// --- Gap Risk Tests ---

func TestGetGapRiskMultiplier(t *testing.T) {
	tests := []struct {
		gapCount       int
		maxConsecutive int
		expected       float64
	}{
		{0, 0, 1.5},  // Minimal risk
		{2, 1, 2.0},  // Some risk
		{4, 2, 2.5},  // Moderate risk
		{6, 3, 3.0},  // High risk
	}

	for _, tt := range tests {
		result := risk.GetGapRiskMultiplier(tt.gapCount, tt.maxConsecutive)
		if result != tt.expected {
			t.Errorf("GetGapRiskMultiplier(%d, %d) = %.1f, want %.1f",
				tt.gapCount, tt.maxConsecutive, result, tt.expected)
		}
	}
}

// --- Position Sizer Integration Tests ---

func TestPositionSizer_CalculateSimple(t *testing.T) {
	sizer := risk.NewPositionSizer(100000000) // 100M VND

	result, err := sizer.CalculateSimple(
		50000, // Entry
		47000, // Stop (6% away)
		0.015, // 1.5% risk
		2500,  // ATR
	)

	if err != nil {
		t.Fatalf("CalculateSimple error: %v", err)
	}

	if !result.ShouldTrade {
		t.Error("Expected ShouldTrade = true")
	}
	if result.PositionSize <= 0 {
		t.Error("Expected position size > 0")
	}
	if result.PositionPercent > 20 {
		t.Errorf("Position percent %.1f%% exceeds 20%% limit", result.PositionPercent)
	}

	t.Logf("Position: %d shares, Value: %.0f VND (%.1f%%), Risk: %.2f%%",
		result.PositionSize, result.PositionValue, result.PositionPercent, result.RiskPercent)
}

func TestPositionSizer_Calculate_Full(t *testing.T) {
	sizer := risk.NewPositionSizer(100000000)

	result, err := sizer.Calculate(
		52000,   // Entry
		49000,   // Stop
		9,       // Trade score
		"bull",  // Market regime
		2500,    // ATR
		0.4,     // Max correlation (low)
		"VCB",   // Correlated with
		1,       // Gap count
		0,       // Max consecutive gaps
	)

	if err != nil {
		t.Fatalf("Calculate error: %v", err)
	}

	if !result.ShouldTrade {
		t.Error("Expected ShouldTrade = true")
	}
	if result.GapRiskMultiplier < 1.5 {
		t.Error("Expected gap risk multiplier >= 1.5")
	}

	t.Logf("Full result: Size=%d, Value=%.0f, Risk=%.2f%%, VolFactor=%.1f, CorrFactor=%.1f, GapMult=%.1f",
		result.PositionSize, result.PositionValue, result.RiskPercent,
		result.VolatilityFactor, result.CorrelationFactor, result.GapRiskMultiplier)
}

func TestPositionSizer_RejectsLowScore(t *testing.T) {
	sizer := risk.NewPositionSizer(100000000)

	result, err := sizer.Calculate(
		50000, 47000, 5, "bull", 2500, 0.3, "", 0, 0,
	)

	if err != risk.ErrScoreTooLow {
		t.Errorf("Expected ErrScoreTooLow, got: %v", err)
	}
	if result.ShouldTrade {
		t.Error("Expected ShouldTrade = false for low score")
	}
}

func TestPositionSizer_RejectsHighCorrelation(t *testing.T) {
	sizer := risk.NewPositionSizer(100000000)

	result, err := sizer.Calculate(
		50000, 47000, 9, "bull", 2500, 0.9, "VCB", 0, 0,
	)

	if err != risk.ErrCorrelationTooHigh {
		t.Errorf("Expected ErrCorrelationTooHigh, got: %v", err)
	}
	if result.ShouldTrade {
		t.Error("Expected ShouldTrade = false for high correlation")
	}
}
