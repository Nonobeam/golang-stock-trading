package signals

import (
	"fmt"
	"math"
	"time"
)

// MeanReversionDetector identifies mean reversion entry setups in ranging markets.
// Theory: In range-bound markets (ADX < 20), prices oscillate between support and resistance.
// Buy at support when oversold, sell at resistance when overbought.
type MeanReversionDetector struct {
	config MeanReversionConfig
}

// NewMeanReversionDetector creates a new mean reversion detector.
func NewMeanReversionDetector(config MeanReversionConfig) *MeanReversionDetector {
	return &MeanReversionDetector{
		config: config,
	}
}

// Scan analyzes market data for mean reversion entry setup.
func (d *MeanReversionDetector) Scan(data *MarketData, srLevels *SRLevels, marketRegime map[string]interface{}) (*ScanResult, error) {
	// Step 1: Check if market is ranging
	rangingResult := d.checkRangingMarket(data.ADX, marketRegime)

	if !rangingResult["is_ranging"].(bool) {
		return &ScanResult{
			SetupDetected: false,
			SetupType:     "Mean Reversion",
			Reason:        "Market not ranging",
			Issues:        []string{rangingResult["reason"].(string)},
			Stage:         "Range Check",
		}, nil
	}

	// Step 2: Analyze range validity
	rangeAnalysis := d.analyzeRange(data, srLevels)

	if !rangeAnalysis["valid_range"].(bool) {
		issues, _ := rangeAnalysis["issues"].([]string)
		return &ScanResult{
			SetupDetected: false,
			SetupType:     "Mean Reversion",
			Reason:        "No valid trading range",
			Issues:        issues,
			Stage:         "Range Analysis",
		}, nil
	}

	// Step 3: Check if at extreme level
	extremeCheck := d.checkAtExtreme(data.CurrentClose, rangeAnalysis)

	if !extremeCheck["at_extreme"].(bool) {
		return &ScanResult{
			SetupDetected: false,
			SetupType:     "Mean Reversion",
			Reason:        extremeCheck["reason"].(string),
			Stage:         "Level Check",
		}, nil
	}

	// Step 4: Check oscillator confirmations
	direction := extremeCheck["direction"].(string)
	oscillatorCheck := d.checkOscillators(data, direction)

	if !oscillatorCheck["confirmed"].(bool) {
		return &ScanResult{
			SetupDetected: false,
			SetupType:     "Mean Reversion",
			Reason:        "Oscillators not confirming",
			Stage:         "Oscillator Check",
		}, nil
	}

	// Step 5: Check for reversal signal
	reversalCheck := d.checkReversalSignal(data, direction)

	if !reversalCheck["reversal_detected"].(bool) {
		return &ScanResult{
			SetupDetected: false,
			SetupType:     "Mean Reversion",
			Reason:        "No reversal signal yet",
			Stage:         "Reversal Signal Check",
		}, nil
	}

	// Step 6: All conditions met - ENTRY SIGNAL
	signal := d.generateMeanReversionSignal(data, rangeAnalysis, extremeCheck, oscillatorCheck, reversalCheck)

	return &ScanResult{
		SetupDetected: true,
		SetupType:     "Mean Reversion",
		Signal:        signal,
		Stage:         "ENTRY SIGNAL",
	}, nil
}

// checkRangingMarket confirms market is ranging (not trending).
func (d *MeanReversionDetector) checkRangingMarket(adx float64, marketRegime map[string]interface{}) map[string]interface{} {
	// Check ADX
	if adx > d.config.MaxADX {
		return map[string]interface{}{
			"is_ranging": false,
			"adx":        adx,
			"reason":     fmt.Sprintf("ADX %.1f > %.0f (trending - don't use mean reversion)", adx, d.config.MaxADX),
		}
	}

	// Check market regime
	regime, ok := marketRegime["vn_market_regime"].(string)
	if ok && regime != "range_bound" {
		return map[string]interface{}{
			"is_ranging": false,
			"adx":        adx,
			"regime":     regime,
			"reason":     fmt.Sprintf("Market regime %s (not range_bound)", regime),
		}
	}

	return map[string]interface{}{
		"is_ranging": true,
		"adx":        adx,
		"regime":     regime,
		"reason":     fmt.Sprintf("ADX %.1f ≤ %.0f (ranging)", adx, d.config.MaxADX),
	}
}

// analyzeRange validates if there's a valid trading range.
func (d *MeanReversionDetector) analyzeRange(data *MarketData, srLevels *SRLevels) map[string]interface{} {
	if srLevels == nil {
		return map[string]interface{}{
			"valid_range": false,
			"issues":      []string{"No support/resistance levels provided"},
		}
	}

	support := srLevels.Support
	resistance := srLevels.Resistance

	// Calculate range metrics
	rangeSize := resistance - support
	avgPrice := (support + resistance) / 2
	rangePercent := (rangeSize / avgPrice) * 100

	issues := []string{}

	// Check range width
	if rangePercent < d.config.MinRangeWidthPercent {
		issues = append(issues, fmt.Sprintf("Range too tight (%.1f%% < %.1f%%)", rangePercent, d.config.MinRangeWidthPercent))
	} else if rangePercent > d.config.MaxRangeWidthPercent {
		issues = append(issues, fmt.Sprintf("Range too wide (%.1f%% > %.1f%%)", rangePercent, d.config.MaxRangeWidthPercent))
	}

	// Count tests of support and resistance (last 60 bars)
	lows := data.Lows
	highs := data.Highs

	lookback := 60
	if len(lows) < lookback {
		lookback = len(lows)
	}

	recentLows := lows[len(lows)-lookback:]
	recentHighs := highs[len(highs)-lookback:]

	supportTests := 0
	for _, low := range recentLows {
		if math.Abs(low-support)/support < 0.02 { // Within 2%
			supportTests++
		}
	}

	resistanceTests := 0
	for _, high := range recentHighs {
		if math.Abs(high-resistance)/resistance < 0.02 { // Within 2%
			resistanceTests++
		}
	}

	if supportTests < d.config.MinRangeTests {
		issues = append(issues, fmt.Sprintf("Support only tested %d times (need %d)", supportTests, d.config.MinRangeTests))
	}

	if resistanceTests < d.config.MinRangeTests {
		issues = append(issues, fmt.Sprintf("Resistance only tested %d times (need %d)", resistanceTests, d.config.MinRangeTests))
	}

	valid := len(issues) == 0

	return map[string]interface{}{
		"valid_range":      valid,
		"support":          support,
		"resistance":       resistance,
		"range_size":       rangeSize,
		"range_percent":    rangePercent,
		"support_tests":    supportTests,
		"resistance_tests": resistanceTests,
		"issues":           issues,
	}
}

// checkAtExtreme checks if price is at support (buy) or resistance (sell).
func (d *MeanReversionDetector) checkAtExtreme(currentPrice float64, rangeAnalysis map[string]interface{}) map[string]interface{} {
	support := rangeAnalysis["support"].(float64)
	resistance := rangeAnalysis["resistance"].(float64)

	// Calculate distances
	distanceFromSupport := math.Abs(currentPrice - support)
	distanceFromResistance := math.Abs(currentPrice - resistance)

	distanceSupportPercent := (distanceFromSupport / support) * 100
	distanceResistancePercent := (distanceFromResistance / resistance) * 100

	// Check if at support (within configured proximity)
	if distanceSupportPercent <= d.config.ProximityPercent {
		return map[string]interface{}{
			"at_extreme":       true,
			"direction":        "long",
			"level":            support,
			"distance_percent": distanceSupportPercent,
			"reason":           fmt.Sprintf("At support (%.1f%% away)", distanceSupportPercent),
		}
	}

	// Check if at resistance
	if distanceResistancePercent <= d.config.ProximityPercent {
		return map[string]interface{}{
			"at_extreme":       true,
			"direction":        "short",
			"level":            resistance,
			"distance_percent": distanceResistancePercent,
			"reason":           fmt.Sprintf("At resistance (%.1f%% away)", distanceResistancePercent),
		}
	}

	// In middle of range
	return map[string]interface{}{
		"at_extreme": false,
		"reason":     fmt.Sprintf("In middle of range (support %.1f%% away, resistance %.1f%% away)", distanceSupportPercent, distanceResistancePercent),
	}
}

// checkOscillators checks if oscillators confirm the setup.
func (d *MeanReversionDetector) checkOscillators(data *MarketData, direction string) map[string]interface{} {
	confirmations := []string{}
	confirmationCount := 0

	if direction == "long" {
		// For longs at support: need oversold
		if data.RSI < d.config.RSIOversold {
			confirmationCount++
			confirmations = append(confirmations, fmt.Sprintf("RSI oversold at %.1f", data.RSI))
		}

		if data.StochK < 20 {
			confirmationCount++
			confirmations = append(confirmations, fmt.Sprintf("Stochastic oversold at %.1f", data.StochK))
		}

		// Bollinger %B (if available) - simplified check
		// In production, this would use actual BB %B calculation
	} else {
		// For shorts at resistance: need overbought
		if data.RSI > d.config.RSIOverbought {
			confirmationCount++
			confirmations = append(confirmations, fmt.Sprintf("RSI overbought at %.1f", data.RSI))
		}

		if data.StochK > 80 {
			confirmationCount++
			confirmations = append(confirmations, fmt.Sprintf("Stochastic overbought at %.1f", data.StochK))
		}
	}

	confirmed := confirmationCount >= 2

	return map[string]interface{}{
		"confirmed":          confirmed,
		"confirmation_count": confirmationCount,
		"confirmations":      confirmations,
	}
}

// checkReversalSignal checks for reversal candlestick pattern.
func (d *MeanReversionDetector) checkReversalSignal(data *MarketData, direction string) map[string]interface{} {
	currentOpen := data.CurrentOpen
	currentHigh := data.CurrentHigh
	currentLow := data.CurrentLow
	currentClose := data.CurrentClose

	if len(data.Closes) < 2 {
		return map[string]interface{}{
			"reversal_detected": false,
		}
	}

	prevOpen := data.Opens[len(data.Opens)-2]
	prevHigh := data.Highs[len(data.Highs)-2]
	prevLow := data.Lows[len(data.Lows)-2]
	prevClose := data.Closes[len(data.Closes)-2]

	body := math.Abs(currentClose - currentOpen)

	if direction == "long" {
		// Look for bullish reversal
		candle := DetectBullishCandle(currentOpen, currentHigh, currentLow, currentClose,
			prevOpen, prevHigh, prevLow, prevClose)

		if candle != nil && candle.IsBullish {
			return map[string]interface{}{
				"reversal_detected": true,
				"pattern":           candle.PatternName,
				"strength":          candle.Strength,
			}
		}

		// Simple green candle
		if currentClose > currentOpen {
			return map[string]interface{}{
				"reversal_detected": true,
				"pattern":           "Green Candle",
				"strength":          "moderate",
			}
		}
	} else {
		// For shorts: look for bearish reversal
		upperWick := currentHigh - max(currentOpen, currentClose)

		isShootingStar := body > 0 && upperWick > body*2
		isRed := currentClose < currentOpen

		if isShootingStar && isRed {
			return map[string]interface{}{
				"reversal_detected": true,
				"pattern":           "Bearish Shooting Star",
				"strength":          "strong",
			}
		} else if isRed {
			return map[string]interface{}{
				"reversal_detected": true,
				"pattern":           "Red Candle",
				"strength":          "moderate",
			}
		}
	}

	return map[string]interface{}{
		"reversal_detected": false,
	}
}

// generateMeanReversionSignal creates an EntrySignal for mean reversion setup.
func (d *MeanReversionDetector) generateMeanReversionSignal(
	data *MarketData,
	rangeAnalysis map[string]interface{},
	extremeCheck map[string]interface{},
	oscillatorCheck map[string]interface{},
	reversalCheck map[string]interface{},
) *EntrySignal {
	direction := extremeCheck["direction"].(string)
	level := extremeCheck["level"].(float64)

	entryPrice := data.CurrentClose
	stopLoss := d.calculateMeanReversionStop(level, direction)
	targets := d.calculateMeanReversionTarget(
		rangeAnalysis["support"].(float64),
		rangeAnalysis["resistance"].(float64),
		entryPrice,
		direction,
	)
	confidence := d.calculateMeanReversionConfidence(rangeAnalysis, oscillatorCheck, reversalCheck)

	return &EntrySignal{
		Symbol:     data.Symbol,
		Type:       SignalTypeMeanReversion,
		EntryPrice: entryPrice,
		StopLoss:   stopLoss,
		Targets:    targets,
		Confidence: confidence,
		Timestamp:  time.Now(),
		SetupDetails: map[string]interface{}{
			"direction":        direction,
			"range_percent":    rangeAnalysis["range_percent"],
			"support_tests":    rangeAnalysis["support_tests"],
			"resistance_tests": rangeAnalysis["resistance_tests"],
			"oscillator_count": oscillatorCheck["confirmation_count"],
			"reversal_pattern": reversalCheck["pattern"],
			"warning":          "Use tight stops - ranges can break",
		},
	}
}

// calculateMeanReversionStop calculates tight stop for mean reversion.
func (d *MeanReversionDetector) calculateMeanReversionStop(level float64, direction string) float64 {
	if direction == "long" {
		// Stop below support
		return roundToNearest(level*0.97, 100)
	}
	// Stop above resistance
	return roundToNearest(level*1.03, 100)
}

// calculateMeanReversionTarget calculates targets for mean reversion.
func (d *MeanReversionDetector) calculateMeanReversionTarget(
	support, resistance, entry float64,
	direction string,
) map[string]float64 {
	midRange := (support + resistance) / 2

	if direction == "long" {
		return map[string]float64{
			"target1": roundToNearest(midRange, 100),
			"target2": roundToNearest(resistance*0.99, 100),
		}
	}
	// For shorts
	return map[string]float64{
		"target1": roundToNearest(midRange, 100),
		"target2": roundToNearest(support*1.01, 100),
	}
}

// calculateMeanReversionConfidence calculates confidence level.
func (d *MeanReversionDetector) calculateMeanReversionConfidence(
	rangeAnalysis map[string]interface{},
	oscillatorCheck map[string]interface{},
	reversalCheck map[string]interface{},
) ConfidenceLevel {
	score := 0

	// Strong range (many tests)
	supportTests := rangeAnalysis["support_tests"].(int)
	resistanceTests := rangeAnalysis["resistance_tests"].(int)

	if supportTests >= 4 && resistanceTests >= 4 {
		score += 2
	} else if supportTests >= 3 && resistanceTests >= 3 {
		score += 1
	}

	// Multiple oscillator confirmations
	confirmationCount := oscillatorCheck["confirmation_count"].(int)
	if confirmationCount >= 3 {
		score += 2
	} else if confirmationCount >= 2 {
		score += 1
	}

	// Strong reversal pattern
	strength, ok := reversalCheck["strength"].(string)
	if ok {
		if strength == "strong" {
			score += 2
		} else if strength == "moderate" {
			score += 1
		}
	}

	if score >= 5 {
		return ConfidenceHigh
	} else if score >= 3 {
		return ConfidenceModerate
	}
	return ConfidenceLow
}

// Helper functions
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
