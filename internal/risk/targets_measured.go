package risk

import (
	"fmt"
	"strings"
)

// CalculateMeasuredMove calculates targets by projecting consolidation range.
//
// Formula: Target = Breakout + (Range × Multiple)
// Where Range = ConsolidationHigh - ConsolidationLow
//
// Example:
//
//	params := MeasuredMoveParams{
//	    ConsolidationLow: 48000,
//	    ConsolidationHigh: 52000,
//	    BreakoutPrice: 52500,
//	}
//	result := CalculateMeasuredMove(params)
func CalculateMeasuredMove(params MeasuredMoveParams) MeasuredMoveResult {
	rangeSize := params.ConsolidationHigh - params.ConsolidationLow

	if rangeSize <= 0 {
		return MeasuredMoveResult{
			Method: "Measured Move",
		}
	}

	multiples := params.Multiples
	if len(multiples) == 0 {
		multiples = []float64{1.0, 1.5, 2.0}
	}

	targets := make([]TargetInfo, len(multiples))
	for i, mult := range multiples {
		targetDistance := rangeSize * mult
		targetPrice := params.BreakoutPrice + targetDistance

		targets[i] = TargetInfo{
			TargetNumber:      i + 1,
			TargetPrice:       roundToNearest100(targetPrice),
			DistanceFromEntry: targetDistance,
			DistancePercent:   (targetDistance / params.BreakoutPrice) * 100,
			RMultiple:         mult,
			Method:            fmt.Sprintf("%.1f× range", mult),
		}
	}

	rangePercent := (rangeSize / params.ConsolidationLow) * 100

	return MeasuredMoveResult{
		Method:             "Measured Move",
		ConsolidationLow:   params.ConsolidationLow,
		ConsolidationHigh:  params.ConsolidationHigh,
		ConsolidationRange: rangeSize,
		RangePercent:       rangePercent,
		BreakoutPrice:      params.BreakoutPrice,
		Targets:            targets,
		Summary:            generateMeasuredSummary(targets),
	}
}

func generateMeasuredSummary(targets []TargetInfo) string {
	parts := make([]string, len(targets))
	for i, t := range targets {
		parts[i] = fmt.Sprintf("T%d: %.0f (%.1f× range, +%.1f%%)",
			t.TargetNumber, t.TargetPrice, t.RMultiple, t.DistancePercent)
	}
	return strings.Join(parts, " | ")
}
