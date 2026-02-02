package risk

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// CalculateTechnicalTargets calculates targets from technical resistance levels.
//
// Example:
//
//	levels := []float64{55000, 58500, 62000, 65000}
//	result := CalculateTechnicalTargets(52000, 49000, levels, true)
//	// Returns top 3 valid resistance levels with R-multiples
func CalculateTechnicalTargets(entry, stop float64, resistanceLevels []float64, isLong bool) TechnicalResult {
	if len(resistanceLevels) == 0 {
		return TechnicalResult{
			Method:     "Technical Resistance",
			EntryPrice: entry,
			StopLoss:   stop,
		}
	}

	risk := math.Abs(entry - stop)

	var validLevels []float64
	if isLong {
		for _, level := range resistanceLevels {
			if level > entry {
				validLevels = append(validLevels, level)
			}
		}
		sort.Float64s(validLevels)
	} else {
		for _, level := range resistanceLevels {
			if level < entry {
				validLevels = append(validLevels, level)
			}
		}
		sort.Sort(sort.Reverse(sort.Float64Slice(validLevels)))
	}

	if len(validLevels) == 0 {
		return TechnicalResult{
			Method:     "Technical Resistance",
			EntryPrice: entry,
			StopLoss:   stop,
		}
	}

	maxTargets := 3
	if len(validLevels) > maxTargets {
		validLevels = validLevels[:maxTargets]
	}

	targets := make([]TargetInfo, len(validLevels))
	for i, level := range validLevels {
		distance := math.Abs(level - entry)
		distancePercent := (distance / entry) * 100
		rMultiple := distance / risk

		targets[i] = TargetInfo{
			TargetNumber:      i + 1,
			TargetPrice:       roundToNearest100(level),
			DistanceFromEntry: distance,
			DistancePercent:   distancePercent,
			RMultiple:         rMultiple,
			Method:            "Technical",
		}
	}

	posType := "long"
	if !isLong {
		posType = "short"
	}

	return TechnicalResult{
		Method:              "Technical Resistance",
		EntryPrice:          entry,
		StopLoss:            stop,
		RiskPerShare:        risk,
		PositionType:        posType,
		Targets:             targets,
		AllResistanceLevels: resistanceLevels,
		Summary:             generateTechSummary(targets),
	}
}

func generateTechSummary(targets []TargetInfo) string {
	parts := make([]string, len(targets))
	for i, t := range targets {
		parts[i] = fmt.Sprintf("T%d: %.0f (%.1fR, +%.1f%%)",
			t.TargetNumber, t.TargetPrice, t.RMultiple, t.DistancePercent)
	}
	return strings.Join(parts, " | ")
}
