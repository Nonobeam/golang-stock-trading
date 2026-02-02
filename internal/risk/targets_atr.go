package risk

import (
	"fmt"
	"strings"
)

// CalculateATRTargets calculates targets based on Average True Range.
//
// Example:
//
//	result := CalculateATRTargets(52000, 2500, []float64{2.0, 3.0, 4.0}, true)
//	// Returns targets based on ATR volatility
func CalculateATRTargets(entry, atr float64, atrMultiples []float64, isLong bool) ATRResult {
	if atr <= 0 {
		return ATRResult{
			Method:     "ATR-Based",
			EntryPrice: entry,
			ATR:        atr,
		}
	}

	if len(atrMultiples) == 0 {
		atrMultiples = []float64{2.0, 3.0, 4.0}
	}

	direction := 1.0
	if !isLong {
		direction = -1.0
	}

	targets := make([]TargetInfo, len(atrMultiples))
	for i, mult := range atrMultiples {
		targetDistance := atr * mult
		targetPrice := entry + (direction * targetDistance)

		targets[i] = TargetInfo{
			TargetNumber:      i + 1,
			TargetPrice:       roundToNearest100(targetPrice),
			DistanceFromEntry: targetDistance,
			DistancePercent:   (targetDistance / entry) * 100,
			RMultiple:         mult,
			Method:            "ATR",
		}
	}

	posType := "long"
	if !isLong {
		posType = "short"
	}

	return ATRResult{
		Method:       "ATR-Based",
		EntryPrice:   entry,
		ATR:          atr,
		ATRPercent:   (atr / entry) * 100,
		PositionType: posType,
		Targets:      targets,
		Summary:      generateATRSummary(targets),
	}
}

func generateATRSummary(targets []TargetInfo) string {
	parts := make([]string, len(targets))
	for i, t := range targets {
		parts[i] = fmt.Sprintf("T%d: %.0f (%.1f×ATR, +%.1f%%)",
			t.TargetNumber, t.TargetPrice, t.RMultiple, t.DistancePercent)
	}
	return strings.Join(parts, " | ")
}
