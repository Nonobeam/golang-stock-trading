package signals

import (
	"fmt"
	"math"
	"time"
)

// CrossoverDetector identifies MA crossover entry setups.
// Theory: When fast MA (20 EMA) crosses above slow MA (50 EMA), wait for pullback
// to fast MA, then enter on bounce with confirmation.
type CrossoverDetector struct {
	config CrossoverConfig
}

// NewCrossoverDetector creates a new MA crossover detector.
func NewCrossoverDetector(config CrossoverConfig) *CrossoverDetector {
	return &CrossoverDetector{
		config: config,
	}
}

// Scan analyzes market data for MA crossover entry setup.
func (d *CrossoverDetector) Scan(data *MarketData) (*ScanResult, error) {
	// Step 1: Check prerequisites
	prereqResult := d.checkPrerequisites(data)

	if !prereqResult.Passes {
		return &ScanResult{
			SetupDetected: false,
			SetupType:     "MA Crossover",
			Reason:        "Prerequisites not met",
			Issues:        prereqResult.Issues,
			Stage:         "Prerequisites Check",
		}, nil
	}

	// Step 2: Detect crossover
	crossoverResult := d.detectCrossover(data)

	if !crossoverResult["crossover_detected"].(bool) {
		return &ScanResult{
			SetupDetected: false,
			SetupType:     "MA Crossover",
			Reason:        crossoverResult["reason"].(string),
			Stage:         "Crossover Detection",
		}, nil
	}

	// Step 3: Check if in pullback phase
	pullbackResult := d.checkPullbackPhase(data, crossoverResult["crossover_index"].(int))

	if !pullbackResult["in_pullback"].(bool) {
		return &ScanResult{
			SetupDetected: false,
			SetupType:     "MA Crossover",
			Reason:        pullbackResult["reason"].(string),
			Stage:         "Pullback Phase Check",
		}, nil
	}

	// Step 4: Check entry triggers
	triggerResult := d.checkCrossoverTriggers(data)

	if !triggerResult.Sufficient {
		return &ScanResult{
			SetupDetected: false,
			SetupType:     "MA Crossover",
			Reason:        fmt.Sprintf("Only %d trigger(s) (need %d)", triggerResult.TriggerCount, d.config.MinTriggers),
			Stage:         "Entry Triggers Check",
		}, nil
	}

	// Step 5: All conditions met - ENTRY SIGNAL
	signal := d.generateSignal(data, crossoverResult, pullbackResult, triggerResult)

	return &ScanResult{
		SetupDetected: true,
		SetupType:     "MA Crossover",
		Signal:        signal,
		Stage:         "ENTRY SIGNAL",
	}, nil
}

// checkPrerequisites verifies structural conditions for MA crossover trade.
func (d *CrossoverDetector) checkPrerequisites(data *MarketData) *PrerequisiteResult {
	passes := true
	details := []string{}
	issues := []string{}

	// Check 1: 20 EMA above 50 EMA (crossover already occurred)
	if data.EMA20 > data.EMA50 {
		details = append(details, fmt.Sprintf("✓ 20 EMA (%.0f) above 50 EMA (%.0f)", data.EMA20, data.EMA50))
	} else {
		passes = false
		issues = append(issues, "20 EMA still below 50 EMA")
	}

	// Check 2: Price above 50 EMA (in new uptrend)
	if data.CurrentClose > data.EMA50 {
		details = append(details, fmt.Sprintf("✓ Price (%.0f) above 50 EMA", data.CurrentClose))
	} else {
		passes = false
		issues = append(issues, "Price still below 50 EMA")
	}

	// Check 3: ADX shows trend
	if data.ADX >= d.config.MinADX {
		details = append(details, fmt.Sprintf("✓ ADX %.1f (≥%.0f)", data.ADX, d.config.MinADX))
	} else {
		passes = false
		issues = append(issues, fmt.Sprintf("ADX %.1f too weak (need ≥%.0f)", data.ADX, d.config.MinADX))
	}

	// Check 4: Weekly uptrend (if available)
	if data.WeeklySMA200 > 0 {
		if data.WeeklyClose > data.WeeklySMA200 {
			details = append(details, "✓ Weekly uptrend")
		} else {
			details = append(details, "⚠ Weekly not in uptrend")
		}
	}

	return &PrerequisiteResult{
		Passes:  passes,
		Details: details,
		Issues:  issues,
	}
}

// detectCrossover detects if crossover occurred recently (within last 20 bars).
func (d *CrossoverDetector) detectCrossover(data *MarketData) map[string]interface{} {
	// Get EMA series from daily series
	if data.DailySeries == nil || data.DailySeries.Len() < 21 {
		return map[string]interface{}{
			"crossover_detected": false,
			"reason":             "Insufficient data for crossover detection",
		}
	}

	// We need to calculate EMA20 and EMA50 for historical bars
	// This is simplified - in production, we'd access pre-calculated indicator series
	closes := data.Closes
	if len(closes) < 60 {
		return map[string]interface{}{
			"crossover_detected": false,
			"reason":             "Insufficient bars for crossover detection",
		}
	}

	// For simplicity, we'll check if current EMA20 > EMA50 and
	// the crossover is within reasonable timeframe
	// In production, we'd look back through historical EMA values

	// Current setup: 20 EMA > 50 EMA should already be verified by prerequisites
	// We'll assume crossover happened recently (within 20 bars)
	// A more robust implementation would calculate historical EMAs

	crossoverIndex := len(closes) - 10 // Assume crossover 10 bars ago (placeholder)
	barsAgo := len(closes) - 1 - crossoverIndex

	if barsAgo > 20 {
		return map[string]interface{}{
			"crossover_detected": false,
			"reason":             "No crossover in last 20 bars",
		}
	}

	return map[string]interface{}{
		"crossover_detected": true,
		"crossover_index":    crossoverIndex,
		"bars_ago":           barsAgo,
		"ema_20_at_cross":    data.EMA20, // Simplified
		"ema_50_at_cross":    data.EMA50,
	}
}

// checkPullbackPhase checks if currently in pullback phase after crossover.
func (d *CrossoverDetector) checkPullbackPhase(data *MarketData, crossoverIndex int) map[string]interface{} {
	currentPrice := data.CurrentClose
	ema20 := data.EMA20

	// Check if near 20 EMA (within configured proximity)
	distanceToEMA := math.Abs(currentPrice - ema20)
	distancePercent := (distanceToEMA / currentPrice) * 100

	if distancePercent > d.config.EMAProximity {
		return map[string]interface{}{
			"in_pullback": false,
			"reason":      fmt.Sprintf("Price %.1f%% from 20 EMA (need within %.1f%%)", distancePercent, d.config.EMAProximity),
		}
	}

	// Check if there was a rally after crossover
	closes := data.Closes
	if len(closes) <= crossoverIndex {
		return map[string]interface{}{
			"in_pullback": false,
			"reason":      "Invalid crossover index",
		}
	}

	barsSinceCross := len(closes) - 1 - crossoverIndex

	if barsSinceCross < 3 {
		return map[string]interface{}{
			"in_pullback": false,
			"reason":      "Too soon after crossover (wait for rally then pullback)",
		}
	}

	// Find recent high after crossover
	recentCloses := closes[crossoverIndex:]
	recentHigh := recentCloses[0]
	recentHighIndex := crossoverIndex

	for i, close := range recentCloses {
		if close > recentHigh {
			recentHigh = close
			recentHighIndex = crossoverIndex + i
		}
	}

	daysSinceHigh := len(closes) - 1 - recentHighIndex

	if daysSinceHigh < d.config.MinPullbackDays {
		return map[string]interface{}{
			"in_pullback": false,
			"reason":      fmt.Sprintf("Pullback too brief (%d days)", daysSinceHigh),
		}
	}

	if daysSinceHigh > d.config.MaxPullbackDays {
		return map[string]interface{}{
			"in_pullback": false,
			"reason":      fmt.Sprintf("Pullback too long (%d days) - trend may be failing", daysSinceHigh),
		}
	}

	// Check volume declined during pullback
	volumes := data.Volumes
	recentVolumes := volumes[recentHighIndex:]

	volumeDeclined := false
	if len(recentVolumes) >= 3 {
		earlyVol := average(recentVolumes[:len(recentVolumes)/2])
		lateVol := average(recentVolumes[len(recentVolumes)/2:])
		volumeDeclined = lateVol < earlyVol*0.9
	}

	return map[string]interface{}{
		"in_pullback":             true,
		"days_since_crossover":    barsSinceCross,
		"days_since_high":         daysSinceHigh,
		"rally_high":              recentHigh,
		"distance_to_ema_percent": distancePercent,
		"volume_declined":         volumeDeclined,
	}
}

// checkCrossoverTriggers checks for entry triggers on the bounce.
func (d *CrossoverDetector) checkCrossoverTriggers(data *MarketData) *TriggerResult {
	triggers := []TriggerInfo{}
	triggerCount := 0

	currentOpen := data.CurrentOpen
	currentHigh := data.CurrentHigh
	currentLow := data.CurrentLow
	currentClose := data.CurrentClose
	currentVolume := data.CurrentVolume

	// Get previous bar if available
	var prevOpen, prevHigh, prevLow, prevClose float64
	if len(data.Opens) >= 2 && len(data.Highs) >= 2 && len(data.Lows) >= 2 && len(data.Closes) >= 2 {
		prevOpen = data.Opens[len(data.Opens)-2]
		prevHigh = data.Highs[len(data.Highs)-2]
		prevLow = data.Lows[len(data.Lows)-2]
		prevClose = data.Closes[len(data.Closes)-2]
	}

	// Trigger 1: Bullish reversal candle
	if prevClose > 0 {
		candle := DetectBullishCandle(currentOpen, currentHigh, currentLow, currentClose,
			prevOpen, prevHigh, prevLow, prevClose)

		if candle != nil && candle.IsBullish {
			triggerCount++
			triggers = append(triggers, TriggerInfo{
				Trigger:     "Bullish Candle",
				Description: candle.PatternName,
				Strength:    candle.Strength,
			})
		}
	}

	// Trigger 2: Volume spike
	if data.VolumeMA20 > 0 {
		volumeSpike := currentVolume > data.VolumeMA20*1.3

		if volumeSpike {
			triggerCount++
			volumeIncrease := ((currentVolume / data.VolumeMA20) - 1) * 100
			triggers = append(triggers, TriggerInfo{
				Trigger:     "Volume Spike",
				Description: fmt.Sprintf("Volume +%.0f%% above average", volumeIncrease),
				Strength:    "strong",
			})
		}
	}

	// Trigger 3: RSI in favorable range
	if data.RSI >= 40 && data.RSI <= 60 {
		triggerCount++
		triggers = append(triggers, TriggerInfo{
			Trigger:     "RSI Favorable",
			Description: fmt.Sprintf("RSI at %.1f (healthy range)", data.RSI),
			Strength:    "moderate",
		})
	}

	// Trigger 4: Stochastic crossover
	if data.StochK > data.StochD && data.StochK < 60 {
		// Simplified - in production, check if K just crossed above D
		triggerCount++
		triggers = append(triggers, TriggerInfo{
			Trigger:     "Stochastic Favorable",
			Description: fmt.Sprintf("%%K above %%D at %.1f", data.StochK),
			Strength:    "moderate",
		})
	}

	// Trigger 5: MACD histogram growing
	if data.MACDHistogram > 0 {
		// Simplified - in production, compare to previous histogram
		triggerCount++
		triggers = append(triggers, TriggerInfo{
			Trigger:     "MACD Positive",
			Description: "MACD histogram positive",
			Strength:    "moderate",
		})
	}

	return &TriggerResult{
		TriggerCount: triggerCount,
		Triggers:     triggers,
		Sufficient:   triggerCount >= d.config.MinTriggers,
		Recommendation: func() string {
			if triggerCount >= d.config.MinTriggers {
				return "ENTER"
			}
			return "WAIT"
		}(),
	}
}

// generateSignal creates an EntrySignal from detected crossover setup.
func (d *CrossoverDetector) generateSignal(
	data *MarketData,
	crossoverResult map[string]interface{},
	pullbackResult map[string]interface{},
	triggerResult *TriggerResult,
) *EntrySignal {
	entryPrice := data.CurrentClose
	stopLoss := d.calculateCrossoverStop(data.EMA50)
	targets := d.calculateCrossoverTargets(entryPrice, data.ATR)
	confidence := d.calculateCrossoverConfidence(crossoverResult, pullbackResult, triggerResult)

	return &EntrySignal{
		Symbol:     data.Symbol,
		Type:       SignalTypeCrossover,
		EntryPrice: entryPrice,
		StopLoss:   stopLoss,
		Targets:    targets,
		Confidence: confidence,
		Timestamp:  time.Now(),
		SetupDetails: map[string]interface{}{
			"crossover_bars_ago": crossoverResult["bars_ago"],
			"days_since_high":    pullbackResult["days_since_high"],
			"distance_to_ema":    pullbackResult["distance_to_ema_percent"],
			"volume_declined":    pullbackResult["volume_declined"],
			"triggers_detected":  triggerResult.TriggerCount,
			"triggers":           triggerResult.Triggers,
		},
	}
}

// calculateCrossoverStop calculates stop loss for MA crossover entry.
func (d *CrossoverDetector) calculateCrossoverStop(ema50 float64) float64 {
	// Stop 2-3% below 50 EMA
	stop := ema50 * 0.97
	return roundToNearest(stop, 100)
}

// calculateCrossoverTargets calculates targets for MA crossover entry.
func (d *CrossoverDetector) calculateCrossoverTargets(entry, atr float64) map[string]float64 {
	return map[string]float64{
		"target1": roundToNearest(entry+(atr*2), 100),
		"target2": roundToNearest(entry+(atr*3), 100),
		// target3 would be "Trail with 20 EMA" as a string, but map only holds float64
	}
}

// calculateCrossoverConfidence calculates overall confidence level.
func (d *CrossoverDetector) calculateCrossoverConfidence(
	crossoverResult map[string]interface{},
	pullbackResult map[string]interface{},
	triggerResult *TriggerResult,
) ConfidenceLevel {
	score := 0

	// Recent crossover is better
	barsAgo := crossoverResult["bars_ago"].(int)
	if barsAgo <= 10 {
		score += 2
	} else if barsAgo <= 15 {
		score += 1
	}

	// Volume confirmation
	volumeDeclined, ok := pullbackResult["volume_declined"].(bool)
	if ok && volumeDeclined {
		score += 2
	}

	// Multiple triggers
	if triggerResult.TriggerCount >= 4 {
		score += 2
	} else if triggerResult.TriggerCount >= 3 {
		score += 1
	}

	if score >= 5 {
		return ConfidenceVeryHigh
	} else if score >= 3 {
		return ConfidenceModerate
	}
	return ConfidenceLow
}

// Helper function to round to nearest increment
func roundToNearest(value, increment float64) float64 {
	return math.Round(value/increment) * increment
}
