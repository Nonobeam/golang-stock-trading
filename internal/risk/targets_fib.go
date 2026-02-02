package risk

import (
	"fmt"
	"strings"
)

// CalculateFibonacciExtensions calculates Fibonacci extension targets from swing points.
//
// Formula: Target = C + ((B - A) × Fib_Ratio)
// Where A = swing low, B = swing high, C = pullback low (entry)
//
// Example:
//
//	params := FibonacciParams{
//	    SwingLow: 45000,
//	    SwingHigh: 60000,
//	    PullbackLow: 52000,
//	}
//	result := CalculateFibonacciExtensions(params)
func CalculateFibonacciExtensions(params FibonacciParams) FibonacciResult {
	waveSize := params.SwingHigh - params.SwingLow

	if waveSize <= 0 {
		return FibonacciResult{
			Method: "Fibonacci Extensions",
		}
	}

	fibRatios := params.FibRatios
	if len(fibRatios) == 0 {
		fibRatios = []float64{0.618, 1.0, 1.618, 2.618}
	}

	targets := make([]TargetInfo, len(fibRatios))
	for i, ratio := range fibRatios {
		extensionDistance := waveSize * ratio
		targetPrice := params.PullbackLow + extensionDistance

		targets[i] = TargetInfo{
			TargetNumber:      i + 1,
			TargetPrice:       roundToNearest100(targetPrice),
			DistanceFromEntry: extensionDistance,
			DistancePercent:   (extensionDistance / params.PullbackLow) * 100,
			RMultiple:         ratio,
			Method:            fmt.Sprintf("Fib %.1f%%", ratio*100),
		}
	}

	return FibonacciResult{
		Method:      "Fibonacci Extensions",
		SwingLow:    params.SwingLow,
		SwingHigh:   params.SwingHigh,
		PullbackLow: params.PullbackLow,
		WaveSize:    waveSize,
		WavePercent: (waveSize / params.SwingLow) * 100,
		Targets:     targets,
		Summary:     generateFibSummary(targets),
	}
}

func generateFibSummary(targets []TargetInfo) string {
	parts := make([]string, len(targets))
	for i, t := range targets {
		parts[i] = fmt.Sprintf("T%d: %.0f (Fib %.1f%%, +%.1f%%)",
			t.TargetNumber, t.TargetPrice, t.RMultiple*100, t.DistancePercent)
	}
	return strings.Join(parts, " | ")
}
