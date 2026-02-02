package scanner

import (
	"sync"

	"github.com/nonobeam/golang-stock-trading/internal/data"
)

// BarCache stores recent OHLCV bars for each symbol in memory
type BarCache struct {
	mu      sync.RWMutex
	bars    map[string][]data.OHLCV // symbol -> bars
	maxSize int                     // Maximum bars to keep per symbol
}

// NewBarCache creates a new bar cache
func NewBarCache(maxSize int) *BarCache {
	return &BarCache{
		bars:    make(map[string][]data.OHLCV),
		maxSize: maxSize,
	}
}

// Add adds a new bar for a symbol
func (c *BarCache) Add(symbol string, bar data.OHLCV) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get existing bars or create new slice
	bars, exists := c.bars[symbol]
	if !exists {
		bars = make([]data.OHLCV, 0, c.maxSize)
	}

	// Append new bar
	bars = append(bars, bar)

	// Keep only last maxSize bars
	if len(bars) > c.maxSize {
		bars = bars[len(bars)-c.maxSize:]
	}

	c.bars[symbol] = bars
}

// Get returns bars for a symbol
func (c *BarCache) Get(symbol string) []data.OHLCV {
	c.mu.RLock()
	defer c.mu.RUnlock()

	bars, exists := c.bars[symbol]
	if !exists {
		return nil
	}

	// Return a copy to prevent external modifications
	result := make([]data.OHLCV, len(bars))
	copy(result, bars)
	return result
}

// GetLast returns the last N bars for a symbol
func (c *BarCache) GetLast(symbol string, n int) []data.OHLCV {
	c.mu.RLock()
	defer c.mu.RUnlock()

	bars, exists := c.bars[symbol]
	if !exists || len(bars) == 0 {
		return nil
	}

	// Get last n bars
	start := len(bars) - n
	if start < 0 {
		start = 0
	}

	result := make([]data.OHLCV, len(bars)-start)
	copy(result, bars[start:])
	return result
}

// HasEnoughBars checks if symbol has enough bars for analysis
func (c *BarCache) HasEnoughBars(symbol string, minBars int) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	bars, exists := c.bars[symbol]
	return exists && len(bars) >= minBars
}

// Clear removes all bars for a symbol
func (c *BarCache) Clear(symbol string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.bars, symbol)
}

// ClearAll removes all bars
func (c *BarCache) ClearAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.bars = make(map[string][]data.OHLCV)
}

// Size returns the number of bars for a symbol
func (c *BarCache) Size(symbol string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	bars, exists := c.bars[symbol]
	if !exists {
		return 0
	}
	return len(bars)
}

// Symbols returns all symbols in cache
func (c *BarCache) Symbols() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	symbols := make([]string, 0, len(c.bars))
	for symbol := range c.bars {
		symbols = append(symbols, symbol)
	}
	return symbols
}
