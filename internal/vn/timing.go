// Package vn provides Vietnamese market timing utilities.
package vn

import (
	"time"
)

// TradingWindow represents a Vietnamese market liquidity window.
type TradingWindow struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Priority  int // 1=highest liquidity, 3=lowest
}

// LiquidityWindowOptimizer optimizes trade timing for Vietnamese market liquidity.
type LiquidityWindowOptimizer struct{}

// NewLiquidityWindowOptimizer creates a new optimizer.
func NewLiquidityWindowOptimizer() *LiquidityWindowOptimizer {
	return &LiquidityWindowOptimizer{}
}

// GetOptimalWindow returns the best trading window for a given target date.
func (o *LiquidityWindowOptimizer) GetOptimalWindow(targetDate time.Time) *TradingWindow {
	// Vietnamese market high-liquidity windows (Vietnam time GMT+7)
	year, month, day := targetDate.Date()
	loc := time.FixedZone("GMT+7", 7*3600)
	
	windows := []TradingWindow{
		{
			Name:      "Morning Peak",
			StartTime: time.Date(year, month, day, 9, 45, 0, 0, loc),
			EndTime:   time.Date(year, month, day, 10, 30, 0, 0, loc),
			Priority:  1, // Highest liquidity
		},
		{
			Name:      "Afternoon Session Start",
			StartTime: time.Date(year, month, day, 13, 0, 0, 0, loc),
			EndTime:   time.Date(year, month, day, 13, 30, 0, 0, loc),
			Priority:  2,
		},
		{
			Name:      "Closing Auction Prep",
			StartTime: time.Date(year, month, day, 14, 15, 0, 0, loc),
			EndTime:   time.Date(year, month, day, 14, 30, 0, 0, loc),
			Priority:  3,
		},
	}
	
	// Return highest priority window
	return &windows[0]
}

// ShouldExecuteNow determines if current time is in a good liquidity window.
func (o *LiquidityWindowOptimizer) ShouldExecuteNow(now time.Time) bool {
	optimal := o.GetOptimalWindow(now)
	return now.After(optimal.StartTime) && now.Before(optimal.EndTime)
}

// OptimizeExitTiming returns the optimal execution time for an exit order.
// Considers T+2 settlement (completes 12:30pm on T+2) and liquidity windows.
func (o *LiquidityWindowOptimizer) OptimizeExitTiming(targetDate time.Time) time.Time {
	// For same-day cash availability on T+2, execute before 11am
	year, month, day := targetDate.Date()
	loc := time.FixedZone("GMT+7", 7*3600)
	
	earlyWindow := time.Date(year, month, day, 10, 0, 0, 0, loc) // 10am = morning peak
	
	// If target is today and we're still early, return morning peak time
	if targetDate.Year() == year && targetDate.Month() == month && targetDate.Day() == day {
		if time.Now().Before(earlyWindow) {
			return earlyWindow
		}
	}
	
	// Otherwise return optimal window for target date
	optimal := o.GetOptimalWindow(targetDate)
	return optimal.StartTime
}

// T2SettlementInfo contains T+2 settlement information.
type T2SettlementInfo struct {
	TradeDate       time.Time
	SettlementDate  time.Time
	SettlementTime  time.Time // 12:30pm on T+2
	CashAvailableAt time.Time
}

// CalculateT2Settlement calculates T+2 settlement details for Vietnamese market.
func CalculateT2Settlement(tradeDate time.Time) *T2SettlementInfo {
	loc := time.FixedZone("GMT+7", 7*3600)
	
	// T+2 = 2 business days after trade date
	settlementDate := addBusinessDays(tradeDate, 2)
	
	// Settlement completes at 12:30pm on T+2 (post-KRX integration May 2025)
	settlementTime := time.Date(
		settlementDate.Year(),
		settlementDate.Month(),
		settlementDate.Day(),
		12, 30, 0, 0, loc,
	)
	
	return &T2SettlementInfo{
		TradeDate:       tradeDate,
		SettlementDate:  settlementDate,
		SettlementTime:  settlementTime,
		CashAvailableAt: settlementTime,
	}
}

// addBusinessDays adds business days (excluding weekends and Vietnamese holidays).
func addBusinessDays(date time.Time, days int) time.Time {
	current := date
	added := 0
	
	for added < days {
		current = current.AddDate(0, 0, 1)
		
		// Skip weekends
		if current.Weekday() == time.Saturday || current.Weekday() == time.Sunday {
			continue
		}
		
		// TODO: Skip Vietnamese public holidays (Tet, etc.)
		
		added++
	}
	
	return current
}
