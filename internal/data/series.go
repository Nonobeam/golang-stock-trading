package data

import (
	"errors"
	"sync"
)

// ErrInsufficientData is returned when there's not enough data for calculation.
var ErrInsufficientData = errors.New("insufficient data points")

// Series represents a sliding window OHLCV time series.
type Series struct {
	data    []OHLCV
	maxSize int
	mu      sync.RWMutex
}

// NewSeries creates a new Series with the specified maximum size.
// If maxSize <= 0, the series has unlimited capacity.
func NewSeries(maxSize int) *Series {
	capacity := maxSize
	if capacity <= 0 {
		capacity = 1000 // default capacity for unlimited
	}
	return &Series{
		data:    make([]OHLCV, 0, capacity),
		maxSize: maxSize,
	}
}

// Append adds a new OHLCV bar to the series.
// If maxSize is exceeded, the oldest bar is removed.
func (s *Series) Append(bar OHLCV) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = append(s.data, bar)
	if s.maxSize > 0 && len(s.data) > s.maxSize {
		s.data = s.data[1:]
	}
}

// Len returns the number of bars in the series.
func (s *Series) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

// Get returns the bar at the specified index (0 = oldest).
func (s *Series) Get(index int) (OHLCV, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if index < 0 || index >= len(s.data) {
		return OHLCV{}, errors.New("index out of range")
	}
	return s.data[index], nil
}

// Last returns the most recent bar.
func (s *Series) Last() (OHLCV, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.data) == 0 {
		return OHLCV{}, ErrInsufficientData
	}
	return s.data[len(s.data)-1], nil
}

// LastN returns the last n bars (most recent last).
func (s *Series) LastN(n int) ([]OHLCV, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if n <= 0 {
		return nil, errors.New("n must be positive")
	}
	if len(s.data) < n {
		return nil, ErrInsufficientData
	}

	result := make([]OHLCV, n)
	copy(result, s.data[len(s.data)-n:])
	return result, nil
}

// Closes returns all closing prices as a slice.
func (s *Series) Closes() []float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	closes := make([]float64, len(s.data))
	for i, bar := range s.data {
		closes[i] = bar.Close
	}
	return closes
}

// Highs returns all high prices as a slice.
func (s *Series) Highs() []float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	highs := make([]float64, len(s.data))
	for i, bar := range s.data {
		highs[i] = bar.High
	}
	return highs
}

// Lows returns all low prices as a slice.
func (s *Series) Lows() []float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lows := make([]float64, len(s.data))
	for i, bar := range s.data {
		lows[i] = bar.Low
	}
	return lows
}

// Volumes returns all volumes as a slice.
func (s *Series) Volumes() []float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	volumes := make([]float64, len(s.data))
	for i, bar := range s.data {
		volumes[i] = bar.Volume
	}
	return volumes
}

// All returns a copy of all bars in the series.
func (s *Series) All() []OHLCV {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]OHLCV, len(s.data))
	copy(result, s.data)
	return result
}

// Clear removes all data from the series.
func (s *Series) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = s.data[:0]
}

// HighestHigh returns the highest high in the last n bars.
func (s *Series) HighestHigh(n int) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.data) < n {
		return 0, ErrInsufficientData
	}

	highest := s.data[len(s.data)-n].High
	for i := len(s.data) - n + 1; i < len(s.data); i++ {
		if s.data[i].High > highest {
			highest = s.data[i].High
		}
	}
	return highest, nil
}

// LowestLow returns the lowest low in the last n bars.
func (s *Series) LowestLow(n int) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.data) < n {
		return 0, ErrInsufficientData
	}

	lowest := s.data[len(s.data)-n].Low
	for i := len(s.data) - n + 1; i < len(s.data); i++ {
		if s.data[i].Low < lowest {
			lowest = s.data[i].Low
		}
	}
	return lowest, nil
}
