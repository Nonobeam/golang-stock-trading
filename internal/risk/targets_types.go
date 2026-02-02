package risk

// TargetInfo holds detailed information about a single target.
type TargetInfo struct {
	TargetNumber      int     // 1, 2, 3, etc.
	TargetPrice       float64 // Calculated target price
	DistanceFromEntry float64 // Absolute distance from entry
	DistancePercent   float64 // Percentage distance from entry
	RMultiple         float64 // R-multiple if applicable
	Method            string  // Which method calculated this
	Confidence        string  // "High", "Moderate", "Low" (for consensus)
}

// RMultipleResult holds R-multiple target calculation results.
type RMultipleResult struct {
	Method       string
	EntryPrice   float64
	StopLoss     float64
	RiskPerShare float64
	RiskPercent  float64
	PositionType string // "long" or "short"
	Targets      []TargetInfo
	Summary      string
}

// ATRResult holds ATR-based target calculation results.
type ATRResult struct {
	Method       string
	EntryPrice   float64
	ATR          float64
	ATRPercent   float64
	PositionType string
	Targets      []TargetInfo
	Summary      string
}

// TechnicalResult holds technical resistance target results.
type TechnicalResult struct {
	Method              string
	EntryPrice          float64
	StopLoss            float64
	RiskPerShare        float64
	PositionType        string
	Targets             []TargetInfo
	AllResistanceLevels []float64
	Summary             string
}

// FibonacciResult holds Fibonacci extension target results.
type FibonacciResult struct {
	Method      string
	SwingLow    float64
	SwingHigh   float64
	PullbackLow float64
	WaveSize    float64
	WavePercent float64
	Targets     []TargetInfo
	Summary     string
}

// MeasuredMoveResult holds measured move target results.
type MeasuredMoveResult struct {
	Method             string
	ConsolidationLow   float64
	ConsolidationHigh  float64
	ConsolidationRange float64
	RangePercent       float64
	BreakoutPrice      float64
	Targets            []TargetInfo
	Summary            string
}

// TrailingStopResult holds trailing stop calculation results.
type TrailingStopResult struct {
	Method                 string
	EntryPrice             float64
	CurrentPrice           float64
	HighestPriceReached    float64
	TrailingStop           float64
	DistanceFromCurrent    float64
	DistancePercent        float64
	LockedInProfit         float64
	ProfitPercentIfStopped float64
	AboveEntry             bool
	Recommendation         string // "Hold - trailing" or "EXIT - stop hit"
}

// Consensus Target represents a target where multiple methods agree.
type ConsensusTarget struct {
	ConsensusNumber int
	TargetPrice     float64
	NumMethodsAgree int
	Methods         []string
	Confidence      string // "High" (3+ methods), "Moderate" (2 methods)
}

// StrategyTarget defines a single target in the scaling strategy.
type StrategyTarget struct {
	Name         string   // "target_1", "target_2", "trailing"
	Price        float64  // 0 for trailing
	SellPercent  int      // Percentage of position to sell
	RMultiple    float64  // R-multiple for this target
	MethodsAgree []string // Which methods agree on this target
	Confidence   string   // Confidence level
	Rationale    string   // Explanation
}

// TargetStrategy defines recommended profit-taking approach.
type TargetStrategy struct {
	Targets []StrategyTarget
}

// ComprehensiveResult holds results from all target methods.
type ComprehensiveResult struct {
	EntryPrice          float64
	StopLoss            float64
	AllMethods          map[string]interface{} // Map of method name to result
	ConsensusTargets    []ConsensusTarget
	RecommendedStrategy TargetStrategy
}
