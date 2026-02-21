// Package position provides exit decision logic for automated graduated profit-taking.
package position

import (
	"time"
)

// Signal type constants (avoiding import cycle with signals package)
const (
	SignalTypeSellTarget1   = "SELL_TARGET1"
	SignalTypeSellTarget2   = "SELL_TARGET2"
	SignalTypeSellTarget3   = "SELL_TARGET3"
	SignalTypeSellEmergency = "SELL_EMERGENCY"
)

// FloorHitResult holds the outcome of a floor-hit check for a single position.
type FloorHitResult struct {
	IsFloorHit      bool
	FloorHitDays    int        // updated consecutive count
	LastFloorDate   *time.Time // updated date
	WasReset        bool       // true if counter was reset (non-consecutive prior hit)
}

// FloorHitThreshold is how close to the floor price counts as "at floor" (0.1%).
const FloorHitThreshold = 0.001

// ExitDecision represents a decision to exit a position.
type ExitDecision struct {
	SignalType      string    // SELL_TARGET1, SELL_TARGET2, SELL_TARGET3, SELL_EMERGENCY
	TargetLevel     int      // 1, 2, 3, or 0 for emergency
	ExitPercentage  int      // 30, 30, 40, or 100
	Shares          int      // Calculated shares to exit
	Reason          string   // Detailed explanation
	Priority        int      // 1=emergency, 2=target hit, 3=trailing stop
	Timestamp       time.Time
}

// ExitEvaluatorConfig holds configuration for exit evaluation.
type ExitEvaluatorConfig struct {
	Target1ProfitPercent float64 // Default: 15.0%
	Target2ProfitPercent float64 // Default: 25.0%
	EmergencyFloorThreshold float64 // Default: 30.0% floor-hit probability
}

// DefaultExitEvaluatorConfig returns default configuration.
func DefaultExitEvaluatorConfig() ExitEvaluatorConfig {
	return ExitEvaluatorConfig{
		Target1ProfitPercent:    15.0,
		Target2ProfitPercent:    25.0,
		EmergencyFloorThreshold: 30.0,
	}
}

// ExitEvaluator evaluates positions for exit conditions.
type ExitEvaluator struct {
	config ExitEvaluatorConfig
}

// NewExitEvaluator creates a new exit evaluator with given config.
func NewExitEvaluator(config ExitEvaluatorConfig) *ExitEvaluator {
	return &ExitEvaluator{config: config}
}

// EvaluatePosition checks a position against all exit criteria.
// Returns ExitDecision if position should be exited, nil otherwise.
func (e *ExitEvaluator) EvaluatePosition(pos *DBPosition, currentPrice float64, floorHitProb float64) *ExitDecision {
	// Priority 1: Check emergency exits first
	if decision := e.CheckEmergencyExit(pos, currentPrice, floorHitProb); decision != nil {
		return decision
	}
	
	// Priority 2: Check Target 1 exit
	if !pos.Target1Filled {
		if decision := e.ShouldExitTarget1(pos, currentPrice); decision != nil {
			return decision
		}
	}
	
	// Priority 3: Check Target 2 exit
	if pos.Target1Filled && !pos.Target2Filled {
		if decision := e.ShouldExitTarget2(pos, currentPrice); decision != nil {
			return decision
		}
	}
	
	// Priority 4: Check Target 3 (trailing stop) exit
	if pos.Target1Filled && pos.Target2Filled {
		if decision := e.ShouldExitTarget3(pos, currentPrice); decision != nil {
			return decision
		}
	}
	
	// No exit condition met
	return nil
}

// CheckEmergencyExit checks for conditions requiring immediate full exit.
func (e *ExitEvaluator) CheckEmergencyExit(pos *DBPosition, currentPrice float64, floorHitProb float64) *ExitDecision {
	// Emergency condition 1: Floor-hit probability > threshold
	if floorHitProb > e.config.EmergencyFloorThreshold {
		return &ExitDecision{
			SignalType:     SignalTypeSellEmergency,
			TargetLevel:    0,
			ExitPercentage: 100,
			Shares:         pos.CurrentShares,
			Reason:         "Emergency exit: Floor-hit probability exceeded threshold",
			Priority:       1,
			Timestamp:      time.Now(),
		}
	}
	
	// Emergency condition 2: Consecutive floor hits (3+ days)
	if pos.FloorHitDays >= 3 {
		return &ExitDecision{
			SignalType:     SignalTypeSellEmergency,
			TargetLevel:    0,
			ExitPercentage: 100,
			Shares:         pos.CurrentShares,
			Reason:         "Emergency exit: 3+ consecutive floor hits",
			Priority:       1,
			Timestamp:      time.Now(),
		}
	}
	
	return nil
}

// ShouldExitTarget1 checks if Target 1 exit criteria are met (+15% profit OR resistance).
func (e *ExitEvaluator) ShouldExitTarget1(pos *DBPosition, currentPrice float64) *ExitDecision {
	if pos.Target1Filled {
		return nil // Already exited
	}
	
	profitPercent := ((currentPrice - pos.EntryPrice) / pos.EntryPrice) * 100
	
	// Condition: Profit >= 15% OR price >= target1
	if profitPercent >= e.config.Target1ProfitPercent || currentPrice >= pos.Target1 {
		shares := calculateExitShares(pos.InitialShares, 30) // 30% of position
		
		return &ExitDecision{
			SignalType:     SignalTypeSellTarget1,
			TargetLevel:    1,
			ExitPercentage: 30,
			Shares:         shares,
			Reason:         "Target 1: +15% profit threshold reached",
			Priority:       2,
			Timestamp:      time.Now(),
		}
	}
	
	return nil
}

// ShouldExitTarget2 checks if Target 2 exit criteria are met (+25% profit AND cleared resistance).
func (e *ExitEvaluator) ShouldExitTarget2(pos *DBPosition, currentPrice float64) *ExitDecision {
	if !pos.Target1Filled || pos.Target2Filled {
		return nil // Target1 must be filled first, Target2 not yet filled
	}
	
	profitPercent := ((currentPrice - pos.EntryPrice) / pos.EntryPrice) * 100
	
	// Condition: Profit >= 25% AND price >= target2
	if profitPercent >= e.config.Target2ProfitPercent && currentPrice >= pos.Target2 {
		shares := calculateExitShares(pos.InitialShares, 30) // 30% of initial position
		
		return &ExitDecision{
			SignalType:     SignalTypeSellTarget2,
			TargetLevel:    2,
			ExitPercentage: 30,
			Shares:         shares,
			Reason:         "Target 2: +25% profit and resistance cleared",
			Priority:       2,
			Timestamp:      time.Now(),
		}
	}
	
	return nil
}

// ShouldExitTarget3 checks if Target 3 (trailing stop) exit criteria are met.
func (e *ExitEvaluator) ShouldExitTarget3(pos *DBPosition, currentPrice float64) *ExitDecision {
	if !pos.Target1Filled || !pos.Target2Filled {
		return nil // Both targets must be filled first
	}
	
	// TODO: Integrate with existing trailing stop logic in internal/risk/targets_trailing.go
	// For now, placeholder - trailing stop hit check would go here
	
	return nil
}

// calculateExitShares calculates shares to exit based on percentage of initial position.
func calculateExitShares(initialShares int, percentage int) int {
	shares := (initialShares * percentage) / 100
	return shares
}

// CheckFloorHit detects whether the position's stock hit the -7% floor today and
// updates the in-memory consecutive counter.  It must be called once per position per day.
//
// Parameters:
//   - pos:          position to update (FloorHitDays and LastFloorDate are mutated)
//   - currentPrice: current market price (from SymbolInfo.LastPrice)
//   - floorPrice:   today's floor price (from SymbolInfo.Floor)
//   - today:        the trading date being evaluated (avoids clock dependency in tests)
//
// Returns a FloorHitResult with the updated counter state.  The caller is responsible
// for persisting FloorHitDays and LastFloorDate back to the database.
func CheckFloorHit(pos *DBPosition, currentPrice, floorPrice float64, today time.Time) FloorHitResult {
	// A stock is "at floor" when the current price is within FloorHitThreshold of the floor price.
	atFloor := floorPrice > 0 && currentPrice <= floorPrice*(1+FloorHitThreshold)

	if !atFloor {
		// Not at floor – counter stays unchanged; no persistence needed.
		return FloorHitResult{IsFloorHit: false, FloorHitDays: pos.FloorHitDays, LastFloorDate: pos.LastFloorDate}
	}

	today = today.Truncate(24 * time.Hour)
	newCounterDays := 1
	wasReset := false

	if pos.LastFloorDate != nil {
		lastDate := pos.LastFloorDate.Truncate(24 * time.Hour)
		daysSinceLast := int(today.Sub(lastDate).Hours() / 24)

		switch daysSinceLast {
		case 0:
			// Same day (called twice within one run) – no change.
			newCounterDays = pos.FloorHitDays
		case 1:
			// Consecutive trading day – increment.
			newCounterDays = pos.FloorHitDays + 1
		default:
			// Gap in floor hits – reset counter.
			newCounterDays = 1
			wasReset = true
		}
	}

	// Mutate the position so CheckEmergencyExit sees the updated counter immediately.
	pos.FloorHitDays = newCounterDays
	pos.LastFloorDate = &today

	return FloorHitResult{
		IsFloorHit:    true,
		FloorHitDays:  newCounterDays,
		LastFloorDate: &today,
		WasReset:      wasReset,
	}
}


// DBPosition represents a database position record with exit tracking fields.
type DBPosition struct {
	PositionID    string
	Symbol        string
	EntryPrice    float64
	InitialShares int
	CurrentShares int
	Target1       float64
	Target2       float64
	Target3       float64
	
	// Exit tracking
	Target1Filled      bool
	Target2Filled      bool
	TrailingStopActive bool
	Target1ExitPrice   *float64
	Target2ExitPrice   *float64
	Target1ExitDate    *time.Time
	Target2ExitDate    *time.Time
	
	// Vietnamese market tracking
	CeilingHitDate *time.Time
	CeilingLockDays int
	FloorHitDays    int
	LastFloorDate   *time.Time
}
