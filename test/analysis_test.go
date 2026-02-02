package test

import (
	"testing"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/analysis"
	"github.com/nonobeam/golang-stock-trading/internal/data"
)

// --- Analysis Manager Tests ---

func TestStockAnalyzer_AddBarAndAnalyze(t *testing.T) {
	analyzer := analysis.NewStockAnalyzer("VNM", 100)

	// Add 60 bars (enough for all indicators)
	basePrice := 50000.0
	for i := 0; i < 60; i++ {
		// Simulate slight uptrend with volatility
		price := basePrice + float64(i)*100 + float64(i%5)*50
		bar := data.NewOHLCV(
			time.Now().Add(time.Duration(i)*time.Minute),
			price-50, price+100, price-100, price, 1000000,
		)
		analyzer.AddBar(bar)
	}

	if analyzer.BarCount() != 60 {
		t.Errorf("BarCount() = %d, want 60", analyzer.BarCount())
	}

	result, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error: %v", err)
	}

	// Check that indicators are calculated
	if result.SMA20 == 0 {
		t.Error("SMA20 should not be zero")
	}
	if result.RSI == 0 {
		t.Error("RSI should not be zero")
	}
	if result.ATR == 0 {
		t.Error("ATR should not be zero")
	}

	t.Logf("Analysis result: SMA20=%.2f, RSI=%.2f, ATR=%.2f, ADX=%.2f",
		result.SMA20, result.RSI, result.ATR, result.ADX)
}

func TestAnalysisManager_MultipleSymbols(t *testing.T) {
	manager := analysis.NewAnalysisManager()

	symbols := []string{"VNM", "HPG", "FPT"}
	for _, sym := range symbols {
		analyzer := manager.GetOrCreate(sym)
		if analyzer.Symbol() != sym {
			t.Errorf("Symbol mismatch: got %s, want %s", analyzer.Symbol(), sym)
		}
	}

	tracked := manager.Symbols()
	if len(tracked) != 3 {
		t.Errorf("Symbols() returned %d, want 3", len(tracked))
	}
}
