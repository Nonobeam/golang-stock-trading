package regime

import (
	"fmt"
)

// RegimeDetector performs multi-factor regime detection.
type RegimeDetector struct {
	config RegimeConfig
}

// NewRegimeDetector creates a new regime detector with configuration.
func NewRegimeDetector(config RegimeConfig) *RegimeDetector {
	return &RegimeDetector{
		config: config,
	}
}

// NewDefaultRegimeDetector creates a detector with default configuration.
func NewDefaultRegimeDetector() *RegimeDetector {
	return NewRegimeDetector(DefaultRegimeConfig())
}

// DetectRegime performs complete multi-factor regime detection.
func (d *RegimeDetector) DetectRegime(
	data *MarketData,
	previousRegime RegimeType,
) *RegimeResult {

	// Factor 1: Basic regime (price & MA based)
	basicRegime, basicScore, basicFactors := ClassifyBasicRegime(
		data.CurrentPrice,
		data.MA50,
		data.MA200,
		data.Highs,
		data.Lows,
		d.config,
	)

	// Factor 2: ADX (trend strength)
	adxScore, adxAssessment := d.assessADX(data.ADX)

	// Factor 3: Directional movement
	diScore, diAssessment := d.assessDirectionalMovement(data.PlusDI, data.MinusDI)

	// Factor 4: Volatility
	volatilityScore, volatilityAssessment := d.assessVolatility(
		data.ATR,
		data.CurrentPrice,
		data.Highs,
		data.Lows,
		data.Closes,
	)

	// Factor 5: Volume pattern
	volumeScore, volumeAssessment := d.assessVolumePattern(data.Volumes, data.Closes)

	// Combine all factor scores
	totalScore := basicScore + adxScore + diScore + volumeScore + volatilityScore

	// Normalize score to -10 to +10 range
	if totalScore > 10 {
		totalScore = 10
	} else if totalScore < -10 {
		totalScore = -10
	}

	// Classify regime with hysteresis
	regime, confidence := ClassifyWithHysteresis(totalScore, previousRegime, d.config)

	// Determine if regime changed
	regimeChanged := previousRegime != "" && regime != previousRegime

	// Get strategy recommendation
	description, strategy := d.getRegimeDescription(regime)
	positionMultiplier := d.getPositionMultiplier(regime)

	// Build factors map
	factors := map[string]interface{}{
		"basic_regime": map[string]interface{}{
			"regime":  basicRegime,
			"score":   basicScore,
			"factors": basicFactors,
		},
		"adx": map[string]interface{}{
			"value":      data.ADX,
			"score":      adxScore,
			"assessment": adxAssessment,
		},
		"directional_movement": map[string]interface{}{
			"plus_di":    data.PlusDI,
			"minus_di":   data.MinusDI,
			"score":      diScore,
			"assessment": diAssessment,
		},
		"volatility": map[string]interface{}{
			"atr":         data.ATR,
			"atr_percent": (data.ATR / data.CurrentPrice) * 100,
			"score":       volatilityScore,
			"assessment":  volatilityAssessment,
		},
		"volume": map[string]interface{}{
			"score":      volumeScore,
			"assessment": volumeAssessment,
		},
	}

	return &RegimeResult{
		Regime:                 regime,
		RegimeScore:            totalScore,
		Confidence:             confidence,
		Description:            description,
		PriceToMA200:           data.CurrentPrice / data.MA200,
		MA50ToMA200:            data.MA50 / data.MA200,
		TrendStructure:         getTrendType(data.Highs, data.Lows),
		Factors:                factors,
		RegimeChanged:          regimeChanged,
		Timestamp:              data.Timestamp,
		StrategyRecommendation: strategy,
		PositionSizeMultiplier: positionMultiplier,
	}
}

// assessADX scores trend strength using ADX.
func (d *RegimeDetector) assessADX(adx float64) (int, string) {
	if adx >= 50 {
		return 2, fmt.Sprintf("Very strong trend (ADX %.1f)", adx)
	} else if adx >= 40 {
		return 1, fmt.Sprintf("Strong trend (ADX %.1f)", adx)
	} else if adx >= d.config.MinADXTrend {
		return 0, fmt.Sprintf("Moderate trend (ADX %.1f)", adx)
	} else if adx >= 20 {
		return -1, fmt.Sprintf("Weak trend (ADX %.1f)", adx)
	}
	return -2, fmt.Sprintf("No trend / Ranging (ADX %.1f)", adx)
}

// assessDirectionalMovement scores +DI vs -DI.
func (d *RegimeDetector) assessDirectionalMovement(plusDI, minusDI float64) (int, string) {
	diDiff := plusDI - minusDI

	if diDiff > 20 {
		return 2, fmt.Sprintf("+DI strongly above -DI (diff %.1f)", diDiff)
	} else if diDiff > 10 {
		return 1, fmt.Sprintf("+DI above -DI (diff %.1f)", diDiff)
	} else if diDiff > -10 {
		return 0, "+DI and -DI balanced"
	} else if diDiff > -20 {
		return -1, fmt.Sprintf("-DI above +DI (diff %.1f)", -diDiff)
	}
	return -2, fmt.Sprintf("-DI strongly above +DI (diff %.1f)", -diDiff)
}

// assessVolatility scores volatility level and trend.
func (d *RegimeDetector) assessVolatility(
	atr, currentPrice float64,
	highs, lows, closes []float64,
) (int, string) {

	atrPercent := (atr / currentPrice) * 100

	if atrPercent < 3 {
		return 0, fmt.Sprintf("Low volatility (ATR %.1f%%) - consolidation", atrPercent)
	} else if atrPercent < 5 {
		return 0, fmt.Sprintf("Normal volatility (ATR %.1f%%)", atrPercent)
	} else if atrPercent < 8 {
		return -1, fmt.Sprintf("High volatility (ATR %.1f%%)", atrPercent)
	}
	return -2, fmt.Sprintf("Extreme volatility (ATR %.1f%%) - reduce size", atrPercent)
}

// assessVolumePattern checks if volume confirms price direction.
func (d *RegimeDetector) assessVolumePattern(volumes, closes []float64) (int, string) {
	if len(volumes) != len(closes) || len(volumes) < 2 {
		return 0, "Insufficient data"
	}

	// Look at last 20 days
	lookback := 20
	if len(volumes) < lookback {
		lookback = len(volumes)
	}

	recentVolumes := volumes[len(volumes)-lookback:]
	recentCloses := closes[len(closes)-lookback:]

	// Separate up days and down days
	var upDayVolumes, downDayVolumes []float64

	for i := 1; i < len(recentCloses); i++ {
		if recentCloses[i] > recentCloses[i-1] {
			upDayVolumes = append(upDayVolumes, recentVolumes[i])
		} else if recentCloses[i] < recentCloses[i-1] {
			downDayVolumes = append(downDayVolumes, recentVolumes[i])
		}
	}

	if len(upDayVolumes) == 0 || len(downDayVolumes) == 0 {
		return 0, "Insufficient price variation"
	}

	avgUpVolume := average(upDayVolumes)
	avgDownVolume := average(downDayVolumes)

	volumeRatio := avgUpVolume / avgDownVolume

	if volumeRatio > 1.3 {
		percentHigher := (volumeRatio - 1) * 100
		return 1, fmt.Sprintf("Volume confirms uptrend (up days %.0f%% higher)", percentHigher)
	} else if volumeRatio > 1.1 {
		return 0, "Volume slightly favors upside"
	} else if volumeRatio > 0.9 {
		return 0, "Volume balanced"
	} else if volumeRatio > 0.7 {
		return -1, "Volume favors downside"
	}

	percentHigher := (1/volumeRatio - 1) * 100
	return -2, fmt.Sprintf("Volume confirms downtrend (down days %.0f%% higher)", percentHigher)
}

// getRegimeDescription returns description and strategy for regime.
func (d *RegimeDetector) getRegimeDescription(regime RegimeType) (string, string) {
	descriptions := map[RegimeType]string{
		RegimeStrongBull: "Strong Bull Market",
		RegimeMildBull:   "Mild Bull Market",
		RegimeRangeBound: "Range-bound Market",
		RegimeMildBear:   "Mild Bear Market",
		RegimeStrongBear: "Strong Bear Market",
	}

	strategies := map[RegimeType]string{
		RegimeStrongBull: "Aggressive long positions, hold winners",
		RegimeMildBull:   "Standard long positions, normal risk",
		RegimeRangeBound: "Mean reversion, tight stops, reduced size",
		RegimeMildBear:   "Defensive, small positions, high selectivity",
		RegimeStrongBear: "Minimal trading, cash position, wait for clarity",
	}

	return descriptions[regime], strategies[regime]
}

// getPositionMultiplier returns position sizing multiplier for regime.
func (d *RegimeDetector) getPositionMultiplier(regime RegimeType) float64 {
	multipliers := map[RegimeType]float64{
		RegimeStrongBull: 1.0,
		RegimeMildBull:   1.0,
		RegimeRangeBound: 0.75,
		RegimeMildBear:   0.5,
		RegimeStrongBear: 0.25,
	}
	return multipliers[regime]
}

// getTrendType extracts trend type from historical data.
func getTrendType(highs, lows []float64) string {
	if len(highs) < 60 || len(lows) < 60 {
		return "unknown"
	}

	structure := AnalyzeTrendStructure(highs, lows, 60)
	return structure.Type
}

// Helper function
func average(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
