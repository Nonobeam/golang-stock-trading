package position_test

import (
	"testing"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/position"
)

func TestExitEvaluator_ShouldExitTarget1(t *testing.T) {
	evaluator := position.NewExitEvaluator(position.DefaultExitEvaluatorConfig())
	
	tests := []struct {
		name          string
		pos           *position.DBPosition
		currentPrice  float64
		shouldExit    bool
	}{
		{
			name: "should exit when profit >= 15%",
			pos: &position.DBPosition{
				EntryPrice:    30000,
				Target1:       34500,
				InitialShares: 100,
				Target1Filled: false,
			},
			currentPrice: 34500, // +15% profit
			shouldExit:   true,
		},
		{
			name: "should not exit when profit < 15%",
			pos: &position.DBPosition{
				EntryPrice:    30000,
				Target1:       34500,
				InitialShares: 100,
				Target1Filled: false,
			},
			currentPrice: 33000, // +10% profit
			shouldExit:   false,
		},
		{
			name: "should not exit when target1 already filled",
			pos: &position.DBPosition{
				EntryPrice:    30000,
				Target1:       34500,
				InitialShares: 100,
				Target1Filled: true,
			},
			currentPrice: 35000,
			shouldExit:   false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := evaluator.ShouldExitTarget1(tt.pos, tt.currentPrice)
			
			if tt.shouldExit {
				if decision == nil {
					t.Error("Expected exit decision, got nil")
				}
				if decision.SignalType != "SELL_TARGET1" {
					t.Errorf("Expected SELL_TARGET1, got %v", decision.SignalType)
				}
				if decision.ExitPercentage != 30 {
					t.Errorf("Expected 30%% exit, got %d%%", decision.ExitPercentage)
				}
			} else {
				if decision != nil {
					t.Errorf("Expected no exit decision, got %v", decision)
				}
			}
		})
	}
}

func TestExitEvaluator_CheckEmergencyExit(t *testing.T) {
	evaluator := position.NewExitEvaluator(position.DefaultExitEvaluatorConfig())
	
	tests := []struct {
		name         string
		pos          *position.DBPosition
		floorHitProb float64
		shouldExit   bool
	}{
		{
			name: "should exit when floor-hit > 30%",
			pos: &position.DBPosition{
				CurrentShares: 100,
			},
			floorHitProb: 35.0,
			shouldExit:   true,
		},
		{
			name: "should exit on 3+ consecutive floor hits",
			pos: &position.DBPosition{
				CurrentShares: 100,
				FloorHitDays:  3,
			},
			floorHitProb: 10.0,
			shouldExit:   true,
		},
		{
			name: "should not exit when floor-hit < 30% and no consecutive hits",
			pos: &position.DBPosition{
				CurrentShares: 100,
				FloorHitDays:  1,
			},
			floorHitProb: 20.0,
			shouldExit:   false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := evaluator.CheckEmergencyExit(tt.pos, 0, tt.floorHitProb)
			
			if tt.shouldExit {
				if decision == nil {
					t.Error("Expected emergency exit decision, got nil")
				}
				if decision.SignalType != "SELL_EMERGENCY" {
					t.Errorf("Expected SELL_EMERGENCY, got %v", decision.SignalType)
				}
				if decision.ExitPercentage != 100 {
					t.Errorf("Expected 100%% exit, got %d%%", decision.ExitPercentage)
				}
			} else {
				if decision != nil {
					t.Errorf("Expected no emergency exit, got %v", decision)
				}
			}
		})
	}
}

func TestCheckFloorHit(t *testing.T) {
	today := time.Date(2026, 2, 19, 15, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)
	twoDaysAgo := today.AddDate(0, 0, -2)

	// A stock at 30,000 with a -7% floor at 27,900.
	const floorPrice = 27900.0

	tests := []struct {
		name              string
		pos               *position.DBPosition
		currentPrice      float64
		wantIsFloorHit    bool
		wantFloorHitDays  int
		wantWasReset      bool
	}{
		{
			name: "first floor hit: counter starts at 1",
			pos: &position.DBPosition{
				FloorHitDays:  0,
				LastFloorDate: nil,
			},
			currentPrice:     floorPrice, // exactly at floor
			wantIsFloorHit:   true,
			wantFloorHitDays: 1,
			wantWasReset:     false,
		},
		{
			name: "consecutive floor hit: counter increments",
			pos: &position.DBPosition{
				FloorHitDays:  2,
				LastFloorDate: &yesterday,
			},
			currentPrice:     floorPrice - 1, // below floor (still counts)
			wantIsFloorHit:   true,
			wantFloorHitDays: 3,
			wantWasReset:     false,
		},
		{
			name: "non-consecutive: counter resets to 1",
			pos: &position.DBPosition{
				FloorHitDays:  2,
				LastFloorDate: &twoDaysAgo,
			},
			currentPrice:     floorPrice,
			wantIsFloorHit:   true,
			wantFloorHitDays: 1,
			wantWasReset:     true,
		},
		{
			name: "price above floor: no hit",
			pos: &position.DBPosition{
				FloorHitDays:  1,
				LastFloorDate: &yesterday,
			},
			currentPrice:     floorPrice * 1.05, // 5% above floor – not at floor
			wantIsFloorHit:   false,
			wantFloorHitDays: 1, // unchanged
			wantWasReset:     false,
		},
		{
			name: "same-day re-evaluation: counter stays the same",
			pos: &position.DBPosition{
				FloorHitDays:  2,
				LastFloorDate: &today,
			},
			currentPrice:     floorPrice,
			wantIsFloorHit:   true,
			wantFloorHitDays: 2, // no increment when same day
			wantWasReset:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := position.CheckFloorHit(tt.pos, tt.currentPrice, floorPrice, today)

			if result.IsFloorHit != tt.wantIsFloorHit {
				t.Errorf("IsFloorHit = %v, want %v", result.IsFloorHit, tt.wantIsFloorHit)
			}
			if result.FloorHitDays != tt.wantFloorHitDays {
				t.Errorf("FloorHitDays = %d, want %d", result.FloorHitDays, tt.wantFloorHitDays)
			}
			if result.WasReset != tt.wantWasReset {
				t.Errorf("WasReset = %v, want %v", result.WasReset, tt.wantWasReset)
			}

			// Verify in-memory mutation
			if result.IsFloorHit && tt.pos.FloorHitDays != tt.wantFloorHitDays {
				t.Errorf("pos.FloorHitDays mutated to %d, want %d", tt.pos.FloorHitDays, tt.wantFloorHitDays)
			}
		})
	}
}

