package signals

import (
	"fmt"
	"math"
	"sort"
)

// BreakoutDetector identifies breakout entry setups from consolidation patterns.
type BreakoutDetector struct {
	config BreakoutConfig
}

// NewBreakoutDetector creates a new breakout detector with configuration.
func NewBreakoutDetector(config BreakoutConfig) *BreakoutDetector {
	return &BreakoutDetector{
		config: config,
	}
}

// Scan analyzes market data for breakout entry setup.
func (d *BreakoutDetector) Scan(data *MarketData) (*ScanResult, error) {
	// Phase 1: Detect consolidation pattern
	consolidation := d.detectConsolidation(data)

	if !consolidation.IsValid {
		return &ScanResult{
			SetupDetected:        false,
			SetupType:            "Breakout",
			Reason:               "No valid consolidation pattern",
			Issues:               consolidation.Issues,
			Stage:                "Consolidation Detection",
			PrerequisitesDetails: consolidation,
		}, nil
	}

	// Phase 2: Detect breakout confirmation
	breakoutResult := d.detectBreakout(data, consolidation)

	return breakoutResult, nil
}

// detectConsolidation identifies consolidation patterns.
func (d *BreakoutDetector) detectConsolidation(data *MarketData) *ConsolidationResult {
	highs := data.Highs
	lows := data.Lows
	closes := data.Closes

	lookback := d.config.MaxConsolidationDays
	if len(closes) < lookback {
		lookback = len(closes)
	}

	if lookback < d.config.MinConsolidationDays {
		return &ConsolidationResult{
			IsValid: false,
			Issues:  []string{fmt.Sprintf("Insufficient data (%d bars, need %d)", lookback, d.config.MinConsolidationDays)},
		}
	}

	recentHighs := highs[len(highs)-lookback:]
	recentLows := lows[len(lows)-lookback:]
	recentCloses := closes[len(closes)-lookback:]

	// Find consolidation range
	consolidationHigh := recentHighs[0]
	consolidationLow := recentLows[0]

	for _, h := range recentHighs {
		if h > consolidationHigh {
			consolidationHigh = h
		}
	}

	for _, l := range recentLows {
		if l < consolidationLow {
			consolidationLow = l
		}
	}

	// Calculate range percentage
	avgPrice := average(recentCloses)
	rangePercent := ((consolidationHigh - consolidationLow) / avgPrice) * 100

	// Count days within range
	daysInRange := 0
	for i := range recentHighs {
		if recentHighs[i] <= consolidationHigh && recentLows[i] >= consolidationLow {
			daysInRange++
		}
	}

	// Count resistance tests (price within 1% of high)
	resistanceTests := d.countResistanceTests(recentHighs, recentCloses, consolidationHigh, 1.0)

	// Validate consolidation
	issues := []string{}

	if daysInRange < d.config.MinConsolidationDays {
		issues = append(issues, fmt.Sprintf("Only %d days in range (need >= %d)", daysInRange, d.config.MinConsolidationDays))
	}

	if daysInRange > d.config.MaxConsolidationDays {
		issues = append(issues, fmt.Sprintf("%d days in range exceeds max %d", daysInRange, d.config.MaxConsolidationDays))
	}

	if rangePercent < d.config.MinRangePercent {
		issues = append(issues, fmt.Sprintf("Range %.1f%% too narrow (min %.1f%%)", rangePercent, d.config.MinRangePercent))
	}

	if rangePercent > d.config.MaxRangePercent {
		issues = append(issues, fmt.Sprintf("Range %.1f%% too wide (max %.1f%%)", rangePercent, d.config.MaxRangePercent))
	}

	if resistanceTests < d.config.MinResistanceTests {
		issues = append(issues, fmt.Sprintf("Only %d resistance tests (need >= %d)", resistanceTests, d.config.MinResistanceTests))
	}

	if resistanceTests > d.config.MaxResistanceTests {
		issues = append(issues, fmt.Sprintf("%d resistance tests exceeds max %d (too many)", resistanceTests, d.config.MaxResistanceTests))
	}

	isValid := len(issues) == 0

	return &ConsolidationResult{
		IsValid:           isValid,
		ConsolidationHigh: consolidationHigh,
		ConsolidationLow:  consolidationLow,
		RangePercent:      rangePercent,
		DaysInRange:       daysInRange,
		ResistanceTests:   resistanceTests,
		Issues:            issues,
	}
}

// countResistanceTests counts how many times price tested resistance.
func (d *BreakoutDetector) countResistanceTests(highs, closes []float64, resistance float64, tolerancePercent float64) int {
	if len(highs) != len(closes) {
		return 0
	}

	tolerance := resistance * (tolerancePercent / 100.0)
	tests := 0
	inTest := false

	for i := range highs {
		nearResistance := math.Abs(highs[i]-resistance) <= tolerance || closes[i] >= resistance*(1.0-tolerancePercent/100.0)

		if nearResistance && !inTest {
			tests++
			inTest = true
		} else if !nearResistance {
			inTest = false
		}
	}

	return tests
}

// detectBreakout confirms breakout from consolidation.
func (d *BreakoutDetector) detectBreakout(data *MarketData, consolidation *ConsolidationResult) *ScanResult {
	// Criterion 2.2: CRITICAL - Exclude breakout signals if stock hit ceiling today
	// Vietnam market: Stocks that hit ceiling have delayed execution risk
	if data.HitCeilingToday {
		return &ScanResult{
			SetupDetected:        false,
			SetupType:            "Breakout",
			Reason:               "Stock hit ceiling today - breakout entry excluded (Criterion 2.2)",
			Issues:               []string{"Ceiling hit detected - wait for Day 2 entry between breakout and breakout+3%"},
			Stage:                "Vietnam Market Constraints",
			PrerequisitesDetails: consolidation,
			TriggersDetails:      map[string]interface{}{"ceiling_hit": true, "ceiling_price": data.CeilingPrice},
		}
	}

	issues := []string{}
	checks := map[string]bool{}

	// Check 1: Close above consolidation high
	breakoutDistance := data.CurrentClose - consolidation.ConsolidationHigh
	aboveResistance := data.CurrentClose > consolidation.ConsolidationHigh

	if !aboveResistance {
		issues = append(issues, fmt.Sprintf("Close %.0f not above resistance %.0f", data.CurrentClose, consolidation.ConsolidationHigh))
		checks["price_above_resistance"] = false
	} else {
		checks["price_above_resistance"] = true
	}

	// Check 2: Breakout distance >= 0.5 × ATR
	minBreakoutDistance := d.config.MinBreakoutATR * data.ATR
	if breakoutDistance < minBreakoutDistance {
		issues = append(issues, fmt.Sprintf("Breakout only %.0f (need %.0f = %.1f×ATR)", breakoutDistance, minBreakoutDistance, d.config.MinBreakoutATR))
		checks["breakout_distance"] = false
	} else {
		checks["breakout_distance"] = true
	}

	// Check 3: Volume validation with DUAL threshold (Criterion 2.13)
	// Must satisfy BOTH conditions:
	// 1. Volume >= 90th percentile
	// 2. Volume >= 2.0× median volume
	volumePercentile := data.VolumePercentile
	volumeSatisfiesPercentile := volumePercentile >= d.config.VolumePercentileMin

	// Calculate median volume from last 30 days
	medianVolume := calculateMedianVolume(data.Volumes)
	volumeSatisfiesMedian := data.CurrentVolume >= (2.0 * medianVolume)

	if !volumeSatisfiesPercentile || !volumeSatisfiesMedian {
		if !volumeSatisfiesPercentile {
			issues = append(issues, fmt.Sprintf("Volume only %.0fth percentile (need >=%.0f) - Criterion 2.13",
				volumePercentile, d.config.VolumePercentileMin))
		}
		if !volumeSatisfiesMedian {
			issues = append(issues, fmt.Sprintf("Volume %.0f < 2.0x median %.0f (need >=%.0f) - Criterion 2.13",
				data.CurrentVolume, medianVolume, 2.0*medianVolume))
		}
		checks["volume_confirmation"] = false
	} else {
		checks["volume_confirmation"] = true
	}

	// Optional checks (warnings, not failures)
	warnings := []string{}

	// Check 4: Weekly uptrend (preferred but not required)
	weeklyUptrend := false
	if data.WeeklyClose > 0 && data.WeeklySMA200 > 0 {
		weeklyUptrend = data.WeeklyClose > data.WeeklySMA200
		if !weeklyUptrend {
			warnings = append(warnings, "Weekly timeframe not in uptrend")
		} else {
			checks["weekly_uptrend"] = true
		}
	}

	// Check 5: RSI not overextended (< 75)
	if data.RSI > d.config.MaxRSI {
		warnings = append(warnings, fmt.Sprintf("RSI %.1f overbought (may be overextended)", data.RSI))
		checks["rsi_not_overextended"] = false
	} else {
		checks["rsi_not_overextended"] = true
	}

	// Determine if breakout is confirmed
	requiredChecks := []string{"price_above_resistance", "breakout_distance", "volume_confirmation"}
	allRequiredPass := true
	for _, check := range requiredChecks {
		if !checks[check] {
			allRequiredPass = false
			break
		}
	}

	if !allRequiredPass {
		return &ScanResult{
			SetupDetected:        false,
			SetupType:            "Breakout",
			Reason:               "Breakout criteria not met",
			Issues:               issues,
			Stage:                "Breakout Confirmation",
			PrerequisitesDetails: consolidation,
			TriggersDetails:      checks,
		}
	}

	// Generate signal
	signal := d.generateBreakoutSignal(data, consolidation, checks, weeklyUptrend, volumePercentile)

	result := &ScanResult{
		SetupDetected:        true,
		SetupType:            "Breakout",
		Signal:               signal,
		PrerequisitesDetails: consolidation,
		TriggersDetails:      checks,
	}

	if len(warnings) > 0 {
		result.Issues = warnings
	}

	return result
}

// generateBreakoutSignal creates an EntrySignal for breakout setup.
func (d *BreakoutDetector) generateBreakoutSignal(
	data *MarketData,
	consolidation *ConsolidationResult,
	checks map[string]bool,
	weeklyUptrend bool,
	volumePercentile float64,
) *EntrySignal {
	// Entry price = current close
	entryPrice := data.CurrentClose

	// Stop loss = 2% below consolidation low
	stopLoss := consolidation.ConsolidationLow * 0.98

	// Targets = Measured move (range of consolidation projected upward)
	consolidationRange := consolidation.ConsolidationHigh - consolidation.ConsolidationLow
	targets := map[string]float64{
		"target1": entryPrice + consolidationRange,         // 1× range
		"target2": entryPrice + (1.5 * consolidationRange), // 1.5× range
		"target3": entryPrice + (2.0 * consolidationRange), // 2× range
	}

	// Confidence: High if volume > 95th percentile AND weekly uptrend
	confidence := ConfidenceModerate
	if volumePercentile >= 95 && weeklyUptrend {
		confidence = ConfidenceHigh
	} else if volumePercentile >= 95 || weeklyUptrend {
		confidence = ConfidenceHigh
	}

	// Setup details
	setupDetails := map[string]interface{}{
		"consolidation_high":  consolidation.ConsolidationHigh,
		"consolidation_low":   consolidation.ConsolidationLow,
		"consolidation_range": consolidationRange,
		"range_percent":       consolidation.RangePercent,
		"days_in_range":       consolidation.DaysInRange,
		"resistance_tests":    consolidation.ResistanceTests,
		"breakout_distance":   entryPrice - consolidation.ConsolidationHigh,
		"breakout_percent":    ((entryPrice - consolidation.ConsolidationHigh) / consolidation.ConsolidationHigh) * 100,
		"volume_percentile":   volumePercentile,
		"weekly_uptrend":      weeklyUptrend,
		"rsi":                 data.RSI,
		"checks":              checks,
	}

	return &EntrySignal{
		Symbol:       data.Symbol,
		Type:         SignalTypeBreakout,
		EntryPrice:   entryPrice,
		StopLoss:     stopLoss,
		Targets:      targets,
		Confidence:   confidence,
		Timestamp:    data.Timestamp,
		SetupDetails: setupDetails,
	}
}

// calculateMedianVolume calculates the median from a slice of volumes.
// Used for Criterion 2.13 dual-threshold volume validation.
func calculateMedianVolume(volumes []float64) float64 {
	if len(volumes) == 0 {
		return 0
	}

	// Use last 30 days for median calculation
	lookback := 30
	if len(volumes) < lookback {
		lookback = len(volumes)
	}

	recentVolumes := make([]float64, lookback)
	copy(recentVolumes, volumes[len(volumes)-lookback:])

	// Sort to find median
	sort.Float64s(recentVolumes)

	// Return median
	mid := len(recentVolumes) / 2
	if len(recentVolumes)%2 == 0 {
		return (recentVolumes[mid-1] + recentVolumes[mid]) / 2
	}
	return recentVolumes[mid]
}
