package test

import (
	"testing"

	"github.com/nonobeam/golang-stock-trading/internal/position"
)

func TestManagedTrackerAutoAdjust(t *testing.T) {
	rules := position.DefaultStopAdjustmentRule()
	tracker := position.NewManagedPositionTracker(&rules)

	// Add position
	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		RiskPercent:  1.5,
		PositionType: "long",
		Targets: []position.Target{
			{TargetNumber: 1, TargetPrice: 58000, PercentToSell: 25, RMultiple: 2.0},
		},
	})

	// Update to +1R with auto-adjust
	result := tracker.UpdatePositionWithStopManagement(
		pos.PositionID,
		55000, // +1R
		nil,
		true, // auto-adjust
	)

	if result.Error != "" {
		t.Fatalf("Unexpected error: %s", result.Error)
	}

	if !result.StopAdjusted {
		t.Error("Expected stop to be adjusted")
	}

	// Verify stop was actually changed
	updatedPos, _ := tracker.GetPosition(pos.PositionID)
	if updatedPos.StopLoss <= 49000 {
		t.Errorf("Expected stop to be raised from 49000, got %.0f", updatedPos.StopLoss)
	}
}

func TestManagedTrackerSuggestOnly(t *testing.T) {
	rules := position.DefaultStopAdjustmentRule()
	tracker := position.NewManagedPositionTracker(&rules)

	// Add position
	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		RiskPercent:  1.5,
		PositionType: "long",
	})

	// Update with suggest-only mode
	result := tracker.UpdatePositionWithStopManagement(
		pos.PositionID,
		55000,
		nil,
		false, // suggest only
	)

	if result.Error != "" {
		t.Fatalf("Unexpected error: %s", result.Error)
	}

	// Should have suggestion but not applied
	if result.StopAdjusted {
		t.Error("Expected stop NOT to be auto-adjusted in suggest mode")
	}

	if result.StopAdjustment == nil {
		t.Error("Expected suggestion to be returned")
	}

	// Verify stop was NOT changed
	updatedPos, _ := tracker.GetPosition(pos.PositionID)
	if updatedPos.StopLoss != 49000 {
		t.Errorf("Expected stop unchanged at 49000, got %.0f", updatedPos.StopLoss)
	}
}

func TestManagedTrackerMultipleAdjustments(t *testing.T) {
	rules := position.DefaultStopAdjustmentRule()
	rules.TrailingMethod = position.TrailingMethodATR

	tracker := position.NewManagedPositionTracker(&rules)

	// Add position
	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		RiskPercent:  1.5,
		PositionType: "long",
	})

	indicators := &position.Indicators{ATR: 2500}

	// First update: +1R → breakeven
	tracker.UpdatePositionWithStopManagement(pos.PositionID, 55000, indicators, true)

	// Second update: higher price → trailing stop
	tracker.UpdatePositionWithStopManagement(pos.PositionID, 58000, indicators, true)

	// Third update: even higher → trailing moves up
	tracker.UpdatePositionWithStopManagement(pos.PositionID, 61000, indicators, true)

	// Check history
	history := tracker.GetStopAdjustmentHistory(pos.PositionID)
	if history == nil {
		t.Fatal("Expected history")
	}

	if len(history.Adjustments) < 2 {
		t.Errorf("Expected at least 2 adjustments, got %d", len(history.Adjustments))
	}
}

func TestManagedTrackerHistoryRecording(t *testing.T) {
	rules := position.DefaultStopAdjustmentRule()
	tracker := position.NewManagedPositionTracker(&rules)

	// Add position
	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		RiskPercent:  1.5,
		PositionType: "long",
	})

	// Trigger adjustment
	tracker.UpdatePositionWithStopManagement(pos.PositionID, 55000, nil, true)

	// Get history
	history := tracker.GetStopAdjustmentHistory(pos.PositionID)

	if history == nil {
		t.Fatal("Expected history to be recorded")
	}

	if len(history.Adjustments) != 1 {
		t.Fatalf("Expected 1 adjustment, got %d", len(history.Adjustments))
	}

	adj := history.Adjustments[0]
	if adj.OldStop != 49000 {
		t.Errorf("Expected old stop 49000, got %.0f", adj.OldStop)
	}

	if adj.Reason != position.ReasonBreakeven {
		t.Errorf("Expected BREAKEVEN reason, got %s", adj.Reason)
	}

	if adj.CurrentPrice != 55000 {
		t.Errorf("Expected current price 55000, got %.0f", adj.CurrentPrice)
	}
}

func TestManagedTrackerSummaryGeneration(t *testing.T) {
	rules := position.DefaultStopAdjustmentRule()
	tracker := position.NewManagedPositionTracker(&rules)

	// Add position
	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		RiskPercent:  1.5,
		PositionType: "long",
	})

	// Trigger adjustment
	tracker.UpdatePositionWithStopManagement(pos.PositionID, 55000, nil, true)

	// Get summary
	summary := tracker.GetStopAdjustmentSummary(pos.PositionID)

	if summary == "" {
		t.Error("Expected non-empty summary")
	}

	if len(summary) < 50 {
		t.Error("Expected detailed summary")
	}
}

func TestManagedTrackerNoAdjustmentNeeded(t *testing.T) {
	rules := position.DefaultStopAdjustmentRule()
	tracker := position.NewManagedPositionTracker(&rules)

	// Add position
	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		RiskPercent:  1.5,
		PositionType: "long",
	})

	// Update with small profit (less than 1R)
	result := tracker.UpdatePositionWithStopManagement(pos.PositionID, 53000, nil, true)

	if result.Error != "" {
		t.Fatalf("Unexpected error: %s", result.Error)
	}

	if result.StopAdjusted {
		t.Error("Expected no adjustment at less than 1R")
	}
}
