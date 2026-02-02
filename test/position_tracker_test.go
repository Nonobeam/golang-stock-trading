package test

import (
	"testing"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/position"
)

func TestAddPosition(t *testing.T) {
	tracker := position.NewPositionTracker()

	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		RiskPercent:  1.5,
		PositionType: "long",
		SetupType:    "pullback",
		TradeScore:   9,
		Targets: []position.Target{
			{TargetNumber: 1, TargetPrice: 58000, PercentToSell: 25, RMultiple: 2.0},
			{TargetNumber: 2, TargetPrice: 61000, PercentToSell: 25, RMultiple: 3.0},
		},
	})

	if pos.PositionID == "" {
		t.Error("Expected position ID to be generated")
	}

	if pos.Ticker != "VNM" {
		t.Errorf("Expected ticker VNM, got %s", pos.Ticker)
	}

	if pos.SharesRemaining != 500 {
		t.Errorf("Expected 500 shares remaining, got %d", pos.SharesRemaining)
	}

	// Check risk calculation
	expectedRiskPerShare := 3000.0 // 52000 - 49000
	if pos.RiskPerShare != expectedRiskPerShare {
		t.Errorf("Expected risk per share %.0f, got %.0f", expectedRiskPerShare, pos.RiskPerShare)
	}

	// Check position value
	expectedValue := 52000.0 * 500
	if pos.PositionValue != expectedValue {
		t.Errorf("Expected position value %.0f, got %.0f", expectedValue, pos.PositionValue)
	}

	// Check extremes initialized to entry
	if pos.HighestPriceReached != 52000 {
		t.Errorf("Expected highest price 52000, got %.0f", pos.HighestPriceReached)
	}
	if pos.LowestPriceReached != 52000 {
		t.Errorf("Expected lowest price 52000, got %.0f", pos.LowestPriceReached)
	}
}

func TestUpdatePositionPrice(t *testing.T) {
	tracker := position.NewPositionTracker()

	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		RiskPercent:  1.5,
		PositionType: "long",
	})

	// Update price upward
	result := tracker.UpdatePositionPrice(pos.PositionID, 55000, nil)

	if result.Error != "" {
		t.Errorf("Unexpected error: %s", result.Error)
	}

	if result.Metrics.CurrentPrice != 55000 {
		t.Errorf("Expected current price 55000, got %.0f", result.Metrics.CurrentPrice)
	}

	// Check extremes updated
	updatedPos, _ := tracker.GetPosition(pos.PositionID)
	if updatedPos.HighestPriceReached != 55000 {
		t.Errorf("Expected highest price 55000, got %.0f", updatedPos.HighestPriceReached)
	}

	// Update price downward
	tracker.UpdatePositionPrice(pos.PositionID, 51000, nil)
	updatedPos, _ = tracker.GetPosition(pos.PositionID)

	if updatedPos.LowestPriceReached != 51000 {
		t.Errorf("Expected lowest price 51000, got %.0f", updatedPos.LowestPriceReached)
	}

	// Highest should still be 55000
	if updatedPos.HighestPriceReached != 55000 {
		t.Errorf("Expected highest price still 55000, got %.0f", updatedPos.HighestPriceReached)
	}
}

func TestPartialExit(t *testing.T) {
	tracker := position.NewPositionTracker()

	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		RiskPercent:  1.5,
		PositionType: "long",
	})

	// Partial exit at target
	result := tracker.PartialExit(pos.PositionID, 58000, 125, "Target 1 hit")

	if result.Error != "" {
		t.Errorf("Unexpected error: %s", result.Error)
	}

	if result.SharesSold != 125 {
		t.Errorf("Expected 125 shares sold, got %d", result.SharesSold)
	}

	if result.SharesRemaining != 375 {
		t.Errorf("Expected 375 shares remaining, got %d", result.SharesRemaining)
	}

	// Check P&L calculation: (58000 - 52000) * 125 = 750000
	expectedPL := 750000.0
	if result.ExitPL != expectedPL {
		t.Errorf("Expected exit P&L %.0f, got %.0f", expectedPL, result.ExitPL)
	}

	// Check R-multiple: 750000 / (3000 * 125) = 2.0
	expectedR := 2.0
	if result.ExitR != expectedR {
		t.Errorf("Expected exit R %.1f, got %.1f", expectedR, result.ExitR)
	}

	if result.FullyClosed {
		t.Error("Position should not be fully closed")
	}
}

func TestClosePosition(t *testing.T) {
	tracker := position.NewPositionTracker()

	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		RiskPercent:  1.5,
		PositionType: "long",
	})

	// Close entire position
	result := tracker.ClosePosition(pos.PositionID, 55000, "Manual close")

	if result.Error != "" {
		t.Errorf("Unexpected error: %s", result.Error)
	}

	if result.SharesSold != 500 {
		t.Errorf("Expected 500 shares sold, got %d", result.SharesSold)
	}

	if !result.FullyClosed {
		t.Error("Position should be fully closed")
	}

	// Verify position is removed from active
	_, err := tracker.GetPosition(pos.PositionID)
	if err == nil {
		t.Error("Expected error getting closed position")
	}

	// Verify position is in closed list
	closedPositions := tracker.GetClosedPositions()
	if len(closedPositions) != 1 {
		t.Errorf("Expected 1 closed position, got %d", len(closedPositions))
	}
}

func TestGetAllPositionsSummary(t *testing.T) {
	tracker := position.NewPositionTracker()

	// Add multiple positions
	pos1 := tracker.AddPosition(position.AddPositionParams{
		Ticker:     "VNM",
		EntryPrice: 52000,
		Shares:     500,
		StopLoss:   49000,
	})

	pos2 := tracker.AddPosition(position.AddPositionParams{
		Ticker:     "VCB",
		EntryPrice: 85000,
		Shares:     200,
		StopLoss:   81000,
	})

	// Update prices
	tracker.UpdatePositionPrice(pos1.PositionID, 54000, nil)
	tracker.UpdatePositionPrice(pos2.PositionID, 87000, nil)

	summary := tracker.GetAllPositionsSummary()

	if summary.NumPositions != 2 {
		t.Errorf("Expected 2 positions, got %d", summary.NumPositions)
	}

	// Check total value: (54000 * 500) + (87000 * 200) = 27000000 + 17400000 = 44400000
	expectedValue := 44400000.0
	if summary.TotalValue != expectedValue {
		t.Errorf("Expected total value %.0f, got %.0f", expectedValue, summary.TotalValue)
	}

	// Check unrealized P&L
	// VNM: (54000 - 52000) * 500 = 1000000
	// VCB: (87000 - 85000) * 200 = 400000
	expectedPL := 1400000.0
	if summary.TotalUnrealizedPL != expectedPL {
		t.Errorf("Expected total unrealized P&L %.0f, got %.0f", expectedPL, summary.TotalUnrealizedPL)
	}
}

func TestPositionNotFound(t *testing.T) {
	tracker := position.NewPositionTracker()

	// Try to update non-existent position
	result := tracker.UpdatePositionPrice("FAKE_ID", 50000, nil)
	if result.Error == "" {
		t.Error("Expected error for non-existent position")
	}

	// Try to exit non-existent position
	exitResult := tracker.PartialExit("FAKE_ID", 50000, 100, "test")
	if exitResult.Error == "" {
		t.Error("Expected error for non-existent position")
	}
}

func TestInsufficientShares(t *testing.T) {
	tracker := position.NewPositionTracker()

	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:     "VNM",
		EntryPrice: 52000,
		Shares:     500,
		StopLoss:   49000,
	})

	// Try to sell more shares than available
	result := tracker.PartialExit(pos.PositionID, 55000, 1000, "test")

	if result.Error == "" {
		t.Error("Expected error for insufficient shares")
	}
}

func TestEmptyPortfolioSummary(t *testing.T) {
	tracker := position.NewPositionTracker()

	summary := tracker.GetAllPositionsSummary()

	if summary.NumPositions != 0 {
		t.Errorf("Expected 0 positions, got %d", summary.NumPositions)
	}

	if summary.TotalValue != 0 {
		t.Errorf("Expected 0 total value, got %.0f", summary.TotalValue)
	}
}

func TestPositionWithCustomTime(t *testing.T) {
	tracker := position.NewPositionTracker()

	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:     "VNM",
		EntryPrice: 52000,
		Shares:     500,
		StopLoss:   49000,
	})

	// Update with specific timestamp
	customTime := time.Now().Add(-24 * time.Hour)
	result := tracker.UpdatePositionPrice(pos.PositionID, 55000, &customTime)

	if result.Timestamp != customTime {
		t.Error("Expected custom timestamp to be used")
	}
}
