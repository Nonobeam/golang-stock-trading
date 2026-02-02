// Package backtest provides a comprehensive backtesting engine for the trading system.
// It simulates historical trading to validate strategy profitability before live trading.
package backtest

import (
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/position"
)

// BacktestConfig defines all configuration parameters for a backtest run.
type BacktestConfig struct {
	// Data configuration
	Symbol    string    `json:"symbol"`    // Stock symbol to backtest (e.g., "VCB")
	DataPath  string    `json:"dataPath"`  // Path to historical CSV data
	StartDate time.Time `json:"startDate"` // Backtest start date
	EndDate   time.Time `json:"endDate"`   // Backtest end date

	// Capital configuration
	InitialCapital float64 `json:"initialCapital"` // Starting capital in VND

	// Trading rules
	MinSignalScore int     `json:"minSignalScore"` // Minimum signal score to trade (e.g., 7)
	MaxPositions   int     `json:"maxPositions"`   // Maximum concurrent positions (e.g., 3)
	RiskPerTrade   float64 `json:"riskPerTrade"`   // Risk per trade as percentage (e.g., 0.01 for 1%)

	// Execution settings
	Commission float64 `json:"commission"` // Commission as percentage (e.g., 0.0015 for 0.15%)
	Slippage   float64 `json:"slippage"`   // Slippage as percentage (e.g., 0.001 for 0.1%)

	// Output settings
	OutputPath string `json:"outputPath"` // Directory for reports
	DryRun     bool   `json:"dryRun"`     // If true, validate only, don't run full backtest
	Verbose    bool   `json:"verbose"`    // Enable verbose logging
}

// BacktestResult contains the complete results of a backtest run.
type BacktestResult struct {
	Config         *BacktestConfig  `json:"config"`
	StartDate      time.Time        `json:"startDate"`
	EndDate        time.Time        `json:"endDate"`
	TradingDays    int              `json:"tradingDays"`
	InitialCapital float64          `json:"initialCapital"`
	FinalCapital   float64          `json:"finalCapital"`
	ClosedTrades   []*ClosedTrade   `json:"closedTrades"`
	EquityCurve    []EquityPoint    `json:"equityCurve"`
	Metrics        *BacktestMetrics `json:"metrics"`
}

// BacktestPosition represents an open position during backtest.
type BacktestPosition struct {
	*position.Position // Embed existing position type

	// Basic position info
	EntryDate  time.Time `json:"entryDate"`
	EntryPrice float64   `json:"entryPrice"`
	Shares     int       `json:"shares"`

	// Signal metadata
	SignalID    string `json:"signalId"`
	SignalType  string `json:"signalType"`  // "pullback", "breakout", etc.
	SignalScore int    `json:"signalScore"` // 0-13 score
	EntryReason string `json:"entryReason"`

	// Targets and stops
	InitialStop float64   `json:"initialStop"`
	CurrentStop float64   `json:"currentStop"`
	Targets     []float64 `json:"targets"`
	TargetsHit  []bool    `json:"targetsHit"`

	// Risk tracking
	InitialRisk      float64 `json:"initialRisk"`      // Entry - Stop in VND
	CapitalAllocated float64 `json:"capitalAllocated"` // Total capital used

	// Performance tracking
	MAE float64 `json:"mae"` // Maximum Adverse Excursion
	MFE float64 `json:"mfe"` // Maximum Favorable Excursion
}

// ClosedTrade represents a completed trade with all details.
type ClosedTrade struct {
	// Identification
	Symbol      string `json:"symbol"`
	SignalType  string `json:"signalType"`
	SignalScore int    `json:"signalScore"`

	// Entry details
	EntryDate  time.Time `json:"entryDate"`
	EntryPrice float64   `json:"entryPrice"`
	Shares     int       `json:"shares"`

	// Exit details
	ExitDate   time.Time `json:"exitDate"`
	ExitPrice  float64   `json:"exitPrice"`
	ExitReason string    `json:"exitReason"` // "stop", "target1", "target2", "target3", "trailing", "end_of_backtest"

	// Performance
	PnL         float64 `json:"pnl"`         // Profit/loss in VND
	PnLPercent  float64 `json:"pnlPercent"`  // Profit/loss as percentage
	RMultiple   float64 `json:"rMultiple"`   // Actual gain/loss vs initial risk
	HoldingDays int     `json:"holdingDays"` // Days held

	// Risk metrics
	InitialRisk float64 `json:"initialRisk"` // Entry - Stop in VND
	MAE         float64 `json:"mae"`         // Maximum Adverse Excursion
	MFE         float64 `json:"mfe"`         // Maximum Favorable Excursion

	// Costs
	Commission float64 `json:"commission"` // Total commission paid (entry + exit)
	Slippage   float64 `json:"slippage"`   // Total slippage cost
}

// EquityPoint represents capital at a specific date.
type EquityPoint struct {
	Date   time.Time `json:"date"`
	Equity float64   `json:"equity"`
}

// SimulatedFill represents the result of a simulated trade execution.
type SimulatedFill struct {
	Success         bool    `json:"success"`
	FilledPrice     float64 `json:"filledPrice"`
	FilledShares    int     `json:"filledShares"`
	Commission      float64 `json:"commission"`
	Slippage        float64 `json:"slippage"`
	TotalCost       float64 `json:"totalCost"`       // For buys: price × shares × (1 + commission)
	NetProceeds     float64 `json:"netProceeds"`     // For sells: price × shares × (1 - commission)
	RejectionReason string  `json:"rejectionReason"` // If Success == false
}

// CapitalTracker tracks available capital and T+2 settlements.
type CapitalTracker struct {
	currentCash        float64
	pendingSettlements map[string]float64 // date string -> amount
}

// NewCapitalTracker creates a new capital tracker.
func NewCapitalTracker(initialCash float64) *CapitalTracker {
	return &CapitalTracker{
		currentCash:        initialCash,
		pendingSettlements: make(map[string]float64),
	}
}

// GetAvailableCash returns currently available cash (excludes pending settlements).
func (c *CapitalTracker) GetAvailableCash() float64 {
	return c.currentCash
}

// DeductCash immediately deducts cash (for buy orders).
func (c *CapitalTracker) DeductCash(amount float64) {
	c.currentCash -= amount
}

// AddPendingSettlement schedules cash to be available on T+2 (for sell orders).
func (c *CapitalTracker) AddPendingSettlement(amount float64, settleDate time.Time) {
	dateKey := settleDate.Format("2006-01-02")
	c.pendingSettlements[dateKey] += amount
}

// SettlePendingCash processes settlements for the current date.
func (c *CapitalTracker) SettlePendingCash(currentDate time.Time) {
	dateKey := currentDate.Format("2006-01-02")
	if amount, exists := c.pendingSettlements[dateKey]; exists {
		c.currentCash += amount
		delete(c.pendingSettlements, dateKey)
	}
}

// GetTotalCapital returns current cash + pending settlements.
func (c *CapitalTracker) GetTotalCapital() float64 {
	total := c.currentCash
	for _, amount := range c.pendingSettlements {
		total += amount
	}
	return total
}

// BacktestMetrics contains comprehensive performance statistics.
type BacktestMetrics struct {
	// Trade statistics
	TotalTrades   int     `json:"totalTrades"`
	WinningTrades int     `json:"winningTrades"`
	LosingTrades  int     `json:"losingTrades"`
	WinRate       float64 `json:"winRate"` // Percentage

	// P&L statistics
	TotalPnL        float64 `json:"totalPnL"`
	TotalPnLPercent float64 `json:"totalPnLPercent"`
	AvgWin          float64 `json:"avgWin"`
	AvgLoss         float64 `json:"avgLoss"`
	LargestWin      float64 `json:"largestWin"`
	LargestLoss     float64 `json:"largestLoss"`
	ProfitFactor    float64 `json:"profitFactor"` // Total wins / Total losses

	// R-multiple statistics
	AvgRMultiple float64   `json:"avgRMultiple"` // Same as expectancy
	Expectancy   float64   `json:"expectancy"`   // Average R-multiple per trade
	RMultiples   []float64 `json:"rMultiples"`   // All R-multiples

	// Risk metrics
	MaxDrawdown        float64 `json:"maxDrawdown"`        // In VND
	MaxDrawdownPercent float64 `json:"maxDrawdownPercent"` // As percentage
	SharpeRatio        float64 `json:"sharpeRatio"`
	SortinoRatio       float64 `json:"sortinoRatio"`
	CalmarRatio        float64 `json:"calmarRatio"` // Return / Max DD

	// Time metrics
	AvgHoldingDays int `json:"avgHoldingDays"`

	// Streak tracking
	LongestWinStreak  int `json:"longestWinStreak"`
	LongestLossStreak int `json:"longestLossStreak"`

	// Breakdowns
	BySignalType map[string]*SignalTypeMetrics `json:"bySignalType,omitempty"`
	ByRegime     map[string]*RegimeMetrics     `json:"byRegime,omitempty"`
}

// SignalTypeMetrics contains performance for a specific signal type.
type SignalTypeMetrics struct {
	SignalType   string  `json:"signalType"`
	TotalTrades  int     `json:"totalTrades"`
	WinRate      float64 `json:"winRate"`
	AvgRMultiple float64 `json:"avgRMultiple"`
	ProfitFactor float64 `json:"profitFactor"`
	TotalPnL     float64 `json:"totalPnL"`
}

// RegimeMetrics contains performance for a specific market regime.
type RegimeMetrics struct {
	Regime       string  `json:"regime"`
	TotalTrades  int     `json:"totalTrades"`
	WinRate      float64 `json:"winRate"`
	AvgRMultiple float64 `json:"avgRMultiple"`
	ProfitFactor float64 `json:"profitFactor"`
	TotalPnL     float64 `json:"totalPnL"`
}
