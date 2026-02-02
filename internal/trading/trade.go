package trading

import (
	"fmt"
	"sync"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/statistics"
)

// TradeLog provides in-memory storage for completed trades
type TradeLog struct {
	trades []statistics.Trade
	mu     sync.RWMutex
}

// NewTradeLog creates a new empty trade log
func NewTradeLog() *TradeLog {
	return &TradeLog{
		trades: make([]statistics.Trade, 0),
	}
}

// AddTrade adds a completed trade to the log
func (tl *TradeLog) AddTrade(trade statistics.Trade) error {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if trade.Symbol == "" {
		return fmt.Errorf("trade symbol cannot be empty")
	}
	if trade.ExitTime.Before(trade.EntryTime) {
		return fmt.Errorf("exit time cannot be before entry time")
	}

	tl.trades = append(tl.trades, trade)
	return nil
}

// GetAllTrades returns all trades in the log
func (tl *TradeLog) GetAllTrades() []statistics.Trade {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	result := make([]statistics.Trade, len(tl.trades))
	copy(result, tl.trades)
	return result
}

// GetTradesBySymbol returns all trades for a specific symbol
func (tl *TradeLog) GetTradesBySymbol(symbol string) []statistics.Trade {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	result := make([]statistics.Trade, 0)
	for _, trade := range tl.trades {
		if trade.Symbol == symbol {
			result = append(result, trade)
		}
	}
	return result
}

// GetTradesByDate returns all trades within a date range (inclusive)
func (tl *TradeLog) GetTradesByDate(start, end time.Time) []statistics.Trade {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	result := make([]statistics.Trade, 0)
	for _, trade := range tl.trades {
		if (trade.ExitTime.Equal(start) || trade.ExitTime.After(start)) &&
			(trade.ExitTime.Equal(end) || trade.ExitTime.Before(end)) {
			result = append(result, trade)
		}
	}
	return result
}

// Count returns the total number of trades
func (tl *TradeLog) Count() int {
	tl.mu.RLock()
	defer tl.mu.RUnlock()
	return len(tl.trades)
}

// Clear removes all trades from the log (useful for testing)
func (tl *TradeLog) Clear() {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.trades = make([]statistics.Trade, 0)
}
