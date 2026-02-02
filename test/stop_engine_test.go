package test

import (
	"testing"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/position"
)

func TestBreakevenStopAt1R(t *testing.T) {
	rules := position.DefaultStopAdjustmentRule()
	rules.MoveToBreakevenAtR = 1.0
	rules.BreakevenBuffer = 0.5

	engine := position.NewStopManagementEngine(&rules)

	pos := &position.Position{
		PositionID:          "TEST_1",
		Ticker:              "VNM",
		EntryPrice:          52000,
		StopLoss:            49000,
		RiskPerShare:        3000,
		SharesRemaining:     500,
		PositionType:        "long",
		HighestPriceReached: 55000,
		EntryDate:           time.Now().AddDate(0, 0, -5),
	}

	// At +1R (55000 = 52000 + 3000)
	result := engine.EvaluateStopAdjustment(pos, 55000, nil)

	if result == nil {
		t.Fatal("Expected adjustment at +1R")
	}

	if result.Reason != position.ReasonBreakeven {
		t.Errorf("Expected BREAKEVEN reason, got %s", result.Reason)
	}

	// Breakeven = 52000 × 1.005 = 52260, rounded to 52300
	expectedStop := 52300.0
	if result.NewStop != expectedStop {
		t.Errorf("Expected new stop %.0f, got %.0f", expectedStop, result.NewStop)
	}
}

func TestBreakevenBufferCalculation(t *testing.T) {
	rules := position.DefaultStopAdjustmentRule()
	rules.BreakevenBuffer = 1.0 // 1% buffer

	engine := position.NewStopManagementEngine(&rules)

	pos := &position.Position{
		PositionID:          "TEST_2",
		EntryPrice:          50000,
		StopLoss:            47000,
		RiskPerShare:        3000,
		SharesRemaining:     500,
		PositionType:        "long",
		HighestPriceReached: 53000,
		EntryDate:           time.Now().AddDate(0, 0, -5),
	}

	result := engine.EvaluateStopAdjustment(pos, 53000, nil)

	if result == nil {
		t.Fatal("Expected adjustment")
	}

	// 50000 × 1.01 = 50500
	expectedStop := 50500.0
	if result.NewStop != expectedStop {
		t.Errorf("Expected new stop %.0f, got %.0f", expectedStop, result.NewStop)
	}
}

func TestTargetBasedStopWhenT2Hit(t *testing.T) {
	rules := position.DefaultStopAdjustmentRule()
	engine := position.NewStopManagementEngine(&rules)

	pos := &position.Position{
		PositionID:          "TEST_3",
		EntryPrice:          52000,
		StopLoss:            52300, // Already at breakeven
		RiskPerShare:        3000,
		SharesRemaining:     500,
		PositionType:        "long",
		HighestPriceReached: 62000,
		EntryDate:           time.Now().AddDate(0, 0, -5),
		Targets: []position.Target{
			{TargetNumber: 1, TargetPrice: 58000, PercentToSell: 25, RMultiple: 2.0},
			{TargetNumber: 2, TargetPrice: 61000, PercentToSell: 25, RMultiple: 3.0},
		},
	}

	// Price above T2
	result := engine.EvaluateStopAdjustment(pos, 62000, nil)

	if result == nil {
		t.Fatal("Expected adjustment when T2 hit")
	}

	if result.Reason != position.ReasonTargetHit {
		t.Errorf("Expected TARGET_HIT reason, got %s", result.Reason)
	}

	// Should move to T1 level
	expectedStop := 58000.0
	if result.NewStop != expectedStop {
		t.Errorf("Expected new stop at T1 %.0f, got %.0f", expectedStop, result.NewStop)
	}
}

func TestATRTrailingCalculation(t *testing.T) {
	rules := position.DefaultStopAdjustmentRule()
	rules.TrailingMethod = position.TrailingMethodATR
	rules.ATRMultiplier = 1.5

	engine := position.NewStopManagementEngine(&rules)

	pos := &position.Position{
		PositionID:          "TEST_4",
		EntryPrice:          52000,
		StopLoss:            52300,
		RiskPerShare:        3000,
		SharesRemaining:     500,
		PositionType:        "long",
		HighestPriceReached: 58000,
		EntryDate:           time.Now().AddDate(0, 0, -5),
	}

	indicators := &position.Indicators{
		ATR: 2500,
	}

	result := engine.EvaluateStopAdjustment(pos, 58000, indicators)

	if result == nil {
		t.Fatal("Expected trailing adjustment")
	}

	if result.Reason != position.ReasonTrailingATR {
		t.Errorf("Expected TRAILING_ATR reason, got %s", result.Reason)
	}

	// 58000 - (2500 × 1.5) = 58000 - 3750 = 54250
	expectedStop := 54300.0 // rounded to hundred
	if result.NewStop != expectedStop {
		t.Errorf("Expected new stop %.0f, got %.0f", expectedStop, result.NewStop)
	}
}

func TestEMATrailingCalculation(t *testing.T) {
	rules := position.DefaultStopAdjustmentRule()
	rules.TrailingMethod = position.TrailingMethodEMA
	rules.EMABufferPercent = 1.0

	engine := position.NewStopManagementEngine(&rules)

	pos := &position.Position{
		PositionID:          "TEST_5",
		EntryPrice:          52000,
		StopLoss:            52300,
		RiskPerShare:        3000,
		SharesRemaining:     500,
		PositionType:        "long",
		HighestPriceReached: 58000,
		EntryDate:           time.Now().AddDate(0, 0, -5),
	}

	indicators := &position.Indicators{
		EMA20: 56000,
	}

	result := engine.EvaluateStopAdjustment(pos, 58000, indicators)

	if result == nil {
		t.Fatal("Expected EMA trailing adjustment")
	}

	if result.Reason != position.ReasonTrailingEMA {
		t.Errorf("Expected TRAILING_EMA reason, got %s", result.Reason)
	}

	// 56000 × 0.99 = 55440
	expectedStop := 55400.0
	if result.NewStop != expectedStop {
		t.Errorf("Expected new stop %.0f, got %.0f", expectedStop, result.NewStop)
	}
}

func TestPercentageTrailingCalculation(t *testing.T) {
	rules := position.DefaultStopAdjustmentRule()
	rules.TrailingMethod = position.TrailingMethodPercentage
	rules.TrailingPercentage = 5.0

	engine := position.NewStopManagementEngine(&rules)

	pos := &position.Position{
		PositionID:          "TEST_6",
		EntryPrice:          52000,
		StopLoss:            52300,
		RiskPerShare:        3000,
		SharesRemaining:     500,
		PositionType:        "long",
		HighestPriceReached: 58000,
		EntryDate:           time.Now().AddDate(0, 0, -5),
	}

	// Pass empty indicators to trigger trailing stop calculation
	indicators := &position.Indicators{}

	result := engine.EvaluateStopAdjustment(pos, 58000, indicators)

	if result == nil {
		t.Fatal("Expected percentage trailing adjustment")
	}

	if result.Reason != position.ReasonTrailingPercentage {
		t.Errorf("Expected TRAILING_PERCENTAGE reason, got %s", result.Reason)
	}

	// 58000 × 0.95 = 55100
	expectedStop := 55100.0
	if result.NewStop != expectedStop {
		t.Errorf("Expected new stop %.0f, got %.0f", expectedStop, result.NewStop)
	}
}

func TestNeverWidenValidation(t *testing.T) {
	rules := position.DefaultStopAdjustmentRule()
	rules.NeverWidenStop = true
	rules.TrailingMethod = position.TrailingMethodATR

	engine := position.NewStopManagementEngine(&rules)

	pos := &position.Position{
		PositionID:          "TEST_7",
		EntryPrice:          52000,
		StopLoss:            55000, // Already high
		RiskPerShare:        3000,
		SharesRemaining:     500,
		PositionType:        "long",
		HighestPriceReached: 58000,
		EntryDate:           time.Now().AddDate(0, 0, -5),
	}

	indicators := &position.Indicators{
		ATR: 2500,
	}

	// ATR would calculate 54250, but current stop is 55000
	result := engine.EvaluateStopAdjustment(pos, 58000, indicators)

	// Should be nil because new stop (54250) < current stop (55000)
	if result != nil {
		t.Error("Expected no adjustment when would widen stop")
	}
}

func TestMinimumAdjustmentFilter(t *testing.T) {
	rules := position.DefaultStopAdjustmentRule()
	rules.MinAdjustmentAmount = 500

	engine := position.NewStopManagementEngine(&rules)

	pos := &position.Position{
		PositionID:          "TEST_8",
		EntryPrice:          52000,
		StopLoss:            52250, // Only 50 VND below potential breakeven
		RiskPerShare:        3000,
		SharesRemaining:     500,
		PositionType:        "long",
		HighestPriceReached: 55000,
		EntryDate:           time.Now().AddDate(0, 0, -5),
	}

	result := engine.EvaluateStopAdjustment(pos, 55000, nil)

	// Breakeven = 52260, current = 52250, diff = 10 < 500 min
	// Should be rejected
	if result != nil && result.ShouldAdjust {
		t.Error("Expected no adjustment for small change")
	}
}

func TestSelectBestAdjustment(t *testing.T) {
	rules := position.DefaultStopAdjustmentRule()
	rules.TrailingMethod = position.TrailingMethodATR

	engine := position.NewStopManagementEngine(&rules)

	pos := &position.Position{
		PositionID:          "TEST_9",
		EntryPrice:          52000,
		StopLoss:            49000,
		RiskPerShare:        3000,
		SharesRemaining:     500,
		PositionType:        "long",
		HighestPriceReached: 62000,
		EntryDate:           time.Now().AddDate(0, 0, -5), // 5 days ago
		Targets: []position.Target{
			{TargetNumber: 1, TargetPrice: 58000, PercentToSell: 25, RMultiple: 2.0},
			{TargetNumber: 2, TargetPrice: 61000, PercentToSell: 25, RMultiple: 3.0},
		},
	}

	indicators := &position.Indicators{
		ATR: 2500,
	}

	// With price at 62000:
	// - Breakeven would be ~52260
	// - ATR trailing would be 62000 - 3750 = 58250
	// - Target-based (T2 hit) would be 58000

	result := engine.EvaluateStopAdjustment(pos, 62000, indicators)

	if result == nil {
		t.Fatal("Expected adjustment")
	}

	// ATR trailing (58250) should be highest and selected
	if result.Reason != position.ReasonTrailingATR {
		t.Errorf("Expected TRAILING_ATR (highest stop), got %s", result.Reason)
	}
}
