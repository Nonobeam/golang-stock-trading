package statistics

import (
	"math"
	"sort"
)

// Helper functions shared across statistics package

// mean calculates the average of a dataset
func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// median calculates the median value of a dataset
func median(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}

	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)

	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2.0
	}
	return sorted[n/2]
}

// standardDeviation calculates the standard deviation of a dataset
func standardDeviation(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	m := mean(values)
	var sumSquares float64

	for _, v := range values {
		diff := v - m
		sumSquares += diff * diff
	}

	variance := sumSquares / float64(len(values)-1)
	return math.Sqrt(variance)
}
