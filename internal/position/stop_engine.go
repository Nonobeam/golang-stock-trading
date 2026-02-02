package position

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// StopManagementEngine evaluates and applies stop adjustments.
type StopManagementEngine struct {
	mu      sync.RWMutex
	rules   StopAdjustmentRule
	history map[string]*StopAdjustmentHistory
}

// NewStopManagementEngine creates a new stop management engine.
func NewStopManagementEngine(rules *StopAdjustmentRule) *StopManagementEngine {
	r := DefaultStopAdjustmentRule()
	if rules != nil {
		r = *rules
	}

	return &StopManagementEngine{
		rules:   r,
		history: make(map[string]*StopAdjustmentHistory),
	}
}

// EvaluateStopAdjustment evaluates if stop should be adjusted.
func (e *StopManagementEngine) EvaluateStopAdjustment(
	position *Position,
	currentPrice float64,
	indicators *Indicators,
) *StopAdjustmentResult {
	e.mu.Lock()
	defer e.mu.Unlock()

	currentStop := position.StopLoss
	entryPrice := position.EntryPrice

	// Calculate R-multiple
	var profitPerShare float64
	if position.PositionType == "long" {
		profitPerShare = currentPrice - entryPrice
	} else {
		profitPerShare = entryPrice - currentPrice
	}

	var rMultiple float64
	if position.RiskPerShare > 0 {
		rMultiple = profitPerShare / position.RiskPerShare
	}

	// Collect all potential adjustments
	potentialAdjustments := make([]*StopAdjustmentResult, 0)

	// Rule 1: Breakeven
	if rMultiple >= e.rules.MoveToBreakevenAtR {
		if adj := e.calculateBreakevenStop(position, currentPrice, currentStop); adj != nil {
			potentialAdjustments = append(potentialAdjustments, adj)
		}
	}

	// Rule 2: Target-based
	if e.rules.AdjustOnTargetHit {
		if adj := e.calculateTargetBasedStop(position, currentPrice, currentStop); adj != nil {
			potentialAdjustments = append(potentialAdjustments, adj)
		}
	}

	// Rule 3: Trailing stop
	if indicators != nil {
		if adj := e.calculateTrailingStop(position, currentPrice, currentStop, indicators); adj != nil {
			potentialAdjustments = append(potentialAdjustments, adj)
		}
	}

	// Rule 4: Time stop
	if e.rules.EnableTimeStop {
		if adj := e.calculateTimeStop(position, currentPrice, rMultiple); adj != nil {
			potentialAdjustments = append(potentialAdjustments, adj)
		}
	}

	// Rule 5: Volatility adjustment
	if e.rules.AdjustForVolatility && indicators != nil && indicators.ATR > 0 {
		if adj := e.calculateVolatilityStop(position, currentPrice, currentStop, indicators.ATR); adj != nil {
			potentialAdjustments = append(potentialAdjustments, adj)
		}
	}

	// No adjustments triggered
	if len(potentialAdjustments) == 0 {
		return nil
	}

	// Select best adjustment (highest stop for longs, lowest for shorts)
	var best *StopAdjustmentResult
	if position.PositionType == "long" {
		for _, adj := range potentialAdjustments {
			if best == nil || adj.NewStop > best.NewStop {
				best = adj
			}
		}
	} else {
		for _, adj := range potentialAdjustments {
			if best == nil || adj.NewStop < best.NewStop {
				best = adj
			}
		}
	}

	// Validate adjustment
	if !e.validateAdjustment(position, currentStop, best.NewStop, currentPrice) {
		return nil
	}

	best.ShouldAdjust = true
	return best
}

// ApplyAdjustment applies the adjustment and records it in history.
func (e *StopManagementEngine) ApplyAdjustment(
	position *Position,
	currentPrice float64,
	result *StopAdjustmentResult,
) {
	if result == nil || !result.ShouldAdjust {
		return
	}

	oldStop := position.StopLoss
	position.StopLoss = result.NewStop

	// Record in history
	adj := StopAdjustment{
		Timestamp:    time.Now(),
		OldStop:      oldStop,
		NewStop:      result.NewStop,
		Reason:       result.Reason,
		CurrentPrice: currentPrice,
		ProfitLocked: result.ProfitLocked,
		Details:      result.Details,
	}

	if _, exists := e.history[position.PositionID]; !exists {
		e.history[position.PositionID] = &StopAdjustmentHistory{
			PositionID:  position.PositionID,
			Adjustments: []StopAdjustment{},
		}
	}
	e.history[position.PositionID].AddAdjustment(adj)
}

// GetAdjustmentHistory returns the adjustment history for a position.
func (e *StopManagementEngine) GetAdjustmentHistory(positionID string) *StopAdjustmentHistory {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.history[positionID]
}

// calculateBreakevenStop calculates breakeven stop with buffer.
func (e *StopManagementEngine) calculateBreakevenStop(
	position *Position,
	currentPrice float64,
	currentStop float64,
) *StopAdjustmentResult {
	entryPrice := position.EntryPrice

	var breakevenStop float64
	if position.PositionType == "long" {
		// Add buffer above entry
		breakevenStop = entryPrice * (1 + e.rules.BreakevenBuffer/100)

		// Only adjust if higher than current stop
		if breakevenStop <= currentStop {
			return nil
		}

		// Don't set stop above current price
		if breakevenStop >= currentPrice {
			breakevenStop = currentPrice * 0.99
		}
	} else {
		// Add buffer below entry
		breakevenStop = entryPrice * (1 - e.rules.BreakevenBuffer/100)

		if breakevenStop >= currentStop {
			return nil
		}

		if breakevenStop <= currentPrice {
			breakevenStop = currentPrice * 1.01
		}
	}

	profitLocked := (breakevenStop - entryPrice) * float64(position.SharesRemaining)

	return &StopAdjustmentResult{
		NewStop:      roundToHundred(breakevenStop),
		Reason:       ReasonBreakeven,
		ProfitLocked: profitLocked,
		Details:      fmt.Sprintf("Moved to breakeven +%.1f%% buffer", e.rules.BreakevenBuffer),
	}
}

// calculateTargetBasedStop adjusts stop when targets are hit.
func (e *StopManagementEngine) calculateTargetBasedStop(
	position *Position,
	currentPrice float64,
	currentStop float64,
) *StopAdjustmentResult {
	if len(position.Targets) == 0 || !e.rules.MoveToPreviousTarget {
		return nil
	}

	// Find which targets have been hit
	var targetsHit []Target
	for _, target := range position.Targets {
		if position.PositionType == "long" {
			if currentPrice >= target.TargetPrice {
				targetsHit = append(targetsHit, target)
			}
		} else {
			if currentPrice <= target.TargetPrice {
				targetsHit = append(targetsHit, target)
			}
		}
	}

	if len(targetsHit) == 0 {
		return nil
	}

	var newStopLevel float64
	var reasonText string

	if len(targetsHit) >= 2 {
		// Move to first target
		newStopLevel = position.Targets[0].TargetPrice
		reasonText = fmt.Sprintf("T%d hit, moved to T1", len(targetsHit))
	} else if len(targetsHit) == 1 {
		// Move to entry + half distance to T1
		if position.PositionType == "long" {
			distance := targetsHit[0].TargetPrice - position.EntryPrice
			newStopLevel = position.EntryPrice + distance*0.5
		} else {
			distance := position.EntryPrice - targetsHit[0].TargetPrice
			newStopLevel = position.EntryPrice - distance*0.5
		}
		reasonText = "T1 hit, moved to +0.5R"
	} else {
		return nil
	}

	// Validate this is better than current stop
	if position.PositionType == "long" {
		if newStopLevel <= currentStop {
			return nil
		}
	} else {
		if newStopLevel >= currentStop {
			return nil
		}
	}

	profitLocked := (newStopLevel - position.EntryPrice) * float64(position.SharesRemaining)

	return &StopAdjustmentResult{
		NewStop:      roundToHundred(newStopLevel),
		Reason:       ReasonTargetHit,
		ProfitLocked: profitLocked,
		Details:      reasonText,
	}
}

// calculateTrailingStop calculates trailing stop based on configured method.
func (e *StopManagementEngine) calculateTrailingStop(
	position *Position,
	currentPrice float64,
	currentStop float64,
	indicators *Indicators,
) *StopAdjustmentResult {
	switch e.rules.TrailingMethod {
	case TrailingMethodATR:
		return e.trailByATR(position, currentPrice, currentStop, indicators.ATR)
	case TrailingMethodEMA:
		return e.trailByEMA(position, currentPrice, currentStop, indicators.EMA20)
	case TrailingMethodPercentage:
		return e.trailByPercentage(position, currentPrice, currentStop)
	case TrailingMethodSwing:
		return e.trailBySwing(position, currentPrice, currentStop, indicators.SwingLow)
	default:
		return nil
	}
}

// trailByATR trails stop using ATR.
func (e *StopManagementEngine) trailByATR(
	position *Position,
	currentPrice float64,
	currentStop float64,
	atr float64,
) *StopAdjustmentResult {
	if atr <= 0 {
		return nil
	}

	highest := position.HighestPriceReached

	var newStop float64
	if position.PositionType == "long" {
		newStop = highest - (atr * e.rules.ATRMultiplier)

		// Ensure minimum distance from current price
		minStop := currentPrice * (1 - e.rules.MinATRDistancePercent/100)
		if newStop > minStop {
			newStop = minStop
		}

		if newStop <= currentStop {
			return nil
		}

		if newStop >= currentPrice {
			return nil
		}
	} else {
		newStop = highest + (atr * e.rules.ATRMultiplier)

		maxStop := currentPrice * (1 + e.rules.MinATRDistancePercent/100)
		if newStop < maxStop {
			newStop = maxStop
		}

		if newStop >= currentStop {
			return nil
		}

		if newStop <= currentPrice {
			return nil
		}
	}

	profitLocked := (newStop - position.EntryPrice) * float64(position.SharesRemaining)

	return &StopAdjustmentResult{
		NewStop:      roundToHundred(newStop),
		Reason:       ReasonTrailingATR,
		ProfitLocked: profitLocked,
		Details:      fmt.Sprintf("ATR trail (%.1f× ATR from highest)", e.rules.ATRMultiplier),
	}
}

// trailByEMA trails stop below EMA.
func (e *StopManagementEngine) trailByEMA(
	position *Position,
	currentPrice float64,
	currentStop float64,
	ema float64,
) *StopAdjustmentResult {
	if ema <= 0 {
		return nil
	}

	var newStop float64
	if position.PositionType == "long" {
		newStop = ema * (1 - e.rules.EMABufferPercent/100)

		if newStop <= currentStop {
			return nil
		}

		if newStop >= currentPrice {
			return nil
		}
	} else {
		newStop = ema * (1 + e.rules.EMABufferPercent/100)

		if newStop >= currentStop {
			return nil
		}

		if newStop <= currentPrice {
			return nil
		}
	}

	profitLocked := (newStop - position.EntryPrice) * float64(position.SharesRemaining)

	return &StopAdjustmentResult{
		NewStop:      roundToHundred(newStop),
		Reason:       ReasonTrailingEMA,
		ProfitLocked: profitLocked,
		Details:      fmt.Sprintf("EMA trail (%d EMA - %.1f%%)", e.rules.EMAPeriod, e.rules.EMABufferPercent),
	}
}

// trailByPercentage trails stop by percentage below highest.
func (e *StopManagementEngine) trailByPercentage(
	position *Position,
	currentPrice float64,
	currentStop float64,
) *StopAdjustmentResult {
	highest := position.HighestPriceReached

	var newStop float64
	if position.PositionType == "long" {
		newStop = highest * (1 - e.rules.TrailingPercentage/100)

		if newStop <= currentStop {
			return nil
		}

		if newStop >= currentPrice {
			return nil
		}
	} else {
		newStop = highest * (1 + e.rules.TrailingPercentage/100)

		if newStop >= currentStop {
			return nil
		}

		if newStop <= currentPrice {
			return nil
		}
	}

	profitLocked := (newStop - position.EntryPrice) * float64(position.SharesRemaining)

	return &StopAdjustmentResult{
		NewStop:      roundToHundred(newStop),
		Reason:       ReasonTrailingPercentage,
		ProfitLocked: profitLocked,
		Details:      fmt.Sprintf("%.1f%% trail from highest (%.0f)", e.rules.TrailingPercentage, highest),
	}
}

// trailBySwing trails stop below swing low.
func (e *StopManagementEngine) trailBySwing(
	position *Position,
	currentPrice float64,
	currentStop float64,
	swingLow float64,
) *StopAdjustmentResult {
	if swingLow <= 0 {
		return nil
	}

	var newStop float64
	if position.PositionType == "long" {
		newStop = swingLow * (1 - e.rules.SwingBufferPercent/100)

		if newStop <= currentStop {
			return nil
		}

		if newStop >= currentPrice {
			return nil
		}
	} else {
		newStop = swingLow * (1 + e.rules.SwingBufferPercent/100)

		if newStop >= currentStop {
			return nil
		}

		if newStop <= currentPrice {
			return nil
		}
	}

	profitLocked := (newStop - position.EntryPrice) * float64(position.SharesRemaining)

	return &StopAdjustmentResult{
		NewStop:      roundToHundred(newStop),
		Reason:       ReasonTrailingSwing,
		ProfitLocked: profitLocked,
		Details:      fmt.Sprintf("Swing trail (below %.0f - %.1f%%)", swingLow, e.rules.SwingBufferPercent),
	}
}

// calculateTimeStop calculates time-based stop for stagnant positions.
func (e *StopManagementEngine) calculateTimeStop(
	position *Position,
	currentPrice float64,
	rMultiple float64,
) *StopAdjustmentResult {
	daysHeld := int(time.Since(position.EntryDate).Hours() / 24)

	if daysHeld < e.rules.TimeStopDays {
		return nil
	}

	if rMultiple < e.rules.TimeStopMinR {
		return nil
	}

	var newStop float64
	if position.PositionType == "long" {
		newStop = currentPrice * 0.99 // 1% below to trigger exit
	} else {
		newStop = currentPrice * 1.01
	}

	profitLocked := (currentPrice - position.EntryPrice) * float64(position.SharesRemaining)

	return &StopAdjustmentResult{
		NewStop:      roundToHundred(newStop),
		Reason:       ReasonTimeStop,
		ProfitLocked: profitLocked,
		Details:      fmt.Sprintf("Time stop: %d days with +%.2fR - consider exit", daysHeld, rMultiple),
	}
}

// calculateVolatilityStop adjusts ATR multiplier based on volatility.
func (e *StopManagementEngine) calculateVolatilityStop(
	position *Position,
	currentPrice float64,
	currentStop float64,
	atr float64,
) *StopAdjustmentResult {
	if atr <= 0 {
		return nil
	}

	atrPercent := (atr / currentPrice) * 100

	// Adjust multiplier based on volatility
	var adjustedMultiplier float64
	if atrPercent > 8.0 {
		adjustedMultiplier = e.rules.VolatilityMultiplierMax
	} else if atrPercent > 5.0 {
		adjustedMultiplier = 2.0
	} else if atrPercent > 3.0 {
		adjustedMultiplier = e.rules.ATRMultiplier
	} else {
		adjustedMultiplier = e.rules.VolatilityMultiplierMin
	}

	// Calculate stop with adjusted multiplier
	highest := position.HighestPriceReached

	var newStop float64
	if position.PositionType == "long" {
		newStop = highest - (atr * adjustedMultiplier)

		if newStop <= currentStop {
			return nil
		}

		if newStop >= currentPrice {
			return nil
		}
	} else {
		newStop = highest + (atr * adjustedMultiplier)

		if newStop >= currentStop {
			return nil
		}

		if newStop <= currentPrice {
			return nil
		}
	}

	profitLocked := (newStop - position.EntryPrice) * float64(position.SharesRemaining)

	return &StopAdjustmentResult{
		NewStop:      roundToHundred(newStop),
		Reason:       ReasonVolatilityChange,
		ProfitLocked: profitLocked,
		Details:      fmt.Sprintf("Volatility-adjusted ATR trail (%.1f× ATR, %.1f%% vol)", adjustedMultiplier, atrPercent),
	}
}

// validateAdjustment validates that adjustment meets all rules.
func (e *StopManagementEngine) validateAdjustment(
	position *Position,
	currentStop float64,
	newStop float64,
	currentPrice float64,
) bool {
	// Check minimum adjustment amount
	adjustmentAmount := math.Abs(newStop - currentStop)
	if adjustmentAmount < e.rules.MinAdjustmentAmount {
		return false
	}

	// Never widen stop
	if e.rules.NeverWidenStop {
		if position.PositionType == "long" {
			if newStop < currentStop {
				return false
			}
		} else {
			if newStop > currentStop {
				return false
			}
		}
	}

	// New stop must be below current price (for longs)
	if position.PositionType == "long" {
		if newStop >= currentPrice {
			return false
		}
	} else {
		if newStop <= currentPrice {
			return false
		}
	}

	return true
}

// roundToHundred rounds a price to the nearest 100 VND.
func roundToHundred(price float64) float64 {
	return math.Round(price/100) * 100
}
