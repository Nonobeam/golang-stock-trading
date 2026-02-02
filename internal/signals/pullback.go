package signals

import (
	"fmt"
	"math"
)

// PullbackDetector identifies pullback entry setups in uptrends.
type PullbackDetector struct {
	config PullbackConfig
}

// NewPullbackDetector creates a new pullback detector with configuration.
func NewPullbackDetector(config PullbackConfig) *PullbackDetector {
	return &PullbackDetector{
		config: config,
	}
}

// Scan analyzes market data for pullback entry setup.
// Two-phase detection: Prerequisites → Triggers
func (d *PullbackDetector) Scan(data *MarketData) (*ScanResult, error) {
	// Phase 1: Check prerequisites (structural conditions)
	prereqResult := d.checkPrerequisites(data)
	
	if !prereqResult.Passes {
		return &ScanResult{
			SetupDetected:        false,
			SetupType:            "Pullback",
			Reason:               "Prerequisites not met",
			Issues:               prereqResult.Issues,
			Stage:                "Prerequisites Check",
			PrerequisitesDetails: prereqResult,
		}, nil
	}
	
	// Phase 2: Detect entry triggers
	supports := FindSupportLevels(data.DailySeries, 60)
	triggerResult := d.detectTriggers(data, supports)
	
	if !triggerResult.Sufficient {
		return &ScanResult{
			SetupDetected:        false,
			SetupType:            "Pullback",
			Reason:               triggerResult.Recommendation,
			Issues:               []string{triggerResult.Recommendation},
			Stage:                "Entry Triggers Check",
			PrerequisitesDetails: prereqResult,
			TriggersDetails:      triggerResult,
		}, nil
	}
	
	// Setup detected! Generate entry signal
	signal := d.generateSignal(data, triggerResult, supports)
	
	return &ScanResult{
		SetupDetected:        true,
		SetupType:            "Pullback",
		Signal:               signal,
		PrerequisitesDetails: prereqResult,
		TriggersDetails:      triggerResult,
	}, nil
}

// checkPrerequisites verifies structural conditions for pullback setup.
func (d *PullbackDetector) checkPrerequisites(data *MarketData) *PrerequisiteResult {
	issues := []string{}
	details := []string{}
	
	// Check 1: Price > EMA20 > EMA50 (uptrend structure)
	if data.CurrentClose <= data.EMA20 {
		issues = append(issues, fmt.Sprintf("Price %.0f not above EMA20 %.0f", data.CurrentClose, data.EMA20))
	} else {
		details = append(details, fmt.Sprintf("✓ Price %.0f above EMA20 %.0f", data.CurrentClose, data.EMA20))
	}
	
	if data.EMA20 <= data.EMA50 {
		issues = append(issues, fmt.Sprintf("EMA20 %.0f not above EMA50 %.0f (no uptrend)", data.EMA20, data.EMA50))
	} else {
		details = append(details, fmt.Sprintf("✓ EMA20 %.0f above EMA50 %.0f (uptrend structure)", data.EMA20, data.EMA50))
	}
	
	// Check 2: EMA slopes are positive (compare current vs 5 days ago)
	closes := data.Closes
	if len(closes) >= 25 {
		ema20Slope := calculateEMASlope(closes, 20, 5)
		ema50Slope := calculateEMASlope(closes, 50, 5)
		
		if ema20Slope == "negative" {
			issues = append(issues, "EMA20 slope is negative (trend weakening)")
		} else {
			details = append(details, "✓ EMA20 slope positive")
		}
		
		if ema50Slope == "negative" {
			issues = append(issues, "EMA50 slope is negative")
		} else {
			details = append(details, "✓ EMA50 slope positive")
		}
	}
	
	// Check 3: ADX >= configured minimum (trend strength)
	if data.ADX < d.config.MinADX {
		issues = append(issues, fmt.Sprintf("ADX %.1f below minimum %.1f (weak trend)", data.ADX, d.config.MinADX))
	} else {
		details = append(details, fmt.Sprintf("✓ ADX %.1f shows strong trend", data.ADX))
	}
	
	// Check 4: Weekly uptrend confirmation
	if data.WeeklyClose > 0 && data.WeeklySMA200 > 0 {
		if data.WeeklyClose <= data.WeeklySMA200 {
			issues = append(issues, "Weekly close below SMA200 (weekly downtrend)")
		} else {
			details = append(details, "✓ Weekly uptrend confirmed")
		}
		
		if data.WeeklyRSI <= d.config.MinWeeklyRSI {
			issues = append(issues, fmt.Sprintf("Weekly RSI %.1f too low", data.WeeklyRSI))
		} else {
			details = append(details, fmt.Sprintf("✓ Weekly RSI %.1f healthy", data.WeeklyRSI))
		}
	}
	
	// Check 5: Rally and pullback characteristics
	highs := data.Highs
	if len(highs) >= 30 {
		// Find recent high (last 20 days)
		recentHigh := highs[len(highs)-20]
		for i := len(highs) - 19; i < len(highs); i++ {
			if highs[i] > recentHigh {
				recentHigh = highs[i]
			}
		}
		
		rallyPercent := ((recentHigh - data.EMA20) / data.EMA20) * 100
		
		if rallyPercent < d.config.MinRallyPercent {
			issues = append(issues, fmt.Sprintf("Rally only %.1f%%, need >= %.1f%%", rallyPercent, d.config.MinRallyPercent))
		} else if rallyPercent > d.config.MaxRallyPercent {
			issues = append(issues, fmt.Sprintf("Rally %.1f%% too extended (max %.1f%%)", rallyPercent, d.config.MaxRallyPercent))
		} else {
			details = append(details, fmt.Sprintf("✓ Rally %.1f%% from EMA20 (healthy)", rallyPercent))
		}
		
		// Check pullback duration
		daysSinceHigh := daysSinceHighFunc(highs, 30)
		if daysSinceHigh < d.config.MinPullbackDays {
			issues = append(issues, fmt.Sprintf("Only %d days since high, need >= %d", daysSinceHigh, d.config.MinPullbackDays))
		} else if daysSinceHigh > d.config.MaxPullbackDays {
			issues = append(issues, fmt.Sprintf("%d days since high, exceeds max %d (trend may be broken)", daysSinceHigh, d.config.MaxPullbackDays))
		} else {
			details = append(details, fmt.Sprintf("✓ Pullback duration %d days (ideal)", daysSinceHigh))
		}
	}
	
	// Check 6: Price within proximity of EMA20
	proximityPercent := math.Abs((data.CurrentClose-data.EMA20)/data.EMA20) * 100
	if proximityPercent > d.config.EMAProximityPercent {
		issues = append(issues, fmt.Sprintf("Price %.1f%% from EMA20, exceeds %.1f%% limit", proximityPercent, d.config.EMAProximityPercent))
	} else {
		details = append(details, fmt.Sprintf("✓ Price within %.1f%% of EMA20", proximityPercent))
	}
	
	// Check 7: RSI in favorable range
	if data.RSI < d.config.MinRSI {
		issues = append(issues, fmt.Sprintf("RSI %.1f too low (oversold)", data.RSI))
	} else if data.RSI > d.config.MaxRSI {
		issues = append(issues, fmt.Sprintf("RSI %.1f too high (overbought)", data.RSI))
	} else {
		details = append(details, fmt.Sprintf("✓ RSI %.1f in healthy range", data.RSI))
	}
	
	passes := len(issues) == 0
	
	return &PrerequisiteResult{
		Passes:  passes,
		Details: details,
		Issues:  issues,
	}
}

// detectTriggers identifies specific entry signals.
func (d *PullbackDetector) detectTriggers(data *MarketData, supports []SupportLevel) *TriggerResult {
	triggers := []TriggerInfo{}
	
	// Get current and previous candle data
	closes := data.Closes
	opens := data.Opens
	highs := data.Highs
	lows := data.Lows
	volumes := data.Volumes
	
	if len(closes) < 2 {
		return &TriggerResult{
			TriggerCount:   0,
			Triggers:       triggers,
			Sufficient:     false,
			Recommendation: "Insufficient data for trigger detection",
		}
	}
	
	currentIdx := len(closes) - 1
	prevIdx := currentIdx - 1
	
	// Trigger 1: Bullish candlestick pattern
	pattern := DetectBullishCandle(
		opens[currentIdx], highs[currentIdx], lows[currentIdx], closes[currentIdx],
		opens[prevIdx], highs[prevIdx], lows[prevIdx], closes[prevIdx],
	)
	
	if pattern.IsBullish {
		triggers = append(triggers, TriggerInfo{
			Trigger:     "Bullish Candle",
			Description: pattern.PatternName,
			Strength:    pattern.Strength,
		})
	}
	
	// Trigger 2: Volume spike
	if len(volumes) >= 20 {
		recentVolumes := volumes[len(volumes)-20 : len(volumes)-1] // Exclude current
		avgVolume := average(recentVolumes)
		currentVolume := volumes[currentIdx]
		
		if currentVolume > avgVolume*1.5 {
			volumePercentile := CalculateVolumePercentile(currentVolume, recentVolumes)
			if volumePercentile >= 75 {
				spikePercent := ((currentVolume / avgVolume) - 1.0) * 100
				triggers = append(triggers, TriggerInfo{
					Trigger:     "Volume Spike",
					Description: fmt.Sprintf("Volume +%.0f%% above average (%.0fth percentile)", spikePercent, volumePercentile),
					Strength:    "strong",
				})
			}
		}
	}
	
	// Trigger 3: Stochastic crossover in oversold zone
	if data.StochK > 0 && data.StochD > 0 {
		// Check if we have previous values (simplified - in real implementation, calculate from series)
		if data.StochK > data.StochD && data.StochK < 50 {
			triggers = append(triggers, TriggerInfo{
				Trigger:     "Stochastic Crossover",
				Description: fmt.Sprintf("%%K crossed above %%D at %.1f", data.StochK),
				Strength:    getStochasticStrength(data.StochK),
			})
		}
	}
	
	// Trigger 4: Support confluence
	if CheckSupportConfluence(data.CurrentClose, supports, 3.0) {
		triggers = append(triggers, TriggerInfo{
			Trigger:     "Support Confluence",
			Description: "Price near multiple support levels (EMA20 + swing low/fib)",
			Strength:    "strong",
		})
	}
	
	// Trigger 5: Gap up from support
	if currentIdx > 0 {
		prevClose := closes[prevIdx]
		currentOpen := opens[currentIdx]
		gapPercent := ((currentOpen - prevClose) / prevClose) * 100
		
		nearEMA20 := math.Abs((prevClose-data.EMA20)/data.EMA20)*100 < 2.0
		
		if gapPercent > 1.0 && nearEMA20 {
			triggers = append(triggers, TriggerInfo{
				Trigger:     "Gap Up",
				Description: fmt.Sprintf("%.1f%% gap from near EMA20", gapPercent),
				Strength:    "moderate",
			})
		}
	}
	
	triggerCount := len(triggers)
	sufficient := triggerCount >= d.config.MinTriggers
	
	var recommendation string
	if sufficient {
		recommendation = fmt.Sprintf("%d triggers detected (sufficient)", triggerCount)
	} else {
		recommendation = fmt.Sprintf("Only %d trigger detected (need %d)", triggerCount, d.config.MinTriggers)
	}
	
	return &TriggerResult{
		TriggerCount:   triggerCount,
		Triggers:       triggers,
		Sufficient:     sufficient,
		Recommendation: recommendation,
	}
}

// generateSignal creates an EntrySignal from detected pullback setup.
func (d *PullbackDetector) generateSignal(data *MarketData, triggers *TriggerResult, supports []SupportLevel) *EntrySignal {
	// Entry price = current close
	entryPrice := data.CurrentClose
	
	// Stop loss = 3% below EMA20
	stopLoss := data.EMA20 * 0.97
	
	// Targets based on ATR
	targets := calculateTargets(entryPrice, data.ATR)
	
	// Confidence based on trigger count
	confidence := determineConfidence(triggers.TriggerCount)
	
	// Setup details for transparency
	setupDetails := map[string]interface{}{
		"entry_price":     entryPrice,
		"ema20":           data.EMA20,
		"ema50":           data.EMA50,
		"rsi":             data.RSI,
		"adx":             data.ADX,
		"trigger_count":   triggers.TriggerCount,
		"triggers":        triggers.Triggers,
		"weekly_rsi":      data.WeeklyRSI,
		"stoch_k":         data.StochK,
		"stoch_d":         data.StochD,
		"support_levels":  len(supports),
	}
	
	return &EntrySignal{
		Symbol:       data.Symbol,
		Type:         SignalTypePullback,
		EntryPrice:   entryPrice,
		StopLoss:     stopLoss,
		Targets:      targets,
		Confidence:   confidence,
		Timestamp:    data.Timestamp,
		SetupDetails: setupDetails,
	}
}

// Helper functions

func calculateEMASlope(closes []float64, period int, lookback int) string {
	if len(closes) < period+lookback {
		return "unknown"
	}
	
	// Simplified: Compare current EMA to EMA N days ago
	// In production, calculate actual EMA values
	recentAvg := average(closes[len(closes)-period:])
	pastAvg := average(closes[len(closes)-period-lookback : len(closes)-lookback])
	
	if recentAvg > pastAvg {
		return "positive"
	}
	return "negative"
}

func daysSinceHighFunc(highs []float64, lookback int) int {
	if len(highs) < lookback {
		lookback = len(highs)
	}
	
	recentHighs := highs[len(highs)-lookback:]
	maxHigh := recentHighs[0]
	maxIdx := 0
	
	for i, h := range recentHighs {
		if h > maxHigh {
			maxHigh = h
			maxIdx = i
		}
	}
	
	return len(recentHighs) - maxIdx - 1
}

func calculateTargets(entry, atr float64) map[string]float64 {
	return map[string]float64{
		"target1": entry + (2 * atr),
		"target2": entry + (3 * atr),
		"target3_trail": entry + (4 * atr), // Or "Trail with EMA20"
	}
}

func determineConfidence(triggerCount int) ConfidenceLevel {
	if triggerCount >= 4 {
		return ConfidenceVeryHigh
	} else if triggerCount >= 3 {
		return ConfidenceHigh
	} else if triggerCount >= 2 {
		return ConfidenceModerate
	}
	return ConfidenceLow
}

func getStochasticStrength(stochK float64) string {
	if stochK < 30 {
		return "strong" // Deep oversold crossover
	} else if stochK < 50 {
		return "moderate"
	}
	return "weak"
}
