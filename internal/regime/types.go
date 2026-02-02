// Package regime provides market regime detection and classification.
package regime

import "time"

// RegimeType represents the market regime classification.
type RegimeType string

const (
	RegimeStrongBull RegimeType = "strong_bull"
	RegimeMildBull   RegimeType = "mild_bull"
	RegimeRangeBound RegimeType = "range_bound"
	RegimeMildBear   RegimeType = "mild_bear"
	RegimeStrongBear RegimeType = "strong_bear"
)

// ConfidenceLevel represents confidence in regime classification.
type ConfidenceLevel string

const (
	ConfidenceHigh     ConfidenceLevel = "High"
	ConfidenceModerate ConfidenceLevel = "Moderate"
	ConfidenceLow      ConfidenceLevel = "Low"
)

// RegimeResult holds the complete regime detection result.
type RegimeResult struct {
	Regime           RegimeType      `json:"regime"`
	RegimeScore      int             `json:"regime_score"`
	Confidence       ConfidenceLevel `json:"confidence"`
	Description      string          `json:"description"`
	PriceToMA200     float64         `json:"price_to_ma200"`
	MA50ToMA200      float64         `json:"ma_50_to_200"`
	TrendStructure   string          `json:"trend_structure"`
	Factors          map[string]interface{} `json:"factors"`
	RegimeChanged    bool            `json:"regime_changed"`
	Timestamp        time.Time       `json:"timestamp"`
	
	// Trading guidelines
	StrategyRecommendation string  `json:"strategy_recommendation"`
	PositionSizeMultiplier float64 `json:"position_size_multiplier"`
}

// RegimeConfig holds configurable thresholds for regime detection.
type RegimeConfig struct {
	// Price to MA200 thresholds
	StrongBullThreshold float64 // 1.10 = 10% above
	MildBullLower       float64 // 1.00 = at MA
	MildBullUpper       float64 // 1.10
	MildBearLower       float64 // 0.90 = 10% below
	MildBearUpper       float64 // 1.00
	StrongBearThreshold float64 // 0.90 = more than 10% below
	
	// ADX thresholds
	MinADXTrend    float64 // 25 = minimum for trending
	StrongADXTrend float64 // 30 = strong trend
	
	// Hysteresis margins (prevent whipsaws)
	HysteresisMargin float64 // 0.5 = half a point margin
}

// DefaultRegimeConfig returns Vietnam market-specific configuration.
func DefaultRegimeConfig() RegimeConfig {
	return RegimeConfig{
		StrongBullThreshold: 1.10,
		MildBullLower:       1.00,
		MildBullUpper:       1.10,
		MildBearLower:       0.90,
		MildBearUpper:       1.00,
		StrongBearThreshold: 0.90,
		MinADXTrend:         25.0,
		StrongADXTrend:      30.0,
		HysteresisMargin:    0.5,
	}
}

// TrendStructure represents the trend pattern analysis.
type TrendStructure struct {
	Type         string    `json:"type"` // higher_highs_lows, lower_highs_lows, choppy
	Strength     string    `json:"strength"`
	PeriodHighs  []float64 `json:"period_highs"`
	PeriodLows   []float64 `json:"period_lows"`
}

// TransitionState represents regime transition detection.
type TransitionState struct {
	InTransition       bool     `json:"in_transition"`
	TransitionSeverity string   `json:"transition_severity"` // High, Moderate, Low
	TransitionScore    int      `json:"transition_score"`
	Signals            []string `json:"signals"`
	Recommendation     string   `json:"recommendation"`
	ActionItems        []string `json:"action_items"`
}

// MarketData contains all data needed for regime detection.
type MarketData struct {
	Symbol    string
	Timestamp time.Time
	
	// Price data
	CurrentPrice float64
	Highs        []float64
	Lows         []float64
	Closes       []float64
	Volumes      []float64
	
	// Moving averages
	MA50  float64
	MA200 float64
	
	// Trend indicators
	ADX     float64
	PlusDI  float64
	MinusDI float64
	
	// Volatility
	ATR float64
}

// VNMarketData contains VN-Index specific data.
type VNMarketData struct {
	VNIndexCurrent float64
	VNIndexMA50    float64
	VNIndexMA200   float64
	VNIndexADX     float64
	Timestamp      time.Time
}

// RegimeHistory stores historical regime classifications.
type RegimeHistory struct {
	Date   time.Time  `json:"date"`
	Regime RegimeType `json:"regime"`
	Score  int        `json:"score"`
}

// PositionAdjustment provides position sizing guidance by regime.
type PositionAdjustment struct {
	Multiplier         float64 `json:"multiplier"`
	MaxRiskPerTrade    float64 `json:"max_risk_per_trade"`
	MaxPortfolioRisk   float64 `json:"max_portfolio_risk"`
	MaxConcurrentTrades int     `json:"max_concurrent_trades"`
}
