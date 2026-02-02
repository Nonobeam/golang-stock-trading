package scanner

import "github.com/nonobeam/golang-stock-trading/internal/signals"

// confidenceToScore converts signal confidence to numeric score
func confidenceToScore(confidence signals.ConfidenceLevel) int {
	switch confidence {
	case signals.ConfidenceVeryHigh:
		return 10
	case signals.ConfidenceHigh:
		return 9
	case signals.ConfidenceModerate:
		return 7
	case signals.ConfidenceLow:
		return 6
	default:
		return 5
	}
}
