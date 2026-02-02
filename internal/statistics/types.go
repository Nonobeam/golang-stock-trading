package statistics

import (
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/regime"
)

// Trade represents a completed trading position from entry to exit
type Trade struct {
	Symbol      string            `json:"symbol"`
	EntryTime   time.Time         `json:"entryTime"`
	ExitTime    time.Time         `json:"exitTime"`
	EntryPrice  float64           `json:"entryPrice"`
	ExitPrice   float64           `json:"exitPrice"`
	Quantity    int               `json:"quantity"`
	PnL         float64           `json:"pnl"`
	PnLPercent  float64           `json:"pnlPercent"`
	InitialRisk float64           `json:"initialRisk"`
	MAE_R       float64           `json:"maeR"` // Maximum Adverse Excursion in R-multiples
	MFE_R       float64           `json:"mfeR"` // Maximum Favorable Excursion in R-multiples
	SignalType  string            `json:"signalType"`
	Regime      regime.RegimeType `json:"regime"`
	Score       int               `json:"score"`
}

// StatisticsConfig holds configuration for statistics calculations
type StatisticsConfig struct {
	RiskFreeRate         float64
	PeriodsPerYear       int
	MinimumSampleSize    int
	MinDrawdownThreshold float64
}

// DefaultConfig returns the default configuration for Vietnam market
func DefaultConfig() StatisticsConfig {
	return StatisticsConfig{
		RiskFreeRate:         0.05,
		PeriodsPerYear:       252,
		MinimumSampleSize:    30,
		MinDrawdownThreshold: 0.05,
	}
}

// WinRateMetrics contains win rate calculation results
type WinRateMetrics struct {
	TotalTrades      int     `json:"totalTrades"`
	Winners          int     `json:"winners"`
	Losers           int     `json:"losers"`
	Breakevens       int     `json:"breakevens"`
	WinRate          float64 `json:"winRate"`
	LossRate         float64 `json:"lossRate"`
	BreakevenRate    float64 `json:"breakevenRate"`
	WinRateFormatted string  `json:"winRateFormatted"`
	AverageWin       float64 `json:"averageWin"`
	AverageLoss      float64 `json:"averageLoss"`
}

// ExpectancyMetrics contains expectancy and profitability metrics
type ExpectancyMetrics struct {
	Expectancy      float64 `json:"expectancy"`
	ExpectancyRatio float64 `json:"expectancyRatio"`
	WinRate         float64 `json:"winRate"`
	LossRate        float64 `json:"lossRate"`
	AvgWin          float64 `json:"avgWin"`
	AvgLoss         float64 `json:"avgLoss"`
	TotalProfit     float64 `json:"totalProfit"`
	TotalLoss       float64 `json:"totalLoss"`
	NetProfit       float64 `json:"netProfit"`
	Unit            string  `json:"unit"`
	Interpretation  string  `json:"interpretation"`
	ProfitFactor    float64 `json:"profitFactor"`
	PayoffRatio     float64 `json:"payoffRatio"`
	PayoffFormatted string  `json:"payoffFormatted"`
}

// RiskAdjustedMetrics contains Sharpe, Sortino, and Calmar ratios
type RiskAdjustedMetrics struct {
	SharpeRatio           float64 `json:"sharpeRatio"`
	SortinoRatio          float64 `json:"sortinoRatio"`
	CalmarRatio           float64 `json:"calmarRatio"`
	AnnualReturn          float64 `json:"annualReturn"`
	AnnualStdDev          float64 `json:"annualStdDev"`
	AnnualDownsideDev     float64 `json:"annualDownsideDev"`
	RiskFreeRate          float64 `json:"riskFreeRate"`
	ExcessReturn          float64 `json:"excessReturn"`
	SharpeInterpretation  string  `json:"sharpeInterpretation"`
	SortinoInterpretation string  `json:"sortinoInterpretation"`
	CalmarInterpretation  string  `json:"calmarInterpretation"`
}

// DrawdownPeriod represents a single peak-to-trough-to-recovery cycle
type DrawdownPeriod struct {
	PeakIdx          int        `json:"peakIdx"`
	TroughIdx        int        `json:"troughIdx"`
	RecoveryIdx      *int       `json:"recoveryIdx,omitempty"`
	PeakValue        float64    `json:"peakValue"`
	TroughValue      float64    `json:"troughValue"`
	RecoveryValue    *float64   `json:"recoveryValue,omitempty"`
	DepthPercent     float64    `json:"depthPercent"`
	Duration         int        `json:"duration"`
	RecoveryDuration *int       `json:"recoveryDuration,omitempty"`
	Recovered        bool       `json:"recovered"`
	PeakDate         *time.Time `json:"peakDate,omitempty"`
	TroughDate       *time.Time `json:"troughDate,omitempty"`
	RecoveryDate     *time.Time `json:"recoveryDate,omitempty"`
}

// DrawdownMetrics contains drawdown analysis results
type DrawdownMetrics struct {
	MaxDrawdown            float64          `json:"maxDrawdown"`
	MaxDrawdownPercent     float64          `json:"maxDrawdownPercent"`
	MaxDDPeakIdx           int              `json:"maxDdPeakIdx"`
	MaxDDTroughIdx         int              `json:"maxDdTroughIdx"`
	MaxDDDuration          int              `json:"maxDdDuration"`
	MaxDDPeakValue         float64          `json:"maxDdPeakValue"`
	MaxDDTroughValue       float64          `json:"maxDdTroughValue"`
	Recovered              bool             `json:"recovered"`
	RecoveryIdx            *int             `json:"recoveryIdx,omitempty"`
	RecoveryDuration       *int             `json:"recoveryDuration,omitempty"`
	CurrentDrawdownPercent float64          `json:"currentDrawdownPercent"`
	AvgDrawdownPercent     float64          `json:"avgDrawdownPercent"`
	NumDrawdownPeriods     int              `json:"numDrawdownPeriods"`
	DrawdownPeriods        []DrawdownPeriod `json:"drawdownPeriods"`
	RecoveryFactor         float64          `json:"recoveryFactor"`
	RecoveryInterpretation string           `json:"recoveryInterpretation"`
}

// DistributionMetrics contains performance distribution analysis
type DistributionMetrics struct {
	BySignalType map[string]ExpectancyMetrics `json:"bySignalType"`
	ByRegime     map[string]ExpectancyMetrics `json:"byRegime"`
	ByScoreRange map[string]ExpectancyMetrics `json:"byScoreRange"`
	BestTrades   []Trade                      `json:"bestTrades"`
	WorstTrades  []Trade                      `json:"worstTrades"`
}

// EquityPoint represents a point on the equity curve
type EquityPoint struct {
	Time   time.Time `json:"time"`
	Equity float64   `json:"equity"`
}

// RDistributionMetrics contains R-multiple distribution analysis
type RDistributionMetrics struct {
	MeanR                  float64            `json:"meanR"`
	MedianR                float64            `json:"medianR"`
	StdDev                 float64            `json:"stdDev"`
	Skewness               float64            `json:"skewness"`
	SkewnessInterpretation string             `json:"skewnessInterpretation"`
	Kurtosis               float64            `json:"kurtosis"`
	KurtosisInterpretation string             `json:"kurtosisInterpretation"`
	Percentiles            map[string]float64 `json:"percentiles"` // 10th, 25th, 50th, 75th, 90th
	RBuckets               RBucketStats       `json:"rBuckets"`
	TailAnalysis           TailAnalysis       `json:"tailAnalysis"`
	QualityAssessment      string             `json:"qualityAssessment"`
}

// RBucketStats breaks down trades by R-multiple ranges
type RBucketStats struct {
	LargeLoss  RBucketDetail `json:"largeLoss"`  // < -2R
	MediumLoss RBucketDetail `json:"mediumLoss"` // -2R to -1R
	SmallLoss  RBucketDetail `json:"smallLoss"`  // -1R to 0R
	SmallWin   RBucketDetail `json:"smallWin"`   // 0R to +1R
	MediumWin  RBucketDetail `json:"mediumWin"`  // +1R to +3R
	LargeWin   RBucketDetail `json:"largeWin"`   // +3R to +5R
	HugeWin    RBucketDetail `json:"hugeWin"`    // > +5R
}

// RBucketDetail contains stats for one R-range bucket
type RBucketDetail struct {
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// TailAnalysis examines extreme outcomes
type TailAnalysis struct {
	RightTail TailDetail `json:"rightTail"` // Top 10% winners
	LeftTail  TailDetail `json:"leftTail"`  // Bottom 10% losers
}

// TailDetail contains tail statistics
type TailDetail struct {
	Threshold              float64 `json:"threshold"`
	Count                  int     `json:"count"`
	MeanR                  float64 `json:"meanR"`
	ContributionToTotalPct float64 `json:"contributionToTotalPct"`
}

// MAEMFEMetrics contains aggregate MAE/MFE analysis
type MAEMFEMetrics struct {
	WinnerMAE      WinnerMAEMetrics `json:"winnerMAE"`
	LoserMAE       LoserMAEMetrics  `json:"loserMAE"`
	WinnerMFE      WinnerMFEMetrics `json:"winnerMFE"`
	LoserMFE       LoserMFEMetrics  `json:"loserMFE"`
	StopInsights   []string         `json:"stopInsights"`
	TargetInsights []string         `json:"targetInsights"`
}

// WinnerMAEMetrics analyzes how far winners went against before recovering
type WinnerMAEMetrics struct {
	AvgMAE          float64        `json:"avgMAE"`
	MedianMAE       float64        `json:"medianMAE"`
	MaxMAE          float64        `json:"maxMAE"`
	MAEDistribution map[string]int `json:"maeDistribution"` // Ranges: 0-0.3R, 0.3-0.5R, etc.
	Interpretation  string         `json:"interpretation"`
}

// LoserMAEMetrics analyzes stop effectiveness
type LoserMAEMetrics struct {
	AvgMAE             float64 `json:"avgMAE"`
	AvgFinalLoss       float64 `json:"avgFinalLoss"`
	MaxLoss            float64 `json:"maxLoss"`
	ExcessiveLossCount int     `json:"excessiveLossCount"` // Losses > 1.2R
	ExcessiveLossPct   float64 `json:"excessiveLossPct"`
	Interpretation     string  `json:"interpretation"`
}

// WinnerMFEMetrics analyzes profit capture efficiency
type WinnerMFEMetrics struct {
	AvgMFE         float64 `json:"avgMFE"`
	AvgFinal       float64 `json:"avgFinal"`
	AvgEfficiency  float64 `json:"avgEfficiency"` // (Final / MFE) * 100
	GaveBack50Pct  int     `json:"gaveBack50Pct"` // Count of trades giving back >50% of MFE
	Interpretation string  `json:"interpretation"`
}

// LoserMFEMetrics analyzes false signals (losers that were profitable)
type LoserMFEMetrics struct {
	WereProfitable     int     `json:"wereProfitable"`
	ProfitablePct      float64 `json:"profitablePct"`
	AvgMFEOfProfitable float64 `json:"avgMFEOfProfitable"`
	Interpretation     string  `json:"interpretation"`
}

// TimeMetrics contains time-based performance analytics
type TimeMetrics struct {
	HoldingPeriods HoldingPeriodMetrics `json:"holdingPeriods"`
	Streaks        StreakMetrics        `json:"streaks"`
}

// HoldingPeriodMetrics analyzes trade duration
type HoldingPeriodMetrics struct {
	AllTrades      HoldingStats         `json:"allTrades"`
	Winners        HoldingStats         `json:"winners"`
	Losers         HoldingStats         `json:"losers"`
	OptimalPeriod  OptimalPeriodMetrics `json:"optimalPeriod"`
	TimeDecay      TimeDecayMetrics     `json:"timeDecay"`
	Interpretation string               `json:"interpretation"`
}

// HoldingStats contains duration statistics
type HoldingStats struct {
	AvgDays    float64 `json:"avgDays"`
	MedianDays float64 `json:"medianDays"`
	MinDays    int     `json:"minDays"`
	MaxDays    int     `json:"maxDays"`
	StdDev     float64 `json:"stdDev"`
}

// OptimalPeriodMetrics identifies best holding period ranges
type OptimalPeriodMetrics struct {
	BucketPerformance map[string]PeriodBucket `json:"bucketPerformance"` // "0-5 days", "5-10 days", etc.
	BestPeriod        string                  `json:"bestPeriod"`
	BestAvgR          float64                 `json:"bestAvgR"`
}

// PeriodBucket contains stats for a holding period range
type PeriodBucket struct {
	Count   int     `json:"count"`
	AvgR    float64 `json:"avgR"`
	WinRate float64 `json:"winRate"`
}

// TimeDecayMetrics analyzes if performance degrades over time
type TimeDecayMetrics struct {
	LongHoldAvgR   float64 `json:"longHoldAvgR"`  // > 30 days
	ShortHoldAvgR  float64 `json:"shortHoldAvgR"` // <= 10 days
	DecayPresent   bool    `json:"decayPresent"`
	Recommendation string  `json:"recommendation"`
}

// StreakMetrics analyzes consecutive wins/losses
type StreakMetrics struct {
	MaxWinStreak     int     `json:"maxWinStreak"`
	MaxLossStreak    int     `json:"maxLossStreak"`
	AvgWinStreak     float64 `json:"avgWinStreak"`
	AvgLossStreak    float64 `json:"avgLossStreak"`
	TotalWinStreaks  int     `json:"totalWinStreaks"`
	TotalLossStreaks int     `json:"totalLossStreaks"`
	Interpretation   string  `json:"interpretation"`
}

// StatisticsResult contains all calculated statistics
type StatisticsResult struct {
	WinRate            WinRateMetrics       `json:"winRate"`
	Expectancy         ExpectancyMetrics    `json:"expectancy"`
	RiskAdjusted       RiskAdjustedMetrics  `json:"riskAdjusted"`
	Drawdown           DrawdownMetrics      `json:"drawdown"`
	Distribution       DistributionMetrics  `json:"distribution"`
	RDistribution      RDistributionMetrics `json:"rDistribution"` // NEW
	MAEMFE             MAEMFEMetrics        `json:"maemfe"`        // NEW
	TimeMetrics        TimeMetrics          `json:"timeMetrics"`   // NEW
	EquityCurve        []EquityPoint        `json:"equityCurve"`
	SampleSizeAdequate bool                 `json:"sampleSizeAdequate"`
	SampleSizeWarning  string               `json:"sampleSizeWarning,omitempty"`
	TotalTrades        int                  `json:"totalTrades"`
	InitialBalance     float64              `json:"initialBalance"`
	FinalBalance       float64              `json:"finalBalance"`
}

// VNConfig holds Vietnam-specific market parameters
type VNConfig struct {
	TradingDaysPerYear    int     // 245 for VN (not 252)
	RiskFreeRate          float64 // 0.04-0.05 (VN T-bills)
	AcceptableSharpe      float64 // 0.7 for VN (vs 1.0 US)
	AcceptableMaxDrawdown float64 // 0.20 for VN (vs 0.15 US)
	AcceptableWinRate     float64 // 0.45 for VN (vs 0.50 US)
	AcceptableExpectancy  float64 // 0.25R for VN (vs 0.30R US)
	GapRiskAdjustment     bool    // true - adjust for ±7% limits
	TrackHolidayImpact    bool    // true - Tết analysis
}

// DefaultVNConfig returns default Vietnam market configuration
func DefaultVNConfig() VNConfig {
	return VNConfig{
		TradingDaysPerYear:    245,
		RiskFreeRate:          0.045,
		AcceptableSharpe:      0.7,
		AcceptableMaxDrawdown: 0.20,
		AcceptableWinRate:     0.45,
		AcceptableExpectancy:  0.25,
		GapRiskAdjustment:     true,
		TrackHolidayImpact:    true,
	}
}

// SystemHealth contains system health assessment
type SystemHealth struct {
	Score          int     `json:"score"`  // 0-100
	Rating         string  `json:"rating"` // EXCELLENT, GOOD, FAIR, POOR, FAILING
	ShouldTrade    bool    `json:"shouldTrade"`
	SizeMultiplier float64 `json:"sizeMultiplier"`       // Position size adjustment (1.0 = normal, 0.5 = half, etc.)
	Confidence     string  `json:"confidence,omitempty"` // LOW_CONFIDENCE, UNRELIABLE if sample size small
}

// Recommendation contains actionable trading recommendation
type Recommendation struct {
	Priority int    `json:"priority"` // 1 = highest
	Category string `json:"category"` // STOP, REDUCE_SIZE, FILTER, REGIME, TIMING, CONTINUE
	Action   string `json:"action"`   // "Raise minimum score to 9"
	Reason   string `json:"reason"`   // "Score 7-8 trades have -0.08R expectancy"
	Impact   string `json:"impact"`   // "Would eliminate 15 losing trades"
}

// RegimeBreakdown contains per-regime statistics
type RegimeBreakdown struct {
	RegimeStats           map[string]StatisticsResult `json:"regimeStats"` // Keyed by regime name
	BestRegime            string                      `json:"bestRegime"`
	WorstRegime           string                      `json:"worstRegime"`
	RegimeContributions   map[string]float64          `json:"regimeContributions"` // % of total P&L per regime
	InsufficientDataNotes []string                    `json:"insufficientDataNotes,omitempty"`
}

// VNMetrics contains Vietnam-specific performance metrics
type VNMetrics struct {
	GapSlippageFactor     float64 `json:"gapSlippageFactor"`     // 0.0-1.0, % worse due to gaps
	GapAdjustedExpectancy float64 `json:"gapAdjustedExpectancy"` // Expectancy adjusted for gap risk
	CapitalEfficiency     float64 `json:"capitalEfficiency"`     // % of time capital is usable (T+2 impact)
	EffectiveAnnualReturn float64 `json:"effectiveAnnualReturn"` // Return adjusted for locked capital
	TotalGapLosses        int     `json:"totalGapLosses"`        // Number of trades hitting floor/ceiling
	GapLossPercentage     float64 `json:"gapLossPercentage"`     // % of total losses from gaps
	AvgSlippageBeyondStop float64 `json:"avgSlippageBeyondStop"` // Average % slippage beyond intended stop
}

// ComprehensiveReport contains all integrated statistics and insights
type ComprehensiveReport struct {
	// Base statistics (from existing calculator)
	BaseStats StatisticsResult `json:"baseStats"`

	// Vietnam-specific metrics
	VNMetrics VNMetrics `json:"vnMetrics"`

	// Regime breakdown
	RegimeBreakdown RegimeBreakdown `json:"regimeBreakdown"`

	// System health assessment
	Health SystemHealth `json:"health"`

	// Actionable recommendations
	Recommendations []Recommendation `json:"recommendations"`

	// Sample size assessment
	SampleSizeStatus  string `json:"sampleSizeStatus"` // INSUFFICIENT, WARNING, ADEQUATE, GOOD, EXCELLENT
	SampleSizeWarning string `json:"sampleSizeWarning,omitempty"`

	// Summary fields
	TotalTrades    int     `json:"totalTrades"`
	InitialBalance float64 `json:"initialBalance"`
	FinalBalance   float64 `json:"finalBalance"`
}
