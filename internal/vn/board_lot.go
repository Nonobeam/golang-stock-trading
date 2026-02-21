// Package vn provides Vietnamese market-specific utilities.
package vn

import (
	"fmt"
)

// Exchange represents a Vietnamese stock exchange.
type Exchange string

const (
	ExchangeHOSE Exchange = "HOSE" // Ho Chi Minh Stock Exchange
	ExchangeHNX  Exchange = "HNX"  // Hanoi Stock Exchange  
	ExchangeUPCOM Exchange = "UPCOM" // Unlisted Public Company Market
)

// BoardLotSize returns the board lot size for a given exchange.
func BoardLotSize(exchange Exchange) int {
	switch exchange {
	case ExchangeHOSE:
		return 10 // 10 shares per lot
	case ExchangeHNX:
		return 100 // 100 shares per lot
	case ExchangeUPCOM:
		return 100 // 100 shares per lot
	default:
		return 1 // Default to single shares
	}
}

// RoundToHOSELot rounds shares to HOSE board lot (10 shares).
func RoundToHOSELot(shares int) int {
	return (shares / 10) * 10
}

// RoundToHNXLot rounds shares to HNX board lot (100 shares).
func RoundToHNXLot(shares int) int {
	return (shares / 100) * 100
}

// RoundToBoardLot rounds shares to the appropriate board lot for the exchange.
func RoundToBoardLot(shares int, exchange Exchange) int {
	lot := BoardLotSize(exchange)
	return (shares / lot) * lot
}

// ValidateBoardLot validates that shares conform to exchange board lot rules.
// Returns error if shares are not a multiple of board lot size.
func ValidateBoardLot(shares int, exchange Exchange, allowOddLot bool) error {
	lot := BoardLotSize(exchange)
	
	// Check if shares are a multiple of board lot
	if shares%lot != 0 {
		if !allowOddLot {
			return &BoardLotError{
				Shares:     shares,
				Exchange:   exchange,
				BoardLot:   lot,
				IsValidOddLot: false,
			}
		}
	}
	
	return nil
}

// BoardLotError represents an error when shares don't conform to board lot rules.
type BoardLotError struct {
	Shares        int
	Exchange      Exchange
	BoardLot      int
	IsValidOddLot bool
}

func (e *BoardLotError) Error() string {
	if e.Exchange == ExchangeHOSE {
		return fmt.Sprintf("HOSE requires 10-share lots: %d shares invalid (rounded: %d)", 
			e.Shares, RoundToHOSELot(e.Shares))
	}
	if e.Exchange == ExchangeHNX {
		return fmt.Sprintf("HNX requires 100-share lots: %d shares invalid (rounded: %d)", 
			e.Shares, RoundToHNXLot(e.Shares))
	}
	return fmt.Sprintf("Invalid board lot size: %d shares for exchange %s", e.Shares, e.Exchange)
}

// GetExchange returns the exchange for a given symbol (simplified heuristic).
// In production, this should query a database or API.
func GetExchange(symbol string) Exchange {
	// This is a simplified heuristic - in production, use database lookup
	// HOSE symbols tend to be 3 letters (VIC, VNM, HPG, etc.)
	// HNX symbols can be longer or have different patterns
	
	// For now, default to HOSE (most common)
	return ExchangeHOSE
}
