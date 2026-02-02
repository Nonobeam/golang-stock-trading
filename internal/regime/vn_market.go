package regime

// VNMarketDetector detects overall Vietnam market regime using VN-Index.
type VNMarketDetector struct {
	config RegimeConfig
}

// NewVNMarketDetector creates a new VN-market detector.
func NewVNMarketDetector(config RegimeConfig) *VNMarketDetector {
	return &VNMarketDetector{
		config: config,
	}
}

// DetectVNMarketRegime detects market-wide regime from VN-Index.
func (d *VNMarketDetector) DetectVNMarketRegime(vnData *VNMarketData) *RegimeResult {
	ratio := vnData.VNIndexCurrent / vnData.VNIndexMA200

	var regime RegimeType
	var marketScore int
	var description, strategy string

	// Classify based on position relative to 200 MA
	if ratio >= d.config.StrongBullThreshold {
		regime = RegimeStrongBull
		marketScore = 4
		description = "Strong Bull - VN-Index >10% above 200 MA"
		strategy = "Aggressive longs, high conviction"

	} else if ratio >= d.config.MildBullLower {
		regime = RegimeMildBull
		marketScore = 2
		description = "Mild Bull - VN-Index 0-10% above 200 MA"
		strategy = "Standard long positions"

	} else if ratio >= d.config.MildBearUpper {
		regime = RegimeMildBear
		marketScore = -1
		description = "Mild Bear - VN-Index 0-10% below 200 MA"
		strategy = "Defensive, reduce size 50%"

	} else {
		regime = RegimeStrongBear
		marketScore = -3
		description = "Strong Bear - VN-Index >10% below 200 MA"
		strategy = "Minimal trading, 80% cash"
	}

	// Adjust for ADX (trend strength)
	var trendStrength string
	var adxModifier int

	if vnData.VNIndexADX > 30 {
		trendStrength = "Strong"
		adxModifier = 0
	} else if vnData.VNIndexADX > 20 {
		trendStrength = "Moderate"
		adxModifier = -1
	} else {
		trendStrength = "Weak/Ranging"
		adxModifier = -2

		// Override regime if ranging
		if regime == RegimeMildBull || regime == RegimeMildBear {
			regime = RegimeRangeBound
			description = "Range-bound - Low ADX"
			strategy = "Mean reversion, tight stops"
		}
	}

	finalScore := marketScore + adxModifier

	// Get position adjustment
	positionAdj := d.GetPositionAdjustment(regime)
	expectedBehavior := d.GetExpectedBehavior(regime)

	factors := map[string]interface{}{
		"vn_index_to_ma200":   ratio,
		"percent_from_ma200":  (ratio - 1) * 100,
		"trend_strength":      trendStrength,
		"adx":                 vnData.VNIndexADX,
		"adx_modifier":        adxModifier,
		"position_adjustment": positionAdj,
		"expected_behavior":   expectedBehavior,
	}

	return &RegimeResult{
		Regime:                 regime,
		RegimeScore:            finalScore,
		Confidence:             assignConfidence(finalScore),
		Description:            description,
		PriceToMA200:           ratio,
		MA50ToMA200:            vnData.VNIndexMA50 / vnData.VNIndexMA200,
		Factors:                factors,
		Timestamp:              vnData.Timestamp,
		StrategyRecommendation: strategy,
		PositionSizeMultiplier: positionAdj.Multiplier,
	}
}

// GetPositionAdjustment returns position sizing guidance for regime.
func (d *VNMarketDetector) GetPositionAdjustment(regime RegimeType) PositionAdjustment {
	adjustments := map[RegimeType]PositionAdjustment{
		RegimeStrongBull: {
			Multiplier:          1.0,
			MaxRiskPerTrade:     2.0,
			MaxPortfolioRisk:    8.0,
			MaxConcurrentTrades: 8,
		},
		RegimeMildBull: {
			Multiplier:          1.0,
			MaxRiskPerTrade:     1.5,
			MaxPortfolioRisk:    6.0,
			MaxConcurrentTrades: 6,
		},
		RegimeRangeBound: {
			Multiplier:          0.75,
			MaxRiskPerTrade:     1.0,
			MaxPortfolioRisk:    4.0,
			MaxConcurrentTrades: 5,
		},
		RegimeMildBear: {
			Multiplier:          0.5,
			MaxRiskPerTrade:     1.0,
			MaxPortfolioRisk:    3.0,
			MaxConcurrentTrades: 4,
		},
		RegimeStrongBear: {
			Multiplier:          0.25,
			MaxRiskPerTrade:     0.5,
			MaxPortfolioRisk:    1.5,
			MaxConcurrentTrades: 2,
		},
	}

	if adj, ok := adjustments[regime]; ok {
		return adj
	}
	return adjustments[RegimeRangeBound] // Default
}

// GetExpectedBehavior describes what to expect from individual stocks.
func (d *VNMarketDetector) GetExpectedBehavior(regime RegimeType) string {
	behaviors := map[RegimeType]string{
		RegimeStrongBull: "Most stocks trending up, pullbacks shallow, breakouts follow through",
		RegimeMildBull:   "Select stocks advancing, normal pullbacks, some breakouts work",
		RegimeRangeBound: "Stocks choppy, false breakouts common, mean reversion better",
		RegimeMildBear:   "Most stocks weak, rallies fail, breakouts rare",
		RegimeStrongBear: "Nearly all stocks declining, avoid trading",
	}

	if behavior, ok := behaviors[regime]; ok {
		return behavior
	}
	return "Mixed behavior"
}
