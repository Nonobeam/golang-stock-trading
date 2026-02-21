// Package signals provides entry signal detection for trading setups.
package signals

import (
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/data"
)

// SignalType represents the type of entry setup detected.
type SignalType string

const (
	SignalTypePullback      SignalType = "Pullback"
	SignalTypeBreakout      SignalType = "Breakout"
	SignalTypeCrossover     SignalType = "Crossover"
	SignalTypeMeanReversion SignalType = "MeanReversion"
	
	// Exit signal types for graduated profit-taking
	SignalTypeSellTarget1   SignalType = "SELL_TARGET1"   // 30% exit at +15% profit
	SignalTypeSellTarget2   SignalType = "SELL_TARGET2"   // 30% exit at +25% profit
	SignalTypeSellTarget3   SignalType = "SELL_TARGET3"   // 40% trailing stop exit
	SignalTypeSellEmergency SignalType = "SELL_EMERGENCY" // Full emergency exit
)

// ConfidenceLevel represents the confidence in a detected signal.
type ConfidenceLevel string

const (
	ConfidenceVeryHigh ConfidenceLevel = "Very High"
	ConfidenceHigh     ConfidenceLevel = "High"
	ConfidenceModerate ConfidenceLevel = "Moderate"
	ConfidenceLow      ConfidenceLevel = "Low"
)

// EntrySignal represents a detected trade entry opportunity.
type EntrySignal struct {
	Symbol     string             `json:"symbol"`
	Type       SignalType         `json:"type"`
	EntryPrice float64            `json:"entryPrice"`
	StopLoss   float64            `json:"stopLoss"`
	Targets    map[string]float64 `json:"targets"` // "target1", "target2", "target3"
	Confidence ConfidenceLevel    `json:"confidence"`
	Timestamp  time.Time          `json:"timestamp"`

	// Setup details for transparency
	SetupDetails map[string]interface{} `json:"setupDetails"`
}

// ScanResult represents the result of scanning a single symbol for a setup.
type ScanResult struct {
	SetupDetected bool         `json:"setupDetected"`
	SetupType     string       `json:"setupType"`
	Signal        *EntrySignal `json:"signal,omitempty"`

	// Failure information
	Reason string   `json:"reason,omitempty"`
	Issues []string `json:"issues,omitempty"`
	Stage  string   `json:"stage,omitempty"` // Where in pipeline it failed

	// Additional metadata
	PrerequisitesDetails interface{} `json:"prerequisitesDetails,omitempty"`
	TriggersDetails      interface{} `json:"triggersDetails,omitempty"`
}

// MarketData contains all data needed for signal detection.
type MarketData struct {
	Symbol    string
	Timestamp time.Time

	// Daily price data
	DailySeries   *data.Series
	CurrentClose  float64
	CurrentHigh   float64
	CurrentLow    float64
	CurrentOpen   float64
	CurrentVolume float64

	// Daily indicators
	EMA20         float64
	EMA50         float64
	SMA20         float64
	SMA50         float64
	RSI           float64
	MACD          float64
	MACDSignal    float64
	MACDHistogram float64
	ADX           float64
	PlusDI        float64
	MinusDI       float64
	ATR           float64
	StochK        float64
	StochD        float64

	// Weekly indicators
	WeeklyClose  float64
	WeeklySMA200 float64
	WeeklyRSI    float64
	WeeklyStochK float64
	WeeklyStochD float64

	// Volume statistics
	VolumeMA20       float64
	VolumePercentile float64

	// Vietnam market constraints (Criteria 2.2, 2.3)
	ReferencePrice  float64 // Reference price for daily limits
	CeilingPrice    float64 // Daily ceiling price (+7%)
	FloorPrice      float64 // Daily floor price (-7%)
	HitCeilingToday bool    // Whether stock hit ceiling during the day
	HitFloorToday   bool    // Whether stock hit floor during the day

	// Historical arrays (for pattern detection)
	Highs   []float64
	Lows    []float64
	Closes  []float64
	Volumes []float64
	Opens   []float64
}

// SetupDetector is the interface all setup detectors must implement.
type SetupDetector interface {
	Scan(data *MarketData) (*ScanResult, error)
}

// DataProvider provides market data for signal scanning.
type DataProvider interface {
	GetDailyData(symbol string) (*MarketData, error)
	GetVolumeStats(symbol string) (volumeMA20, volumePercentile float64, error error)
	GetSupportResistance(symbol string) (*SRLevels, error)
	GetMarketRegime(symbol string) (map[string]interface{}, error)
}

// SRLevels holds support and resistance levels for a symbol.
type SRLevels struct {
	Support    float64 // Primary support level
	Resistance float64 // Primary resistance level
}

// PullbackConfig holds configuration for pullback entry detector.
type PullbackConfig struct {
	MinRallyPercent     float64 // Minimum rally before pullback (5.0%)
	MaxRallyPercent     float64 // Maximum rally (20.0%)
	MinPullbackDays     int     // Minimum consolidation time (2)
	MaxPullbackDays     int     // Maximum pullback duration (10)
	EMAProximityPercent float64 // Must be within % of 20 EMA (3.0%)
	MinADX              float64 // Minimum trend strength (25)
	MinWeeklyRSI        float64 // Weekly trend confirmation (50)
	MinRSI              float64 // Minimum daily RSI (40)
	MaxRSI              float64 // Maximum daily RSI (60)
	MinTriggers         int     // Minimum triggers required (2)
}

// DefaultPullbackConfig returns proven pullback configuration.
func DefaultPullbackConfig() PullbackConfig {
	return PullbackConfig{
		MinRallyPercent:     5.0,
		MaxRallyPercent:     20.0,
		MinPullbackDays:     2,
		MaxPullbackDays:     10,
		EMAProximityPercent: 3.0,
		MinADX:              25.0,
		MinWeeklyRSI:        50.0,
		MinRSI:              40.0,
		MaxRSI:              60.0,
		MinTriggers:         2,
	}
}

// BreakoutConfig holds configuration for breakout entry detector.
type BreakoutConfig struct {
	MinConsolidationDays int     // Minimum consolidation (20)
	MaxConsolidationDays int     // Maximum consolidation (60)
	MinRangePercent      float64 // Minimum range (8.0%)
	MaxRangePercent      float64 // Maximum range (25.0%)
	MinResistanceTests   int     // Minimum tests of resistance (2)
	MaxResistanceTests   int     // Maximum tests (3)
	MinBreakoutATR       float64 // Minimum breakout distance (0.5)
	VolumePercentileMin  float64 // Minimum volume percentile (90)
	MaxRSI               float64 // Maximum RSI to avoid overextension (75)
}

// DefaultBreakoutConfig returns proven breakout configuration.
func DefaultBreakoutConfig() BreakoutConfig {
	return BreakoutConfig{
		MinConsolidationDays: 20,
		MaxConsolidationDays: 60,
		MinRangePercent:      8.0,
		MaxRangePercent:      25.0,
		MinResistanceTests:   2,
		MaxResistanceTests:   3,
		MinBreakoutATR:       0.5,
		VolumePercentileMin:  90.0,
		MaxRSI:               75.0,
	}
}

// CrossoverConfig holds configuration for MA crossover entry detector.
type CrossoverConfig struct {
	FastPeriod      int     // Fast EMA period (20)
	SlowPeriod      int     // Slow EMA period (50)
	MinADX          float64 // Minimum ADX for trend strength (20)
	MinPullbackDays int     // Minimum days in pullback (2)
	MaxPullbackDays int     // Maximum days in pullback (7)
	EMAProximity    float64 // Max distance from 20 EMA in percent (3.0)
	MinTriggers     int     // Minimum triggers required (2)
}

// DefaultCrossoverConfig returns proven crossover configuration.
func DefaultCrossoverConfig() CrossoverConfig {
	return CrossoverConfig{
		FastPeriod:      20,
		SlowPeriod:      50,
		MinADX:          20.0,
		MinPullbackDays: 2,
		MaxPullbackDays: 7,
		EMAProximity:    3.0,
		MinTriggers:     2,
	}
}

// MeanReversionConfig holds configuration for mean reversion entry detector.
type MeanReversionConfig struct {
	MaxADX               float64 // Maximum ADX for ranging market (20)
	MinRangeTests        int     // Minimum support/resistance tests (3)
	MinRangeWidthPercent float64 // Minimum range width (8.0%)
	MaxRangeWidthPercent float64 // Maximum range width (25.0%)
	RSIOversold          float64 // RSI oversold threshold (30)
	RSIOverbought        float64 // RSI overbought threshold (70)
	ProximityPercent     float64 // Max distance to support/resistance (3.0%)
}

// DefaultMeanReversionConfig returns proven mean reversion configuration.
func DefaultMeanReversionConfig() MeanReversionConfig {
	return MeanReversionConfig{
		MaxADX:               20.0,
		MinRangeTests:        3,
		MinRangeWidthPercent: 8.0,
		MaxRangeWidthPercent: 25.0,
		RSIOversold:          30.0,
		RSIOverbought:        70.0,
		ProximityPercent:     3.0,
	}
}

// CandlePattern represents a detected candlestick pattern.
type CandlePattern struct {
	IsBullish   bool
	PatternName string
	Strength    string // "strong", "moderate", "weak"
	Patterns    []string
}

// SupportLevel represents a support or resistance price level.
type SupportLevel struct {
	Price      float64
	Type       string  // "Swing Low", "Fibonacci 38.2%", "Round Number"
	Confidence float64 // 0.0 to 1.0
}

// VolumePatternResult holds volume analysis results.
type VolumePatternResult struct {
	Confirms                     bool
	VolumeDeclinedDuringPullback bool
	BounceVolumeSpike            bool
	Description                  string
	EarlyAvgVolume               float64
	LateAvgVolume                float64
	CurrentVolume                float64
}

// PrerequisiteResult holds prerequisite check results.
type PrerequisiteResult struct {
	Passes  bool
	Details []string
	Issues  []string
}

// TriggerResult holds trigger detection results.
type TriggerResult struct {
	TriggerCount   int
	Triggers       []TriggerInfo
	Sufficient     bool
	Recommendation string
}

// TriggerInfo represents a single detected trigger.
type TriggerInfo struct {
	Trigger     string
	Description string
	Strength    string
}

// ConsolidationResult holds consolidation pattern detection results.
type ConsolidationResult struct {
	IsValid           bool
	ConsolidationHigh float64
	ConsolidationLow  float64
	RangePercent      float64
	DaysInRange       int
	ResistanceTests   int
	Issues            []string
}
// ExitSignal represents a decision to exit a position (partial or full).
type ExitSignal struct {
Symbol          string                 `json:"symbol"`
Type            SignalType             `json:"type"`
ExitPrice       float64                `json:"exitPrice"`
ExitPercentage  int                    `json:"exitPercentage"`
TargetLevel     int                    `json:"targetLevel"`
Reason          string                 `json:"reason"`
Timestamp       time.Time              `json:"timestamp"`
CurrentPrice    float64                `json:"currentPrice"`
EntryPrice      float64                `json:"entryPrice"`
ProfitPercent   float64                `json:"profitPercent"`
Details         map[string]interface{} `json:"details,omitempty"`
}

// IsExitSignal returns true if the signal type is an exit signal.
func IsExitSignal(signalType SignalType) bool {
switch signalType {
case SignalTypeSellTarget1, SignalTypeSellTarget2, SignalTypeSellTarget3, SignalTypeSellEmergency:
return true
default:
return false
}
}
