package backtest

import (
	"testing"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/data"
)

func TestGapToCeiling(t *testing.T) {
	simulator := NewTradeSimulator(0.0015, 0.001) // 0.15% commission, 0.1% slippage
	refPrice := 80000.0
	ceiling := refPrice * 1.07 // 85,600

	// Create bar stuck at ceiling
	bar := data.OHLCV{
		Timestamp: time.Now(),
		Open:      ceiling,
		High:      ceiling,
		Low:       ceiling,
		Close:     ceiling,
		Volume:    1000,
	}

	// Try to buy - should be rejected
	fill, err := simulator.SimulateBuy("VCB", 100, ceiling, bar, refPrice)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if fill.Success {
		t.Error("Expected buy to be rejected at ceiling, but it succeeded")
	}

	if fill.RejectionReason == "" {
		t.Error("Expected rejection reason to be set")
	}

	t.Logf("Buy rejected as expected: %s", fill.RejectionReason)
}

func TestGapToFloor(t *testing.T) {
	simulator := NewTradeSimulator(0.0015, 0.001)
	refPrice := 80000.0
	floor := refPrice * 0.93 // 74,400

	// Create bar stuck at floor
	bar := data.OHLCV{
		Timestamp: time.Now(),
		Open:      floor,
		High:      floor,
		Low:       floor,
		Close:     floor,
		Volume:    1000,
	}

	// Try to sell - should apply max slippage
	fill, err := simulator.SimulateSell("VCB", 100, floor, bar, refPrice)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !fill.Success {
		t.Errorf("Expected sell to succeed at floor, got rejection: %s", fill.RejectionReason)
	}

	// At floor, slippage should be doubled
	expectedFillPrice := floor * (1 - 0.001*2)
	if fill.FilledPrice > expectedFillPrice+1 { // Allow small tolerance
		t.Errorf("Expected max slippage at floor, fill price %.2f should be near %.2f",
			fill.FilledPrice, expectedFillPrice)
	}

	t.Logf("Sell at floor with max slippage: %.2f", fill.FilledPrice)
}

func TestNormalBuy(t *testing.T) {
	simulator := NewTradeSimulator(0.0015, 0.001)
	refPrice := 80000.0

	// Normal trading day
	bar := data.OHLCV{
		Timestamp: time.Now(),
		Open:      80500.0,
		High:      81000.0,
		Low:       80000.0,
		Close:     80800.0,
		Volume:    100000,
	}

	// Buy at limit
	limitPrice := 80500.0
	shares := 100

	fill, err := simulator.SimulateBuy("VCB", shares, limitPrice, bar, refPrice)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !fill.Success {
		t.Errorf("Expected successful fill, got rejection: %s", fill.RejectionReason)
	}

	// Verify commission applied
	subtotal := fill.FilledPrice * float64(shares)
	expectedCommission := subtotal * 0.0015
	if fill.Commission < expectedCommission*0.99 || fill.Commission > expectedCommission*1.01 {
		t.Errorf("Expected commission ~%.2f, got %.2f", expectedCommission, fill.Commission)
	}

	// Verify total cost
	expectedTotal := subtotal + fill.Commission
	if fill.TotalCost < expectedTotal*0.99 || fill.TotalCost > expectedTotal*1.01 {
		t.Errorf("Expected total cost ~%.2f, got %.2f", expectedTotal, fill.TotalCost)
	}

	t.Logf("Normal buy: filled at %.2f, commission %.2f, total cost %.2f",
		fill.FilledPrice, fill.Commission, fill.TotalCost)
}

func TestNormalSell(t *testing.T) {
	simulator := NewTradeSimulator(0.0015, 0.001)
	refPrice := 80000.0

	// Normal trading day
	bar := data.OHLCV{
		Timestamp: time.Now(),
		Open:      80500.0,
		High:      81000.0,
		Low:       80000.0,
		Close:     80800.0,
		Volume:    100000,
	}

	// Sell at limit
	limitPrice := 80500.0
	shares := 100

	fill, err := simulator.SimulateSell("VCB", shares, limitPrice, bar, refPrice)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !fill.Success {
		t.Errorf("Expected successful fill, got rejection: %s", fill.RejectionReason)
	}

	// Verify commission deducted from proceeds
	subtotal := fill.FilledPrice * float64(shares)
	expectedCommission := subtotal * 0.0015
	expectedProceeds := subtotal - expectedCommission

	if fill.NetProceeds < expectedProceeds*0.99 || fill.NetProceeds > expectedProceeds*1.01 {
		t.Errorf("Expected net proceeds ~%.2f, got %.2f", expectedProceeds, fill.NetProceeds)
	}

	t.Logf("Normal sell: filled at %.2f, commission %.2f, net proceeds %.2f",
		fill.FilledPrice, fill.Commission, fill.NetProceeds)
}

func TestBuyRejectedBelowLow(t *testing.T) {
	simulator := NewTradeSimulator(0.0015, 0.001)

	bar := data.OHLCV{
		Open:  80500.0,
		High:  81000.0,
		Low:   80000.0,
		Close: 80800.0,
	}

	// Try to buy below the day's low
	limitPrice := 79500.0 // Below bar.Low

	fill, err := simulator.SimulateBuy("VCB", 100, limitPrice, bar, 80000.0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if fill.Success {
		t.Error("Expected buy to be rejected (limit below low)")
	}

	t.Logf("Buy correctly rejected: %s", fill.RejectionReason)
}

func TestSellRejectedAboveHigh(t *testing.T) {
	simulator := NewTradeSimulator(0.0015, 0.001)

	bar := data.OHLCV{
		Open:  80500.0,
		High:  81000.0,
		Low:   80000.0,
		Close: 80800.0,
	}

	// Try to sell above the day's high
	limitPrice := 82000.0 // Above bar.High

	fill, err := simulator.SimulateSell("VCB", 100, limitPrice, bar, 80000.0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if fill.Success {
		t.Error("Expected sell to be rejected (limit above high)")
	}

	t.Logf("Sell correctly rejected: %s", fill.RejectionReason)
}

func TestPriceLimitValidation(t *testing.T) {
	simulator := NewTradeSimulator(0.0015, 0.001)
	refPrice := 80000.0

	// Test HOSE limits (±7%)
	ceiling := refPrice * 1.07
	floor := refPrice * 0.93

	// Within limits should pass
	if err := simulator.ValidateWithinLimits(81000, refPrice, "HOSE"); err != nil {
		t.Errorf("Price 81000 should be within HOSE limits: %v", err)
	}

	// Above ceiling should fail
	if err := simulator.ValidateWithinLimits(ceiling+100, refPrice, "HOSE"); err == nil {
		t.Error("Price above ceiling should fail validation")
	}

	// Below floor should fail
	if err := simulator.ValidateWithinLimits(floor-100, refPrice, "HOSE"); err == nil {
		t.Error("Price below floor should fail validation")
	}

	// Test HNX limits (±10%)
	hnxCeiling := refPrice * 1.10
	if err := simulator.ValidateWithinLimits(hnxCeiling-100, refPrice, "HNX"); err != nil {
		t.Errorf("Price should be within HNX limits: %v", err)
	}
}

func TestGapDetection(t *testing.T) {
	simulator := NewTradeSimulator(0.0015, 0.001)
	prevClose := 80000.0
	ceiling := prevClose * 1.07
	floor := prevClose * 0.93

	// Test gap to ceiling
	ceilingBar := data.OHLCV{
		Open:  ceiling,
		High:  ceiling,
		Low:   ceiling,
		Close: ceiling,
	}
	scenario := simulator.DetectGapScenario(ceilingBar, prevClose)
	if scenario != "gap_to_ceiling" {
		t.Errorf("Expected 'gap_to_ceiling', got '%s'", scenario)
	}

	// Test gap to floor
	floorBar := data.OHLCV{
		Open:  floor,
		High:  floor,
		Low:   floor,
		Close: floor,
	}
	scenario = simulator.DetectGapScenario(floorBar, prevClose)
	if scenario != "gap_to_floor" {
		t.Errorf("Expected 'gap_to_floor', got '%s'", scenario)
	}

	// Test normal day
	normalBar := data.OHLCV{
		Open:  80500,
		High:  81000,
		Low:   80000,
		Close: 80800,
	}
	scenario = simulator.DetectGapScenario(normalBar, prevClose)
	if scenario != "normal" {
		t.Errorf("Expected 'normal', got '%s'", scenario)
	}

	t.Logf("Gap detection working correctly")
}

func TestSlippageApplication(t *testing.T) {
	simulator := NewTradeSimulator(0.0015, 0.001) // 0.1% slippage

	bar := data.OHLCV{
		Open:  80000.0,
		High:  81000.0,
		Low:   79500.0,
		Close: 80500.0,
	}

	// Buy should add slippage
	buyFill, _ := simulator.SimulateBuy("VCB", 100, 80000, bar, 80000)
	if buyFill.Success {
		// Fill price should be higher than base price due to slippage
		basePrice := 80000.0
		expectedWithSlippage := basePrice * 1.001
		if buyFill.FilledPrice < basePrice || buyFill.FilledPrice > expectedWithSlippage*1.01 {
			t.Errorf("Buy slippage not applied correctly: %.2f", buyFill.FilledPrice)
		}
	}

	// Sell should subtract slippage
	sellFill, _ := simulator.SimulateSell("VCB", 100, 80000, bar, 80000)
	if sellFill.Success {
		// Fill price should be lower than base price due to slippage
		basePrice := 80000.0
		expectedWithSlippage := basePrice * 0.999
		if sellFill.FilledPrice > basePrice || sellFill.FilledPrice < expectedWithSlippage*0.99 {
			t.Errorf("Sell slippage not applied correctly: %.2f", sellFill.FilledPrice)
		}
	}

	t.Logf("Slippage correctly applied: buy=%.2f, sell=%.2f",
		buyFill.FilledPrice, sellFill.FilledPrice)
}
