package position

import (
	"encoding/json"
	"math"
	"time"
)

// Position represents a complete trading position from entry until fully closed.
type Position struct {
	// Identification
	PositionID string `json:"positionId"`
	Ticker     string `json:"ticker"`

	// Entry details
	EntryDate     time.Time `json:"entryDate"`
	EntryPrice    float64   `json:"entryPrice"`
	Shares        int       `json:"shares"`
	PositionValue float64   `json:"positionValue"` // EntryPrice × Shares

	// Risk management
	StopLoss     float64 `json:"stopLoss"`
	RiskPerShare float64 `json:"riskPerShare"` // |EntryPrice - StopLoss|
	TotalRisk    float64 `json:"totalRisk"`    // RiskPerShare × Shares
	RiskPercent  float64 `json:"riskPercent"`  // % of account risked

	// Targets
	Targets        []Target `json:"targets"`
	TrailingMethod string   `json:"trailingMethod,omitempty"`

	// Position type
	PositionType string `json:"positionType"` // "long" or "short"

	// Current state
	CurrentPrice float64   `json:"currentPrice"`
	LastUpdated  time.Time `json:"lastUpdated"`

	// Tracking extremes (for MAE/MFE)
	HighestPriceReached float64    `json:"highestPriceReached"`
	LowestPriceReached  float64    `json:"lowestPriceReached"`
	HighestDate         *time.Time `json:"highestDate,omitempty"`
	LowestDate          *time.Time `json:"lowestDate,omitempty"`

	// Partial exits
	SharesRemaining int    `json:"sharesRemaining"`
	Exits           []Exit `json:"exits"`

	// Metadata
	SetupType  string `json:"setupType,omitempty"`
	TradeScore int    `json:"tradeScore,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

// Target represents a profit target with exit percentage.
type Target struct {
	TargetNumber  int     `json:"targetNumber"`
	TargetPrice   float64 `json:"targetPrice"`
	PercentToSell int     `json:"percentToSell"` // Percentage of position to sell
	RMultiple     float64 `json:"rMultiple"`     // How many R this target represents
}

// Exit records a partial or full exit from a position.
type Exit struct {
	Date   time.Time `json:"date"`
	Price  float64   `json:"price"`
	Shares int       `json:"shares"`
	Reason string    `json:"reason"`
}

// Alert represents a position alert with severity and recommended action.
type Alert struct {
	Type     string `json:"type"`     // STOP_HIT, STOP_CLOSE, TARGET_HIT, TARGET_CLOSE, TIME_LONG, LARGE_PROFIT
	Severity string `json:"severity"` // HIGH, MEDIUM, LOW
	Message  string `json:"message"`
	Action   string `json:"action"`
}

// PositionMetrics contains all calculated metrics for a position.
type PositionMetrics struct {
	// Identification
	PositionID   string  `json:"positionId"`
	Ticker       string  `json:"ticker"`
	CurrentPrice float64 `json:"currentPrice"`

	// P&L metrics
	UnrealizedPL         float64 `json:"unrealizedPl"`
	UnrealizedPLPercent  float64 `json:"unrealizedPlPercent"`
	UnrealizedPLPerShare float64 `json:"unrealizedPlPerShare"`
	RMultiple            float64 `json:"rMultiple"`
	RealizedPL           float64 `json:"realizedPl"`
	RealizedShares       int     `json:"realizedShares"`
	TotalPL              float64 `json:"totalPl"`
	AccountImpactPercent float64 `json:"accountImpactPercent"`

	// Position details
	SharesRemaining        int     `json:"sharesRemaining"`
	SharesOriginal         int     `json:"sharesOriginal"`
	PositionRemainingValue float64 `json:"positionRemainingValue"`

	// Extremes (MAE/MFE)
	MAE          float64 `json:"mae"`
	MAEPercent   float64 `json:"maePercent"`
	MAE_R        float64 `json:"maeR"`
	LowestPrice  float64 `json:"lowestPrice"`
	MFE          float64 `json:"mfe"`
	MFEPercent   float64 `json:"mfePercent"`
	MFE_R        float64 `json:"mfeR"`
	HighestPrice float64 `json:"highestPrice"`

	// Time
	DaysInTrade  int     `json:"daysInTrade"`
	HoursInTrade float64 `json:"hoursInTrade"`
	EntryDate    string  `json:"entryDate"`

	// Stop
	StopLoss            float64 `json:"stopLoss"`
	StopDistance        float64 `json:"stopDistance"`
	StopDistancePercent float64 `json:"stopDistancePercent"`
	StopHit             bool    `json:"stopHit"`

	// Targets
	TargetProgress []TargetProgress `json:"targetProgress"`

	// Risk
	RiskRemaining        float64 `json:"riskRemaining"`
	RiskRemainingPercent float64 `json:"riskRemainingPercent"`
	RiskRewardCurrent    float64 `json:"riskRewardCurrent"`
}

// TargetProgress tracks progress toward a single target.
type TargetProgress struct {
	TargetNumber     int     `json:"targetNumber"`
	TargetPrice      float64 `json:"targetPrice"`
	RMultiple        float64 `json:"rMultiple"`
	PercentToSell    int     `json:"percentToSell"`
	DistanceToTarget float64 `json:"distanceToTarget"`
	DistancePercent  float64 `json:"distancePercent"`
	PercentComplete  float64 `json:"percentComplete"`
	TargetHit        bool    `json:"targetHit"`
}

// PortfolioSummary provides an overview of all open positions.
type PortfolioSummary struct {
	NumPositions      int               `json:"numPositions"`
	TotalValue        float64           `json:"totalValue"`
	TotalUnrealizedPL float64           `json:"totalUnrealizedPl"`
	TotalRealizedPL   float64           `json:"totalRealizedPl"`
	TotalPL           float64           `json:"totalPl"`
	TotalRisk         float64           `json:"totalRisk"`
	Positions         []PositionSummary `json:"positions"`
}

// PositionSummary is a brief summary of a single position for portfolio view.
type PositionSummary struct {
	PositionID          string  `json:"positionId"`
	Ticker              string  `json:"ticker"`
	EntryPrice          float64 `json:"entryPrice"`
	CurrentPrice        float64 `json:"currentPrice"`
	SharesRemaining     int     `json:"sharesRemaining"`
	UnrealizedPL        float64 `json:"unrealizedPl"`
	UnrealizedPLPercent float64 `json:"unrealizedPlPercent"`
	RMultiple           float64 `json:"rMultiple"`
	DaysInTrade         int     `json:"daysInTrade"`
	StopDistancePercent float64 `json:"stopDistancePercent"`
}

// ExitResult contains the result of a partial or full exit.
type ExitResult struct {
	PositionID      string    `json:"positionId"`
	ExitPrice       float64   `json:"exitPrice"`
	SharesSold      int       `json:"sharesSold"`
	SharesRemaining int       `json:"sharesRemaining"`
	ExitPL          float64   `json:"exitPl"`
	ExitPLPercent   float64   `json:"exitPlPercent"`
	ExitR           float64   `json:"exitR"`
	Reason          string    `json:"reason"`
	FullyClosed     bool      `json:"fullyClosed"`
	Timestamp       time.Time `json:"timestamp"`
	Error           string    `json:"error,omitempty"`
}

// PriceUpdateResult contains the result of updating a position's price.
type PriceUpdateResult struct {
	PositionID string          `json:"positionId"`
	Metrics    PositionMetrics `json:"metrics"`
	Alerts     []Alert         `json:"alerts"`
	Timestamp  time.Time       `json:"timestamp"`
	Error      string          `json:"error,omitempty"`
}

// Initialize sets up calculated fields after position creation.
func (p *Position) Initialize() {
	if p.SharesRemaining == 0 {
		p.SharesRemaining = p.Shares
	}

	if p.CurrentPrice == 0 {
		p.CurrentPrice = p.EntryPrice
	}

	if p.LastUpdated.IsZero() {
		p.LastUpdated = time.Now()
	}

	// Initialize extremes to entry price
	if p.HighestPriceReached == 0 {
		p.HighestPriceReached = p.EntryPrice
	}
	if p.LowestPriceReached == 0 {
		p.LowestPriceReached = p.EntryPrice
	}

	// Calculate risk per share
	if p.RiskPerShare == 0 && p.StopLoss > 0 {
		p.RiskPerShare = math.Abs(p.EntryPrice - p.StopLoss)
	}

	// Calculate total risk
	if p.TotalRisk == 0 && p.RiskPerShare > 0 {
		p.TotalRisk = p.RiskPerShare * float64(p.Shares)
	}

	// Calculate position value
	if p.PositionValue == 0 {
		p.PositionValue = p.EntryPrice * float64(p.Shares)
	}

	// Default position type
	if p.PositionType == "" {
		p.PositionType = "long"
	}

	// Initialize exits slice
	if p.Exits == nil {
		p.Exits = []Exit{}
	}

	// Initialize targets slice
	if p.Targets == nil {
		p.Targets = []Target{}
	}
}

// ToJSON converts position to JSON bytes.
func (p *Position) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

// PositionFromJSON creates a Position from JSON bytes.
func PositionFromJSON(data []byte) (*Position, error) {
	var p Position
	err := json.Unmarshal(data, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// SavedPositions represents the structure for saving/loading all positions.
type SavedPositions struct {
	Positions       map[string]*Position `json:"positions"`
	ClosedPositions []*Position          `json:"closedPositions"`
	SavedAt         time.Time            `json:"savedAt"`
}
