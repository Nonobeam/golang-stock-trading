package statistics

import (
	"math"
	"sort"
)

// AnalyzeRDistribution performs comprehensive R-multiple distribution analysis
func AnalyzeRDistribution(trades []Trade) RDistributionMetrics {
	if len(trades) == 0 {
		return RDistributionMetrics{}
	}

	// Extract R-multiples (using PnLPercent / InitialRisk as proxy if InitialRisk is set)
	rMultiples := make([]float64, 0, len(trades))
	for _, t := range trades {
		var r float64
		if t.InitialRisk > 0 {
			r = t.PnL / t.InitialRisk
		} else {
			// Fallback to percent if no initial risk set
			r = t.PnLPercent / 100.0
		}
		rMultiples = append(rMultiples, r)
	}

	// Basic statistics
	meanR := mean(rMultiples)
	medianR := median(rMultiples)
	stdDev := standardDeviation(rMultiples)

	// Distribution shape
	skewness := calculateSkewness(rMultiples, meanR, stdDev)
	kurtosis := calculateKurtosis(rMultiples, meanR, stdDev)

	// Percentiles
	percentiles := calculatePercentiles(rMultiples)

	// Bucket analysis
	buckets := bucketRMultiples(trades, rMultiples)

	// Tail analysis
	tailAnalysis := analyzeTails(trades, rMultiples)

	// Quality assessment
	quality := assessDistributionQuality(meanR, stdDev, skewness)

	return RDistributionMetrics{
		MeanR:                  meanR,
		MedianR:                medianR,
		StdDev:                 stdDev,
		Skewness:               skewness,
		SkewnessInterpretation: interpretSkewness(skewness),
		Kurtosis:               kurtosis,
		KurtosisInterpretation: interpretKurtosis(kurtosis),
		Percentiles:            percentiles,
		RBuckets:               buckets,
		TailAnalysis:           tailAnalysis,
		QualityAssessment:      quality,
	}
}

// calculateSkewness calculates the skewness of the distribution
// Skewness = E[(X - μ)³] / σ³
func calculateSkewness(data []float64, mean, stdDev float64) float64 {
	if stdDev == 0 || len(data) == 0 {
		return 0
	}

	n := float64(len(data))
	sum := 0.0
	for _, x := range data {
		sum += math.Pow((x-mean)/stdDev, 3)
	}

	return sum / n
}

// interpretSkewness provides human-readable interpretation
func interpretSkewness(skewness float64) string {
	if skewness > 1.0 {
		return "Highly positively skewed - many small losses, few big wins (IDEAL)"
	} else if skewness > 0.5 {
		return "Positively skewed - big winners offset small losses (GOOD)"
	} else if skewness > -0.5 {
		return "Approximately symmetric - wins and losses similar"
	} else if skewness > -1.0 {
		return "Negatively skewed - many small wins, few big losses (DANGEROUS)"
	}
	return "Highly negatively skewed - big losers offset small wins (VERY BAD)"
}

// calculateKurtosis calculates excess kurtosis (tailedness)
// Kurtosis = E[(X - μ)⁴] / σ⁴ - 3
func calculateKurtosis(data []float64, mean, stdDev float64) float64 {
	if stdDev == 0 || len(data) == 0 {
		return 0
	}

	n := float64(len(data))
	sum := 0.0
	for _, x := range data {
		sum += math.Pow((x-mean)/stdDev, 4)
	}

	return (sum / n) - 3.0
}

// interpretKurtosis provides human-readable interpretation
func interpretKurtosis(kurtosis float64) string {
	if kurtosis > 3 {
		return "Very fat tails - extreme outcomes common (high risk)"
	} else if kurtosis > 1 {
		return "Fat tails - more extreme outcomes than normal"
	} else if kurtosis > -1 {
		return "Normal tails - typical distribution"
	}
	return "Thin tails - few extreme outcomes"
}

// calculatePercentiles calculates key percentiles
func calculatePercentiles(data []float64) map[string]float64 {
	if len(data) == 0 {
		return map[string]float64{}
	}

	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)

	return map[string]float64{
		"10th": percentile(sorted, 10),
		"25th": percentile(sorted, 25),
		"50th": percentile(sorted, 50),
		"75th": percentile(sorted, 75),
		"90th": percentile(sorted, 90),
	}
}

// percentile calculates the nth percentile
func percentile(sortedData []float64, p float64) float64 {
	if len(sortedData) == 0 {
		return 0
	}

	rank := (p / 100.0) * float64(len(sortedData)-1)
	lower := int(math.Floor(rank))
	upper := int(math.Ceil(rank))

	if lower == upper {
		return sortedData[lower]
	}

	// Linear interpolation
	weight := rank - float64(lower)
	return sortedData[lower]*(1-weight) + sortedData[upper]*weight
}

// bucketRMultiples categorizes trades into R-multiple ranges
func bucketRMultiples(trades []Trade, rMultiples []float64) RBucketStats {
	buckets := RBucketStats{}
	total := float64(len(trades))

	if total == 0 {
		return buckets
	}

	for _, r := range rMultiples {
		if r < -2 {
			buckets.LargeLoss.Count++
		} else if r < -1 {
			buckets.MediumLoss.Count++
		} else if r < 0 {
			buckets.SmallLoss.Count++
		} else if r < 1 {
			buckets.SmallWin.Count++
		} else if r < 3 {
			buckets.MediumWin.Count++
		} else if r < 5 {
			buckets.LargeWin.Count++
		} else {
			buckets.HugeWin.Count++
		}
	}

	// Calculate percentages
	buckets.LargeLoss.Percentage = float64(buckets.LargeLoss.Count) / total * 100
	buckets.MediumLoss.Percentage = float64(buckets.MediumLoss.Count) / total * 100
	buckets.SmallLoss.Percentage = float64(buckets.SmallLoss.Count) / total * 100
	buckets.SmallWin.Percentage = float64(buckets.SmallWin.Count) / total * 100
	buckets.MediumWin.Percentage = float64(buckets.MediumWin.Count) / total * 100
	buckets.LargeWin.Percentage = float64(buckets.LargeWin.Count) / total * 100
	buckets.HugeWin.Percentage = float64(buckets.HugeWin.Count) / total * 100

	return buckets
}

// analyzeTails examines extreme outcomes (top and bottom 10%)
func analyzeTails(trades []Trade, rMultiples []float64) TailAnalysis {
	if len(trades) == 0 {
		return TailAnalysis{}
	}

	sorted := make([]float64, len(rMultiples))
	copy(sorted, rMultiples)
	sort.Float64s(sorted)

	// Right tail (top 10% - big winners)
	rightThreshold := percentile(sorted, 90)
	var rightTailR []float64
	var rightTailSum float64
	for _, r := range rMultiples {
		if r >= rightThreshold {
			rightTailR = append(rightTailR, r)
			rightTailSum += r
		}
	}

	// Left tail (bottom 10% - big losers)
	leftThreshold := percentile(sorted, 10)
	var leftTailR []float64
	var leftTailSum float64
	for _, r := range rMultiples {
		if r <= leftThreshold {
			leftTailR = append(leftTailR, r)
			leftTailSum += r
		}
	}

	// Total sum for contribution calculation
	totalSum := 0.0
	for _, r := range rMultiples {
		totalSum += r
	}

	rightContribution := 0.0
	if totalSum != 0 {
		rightContribution = (rightTailSum / totalSum) * 100
	}

	leftContribution := 0.0
	if totalSum != 0 {
		leftContribution = (leftTailSum / totalSum) * 100
	}

	return TailAnalysis{
		RightTail: TailDetail{
			Threshold:              rightThreshold,
			Count:                  len(rightTailR),
			MeanR:                  mean(rightTailR),
			ContributionToTotalPct: rightContribution,
		},
		LeftTail: TailDetail{
			Threshold:              leftThreshold,
			Count:                  len(leftTailR),
			MeanR:                  mean(leftTailR),
			ContributionToTotalPct: leftContribution,
		},
	}
}

// assessDistributionQuality provides overall quality assessment
func assessDistributionQuality(meanR, stdDev, skewness float64) string {
	score := 0

	// Positive expectancy
	if meanR > 0.5 {
		score += 3
	} else if meanR > 0.3 {
		score += 2
	} else if meanR > 0 {
		score += 1
	}

	// Low volatility relative to mean
	if meanR > 0 {
		cvRatio := stdDev / meanR
		if cvRatio < 2 {
			score += 2
		} else if cvRatio < 3 {
			score += 1
		}
	}

	// Positive skewness (big winners, small losers)
	if skewness > 0.5 {
		score += 2
	} else if skewness > 0 {
		score += 1
	}

	// Assessment based on score
	if score >= 6 {
		return "Excellent distribution - consistent profits with big winner potential"
	} else if score >= 4 {
		return "Good distribution - positive expectancy with acceptable risk"
	} else if score >= 2 {
		return "Fair distribution - profitable but inconsistent"
	}
	return "Poor distribution - needs improvement"
}
