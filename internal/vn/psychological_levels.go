// Package vn provides Vietnamese market utilities.
package vn

import (
	"fmt"
)

// PsychologicalLevelDetector detects Vietnamese psychological price levels.
type PsychologicalLevelDetector struct {
	roundLevels []float64
}

// NewPsychologicalLevelDetector creates a new detector.
func NewPsychologicalLevelDetector() *PsychologicalLevelDetector {
	// Vietnamese round numbers (in VND thousands)
	levels := []float64{
		10_000, 20_000, 30_000, 40_000, 50_000,
		60_000, 70_000, 80_000, 90_000, 100_000,
		150_000, 200_000, 250_000, 300_000,
	}
	
	return &PsychologicalLevelDetector{roundLevels: levels}
}

// AdjustTargetForResistance adjusts target price to avoid round number resistance.
// If target is within 2% of a round level, adjust it to 2% below the level.
func (d *PsychologicalLevelDetector) AdjustTargetForResistance(targetPrice float64) (float64, string) {
	for _, level := range d.roundLevels {
		percentDiff := ((targetPrice - level) / level) * 100
		
		// If target within 2% of round level (above or slightly below)
		if percentDiff >= -2  && percentDiff <= 2 {
			adjusted := level * 0.98 // 2% below round level
			reason := "Adjusted to avoid psychological resistance at " + formatVND(level)
			return adjusted, reason
		}
	}
	
	return targetPrice, "No adjustment needed"
}

// formatVND formats price in VND format (simplified).
func formatVND(price float64) string {
	return fmt.Sprintf("%.0f VND", price)
}
