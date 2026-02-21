// Package vn provides Vietnamese market FOL (Foreign Ownership Limit) monitoring.
package vn

import (
	"fmt"
	"time"
)

// FOLMonitor monitors foreign ownership limits for Vietnamese stocks.
type FOLMonitor struct {
	// In production, cache FOL data with 1-hour TTL
	cache map[string]*FOLData
	cacheTTL time.Duration
}

// FOLData contains foreign ownership limit information for a stock.
type FOLData struct {
	Symbol           string
	ForeignOwnership float64   // Current foreign ownership percentage
	ForeignLimit     float64   // Maximum allowed (typically 49% or 100%)
	LastUpdated      time.Time
}

// NewFOLMonitor creates a new FOL monitor.
func NewFOLMonitor() *FOLMonitor {
	return &FOLMonitor{
		cache:    make(map[string]*FOLData),
		cacheTTL: 1 * time.Hour,
	}
}

// CheckFOLRestriction checks if FOL is approaching limits.
// Returns restriction level: nil (safe), warning (>85%), emergency (>95%)
func (m *FOLMonitor) CheckFOLRestriction(symbol string) (*FOLRestriction, error) {
	folData, err := m.getFOLData(symbol)
	if err != nil {
		return nil, err
	}
	
	percentOfLimit := (folData.ForeignOwnership / folData.ForeignLimit) * 100
	
	if percentOfLimit > 95 {
		return &FOLRestriction{
			Level:            "EMERGENCY",
			Action:           "IMMEDIATE_EXIT",
			PercentOfLimit:   percentOfLimit,
			ForeignOwnership: folData.ForeignOwnership,
			ForeignLimit:     folData.ForeignLimit,
			Reason:           fmt.Sprintf("FOL at %.1f%% of %.0f%% limit - emergency exit required", percentOfLimit, folData.ForeignLimit),
		}, nil
	}
	
	if percentOfLimit > 85 {
		return &FOLRestriction{
			Level:            "WARNING",
			Action:           "ACCELERATE_EXITS",
			PercentOfLimit:   percentOfLimit,
			ForeignOwnership: folData.ForeignOwnership,
			ForeignLimit:     folData.ForeignLimit,
			Reason:           fmt.Sprintf("FOL at %.1f%% of %.0f%% limit - accelerate exits", percentOfLimit, folData.ForeignLimit),
		}, nil
	}
	
	return nil, nil // Safe
}

// FOLRestriction represents a FOL-based trading restriction.
type FOLRestriction struct {
	Level            string  // WARNING, EMERGENCY
	Action           string  // ACCELERATE_EXITS, IMMEDIATE_EXIT
	PercentOfLimit   float64
	ForeignOwnership float64
	ForeignLimit     float64
	Reason           string
}

// getFOLData retrieves FOL data (from cache or API).
// In production, this would call VSD (Vietnam Securities Depository) API.
func (m *FOLMonitor) getFOLData(symbol string) (*FOLData, error) {
	// Check cache first
	if cached, ok := m.cache[symbol]; ok {
		if time.Since(cached.LastUpdated) < m.cacheTTL {
			return cached, nil
		}
	}
	
	// TODO: Call VSD API for real data
	// For now, return mock data
	data := &FOLData{
		Symbol:           symbol,
		ForeignOwnership: 45.0, // Mock: 45%
		ForeignLimit:     49.0, // Most stocks have 49% limit
		LastUpdated:      time.Now(),
	}
	
	m.cache[symbol] = data
	return data, nil
}
