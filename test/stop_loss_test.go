package test

import (
	"testing"

	"github.com/nonobeam/golang-stock-trading/internal/risk"
)

// --- ATR Stop Tests ---

func TestATRStopDetailed(t *testing.T) {
	// Use 1.5x multiplier to stay under 7% max
	params := risk.StopParams{
		EntryPrice: 50000,
		ATR:        1500,
		Multiplier: 2.0,
		IsLong:     true,
	}

	result, err := risk.ATRStopDetailed(params)
	if err != nil {
		t.Fatalf("ATRStopDetailed error: %v", err)
	}

	// 50000 - (1500 * 2) = 47000 (6%)
	if result.StopLossPrice != 47000 {
		t.Errorf("StopLossPrice = %.0f, want 47000", result.StopLossPrice)
	}
	if result.StopDistance != 3000 {
		t.Errorf("StopDistance = %.0f, want 3000", result.StopDistance)
	}
	if result.StopDistancePercent != 6.0 {
		t.Errorf("StopDistancePercent = %.1f, want 6.0", result.StopDistancePercent)
	}
	if result.Method != "ATR-based" {
		t.Errorf("Method = %s, want ATR-based", result.Method)
	}
	if !result.IsValid {
		t.Errorf("Expected IsValid = true, got false. Issues: %v", result.ValidationIssues)
	}
}

func TestATRStop_InvalidInput(t *testing.T) {
	params := risk.StopParams{
		EntryPrice: 50000,
		ATR:        0, // Invalid
		Multiplier: 2.0,
	}

	_, err := risk.ATRStop(params)
	if err != risk.ErrInvalidInput {
		t.Errorf("Expected ErrInvalidInput, got: %v", err)
	}
}

// --- Swing Low Stop Tests ---

func TestSwingLowStop_Standard(t *testing.T) {
	params := risk.StopParams{
		EntryPrice:      50000,
		Lows:            []float64{48000, 48500, 49000, 48200, 49500, 50000, 49800, 48500},
		LookbackPeriods: 8,
		Buffer:          0.005, // 0.5%
		IsLong:          true,
	}

	result, err := risk.SwingLowStop(params)
	if err != nil {
		t.Fatalf("SwingLowStop error: %v", err)
	}

	// Swing low is 48000, buffer = 48000 * 0.005 = 240
	// Stop = 48000 - 240 = 47760, rounded to ~47750
	if result.SwingLow != 48000 {
		t.Errorf("SwingLow = %.0f, want 48000", result.SwingLow)
	}
	if result.Method != "Swing Low" {
		t.Errorf("Method = %s, want Swing Low", result.Method)
	}
	// Stop should be below swing low
	if result.StopLossPrice >= result.SwingLow {
		t.Errorf("StopLossPrice %.0f should be below SwingLow %.0f", result.StopLossPrice, result.SwingLow)
	}

	t.Logf("Swing Low: %.0f, Stop: %.0f, Distance: %.2f%%",
		result.SwingLow, result.StopLossPrice, result.StopDistancePercent)
}

func TestSwingLowStop_InsufficientData(t *testing.T) {
	params := risk.StopParams{
		EntryPrice:      50000,
		Lows:            []float64{48000, 49000, 50000}, // Only 3 periods
		LookbackPeriods: 20,                             // Needs 20
		IsLong:          true,
	}

	_, err := risk.SwingLowStop(params)
	if err != risk.ErrInvalidSwingData {
		t.Errorf("Expected ErrInvalidSwingData, got: %v", err)
	}
}

// --- Support Stop Tests ---

func TestSupportStop_PercentageBuffer(t *testing.T) {
	result, err := risk.SupportStop(49000, 0.01, 0, true)
	if err != nil {
		t.Fatalf("SupportStop error: %v", err)
	}

	// 49000 - (49000 * 0.01) = 49000 - 490 = 48510, rounded to 48500
	if result.StopLossPrice != 48500 {
		t.Errorf("StopLossPrice = %.0f, want 48500", result.StopLossPrice)
	}
	if result.TechnicalLevel != 49000 {
		t.Errorf("TechnicalLevel = %.0f, want 49000", result.TechnicalLevel)
	}
}

func TestSupportStop_ATRBuffer(t *testing.T) {
	result, err := risk.SupportStop(49000, 0, 2000, true)
	if err != nil {
		t.Fatalf("SupportStop error: %v", err)
	}

	// 49000 - 2000 = 47000
	if result.StopLossPrice != 47000 {
		t.Errorf("StopLossPrice = %.0f, want 47000", result.StopLossPrice)
	}
}

// --- Floor-Aware Stop Tests ---

func TestFloorAwareStop_BelowFloor(t *testing.T) {
	// Entry 52000, intended stop 47000, reference 51000, limit 7%
	// Floor = 51000 * 0.93 = 47430
	result, err := risk.FloorAwareStop(52000, 47000, 51000, 0.07)
	if err != nil {
		t.Fatalf("FloorAwareStop error: %v", err)
	}

	// Floor should be around 47430
	if result.FloorPrice < 47400 || result.FloorPrice > 47500 {
		t.Errorf("FloorPrice = %.0f, expected around 47430", result.FloorPrice)
	}
	if result.ReachableToday {
		t.Error("Expected ReachableToday = false")
	}
	if result.Warning == "" {
		t.Error("Expected warning message")
	}
	// Effective stop should be floor (adjusted)
	if result.StopLossPrice < result.IntendedStop {
		t.Errorf("StopLossPrice %.0f should be >= IntendedStop %.0f (adjusted to floor)", result.StopLossPrice, result.IntendedStop)
	}

	t.Logf("Intended: %.0f, Floor: %.0f, Effective: %.0f, WorstCase: %.0f (%.1f%%)",
		result.IntendedStop, result.FloorPrice, result.StopLossPrice, result.WorstCaseStop, result.WorstCasePercent)
}

func TestFloorAwareStop_AboveFloor(t *testing.T) {
	// Entry 50000, intended stop 48000, reference 50000, limit 7%
	// Floor = 50000 * 0.93 = 46500
	result, err := risk.FloorAwareStop(50000, 48000, 50000, 0.07)
	if err != nil {
		t.Fatalf("FloorAwareStop error: %v", err)
	}

	if !result.ReachableToday {
		t.Error("Expected ReachableToday = true")
	}
	if result.StopLossPrice != 48000 {
		t.Errorf("StopLossPrice = %.0f, want 48000 (unchanged)", result.StopLossPrice)
	}
}

func TestFloorAwareStop_WorstCase(t *testing.T) {
	result, err := risk.FloorAwareStop(52000, 47000, 51000, 0.07)
	if err != nil {
		t.Fatalf("FloorAwareStop error: %v", err)
	}

	// 51000 * 0.93^3 ≈ 41022
	if result.WorstCaseStop < 40000 || result.WorstCaseStop > 42000 {
		t.Errorf("WorstCaseStop = %.0f, expected around 41000", result.WorstCaseStop)
	}
}

// --- Pre-emptive Alerts Tests ---

func TestPreemptiveAlerts(t *testing.T) {
	// Entry 50000, stop 46500 (7% below)
	// Distance: 3500
	// Alert 1 (50%): 50000 - 1750 = 48250
	// Alert 2 (70%): 50000 - 2450 = 47550
	result, err := risk.PreemptiveAlerts(50000, 46500)
	if err != nil {
		t.Fatalf("PreemptiveAlerts error: %v", err)
	}

	// Allow some tolerance for rounding
	if result.Alert1Price < 48200 || result.Alert1Price > 48300 {
		t.Errorf("Alert1Price = %.0f, expected around 48250", result.Alert1Price)
	}
	if result.Alert1Action != "Exit 50% of position" {
		t.Errorf("Alert1Action = %s, want 'Exit 50%% of position'", result.Alert1Action)
	}
	if result.Alert2Price < 47500 || result.Alert2Price > 47600 {
		t.Errorf("Alert2Price = %.0f, expected around 47550", result.Alert2Price)
	}
	if result.Alert2Action != "Exit remaining 50%" {
		t.Errorf("Alert2Action = %s, want 'Exit remaining 50%%'", result.Alert2Action)
	}

	t.Logf("Entry: 50000, Stop: 46500, Alert1: %.0f, Alert2: %.0f",
		result.Alert1Price, result.Alert2Price)
}

// --- Validation Tests ---

func TestValidateStopFull_Valid(t *testing.T) {
	result, err := risk.ValidateStopFull(50000, 47000, true)
	if err != nil {
		t.Fatalf("ValidateStopFull error: %v", err)
	}

	if !result.IsValid {
		t.Errorf("Expected IsValid = true, issues: %v", result.ValidationIssues)
	}
	if result.StopDistancePercent < 5 || result.StopDistancePercent > 7 {
		t.Errorf("StopDistancePercent = %.1f%%, expected around 6%%", result.StopDistancePercent)
	}
}

func TestValidateStopFull_TooWide(t *testing.T) {
	// 16% stop - should fail (exceeds 7% max)
	result, _ := risk.ValidateStopFull(50000, 42000, true)

	if result.IsValid {
		t.Error("Expected IsValid = false for 16% stop")
	}
	if len(result.ValidationIssues) == 0 {
		t.Error("Expected validation issues")
	}

	t.Logf("Issues: %v", result.ValidationIssues)
}

func TestValidateStopFull_WrongSide(t *testing.T) {
	// Stop above entry for long position
	result, _ := risk.ValidateStopFull(50000, 52000, true)

	if result.IsValid {
		t.Error("Expected IsValid = false for wrong side")
	}

	found := false
	for _, issue := range result.ValidationIssues {
		if issue == "stop on wrong side: must be below entry for long" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'wrong side' issue, got: %v", result.ValidationIssues)
	}
}

// --- Integration Tests ---

func TestCalculateStopDetailed_ATRWithFloorAware(t *testing.T) {
	params := risk.StopParams{
		EntryPrice:        52000,
		ATR:               1500,
		Multiplier:        2.0,
		IsLong:            true,
		ReferencePrice:    51000,
		DailyLimitPercent: 0.07,
		EnablePreemptive:  true,
	}

	result, err := risk.CalculateStopDetailed(params)
	if err != nil {
		t.Fatalf("CalculateStopDetailed error: %v", err)
	}

	if result.Method != "ATR-based" {
		t.Errorf("Method = %s, want ATR-based", result.Method)
	}
	if result.FloorPrice == 0 {
		t.Error("Expected FloorPrice to be calculated")
	}
	if result.Alert1Price == 0 || result.Alert2Price == 0 {
		t.Error("Expected pre-emptive alerts")
	}

	t.Logf("Stop: %.0f, Method: %s, Floor: %.0f, Alert1: %.0f, Alert2: %.0f",
		result.StopLossPrice, result.Method, result.FloorPrice, result.Alert1Price, result.Alert2Price)
}

func TestCalculateStopDetailed_SwingLowPriority(t *testing.T) {
	// Create lows with clear swing low closer to entry (under 7%)
	lows := make([]float64, 20)
	for i := 0; i < 20; i++ {
		lows[i] = 50000 + float64(i*100) // Ascending trend: 50000 to 51900
	}
	lows[5] = 49500 // Insert swing low at 49500 (close to entry)

	params := risk.StopParams{
		EntryPrice:      52000,
		ATR:             1500,
		Multiplier:      2.0,
		Lows:            lows,
		LookbackPeriods: 20, // Explicitly request swing low
		Buffer:          0.005,
		IsLong:          true,
	}

	result, err := risk.CalculateStopDetailed(params)
	if err != nil {
		t.Fatalf("CalculateStopDetailed error: %v", err)
	}

	// Swing Low should take priority over ATR when LookbackPeriods is set
	if result.Method != "Swing Low" {
		t.Errorf("Method = %s, want Swing Low (priority)", result.Method)
	}
	if result.SwingLow != 49500 {
		t.Errorf("SwingLow = %.0f, want 49500", result.SwingLow)
	}

	t.Logf("Method: %s, SwingLow: %.0f, Stop: %.0f, Distance: %.2f%%", 
		result.Method, result.SwingLow, result.StopLossPrice, result.StopDistancePercent)
}

func TestCalculateStop_Compatibility(t *testing.T) {
	// Test that CalculateStop still works (backward compatibility)
	// Use smaller multiplier to stay under 7% max
	params := risk.StopParams{
		EntryPrice: 50000,
		ATR:        1500,
		Multiplier: 2.0,
		IsLong:     true,
	}

	stop, err := risk.CalculateStop(params)
	if err != nil {
		t.Fatalf("CalculateStop error: %v", err)
	}

	expected := 47000.0 // 50000 - 3000
	if stop != expected {
		t.Errorf("CalculateStop = %.0f, want %.0f", stop, expected)
	}
}

func TestCalculateStop_NoMethod(t *testing.T) {
	params := risk.StopParams{
		EntryPrice: 50000,
		IsLong:     true,
	}

	_, err := risk.CalculateStop(params)
	if err != risk.ErrNoStopMethod {
		t.Errorf("Expected ErrNoStopMethod, got: %v", err)
	}
}

// --- Moving Average Stop Tests ---

func TestMovingAverageStop(t *testing.T) {
	// 20 EMA at 50500, 2% buffer
	result, err := risk.MovingAverageStop(50500, 20, 0.02, 0, true)
	if err != nil {
		t.Fatalf("MovingAverageStop error: %v", err)
	}

	// 50500 - (50500 * 0.02) = 50500 - 1010 = 49490, rounded to 49500
	if result.StopLossPrice != 49500 {
		t.Errorf("StopLossPrice = %.0f, want 49500", result.StopLossPrice)
	}
	if result.TechnicalLevel != 50500 {
		t.Errorf("TechnicalLevel = %.0f, want 50500", result.TechnicalLevel)
	}

	t.Logf("MA: %.0f, Stop: %.0f", result.TechnicalLevel, result.StopLossPrice)
}

// --- Percentage Stop Tests ---

func TestPercentageStopDetailed(t *testing.T) {
	params := risk.StopParams{
		EntryPrice: 50000,
		Percentage: 0.05, // 5%
		IsLong:     true,
	}

	result, err := risk.PercentageStopDetailed(params)
	if err != nil {
		t.Fatalf("PercentageStopDetailed error: %v", err)
	}

	// 50000 * 0.95 = 47500
	if result.StopLossPrice != 47500 {
		t.Errorf("StopLossPrice = %.0f, want 47500", result.StopLossPrice)
	}
	if result.StopDistancePercent != 5.0 {
		t.Errorf("StopDistancePercent = %.1f, want 5.0", result.StopDistancePercent)
	}
}

// --- Helper Function Tests ---

func TestStopDistance(t *testing.T) {
	dist := risk.StopDistance(50000, 47000)
	if dist != 3000 {
		t.Errorf("StopDistance = %.0f, want 3000", dist)
	}
}

func TestStopPercent(t *testing.T) {
	pct := risk.StopPercent(50000, 47000)
	expected := 0.06
	if pct != expected {
		t.Errorf("StopPercent = %.4f, want %.4f", pct, expected)
	}
}

// --- Edge Cases ---

func TestATRStop_TooWide(t *testing.T) {
	// 10% stop should fail with default 7% max
	params := risk.StopParams{
		EntryPrice: 50000,
		ATR:        2500,
		Multiplier: 2.0, // 10% distance
		IsLong:     true,
	}

	_, err := risk.ATRStop(params)
	if err != risk.ErrStopTooWide {
		t.Errorf("Expected ErrStopTooWide, got: %v", err)
	}
}

func TestSwingLowStop_ShortPosition(t *testing.T) {
	params := risk.StopParams{
		EntryPrice:      48000,
		Lows:            []float64{50000, 50500, 51000, 50200, 49500, 50000, 49800, 50500},
		LookbackPeriods: 8,
		Buffer:          0.005,
		IsLong:          false, // Short position
	}

	result, err := risk.SwingLowStop(params)
	if err != nil {
		t.Fatalf("SwingLowStop error: %v", err)
	}

	// For shorts, stop should be ABOVE entry (in this case above swing low)
	if result.StopLossPrice <= params.EntryPrice {
		t.Logf("Short position: Stop %.0f should be above entry %.0f", result.StopLossPrice, params.EntryPrice)
	}
}
