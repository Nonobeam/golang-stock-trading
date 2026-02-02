package statistics

import (
	"math"
)

// VNAdjustments handles Vietnam-specific market adjustments
type VNAdjustments struct {
	config VNConfig
}

// NewVNAdjustments creates a new VN adjustments calculator
func NewVNAdjustments(config VNConfig) *VNAdjustments {
	return &VNAdjustments{config: config}
}

// InterpretSharpeForVN interprets Sharpe ratio with VN thresholds
func (vn *VNAdjustments) InterpretSharpeForVN(sharpe float64) string {
	if sharpe >= 2.0 {
		return "Excellent"
	} else if sharpe >= 1.0 {
		return "Good"
	} else if sharpe >= 0.7 {
		return "Fair"
	} else if sharpe >= 0.5 {
		return "Below Average"
	}
	return "Poor"
}

// InterpretSortinoForVN interprets Sortino ratio with VN thresholds
func (vn *VNAdjustments) InterpretSortinoForVN(sortino float64) string {
	if sortino >= 2.5 {
		return "Excellent"
	} else if sortino >= 1.5 {
		return "Good"
	} else if sortino >= 1.0 {
		return "Fair"
	} else if sortino >= 0.7 {
		return "Below Average"
	}
	return "Poor"
}

// InterpretCalmarForVN interprets Calmar ratio with VN thresholds
func (vn *VNAdjustments) InterpretCalmarForVN(calmar float64) string {
	if calmar >= 3.0 {
		return "Excellent"
	} else if calmar >= 2.0 {
		return "Good"
	} else if calmar >= 1.0 {
		return "Fair"
	} else if calmar >= 0.5 {
		return "Below Average"
	}
	return "Poor"
}

// InterpretMaxDDForVN interprets max drawdown with VN acceptable thresholds
func (vn *VNAdjustments) InterpretMaxDDForVN(dd float64) string {
	ddPercent := dd * 100
	if ddPercent < 10 {
		return "Excellent (Low Risk)"
	} else if ddPercent < 15 {
		return "Good (Acceptable)"
	} else if ddPercent < 20 {
		return "Fair (Within VN Limits)"
	} else if ddPercent < 25 {
		return "Warning (High Risk)"
	}
	return "Critical (Excessive Risk)"
}

// CalculateGapSlippageFactor calculates impact of -7%/+7% limits
func (vn *VNAdjustments) CalculateGapSlippageFactor(trades []Trade) float64 {
	if len(trades) == 0 {
		return 0.0
	}

	var gapLosses int
	var totalSlippage float64
	var totalLosses int

	for _, trade := range trades {
		if trade.PnL < 0 {
			totalLosses++
			// Detect if trade likely hit floor (-7% limit)
			lossPercent := (trade.PnL / (trade.EntryPrice * float64(trade.Quantity))) * 100
			if lossPercent <= -6.5 { // Close to -7% floor
				gapLosses++
				// Calculate slippage beyond intended 1R risk
				if trade.InitialRisk > 0 {
					actualLoss := math.Abs(trade.PnL)
					excessLoss := actualLoss - trade.InitialRisk
					if excessLoss > 0 {
						totalSlippage += (excessLoss / trade.InitialRisk)
					}
				}
			}
		}
	}

	if gapLosses == 0 {
		return 0.0
	}

	// Average slippage factor
	avgSlippage := totalSlippage / float64(gapLosses)
	// Weight by frequency of gaps
	gapFrequency := float64(gapLosses) / float64(totalLosses)

	return math.Min(avgSlippage*gapFrequency, 0.5) // Cap at 50%
}

// AdjustExpectancyForGaps adjusts expectancy for gap risk
func (vn *VNAdjustments) AdjustExpectancyForGaps(expectancy, gapFactor float64) float64 {
	return expectancy * (1.0 - gapFactor)
}

// CalculateCapitalEfficiency calculates impact of T+2 settlement
func (vn *VNAdjustments) CalculateCapitalEfficiency(trades []Trade) float64 {
	if len(trades) == 0 {
		return 1.0
	}

	var totalHoldingDays float64
	for _, trade := range trades {
		holdingDays := trade.ExitTime.Sub(trade.EntryTime).Hours() / 24.0
		totalHoldingDays += holdingDays
	}

	avgHoldingDays := totalHoldingDays / float64(len(trades))

	// Capital locked for avg holding + 2 days (T+2)
	capitalLockedDays := avgHoldingDays + 2.0

	// If avg holding is 8 days, capital locked for 10 days
	// Assume trading every day, efficiency = 8/10 = 0.8 (80% usable)
	efficiency := avgHoldingDays / capitalLockedDays

	return efficiency
}

// CalculateVNMetrics calculates all Vietnam-specific metrics
func (vn *VNAdjustments) CalculateVNMetrics(trades []Trade, baseExpectancy float64) VNMetrics {
	gapFactor := vn.CalculateGapSlippageFactor(trades)
	gapAdjustedExp := vn.AdjustExpectancyForGaps(baseExpectancy, gapFactor)
	capEfficiency := vn.CalculateCapitalEfficiency(trades)

	// Count gap losses
	var gapLosses int
	var totalLosses int
	var totalSlippagePct float64

	for _, trade := range trades {
		if trade.PnL < 0 {
			totalLosses++
			lossPercent := (trade.PnL / (trade.EntryPrice * float64(trade.Quantity))) * 100
			if lossPercent <= -6.5 { // Likely hit floor
				gapLosses++
				if trade.InitialRisk > 0 {
					actualLoss := math.Abs(trade.PnL)
					slippagePct := ((actualLoss - trade.InitialRisk) / trade.InitialRisk) * 100
					if slippagePct > 0 {
						totalSlippagePct += slippagePct
					}
				}
			}
		}
	}

	gapLossPercentage := 0.0
	if totalLosses > 0 {
		gapLossPercentage = (float64(gapLosses) / float64(totalLosses)) * 100
	}

	avgSlippage := 0.0
	if gapLosses > 0 {
		avgSlippage = totalSlippagePct / float64(gapLosses)
	}

	return VNMetrics{
		GapSlippageFactor:     gapFactor,
		GapAdjustedExpectancy: gapAdjustedExp,
		CapitalEfficiency:     capEfficiency,
		EffectiveAnnualReturn: 0.0, // Will be calculated later with actual returns
		TotalGapLosses:        gapLosses,
		GapLossPercentage:     gapLossPercentage,
		AvgSlippageBeyondStop: avgSlippage,
	}
}
