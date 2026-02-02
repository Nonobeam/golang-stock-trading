package statistics

import (
	"math"
	"time"
)

// AnalyzeDrawdowns performs comprehensive drawdown analysis
func AnalyzeDrawdowns(equityCurve []float64, dates []time.Time) DrawdownMetrics {
	if len(equityCurve) < 2 {
		return DrawdownMetrics{}
	}

	runningMax := make([]float64, len(equityCurve))
	runningMax[0] = equityCurve[0]

	for i := 1; i < len(equityCurve); i++ {
		runningMax[i] = math.Max(runningMax[i-1], equityCurve[i])
	}

	drawdowns := make([]float64, len(equityCurve))
	for i := range equityCurve {
		if runningMax[i] > 0 {
			drawdowns[i] = (equityCurve[i] - runningMax[i]) / runningMax[i]
		}
	}

	maxDDIdx := 0
	maxDD := 0.0
	for i, dd := range drawdowns {
		if math.Abs(dd) > maxDD {
			maxDD = math.Abs(dd)
			maxDDIdx = i
		}
	}

	peakIdx := 0
	for i := maxDDIdx - 1; i >= 0; i-- {
		if equityCurve[i] >= equityCurve[peakIdx] {
			peakIdx = i
		}
		if equityCurve[i] == runningMax[maxDDIdx] {
			peakIdx = i
			break
		}
	}

	recovered := false
	var recoveryIdx *int
	var recoveryDuration *int

	if maxDDIdx < len(equityCurve)-1 {
		peakValue := equityCurve[peakIdx]
		for i := maxDDIdx + 1; i < len(equityCurve); i++ {
			if equityCurve[i] >= peakValue {
				recovered = true
				idx := i
				recoveryIdx = &idx
				dur := i - maxDDIdx
				recoveryDuration = &dur
				break
			}
		}
	}

	periods := identifyDrawdownPeriods(equityCurve, drawdowns, dates)

	avgDrawdown := 0.0
	significantDD := make([]DrawdownPeriod, 0)
	for _, p := range periods {
		if p.DepthPercent >= 5.0 {
			significantDD = append(significantDD, p)
		}
	}

	if len(significantDD) > 0 {
		totalDD := 0.0
		for _, p := range significantDD {
			totalDD += p.DepthPercent
		}
		avgDrawdown = totalDD / float64(len(significantDD))
	}

	currentDD := math.Abs(drawdowns[len(drawdowns)-1]) * 100

	return DrawdownMetrics{
		MaxDrawdown:            maxDD,
		MaxDrawdownPercent:     maxDD * 100,
		MaxDDPeakIdx:           peakIdx,
		MaxDDTroughIdx:         maxDDIdx,
		MaxDDDuration:          maxDDIdx - peakIdx,
		MaxDDPeakValue:         equityCurve[peakIdx],
		MaxDDTroughValue:       equityCurve[maxDDIdx],
		Recovered:              recovered,
		RecoveryIdx:            recoveryIdx,
		RecoveryDuration:       recoveryDuration,
		CurrentDrawdownPercent: currentDD,
		AvgDrawdownPercent:     avgDrawdown,
		NumDrawdownPeriods:     len(significantDD),
		DrawdownPeriods:        periods,
	}
}

// IdentifyDrawdownPeriods finds all peak-to-recovery cycles
func identifyDrawdownPeriods(equityCurve []float64, drawdownPercent []float64, dates []time.Time) []DrawdownPeriod {
	periods := make([]DrawdownPeriod, 0)
	inDrawdown := false
	var currentPeriod *DrawdownPeriod

	for i := 0; i < len(equityCurve); i++ {
		dd := drawdownPercent[i] * 100

		if dd < -0.1 && !inDrawdown {
			inDrawdown = true
			peakIdx := i
			for j := i - 1; j >= 0; j-- {
				if equityCurve[j] >= equityCurve[peakIdx] {
					peakIdx = j
				} else {
					break
				}
			}

			currentPeriod = &DrawdownPeriod{
				PeakIdx:      peakIdx,
				TroughIdx:    i,
				PeakValue:    equityCurve[peakIdx],
				TroughValue:  equityCurve[i],
				DepthPercent: math.Abs(dd),
			}

			if len(dates) > peakIdx {
				peak := dates[peakIdx]
				currentPeriod.PeakDate = &peak
			}
			if len(dates) > i {
				trough := dates[i]
				currentPeriod.TroughDate = &trough
			}

		} else if inDrawdown && currentPeriod != nil {
			if math.Abs(dd) > currentPeriod.DepthPercent {
				currentPeriod.TroughIdx = i
				currentPeriod.TroughValue = equityCurve[i]
				currentPeriod.DepthPercent = math.Abs(dd)

				if len(dates) > i {
					trough := dates[i]
					currentPeriod.TroughDate = &trough
				}
			}

			if dd >= -0.01 {
				currentPeriod.RecoveryIdx = &i
				recovVal := equityCurve[i]
				currentPeriod.RecoveryValue = &recovVal
				currentPeriod.Duration = i - currentPeriod.PeakIdx
				recovDur := i - currentPeriod.TroughIdx
				currentPeriod.RecoveryDuration = &recovDur
				currentPeriod.Recovered = true

				if len(dates) > i {
					recovery := dates[i]
					currentPeriod.RecoveryDate = &recovery
				}

				periods = append(periods, *currentPeriod)
				inDrawdown = false
				currentPeriod = nil
			}
		}
	}

	if inDrawdown && currentPeriod != nil {
		currentPeriod.Recovered = false
		currentPeriod.Duration = len(equityCurve) - currentPeriod.PeakIdx
		periods = append(periods, *currentPeriod)
	}

	return periods
}

// CalculateRecoveryFactor calculates net profit / max drawdown
func CalculateRecoveryFactor(netProfit float64, maxDrawdown float64) (float64, string) {
	if maxDrawdown == 0 {
		return math.Inf(1), "No drawdown - exceptional"
	}

	factor := netProfit / maxDrawdown
	interpretation := interpretRecoveryFactor(factor)

	return factor, interpretation
}

func interpretRecoveryFactor(factor float64) string {
	if math.IsInf(factor, 1) {
		return "No drawdown - exceptional"
	}

	switch {
	case factor >= 10:
		return "Exceptional (≥10)"
	case factor >= 5:
		return "Excellent (≥5)"
	case factor >= 3:
		return "Good (≥3)"
	case factor >= 2:
		return "Acceptable (≥2)"
	case factor > 0:
		return "Poor but positive"
	default:
		return "Negative - losing money"
	}
}
