package position

import (
	"math"
	"time"
)

// PositionMetricsCalculator calculates comprehensive metrics for positions.
type PositionMetricsCalculator struct{}

// NewMetricsCalculator creates a new metrics calculator.
func NewMetricsCalculator() *PositionMetricsCalculator {
	return &PositionMetricsCalculator{}
}

// CalculateMetrics calculates all position metrics.
func (c *PositionMetricsCalculator) CalculateMetrics(p *Position) PositionMetrics {
	metrics := PositionMetrics{
		PositionID:     p.PositionID,
		Ticker:         p.Ticker,
		CurrentPrice:   p.CurrentPrice,
		SharesOriginal: p.Shares,
	}

	// Basic P&L calculations
	var unrealizedPLPerShare float64
	if p.PositionType == "long" {
		unrealizedPLPerShare = p.CurrentPrice - p.EntryPrice
	} else { // short
		unrealizedPLPerShare = p.EntryPrice - p.CurrentPrice
	}

	// Total unrealized P&L
	metrics.UnrealizedPLPerShare = unrealizedPLPerShare
	metrics.UnrealizedPL = unrealizedPLPerShare * float64(p.SharesRemaining)
	if p.EntryPrice > 0 {
		metrics.UnrealizedPLPercent = (unrealizedPLPerShare / p.EntryPrice) * 100
	}

	// R-multiple
	if p.RiskPerShare > 0 {
		metrics.RMultiple = unrealizedPLPerShare / p.RiskPerShare
	}

	// Account impact
	if p.PositionValue > 0 {
		metrics.AccountImpactPercent = (metrics.UnrealizedPL / p.PositionValue) * p.RiskPercent
	}

	// Realized P&L from partial exits
	metrics.RealizedPL = c.calculateRealizedPL(p)
	metrics.RealizedShares = c.calculateRealizedShares(p)

	// Total P&L
	metrics.TotalPL = metrics.RealizedPL + metrics.UnrealizedPL

	// Position details
	metrics.SharesRemaining = p.SharesRemaining
	metrics.PositionRemainingValue = float64(p.SharesRemaining) * p.CurrentPrice

	// MAE/MFE
	maeMetrics := c.calculateMAEMFE(p)
	metrics.MAE = maeMetrics.MAE
	metrics.MAEPercent = maeMetrics.MAEPercent
	metrics.MAE_R = maeMetrics.MAE_R
	metrics.LowestPrice = maeMetrics.LowestPrice
	metrics.MFE = maeMetrics.MFE
	metrics.MFEPercent = maeMetrics.MFEPercent
	metrics.MFE_R = maeMetrics.MFE_R
	metrics.HighestPrice = maeMetrics.HighestPrice

	// Time metrics
	timeMetrics := c.calculateTimeMetrics(p)
	metrics.DaysInTrade = timeMetrics.Days
	metrics.HoursInTrade = timeMetrics.Hours
	metrics.EntryDate = p.EntryDate.Format("2006-01-02 15:04")

	// Stop metrics
	stopMetrics := c.calculateStopMetrics(p)
	metrics.StopLoss = p.StopLoss
	metrics.StopDistance = stopMetrics.Distance
	metrics.StopDistancePercent = stopMetrics.DistancePercent
	metrics.StopHit = stopMetrics.StopHit

	// Target progress
	metrics.TargetProgress = c.calculateTargetProgress(p)

	// Risk metrics
	riskMetrics := c.calculateRiskMetrics(p, metrics.UnrealizedPL)
	metrics.RiskRemaining = riskMetrics.RiskRemaining
	metrics.RiskRemainingPercent = riskMetrics.RiskRemainingPercent
	metrics.RiskRewardCurrent = riskMetrics.RiskRewardCurrent

	return metrics
}

// calculateRealizedPL calculates total realized P&L from all exits.
func (c *PositionMetricsCalculator) calculateRealizedPL(p *Position) float64 {
	var total float64
	for _, exit := range p.Exits {
		if p.PositionType == "long" {
			total += (exit.Price - p.EntryPrice) * float64(exit.Shares)
		} else {
			total += (p.EntryPrice - exit.Price) * float64(exit.Shares)
		}
	}
	return total
}

// calculateRealizedShares returns total shares exited.
func (c *PositionMetricsCalculator) calculateRealizedShares(p *Position) int {
	var total int
	for _, exit := range p.Exits {
		total += exit.Shares
	}
	return total
}

// MAEMFEResult contains MAE/MFE calculation results.
type MAEMFEResult struct {
	MAE          float64
	MAEPercent   float64
	MAE_R        float64
	LowestPrice  float64
	MFE          float64
	MFEPercent   float64
	MFE_R        float64
	HighestPrice float64
}

// calculateMAEMFE calculates Maximum Adverse and Favorable Excursions.
func (c *PositionMetricsCalculator) calculateMAEMFE(p *Position) MAEMFEResult {
	result := MAEMFEResult{
		LowestPrice:  p.LowestPriceReached,
		HighestPrice: p.HighestPriceReached,
	}

	if p.PositionType == "long" {
		// MAE: How far below entry did it go
		result.MAE = p.EntryPrice - p.LowestPriceReached
		// MFE: How far above entry did it go
		result.MFE = p.HighestPriceReached - p.EntryPrice
	} else { // short
		// MAE: How far above entry did it go
		result.MAE = p.HighestPriceReached - p.EntryPrice
		// MFE: How far below entry did it go
		result.MFE = p.EntryPrice - p.LowestPriceReached
	}

	// As percentages
	if p.EntryPrice > 0 {
		result.MAEPercent = (result.MAE / p.EntryPrice) * 100
		result.MFEPercent = (result.MFE / p.EntryPrice) * 100
	}

	// As R-multiples
	if p.RiskPerShare > 0 {
		result.MAE_R = result.MAE / p.RiskPerShare
		result.MFE_R = result.MFE / p.RiskPerShare
	}

	return result
}

// TimeResult contains time-based metrics.
type TimeResult struct {
	Days      int
	Hours     float64
	TimeDelta time.Duration
}

// calculateTimeMetrics calculates time-based position metrics.
func (c *PositionMetricsCalculator) calculateTimeMetrics(p *Position) TimeResult {
	timeDelta := time.Since(p.EntryDate)
	return TimeResult{
		Days:      int(timeDelta.Hours() / 24),
		Hours:     timeDelta.Hours(),
		TimeDelta: timeDelta,
	}
}

// StopResult contains stop loss related metrics.
type StopResult struct {
	Distance        float64
	DistancePercent float64
	StopHit         bool
	Cushion         float64 // Positive = safe, negative = stop breached
}

// calculateStopMetrics calculates stop loss related metrics.
func (c *PositionMetricsCalculator) calculateStopMetrics(p *Position) StopResult {
	var distance float64
	var stopHit bool

	if p.PositionType == "long" {
		distance = p.CurrentPrice - p.StopLoss
		stopHit = p.CurrentPrice <= p.StopLoss
	} else {
		distance = p.StopLoss - p.CurrentPrice
		stopHit = p.CurrentPrice >= p.StopLoss
	}

	var distancePercent float64
	if p.CurrentPrice > 0 {
		distancePercent = (math.Abs(distance) / p.CurrentPrice) * 100
	}

	return StopResult{
		Distance:        math.Abs(distance),
		DistancePercent: distancePercent,
		StopHit:         stopHit,
		Cushion:         distance,
	}
}

// calculateTargetProgress calculates progress toward each target.
func (c *PositionMetricsCalculator) calculateTargetProgress(p *Position) []TargetProgress {
	if len(p.Targets) == 0 {
		return []TargetProgress{}
	}

	progress := make([]TargetProgress, 0, len(p.Targets))

	for _, target := range p.Targets {
		var distanceToTarget, progressMade, totalDistance float64
		var targetHit bool

		if p.PositionType == "long" {
			distanceToTarget = target.TargetPrice - p.CurrentPrice
			progressMade = p.CurrentPrice - p.EntryPrice
			totalDistance = target.TargetPrice - p.EntryPrice
			targetHit = p.CurrentPrice >= target.TargetPrice
		} else {
			distanceToTarget = p.CurrentPrice - target.TargetPrice
			progressMade = p.EntryPrice - p.CurrentPrice
			totalDistance = p.EntryPrice - target.TargetPrice
			targetHit = p.CurrentPrice <= target.TargetPrice
		}

		var percentComplete float64
		if totalDistance > 0 {
			percentComplete = (progressMade / totalDistance) * 100
			percentComplete = math.Max(0, math.Min(100, percentComplete))
		}

		var distancePercent float64
		if p.CurrentPrice > 0 {
			distancePercent = (math.Abs(distanceToTarget) / p.CurrentPrice) * 100
		}

		progress = append(progress, TargetProgress{
			TargetNumber:     target.TargetNumber,
			TargetPrice:      target.TargetPrice,
			RMultiple:        target.RMultiple,
			PercentToSell:    target.PercentToSell,
			DistanceToTarget: math.Abs(distanceToTarget),
			DistancePercent:  distancePercent,
			PercentComplete:  percentComplete,
			TargetHit:        targetHit,
		})
	}

	return progress
}

// RiskResult contains risk-related metrics.
type RiskResult struct {
	RiskRemaining        float64
	RiskRemainingPercent float64
	RiskRewardCurrent    float64
}

// calculateRiskMetrics calculates current risk metrics.
func (c *PositionMetricsCalculator) calculateRiskMetrics(p *Position, unrealizedPL float64) RiskResult {
	var riskRemaining float64

	// Current risk remaining (if stop is hit from current price)
	if p.PositionType == "long" {
		riskRemaining = (p.CurrentPrice - p.StopLoss) * float64(p.SharesRemaining)
	} else {
		riskRemaining = (p.StopLoss - p.CurrentPrice) * float64(p.SharesRemaining)
	}

	var riskRemainingPercent float64
	if p.PositionValue > 0 {
		riskRemainingPercent = (riskRemaining / p.PositionValue) * 100
	}

	// Current risk/reward (to next unfilled target)
	var riskRewardCurrent float64
	if len(p.Targets) > 0 && riskRemaining > 0 {
		// Find first unfilled target
		var nextTarget *Target
		for i := range p.Targets {
			target := &p.Targets[i]
			if p.PositionType == "long" {
				if p.CurrentPrice < target.TargetPrice {
					nextTarget = target
					break
				}
			} else {
				if p.CurrentPrice > target.TargetPrice {
					nextTarget = target
					break
				}
			}
		}

		if nextTarget != nil {
			potentialReward := math.Abs(nextTarget.TargetPrice-p.CurrentPrice) * float64(p.SharesRemaining)
			if riskRemaining > 0 {
				riskRewardCurrent = potentialReward / riskRemaining
			}
		}
	}

	return RiskResult{
		RiskRemaining:        riskRemaining,
		RiskRemainingPercent: riskRemainingPercent,
		RiskRewardCurrent:    riskRewardCurrent,
	}
}
