package risk

import (
	"errors"
	"fmt"
	"math"

	"github.com/nonobeam/golang-stock-trading/internal/config"
)

// ErrInvalidInput is returned when input parameters are invalid.
var ErrInvalidInput = errors.New("invalid input parameters")

// ErrScoreTooLow is returned when trade score is below minimum.
var ErrScoreTooLow = errors.New("trade score too low - do not trade")

// ErrCorrelationTooHigh is returned when correlation with existing positions is too high.
var ErrCorrelationTooHigh = errors.New("correlation too high - skip trade")

// VN_GAP_RISK_DIVISOR is the fixed divisor for Vietnam market gap risk (Criterion 6.2).
// Accounts for potential 3 consecutive floor days (-19.5% total loss).
const VN_GAP_RISK_DIVISOR = 3.25

// PositionResult holds the complete result of position sizing calculation.
type PositionResult struct {
	PositionSize    int     // Number of shares (rounded to lot size)
	PositionValue   float64 // Total VND to invest
	PositionPercent float64 // % of capital used
	RiskAmount      float64 // Actual VND at risk
	RiskPercent     float64 // Actual % of capital at risk
	RiskPerShare    float64 // VND risk per share
	ShouldTrade     bool    // Whether to proceed with trade

	// Adjustments applied
	BasePositionSize   int     // Before adjustments
	VolatilityFactor   float64 // ATR-based adjustment
	CorrelationFactor  float64 // Correlation adjustment
	GapRiskMultiplier  float64 // Vietnam gap risk
	WasCapitalAdjusted bool    // If capped by capital limit

	// Additional metrics
	ATRPercent               float64 // ATR as % of price
	VolatilityClassification string  // low/normal/high/extreme
	MaxCorrelation           float64 // Highest correlation with existing
	CorrelatedWith           string  // Ticker with highest correlation
	WorstCaseLossPercent     float64 // Potential loss with gap risk (3-day floor scenario)

	// Worst-case loss details (Criterion 2.5)
	WorstCaseLossVND float64 // Worst-case loss amount in VND
	GapRiskWarning   string  // Warning message for worst-case scenario

	// Reason for adjustment or rejection
	Reason string
}

// VolatilityAdjustment holds volatility analysis results.
type VolatilityAdjustment struct {
	Factor         float64
	ATRPercent     float64
	Classification string
}

// FixedRisk calculates position size using fixed-risk method.
// Formula: (Balance * RiskPercent) / StopDistance
func FixedRisk(params RiskParams) (int, error) {
	if params.AccountBalance <= 0 || params.RiskPercent <= 0 {
		return 0, ErrInvalidInput
	}

	stopDistance := math.Abs(params.EntryPrice - params.StopPrice)
	if stopDistance == 0 {
		return 0, errors.New("stop distance cannot be zero")
	}

	riskAmount := params.AccountBalance * params.RiskPercent
	rawSize := riskAmount / stopDistance

	// Round down to nearest lot size
	return RoundToLotSize(rawSize, GetLotSize()), nil
}

// FixedRiskDetailed returns detailed position sizing result.
func FixedRiskDetailed(params RiskParams) (*PositionResult, error) {
	if params.AccountBalance <= 0 || params.RiskPercent <= 0 {
		return nil, ErrInvalidInput
	}
	if params.EntryPrice <= params.StopPrice {
		return nil, errors.New("entry price must be above stop loss for long positions")
	}

	riskPerShare := params.EntryPrice - params.StopPrice
	riskAmount := params.AccountBalance * params.RiskPercent
	rawSize := riskAmount / riskPerShare
	positionSize := RoundToLotSize(rawSize, GetLotSize())

	positionValue := float64(positionSize) * params.EntryPrice
	actualRisk := float64(positionSize) * riskPerShare

	return &PositionResult{
		PositionSize:       positionSize,
		PositionValue:      positionValue,
		PositionPercent:    (positionValue / params.AccountBalance) * 100,
		RiskAmount:         actualRisk,
		RiskPercent:        (actualRisk / params.AccountBalance) * 100,
		RiskPerShare:       riskPerShare,
		ShouldTrade:        positionSize > 0,
		BasePositionSize:   positionSize,
		VolatilityFactor:   1.0,
		CorrelationFactor:  1.0,
		GapRiskMultiplier:  1.0,
		WasCapitalAdjusted: false,
	}, nil
}

// CalculateVolatilityFactor returns adjustment factor based on ATR%.
// Low volatility (<3%): 1.2x, Normal (3-5%): 1.0x, High (5-8%): 0.8x, Extreme (>8%): 0.6x
func CalculateVolatilityFactor(atr, price float64) VolatilityAdjustment {
	if price == 0 {
		return VolatilityAdjustment{Factor: 1.0, ATRPercent: 0, Classification: "unknown"}
	}

	atrPercent := (atr / price) * 100

	var factor float64
	var classification string

	switch {
	case atrPercent < 3.0:
		factor = 1.2
		classification = "low"
	case atrPercent < 5.0:
		factor = 1.0
		classification = "normal"
	case atrPercent < 8.0:
		factor = 0.8
		classification = "high"
	default:
		factor = 0.6
		classification = "extreme"
	}

	return VolatilityAdjustment{
		Factor:         factor,
		ATRPercent:     atrPercent,
		Classification: classification,
	}
}

// ATRBased calculates position size using ATR as stop distance.
func ATRBased(balance, riskPercent, atr, multiplier float64) (int, error) {
	if balance <= 0 || riskPercent <= 0 || atr <= 0 {
		return 0, ErrInvalidInput
	}

	if multiplier <= 0 {
		multiplier = GetDefaultATRMultiplier()
	}

	stopDistance := atr * multiplier
	riskAmount := balance * riskPercent
	rawSize := riskAmount / stopDistance

	return RoundToLotSize(rawSize, GetLotSize()), nil
}

// ScoreBased scales position size based on trade score (0-13).
// Score 0-6: 0.5x, Score 7-8: 1.0x, Score 9-10: 1.25x, Score 11-13: 1.5x
func ScoreBased(baseSize int, score int) int {
	multiplier := GetScoreMultiplier(score)
	scaledSize := float64(baseSize) * multiplier
	return RoundToLotSize(scaledSize, GetLotSize())
}

// GetScoreMultiplier returns position multiplier based on trade score.
func GetScoreMultiplier(score int) float64 {
	switch {
	case score <= 6:
		return 0.5
	case score <= 8:
		return 1.0
	case score <= 10:
		return 1.25
	default:
		return 1.5
	}
}

// GetRiskPercentByScore determines risk % based on trade score and market regime.
func GetRiskPercentByScore(score int, marketRegime string) float64 {
	if score < 7 {
		return 0 // Score too low, don't trade
	}

	var baseRisk float64
	switch {
	case score >= 11:
		baseRisk = 0.02 // 2%
	case score >= 9:
		baseRisk = 0.015 // 1.5%
	default:
		baseRisk = 0.01 // 1%
	}

	// Adjust for market regime
	var multiplier float64
	switch marketRegime {
	case "bull":
		multiplier = 1.0
	case "bear":
		multiplier = 0.5
	case "range":
		multiplier = 0.75
	case "transition":
		multiplier = 0.5
	default:
		multiplier = 1.0
	}

	adjusted := baseRisk * multiplier
	if adjusted > 0.02 {
		return 0.02 // Cap at 2%
	}
	return adjusted
}

// CapitalConstrained adjusts position size if it exceeds available capital.
func CapitalConstrained(size int, price, availableCapital float64) (int, string) {
	requiredCapital := float64(size) * price

	if requiredCapital <= availableCapital {
		return size, ""
	}

	maxSize := int(availableCapital / price)
	adjustedSize := RoundToLotSize(float64(maxSize), GetLotSize())

	return adjustedSize, "position reduced due to capital constraint"
}

// ApplyMaxPositionLimit reduces position if it exceeds max position percent.
func ApplyMaxPositionLimit(positionSize int, entryPrice, totalCapital, maxPositionPercent float64) (int, bool) {
	positionValue := float64(positionSize) * entryPrice
	positionPercent := (positionValue / totalCapital) * 100

	if positionPercent <= maxPositionPercent {
		return positionSize, false
	}

	// Reduce to max position
	maxValue := totalCapital * (maxPositionPercent / 100)
	adjustedSize := int(maxValue / entryPrice)
	return RoundToLotSize(float64(adjustedSize), GetLotSize()), true
}

// GetCorrelationFactor returns position adjustment based on correlation.
// ρ >= 0.85: 0 (don't trade), ρ 0.7-0.85: 0.5, ρ 0.5-0.7: 0.8, else: 1.0
func GetCorrelationFactor(correlation float64) (float64, string) {
	absCorr := math.Abs(correlation)

	switch {
	case absCorr >= 0.85:
		return 0.0, "correlation too high (>0.85) - skip trade"
	case absCorr >= 0.7:
		return 0.5, "high correlation - reduce position 50%"
	case absCorr >= 0.5:
		return 0.8, "moderate correlation - reduce position 20%"
	default:
		return 1.0, "low correlation - no adjustment"
	}
}

// GetGapRiskMultiplier returns gap risk factor for Vietnam market.
// Based on historical gap behavior: 1.5 (minimal) to 3.0 (high risk)
func GetGapRiskMultiplier(gapCount, maxConsecutive int) float64 {
	switch {
	case maxConsecutive >= 3 || gapCount > 5:
		return 3.0 // High gap risk
	case maxConsecutive >= 2 || gapCount > 3:
		return 2.5 // Moderate gap risk
	case gapCount > 1:
		return 2.0 // Some gap risk
	default:
		return 1.5 // Minimal gap risk (still account for VN limits)
	}
}

// RoundToLotSize rounds down to the nearest lot size.
func RoundToLotSize(size float64, lotSize int) int {
	if lotSize <= 0 {
		lotSize = 1
	}
	lots := int(size) / lotSize
	return lots * lotSize
}

// PositionSizer provides comprehensive position sizing with all adjustments.
type PositionSizer struct {
	TotalCapital       float64
	MaxPositionPercent float64
}

// NewPositionSizer creates a new PositionSizer with config defaults.
func NewPositionSizer(totalCapital float64) *PositionSizer {
	cfg := config.Get()
	maxPos := 20.0 // Default 20%
	if cfg.Trading.MaxStopPercent > 0 {
		// Use configured max or default
		maxPos = 20.0
	}

	return &PositionSizer{
		TotalCapital:       totalCapital,
		MaxPositionPercent: maxPos,
	}
}

// Calculate performs comprehensive position sizing with all adjustments.
func (ps *PositionSizer) Calculate(
	entryPrice, stopPrice float64,
	tradeScore int,
	marketRegime string,
	atr float64,
	maxCorrelation float64,
	correlatedWith string,
	gapCount, maxConsecutiveGaps int,
) (*PositionResult, error) {

	// Step 1: Determine risk percentage from score
	riskPercent := GetRiskPercentByScore(tradeScore, marketRegime)
	if riskPercent == 0 {
		return &PositionResult{
			PositionSize: 0,
			ShouldTrade:  false,
			Reason:       "trade score too low (minimum 7 required)",
		}, ErrScoreTooLow
	}

	// Step 2: Calculate base position size FIRST (before gap adjustment)
	// This ensures we apply 3.25 divisor to position size, not risk percent

	// Step 3: Calculate base position using ORIGINAL risk percent
	params := RiskParams{
		AccountBalance: ps.TotalCapital,
		RiskPercent:    riskPercent, // Use original risk, not adjusted
		EntryPrice:     entryPrice,
		StopPrice:      stopPrice,
	}

	baseResult, err := FixedRiskDetailed(params)
	if err != nil {
		return nil, err
	}

	// Step 3a: Apply Vietnam gap risk divisor to POSITION SIZE (Criterion 6.2)
	// This accounts for 3 consecutive floor days (-19.5% potential loss)
	basePositionSize := baseResult.PositionSize
	vnAdjustedPosition := float64(basePositionSize) / VN_GAP_RISK_DIVISOR
	positionSize := RoundToLotSize(vnAdjustedPosition, GetLotSize())

	// Step 4: Apply volatility adjustment
	volAdj := CalculateVolatilityFactor(atr, entryPrice)
	positionSize = RoundToLotSize(float64(positionSize)*volAdj.Factor, GetLotSize())

	// Step 5: Apply capital constraint (max position %)
	positionSize, wasAdjusted := ApplyMaxPositionLimit(
		positionSize, entryPrice, ps.TotalCapital, ps.MaxPositionPercent,
	)

	// Step 6: Apply correlation adjustment
	corrFactor, corrReason := GetCorrelationFactor(maxCorrelation)
	if corrFactor == 0 {
		return &PositionResult{
			PositionSize:   0,
			ShouldTrade:    false,
			MaxCorrelation: maxCorrelation,
			CorrelatedWith: correlatedWith,
			Reason:         corrReason,
		}, ErrCorrelationTooHigh
	}
	positionSize = RoundToLotSize(float64(positionSize)*corrFactor, GetLotSize())

	// Calculate final metrics
	riskPerShare := entryPrice - stopPrice
	positionValue := float64(positionSize) * entryPrice
	actualRisk := float64(positionSize) * riskPerShare

	// Calculate worst-case loss (Criterion 2.5): 3 consecutive floor days = -19.5%
	worstCaseLossVND := positionValue * 0.195 // 19.5% of position value
	worstCaseLossPercent := (worstCaseLossVND / ps.TotalCapital) * 100

	// Format worst-case warning message
	gapWarning := formatWorstCaseWarning(worstCaseLossVND, worstCaseLossPercent)

	return &PositionResult{
		PositionSize:             positionSize,
		PositionValue:            positionValue,
		PositionPercent:          (positionValue / ps.TotalCapital) * 100,
		RiskAmount:               actualRisk,
		RiskPercent:              (actualRisk / ps.TotalCapital) * 100,
		RiskPerShare:             riskPerShare,
		ShouldTrade:              positionSize > 0,
		BasePositionSize:         basePositionSize,
		VolatilityFactor:         volAdj.Factor,
		CorrelationFactor:        corrFactor,
		GapRiskMultiplier:        VN_GAP_RISK_DIVISOR, // Always 3.25
		WasCapitalAdjusted:       wasAdjusted,
		ATRPercent:               volAdj.ATRPercent,
		VolatilityClassification: volAdj.Classification,
		MaxCorrelation:           maxCorrelation,
		CorrelatedWith:           correlatedWith,
		WorstCaseLossPercent:     worstCaseLossPercent,
		WorstCaseLossVND:         worstCaseLossVND,
		GapRiskWarning:           gapWarning,
		Reason:                   "",
	}, nil
}

// CalculateSimple performs position sizing without correlation/gap data.
func (ps *PositionSizer) CalculateSimple(
	entryPrice, stopPrice float64,
	riskPercent float64,
	atr float64,
) (*PositionResult, error) {

	params := RiskParams{
		AccountBalance: ps.TotalCapital,
		RiskPercent:    riskPercent,
		EntryPrice:     entryPrice,
		StopPrice:      stopPrice,
	}

	baseResult, err := FixedRiskDetailed(params)
	if err != nil {
		return nil, err
	}

	positionSize := baseResult.PositionSize

	// Apply volatility adjustment if ATR provided
	var volAdj VolatilityAdjustment
	if atr > 0 {
		volAdj = CalculateVolatilityFactor(atr, entryPrice)
		positionSize = RoundToLotSize(float64(positionSize)*volAdj.Factor, GetLotSize())
	} else {
		volAdj = VolatilityAdjustment{Factor: 1.0, Classification: "unknown"}
	}

	// Apply capital constraint
	positionSize, wasAdjusted := ApplyMaxPositionLimit(
		positionSize, entryPrice, ps.TotalCapital, ps.MaxPositionPercent,
	)

	riskPerShare := entryPrice - stopPrice
	positionValue := float64(positionSize) * entryPrice
	actualRisk := float64(positionSize) * riskPerShare

	return &PositionResult{
		PositionSize:             positionSize,
		PositionValue:            positionValue,
		PositionPercent:          (positionValue / ps.TotalCapital) * 100,
		RiskAmount:               actualRisk,
		RiskPercent:              (actualRisk / ps.TotalCapital) * 100,
		RiskPerShare:             riskPerShare,
		ShouldTrade:              positionSize > 0,
		BasePositionSize:         baseResult.PositionSize,
		VolatilityFactor:         volAdj.Factor,
		CorrelationFactor:        1.0,
		GapRiskMultiplier:        1.0,
		WasCapitalAdjusted:       wasAdjusted,
		ATRPercent:               volAdj.ATRPercent,
		VolatilityClassification: volAdj.Classification,
	}, nil
}

// CalculatePositionSize combines all sizing methods based on params (legacy).
func CalculatePositionSize(params RiskParams, atr float64, atrMultiplier float64, availableCapital float64) (int, string, error) {
	baseSize, err := FixedRisk(params)
	if err != nil {
		return 0, "", err
	}

	if atr > 0 && atrMultiplier > 0 {
		atrSize, err := ATRBased(params.AccountBalance, params.RiskPercent, atr, atrMultiplier)
		if err == nil && atrSize < baseSize {
			baseSize = atrSize
		}
	}

	if params.TradeScore > 0 {
		baseSize = ScoreBased(baseSize, params.TradeScore)
	}

	finalSize, warning := CapitalConstrained(baseSize, params.EntryPrice, availableCapital)

	return finalSize, warning, nil
}

// formatWorstCaseWarning formats the gap risk warning message per Criterion 2.5.
func formatWorstCaseWarning(lossVND, lossPercent float64) string {
	return fmt.Sprintf(
		"⚠️ WARNING: If floor hit, actual exit may be delayed 1-3 days. "+
			"Worst case loss: %s VND (%.2f%% of capital)",
		formatVND(lossVND),
		lossPercent,
	)
}
