package test

import (
	"testing"

	"github.com/nonobeam/golang-stock-trading/internal/position"
)

func TestPnLCalculationsProfit(t *testing.T) {
	tracker := position.NewPositionTracker()

	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		PositionType: "long",
	})

	// Update to profitable price
	tracker.UpdatePositionPrice(pos.PositionID, 55000, nil)
	metrics, _ := tracker.GetPositionMetrics(pos.PositionID)

	// Unrealized P&L: (55000 - 52000) * 500 = 1,500,000
	expectedPL := 1500000.0
	if metrics.UnrealizedPL != expectedPL {
		t.Errorf("Expected unrealized P&L %.0f, got %.0f", expectedPL, metrics.UnrealizedPL)
	}

	// Unrealized P&L percent: (3000 / 52000) * 100 = 5.77%
	expectedPercent := (3000.0 / 52000.0) * 100
	if diff := metrics.UnrealizedPLPercent - expectedPercent; diff > 0.01 || diff < -0.01 {
		t.Errorf("Expected unrealized P&L percent %.2f%%, got %.2f%%", expectedPercent, metrics.UnrealizedPLPercent)
	}

	// R-multiple: 3000 / 3000 = 1.0R
	expectedR := 1.0
	if metrics.RMultiple != expectedR {
		t.Errorf("Expected R-multiple %.1f, got %.1f", expectedR, metrics.RMultiple)
	}
}

func TestPnLCalculationsLoss(t *testing.T) {
	tracker := position.NewPositionTracker()

	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		PositionType: "long",
	})

	// Update to losing price
	tracker.UpdatePositionPrice(pos.PositionID, 50000, nil)
	metrics, _ := tracker.GetPositionMetrics(pos.PositionID)

	// Unrealized P&L: (50000 - 52000) * 500 = -1,000,000
	expectedPL := -1000000.0
	if metrics.UnrealizedPL != expectedPL {
		t.Errorf("Expected unrealized P&L %.0f, got %.0f", expectedPL, metrics.UnrealizedPL)
	}

	// R-multiple: -2000 / 3000 = -0.67R
	expectedR := -2000.0 / 3000.0
	if diff := metrics.RMultiple - expectedR; diff > 0.01 || diff < -0.01 {
		t.Errorf("Expected R-multiple %.2f, got %.2f", expectedR, metrics.RMultiple)
	}
}

func TestMAEMFETracking(t *testing.T) {
	tracker := position.NewPositionTracker()

	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		PositionType: "long",
	})

	// Price sequence: 52000 → 51000 → 50500 → 52500 → 54000
	tracker.UpdatePositionPrice(pos.PositionID, 51000, nil)
	tracker.UpdatePositionPrice(pos.PositionID, 50500, nil)
	tracker.UpdatePositionPrice(pos.PositionID, 52500, nil)
	tracker.UpdatePositionPrice(pos.PositionID, 54000, nil)

	metrics, _ := tracker.GetPositionMetrics(pos.PositionID)

	// MAE: 52000 - 50500 = 1500
	expectedMAE := 1500.0
	if metrics.MAE != expectedMAE {
		t.Errorf("Expected MAE %.0f, got %.0f", expectedMAE, metrics.MAE)
	}

	// MAE percent: 1500 / 52000 * 100 = 2.88%
	expectedMAEPercent := (1500.0 / 52000.0) * 100
	if diff := metrics.MAEPercent - expectedMAEPercent; diff > 0.01 || diff < -0.01 {
		t.Errorf("Expected MAE percent %.2f%%, got %.2f%%", expectedMAEPercent, metrics.MAEPercent)
	}

	// MAE R: 1500 / 3000 = 0.5R
	expectedMAE_R := 0.5
	if diff := metrics.MAE_R - expectedMAE_R; diff > 0.01 || diff < -0.01 {
		t.Errorf("Expected MAE R %.2f, got %.2f", expectedMAE_R, metrics.MAE_R)
	}

	// MFE: 54000 - 52000 = 2000
	expectedMFE := 2000.0
	if metrics.MFE != expectedMFE {
		t.Errorf("Expected MFE %.0f, got %.0f", expectedMFE, metrics.MFE)
	}

	// MFE percent: 2000 / 52000 * 100 = 3.85%
	expectedMFEPercent := (2000.0 / 52000.0) * 100
	if diff := metrics.MFEPercent - expectedMFEPercent; diff > 0.01 || diff < -0.01 {
		t.Errorf("Expected MFE percent %.2f%%, got %.2f%%", expectedMFEPercent, metrics.MFEPercent)
	}

	// Lowest/Highest tracking
	if metrics.LowestPrice != 50500 {
		t.Errorf("Expected lowest price 50500, got %.0f", metrics.LowestPrice)
	}
	if metrics.HighestPrice != 54000 {
		t.Errorf("Expected highest price 54000, got %.0f", metrics.HighestPrice)
	}
}

func TestTargetProgressCalculation(t *testing.T) {
	tracker := position.NewPositionTracker()

	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		PositionType: "long",
		Targets: []position.Target{
			{TargetNumber: 1, TargetPrice: 58000, PercentToSell: 25, RMultiple: 2.0},
			{TargetNumber: 2, TargetPrice: 61000, PercentToSell: 25, RMultiple: 3.0},
		},
	})

	// Update to 55000 (halfway to first target)
	// Entry: 52000, Target: 58000, Distance: 6000, Progress: 3000 = 50%
	tracker.UpdatePositionPrice(pos.PositionID, 55000, nil)
	metrics, _ := tracker.GetPositionMetrics(pos.PositionID)

	if len(metrics.TargetProgress) != 2 {
		t.Fatalf("Expected 2 targets, got %d", len(metrics.TargetProgress))
	}

	// First target: 50% complete
	target1 := metrics.TargetProgress[0]
	if diff := target1.PercentComplete - 50.0; diff > 0.1 || diff < -0.1 {
		t.Errorf("Expected target 1 at 50%% complete, got %.1f%%", target1.PercentComplete)
	}

	if target1.TargetHit {
		t.Error("Target 1 should not be hit yet")
	}

	// Second target: ~33% complete (3000 / 9000)
	target2 := metrics.TargetProgress[1]
	expectedProgress2 := (3000.0 / 9000.0) * 100 // Entry to target2 is 9000
	if diff := target2.PercentComplete - expectedProgress2; diff > 0.1 || diff < -0.1 {
		t.Errorf("Expected target 2 at %.1f%% complete, got %.1f%%", expectedProgress2, target2.PercentComplete)
	}
}

func TestTargetHitDetection(t *testing.T) {
	tracker := position.NewPositionTracker()

	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		PositionType: "long",
		Targets: []position.Target{
			{TargetNumber: 1, TargetPrice: 58000, PercentToSell: 25, RMultiple: 2.0},
		},
	})

	// Update to above target
	tracker.UpdatePositionPrice(pos.PositionID, 58500, nil)
	metrics, _ := tracker.GetPositionMetrics(pos.PositionID)

	if !metrics.TargetProgress[0].TargetHit {
		t.Error("Target 1 should be hit when price >= target price")
	}

	if metrics.TargetProgress[0].PercentComplete != 100 {
		t.Errorf("Expected 100%% complete when hit, got %.0f%%", metrics.TargetProgress[0].PercentComplete)
	}
}

func TestStopDistanceCalculation(t *testing.T) {
	tracker := position.NewPositionTracker()

	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		PositionType: "long",
	})

	// Update price
	tracker.UpdatePositionPrice(pos.PositionID, 51000, nil)
	metrics, _ := tracker.GetPositionMetrics(pos.PositionID)

	// Stop distance: 51000 - 49000 = 2000
	expectedDistance := 2000.0
	if metrics.StopDistance != expectedDistance {
		t.Errorf("Expected stop distance %.0f, got %.0f", expectedDistance, metrics.StopDistance)
	}

	// Stop distance percent: 2000 / 51000 * 100 = 3.92%
	expectedPercent := (2000.0 / 51000.0) * 100
	if diff := metrics.StopDistancePercent - expectedPercent; diff > 0.01 || diff < -0.01 {
		t.Errorf("Expected stop distance percent %.2f%%, got %.2f%%", expectedPercent, metrics.StopDistancePercent)
	}

	if metrics.StopHit {
		t.Error("Stop should not be hit yet")
	}
}

func TestStopHitDetection(t *testing.T) {
	tracker := position.NewPositionTracker()

	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		PositionType: "long",
	})

	// Update price below stop
	tracker.UpdatePositionPrice(pos.PositionID, 48500, nil)
	metrics, _ := tracker.GetPositionMetrics(pos.PositionID)

	if !metrics.StopHit {
		t.Error("Stop should be hit when price <= stop loss")
	}
}

func TestRealizedPLFromPartialExits(t *testing.T) {
	tracker := position.NewPositionTracker()

	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		PositionType: "long",
	})

	// Partial exit 1: 125 shares at 58000
	tracker.PartialExit(pos.PositionID, 58000, 125, "Target 1")

	// Partial exit 2: 125 shares at 61000
	tracker.PartialExit(pos.PositionID, 61000, 125, "Target 2")

	// Update remaining position price
	tracker.UpdatePositionPrice(pos.PositionID, 60000, nil)
	metrics, _ := tracker.GetPositionMetrics(pos.PositionID)

	// Realized P&L: (58000-52000)*125 + (61000-52000)*125 = 750000 + 1125000 = 1875000
	expectedRealizedPL := 1875000.0
	if metrics.RealizedPL != expectedRealizedPL {
		t.Errorf("Expected realized P&L %.0f, got %.0f", expectedRealizedPL, metrics.RealizedPL)
	}

	// Unrealized P&L: (60000 - 52000) * 250 = 2000000
	expectedUnrealizedPL := 2000000.0
	if metrics.UnrealizedPL != expectedUnrealizedPL {
		t.Errorf("Expected unrealized P&L %.0f, got %.0f", expectedUnrealizedPL, metrics.UnrealizedPL)
	}

	// Total P&L
	expectedTotalPL := 3875000.0
	if metrics.TotalPL != expectedTotalPL {
		t.Errorf("Expected total P&L %.0f, got %.0f", expectedTotalPL, metrics.TotalPL)
	}

	// Shares remaining
	if metrics.SharesRemaining != 250 {
		t.Errorf("Expected 250 shares remaining, got %d", metrics.SharesRemaining)
	}
}
