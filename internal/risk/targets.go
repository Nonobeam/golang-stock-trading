package risk

import (
	"sort"
)

// RMultipleTarget calculates target based on R-multiple.
// Target = Entry + (R * (Entry - Stop))
func RMultipleTarget(entry, stop float64, rMultiple float64, isLong bool) float64 {
	risk := StopDistance(entry, stop)

	if isLong {
		return entry + (rMultiple * risk)
	}
	return entry - (rMultiple * risk)
}

// MultipleTargets calculates 1R, 2R, 3R targets.
func MultipleTargets(entry, stop float64, isLong bool) []float64 {
	return []float64{
		RMultipleTarget(entry, stop, 1, isLong),
		RMultipleTarget(entry, stop, 2, isLong),
		RMultipleTarget(entry, stop, 3, isLong),
	}
}

// ATRTarget calculates target based on ATR.
func ATRTarget(entry, atr, multiplier float64, isLong bool) float64 {
	if isLong {
		return entry + (atr * multiplier)
	}
	return entry - (atr * multiplier)
}

// TechnicalTargets filters technical levels for valid targets.
// For longs: returns levels above entry that are at least 1R away.
// For shorts: returns levels below entry that are at least 1R away.
func TechnicalTargets(entry, stop float64, levels []float64, isLong bool) []float64 {
	if len(levels) == 0 {
		return nil
	}

	minR := StopDistance(entry, stop) // 1R minimum

	var validTargets []float64

	for _, level := range levels {
		if isLong {
			// Level must be above entry and at least 1R away
			if level > entry && (level-entry) >= minR {
				validTargets = append(validTargets, level)
			}
		} else {
			// Level must be below entry and at least 1R away
			if level < entry && (entry-level) >= minR {
				validTargets = append(validTargets, level)
			}
		}
	}

	// Sort targets by proximity to entry
	if isLong {
		sort.Float64s(validTargets) // Ascending for longs
	} else {
		sort.Sort(sort.Reverse(sort.Float64Slice(validTargets))) // Descending for shorts
	}

	return validTargets
}

// NextTarget returns the nearest target from a list of levels.
func NextTarget(entry, stop float64, levels []float64, isLong bool) float64 {
	targets := TechnicalTargets(entry, stop, levels, isLong)
	if len(targets) == 0 {
		// Default to 2R if no technical levels
		return RMultipleTarget(entry, stop, 2, isLong)
	}
	return targets[0]
}

// RiskRewardRatio calculates the risk/reward ratio.
func RiskRewardRatio(entry, stop, target float64) float64 {
	risk := StopDistance(entry, stop)
	reward := StopDistance(entry, target)

	if risk == 0 {
		return 0
	}

	return reward / risk
}

// TargetResult holds target calculation results.
type TargetResult struct {
	Target1R         float64
	Target2R         float64
	Target3R         float64
	ATRTarget        float64
	TechnicalTarget  float64
	RiskRewardRatio  float64
}

// CalculateTargets computes all target types.
func CalculateTargets(params TargetParams) TargetResult {
	targets := MultipleTargets(params.EntryPrice, params.StopPrice, params.IsLong)

	result := TargetResult{
		Target1R: targets[0],
		Target2R: targets[1],
		Target3R: targets[2],
	}

	if params.ATR > 0 {
		result.ATRTarget = ATRTarget(params.EntryPrice, params.ATR, 3, params.IsLong)
	}

	if len(params.ResistanceLevels) > 0 {
		result.TechnicalTarget = NextTarget(
			params.EntryPrice,
			params.StopPrice,
			params.ResistanceLevels,
			params.IsLong,
		)
	}

	// Calculate R:R for 2R target
	result.RiskRewardRatio = 2.0

	return result
}
