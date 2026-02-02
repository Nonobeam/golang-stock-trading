package scoring

// TradeSetup holds all input data for trade scoring.
type TradeSetup struct {
	// Price data
	CurrentPrice float64
	EntryPrice   float64
	StopLoss     float64
	Target       float64

	// Indicators (pre-computed)
	EMA20             float64
	EMA50             float64
	WeeklyPrice       float64
	WeeklySMA200      float64
	WeeklyStructure   string // "higher_highs_lows", "lower_highs_lows", "choppy"
	RSI               float64
	MACD              float64
	MACDSignal        float64
	MACDHistogram     float64
	PreviousHistogram float64
	ATR               float64

	// Setup characteristics
	SupportLevel      float64
	SupportType       string // "swing_low", "ma", "previous_resistance", "fibonacci", "none"
	HasConsolidation  bool
	ConsolidationBars int
	VolumeConfirms    bool

	// Context
	VNIndexPrice  float64
	VNIndexMA50   float64
	SectorRS      float64 // Sector relative strength vs market (>1 = outperforming)
	NewsSentiment string  // "positive", "neutral", "negative", "very_negative"

	// Liquidity
	AvgDailyVolume   float64
	AvgDailyTurnover float64
	ZeroVolumeDays   int
}

// ComponentScore holds the result of scoring a single component.
type ComponentScore struct {
	Score    int
	MaxScore int
	MinScore int // For context which can be negative
	Details  []string
	Name     string
}

// LiquidityResult holds the result of liquidity filter checks.
type LiquidityResult struct {
	Passes  bool
	Issues  []string
	Details []string
}

// ScoreSummary provides strengths and weaknesses analysis.
type ScoreSummary struct {
	Strengths      []string
	Weaknesses     []string
	OverallQuality string // "Excellent", "Good", "Acceptable", "Poor"
}

// ScoreResult holds the complete result of trade scoring.
type ScoreResult struct {
	TotalScore    int
	MaxScore      int // 13
	ShouldTrade   bool
	RiskPercent   float64
	Recommendation string

	ComponentScores struct {
		TrendAlignment ComponentScore
		SetupQuality   ComponentScore
		Momentum       ComponentScore
		RiskReward     ComponentScore
		Context        ComponentScore
	}

	Liquidity LiquidityResult
	Summary   ScoreSummary
}
