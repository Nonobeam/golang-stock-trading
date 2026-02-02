package test

import (
	"testing"

	"github.com/nonobeam/golang-stock-trading/internal/analysis/indicators"
	"github.com/nonobeam/golang-stock-trading/internal/vn"
)

// --- Indicator Tests ---

func TestSMA_Calculate(t *testing.T) {
	prices := []float64{50000, 51000, 52000, 53000, 54000}
	sma, err := indicators.CalculateSMA(prices, 3)
	if err != nil {
		t.Fatalf("CalculateSMA error: %v", err)
	}

	expected := (52000.0 + 53000.0 + 54000.0) / 3 // 53000
	if sma != expected {
		t.Errorf("SMA(3) = %v, want %v", sma, expected)
	}
}

func TestEMA_Calculate(t *testing.T) {
	prices := []float64{50000, 51000, 52000, 53000, 54000, 55000}
	ema, err := indicators.CalculateEMA(prices, 3)
	if err != nil {
		t.Fatalf("CalculateEMA error: %v", err)
	}

	// EMA should be calculated
	if ema == 0 {
		t.Error("EMA should not be zero")
	}
	t.Logf("EMA(3) = %.2f", ema)
}

func TestRSI_Calculate(t *testing.T) {
	// Create prices with alternating up/down movements
	prices := make([]float64, 20)
	prices[0] = 50000
	for i := 1; i < 20; i++ {
		if i%2 == 0 {
			prices[i] = prices[i-1] + 500
		} else {
			prices[i] = prices[i-1] - 300
		}
	}

	rsi, err := indicators.CalculateRSI(prices, 14)
	if err != nil {
		t.Fatalf("CalculateRSI error: %v", err)
	}

	if rsi < 0 || rsi > 100 {
		t.Errorf("RSI should be between 0-100, got %v", rsi)
	}
	t.Logf("RSI(14) = %.2f", rsi)
}

// --- Vietnam Market Tests ---

func TestVN_CalculateLimits(t *testing.T) {
	limits := vn.CalculateLimits(50000)

	if limits.Ceiling != 53500 { // 50000 * 1.07 = 53500
		t.Errorf("Ceiling = %v, want 53500", limits.Ceiling)
	}
	if limits.Floor != 46500 { // 50000 * 0.93 = 46500
		t.Errorf("Floor = %v, want 46500", limits.Floor)
	}
}

func TestVN_ValidateOrderPrice(t *testing.T) {
	limits := vn.CalculateLimits(50000)

	// Valid price
	err := vn.ValidateOrderPrice(52000, limits)
	if err != nil {
		t.Errorf("Expected valid price, got error: %v", err)
	}

	// Price above ceiling
	err = vn.ValidateOrderPrice(54000, limits)
	if err != vn.ErrPriceAboveCeiling {
		t.Errorf("Expected ErrPriceAboveCeiling, got: %v", err)
	}

	// Price below floor
	err = vn.ValidateOrderPrice(45000, limits)
	if err != vn.ErrPriceBelowFloor {
		t.Errorf("Expected ErrPriceBelowFloor, got: %v", err)
	}
}

func TestVN_RoundToTick(t *testing.T) {
	tests := []struct {
		price    float64
		expected float64
	}{
		{5555, 5560},   // Under 10K: tick=10
		{25033, 25050}, // 10K-50K: tick=50
		{55555, 55600}, // 50K+: tick=100
	}

	for _, tt := range tests {
		result := vn.RoundToTick(tt.price)
		if result != tt.expected {
			t.Errorf("RoundToTick(%v) = %v, want %v", tt.price, result, tt.expected)
		}
	}
}
