package risk

import (
	"fmt"
	"math"
	"strings"
)

// CalculateRMultipleTargets calculates targets based on R-multiples.
//
// Example:
//
//	result := CalculateRMultipleTargets(52000, 49000, []float64{2.0, 3.0, 4.0}, true)
//	// Returns targets at 58,000 (2R), 61,000 (3R), 64,000 (4R)
func CalculateRMultipleTargets(entry, stop float64, rMultiples []float64, isLong bool) RMultipleResult {
	var risk float64
	var direction float64

	if isLong {
		risk = entry - stop
		direction = 1
	} else {
		risk = stop - entry
		direction = -1
	}

	if risk <= 0 {
		return RMultipleResult{
			Method:     "R-Multiple",
			EntryPrice: entry,
			StopLoss:   stop,
		}
	}

	if len(rMultiples) == 0 {
		rMultiples = []float64{2.0, 3.0, 4.0}
	}

	targets := make([]TargetInfo, len(rMultiples))
	for i, rMult := range rMultiples {
		targetPrice := entry + (direction * risk * rMult)

		targets[i] = TargetInfo{
			TargetNumber:      i + 1,
			TargetPrice:       roundToNearest100(targetPrice),
			DistanceFromEntry: math.Abs(targetPrice - entry),
			DistancePercent:   (math.Abs(targetPrice-entry) / entry) * 100,
			RMultiple:         rMult,
			Method:            "R-Multiple",
		}
	}

	posType := "long"
	if !isLong {
		posType = "short"
	}

	return RMultipleResult{
		Method:       "R-Multiple",
		EntryPrice:   entry,
		StopLoss:     stop,
		RiskPerShare: risk,
		RiskPercent:  (risk / entry) * 100,
		PositionType: posType,
		Targets:      targets,
		Summary:      generateRSummary(targets),
	}
}

func generateRSummary(targets []TargetInfo) string {
	parts := make([]string, len(targets))
	for i, t := range targets {
		parts[i] = fmt.Sprintf("T%d: %.0f (%.1fR, +%.1f%%)",
			t.TargetNumber, t.TargetPrice, t.RMultiple, t.DistancePercent)
	}
	return strings.Join(parts, " | ")
}

func roundToNearest100(val float64) float64 {
	return math.Round(val/100) * 100
}
