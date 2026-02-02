package test

import (
	"strings"
	"testing"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/position"
)

func TestStopHitAlert(t *testing.T) {
	tracker := position.NewPositionTracker()

	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		PositionType: "long",
	})

	// Update price below stop
	result := tracker.UpdatePositionPrice(pos.PositionID, 48500, nil)

	if len(result.Alerts) == 0 {
		t.Fatal("Expected at least one alert")
	}

	// Find STOP_HIT alert
	var foundStopHit bool
	for _, alert := range result.Alerts {
		if alert.Type == position.AlertTypeStopHit {
			foundStopHit = true
			if alert.Severity != position.SeverityHigh {
				t.Errorf("Expected HIGH severity, got %s", alert.Severity)
			}
			if alert.Action != "EXIT IMMEDIATELY" {
				t.Errorf("Expected 'EXIT IMMEDIATELY' action, got %s", alert.Action)
			}
		}
	}

	if !foundStopHit {
		t.Error("Expected STOP_HIT alert")
	}
}

func TestStopCloseAlert(t *testing.T) {
	tracker := position.NewPositionTracker()

	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		PositionType: "long",
	})

	// Update price to within 2% of stop
	// Stop at 49000, 2% above = 49980. So price of 49800 should trigger
	result := tracker.UpdatePositionPrice(pos.PositionID, 49800, nil)

	// Find STOP_CLOSE alert
	var foundStopClose bool
	for _, alert := range result.Alerts {
		if alert.Type == position.AlertTypeStopClose {
			foundStopClose = true
			if alert.Severity != position.SeverityMedium {
				t.Errorf("Expected MEDIUM severity, got %s", alert.Severity)
			}
			if !strings.Contains(alert.Message, "within") {
				t.Errorf("Expected message to contain 'within', got %s", alert.Message)
			}
		}
	}

	if !foundStopClose {
		t.Error("Expected STOP_CLOSE alert when within 2% of stop")
	}
}

func TestTargetHitAlert(t *testing.T) {
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

	// Update price above target
	result := tracker.UpdatePositionPrice(pos.PositionID, 58500, nil)

	// Find TARGET_HIT alert
	var foundTargetHit bool
	for _, alert := range result.Alerts {
		if alert.Type == position.AlertTypeTargetHit {
			foundTargetHit = true
			if !strings.Contains(alert.Message, "Target 1 hit") {
				t.Errorf("Expected message about Target 1 hit, got %s", alert.Message)
			}
			if !strings.Contains(alert.Action, "25%") {
				t.Errorf("Expected action to mention 25%%, got %s", alert.Action)
			}
		}
	}

	if !foundTargetHit {
		t.Error("Expected TARGET_HIT alert when price >= target")
	}
}

func TestTargetCloseAlert(t *testing.T) {
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

	// Update price to 90% of progress (90% of 6000 = 5400, so 52000 + 5400 = 57400)
	result := tracker.UpdatePositionPrice(pos.PositionID, 57400, nil)

	// Find TARGET_CLOSE alert
	var foundTargetClose bool
	for _, alert := range result.Alerts {
		if alert.Type == position.AlertTypeTargetClose {
			foundTargetClose = true
			if alert.Action != "Prepare limit order" {
				t.Errorf("Expected 'Prepare limit order' action, got %s", alert.Action)
			}
		}
	}

	if !foundTargetClose {
		t.Error("Expected TARGET_CLOSE alert when 90% complete")
	}
}

func TestTimeLongAlert(t *testing.T) {
	tracker := position.NewPositionTracker()

	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000,
		PositionType: "long",
	})

	// Manually adjust entry date to 31 days ago
	p, _ := tracker.GetPosition(pos.PositionID)
	p.EntryDate = time.Now().Add(-31 * 24 * time.Hour)

	result := tracker.UpdatePositionPrice(pos.PositionID, 53000, nil)

	// Find TIME_LONG alert
	var foundTimeLong bool
	for _, alert := range result.Alerts {
		if alert.Type == position.AlertTypeTimeLong {
			foundTimeLong = true
			if !strings.Contains(alert.Message, "30 days") && !strings.Contains(alert.Message, "31 days") {
				t.Errorf("Expected message about 30+ days, got %s", alert.Message)
			}
		}
	}

	if !foundTimeLong {
		t.Error("Expected TIME_LONG alert when held >= 30 days")
	}
}

func TestLargeProfitAlert(t *testing.T) {
	tracker := position.NewPositionTracker()

	pos := tracker.AddPosition(position.AddPositionParams{
		Ticker:       "VNM",
		EntryPrice:   52000,
		Shares:       500,
		StopLoss:     49000, // Risk per share = 3000
		PositionType: "long",
	})

	// Update price to 4R profit: 52000 + (4 * 3000) = 64000
	result := tracker.UpdatePositionPrice(pos.PositionID, 64000, nil)

	// Find LARGE_PROFIT alert
	var foundLargeProfit bool
	for _, alert := range result.Alerts {
		if alert.Type == position.AlertTypeLargeProfit {
			foundLargeProfit = true
			if !strings.Contains(alert.Message, "+4.0R") {
				t.Errorf("Expected message about +4.0R, got %s", alert.Message)
			}
		}
	}

	if !foundLargeProfit {
		t.Error("Expected LARGE_PROFIT alert when R >= 4.0")
	}
}

func TestNoAlertsForNormalPosition(t *testing.T) {
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

	// Update to "normal" price - not near stop, not near target, not huge profit
	result := tracker.UpdatePositionPrice(pos.PositionID, 53000, nil)

	if len(result.Alerts) > 0 {
		t.Errorf("Expected no alerts for normal position, got %d alerts", len(result.Alerts))
	}
}
