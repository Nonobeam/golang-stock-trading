package backtest

import (
	"fmt"

	"github.com/nonobeam/golang-stock-trading/internal/data"
)

// TradeSimulator simulates realistic trade execution with Vietnam market constraints
type TradeSimulator struct {
	commission float64 // as percentage, e.g. 0.0015 for 0.15%
	slippage   float64 // as percentage, e.g. 0.001 for 0.1%
}

// NewTradeSimulator creates a new trade simulator
func NewTradeSimulator(commission, slippage float64) *TradeSimulator {
	return &TradeSimulator{
		commission: commission,
		slippage:   slippage,
	}
}

// SimulateBuy simulates buying shares
// Returns SimulatedFill with execution details or rejection reason
func (s *TradeSimulator) SimulateBuy(
	symbol string,
	shares int,
	limitPrice float64,
	bar data.OHLCV,
	referencePrice float64,
) (*SimulatedFill, error) {
	// Check if limit price is within daily range
	// Buys can only execute if limit >= bar.Low
	if limitPrice < bar.Low {
		return &SimulatedFill{
			Success:         false,
			RejectionReason: fmt.Sprintf("limit price %.2f below day's low %.2f", limitPrice, bar.Low),
		}, nil
	}

	// Check Vietnam price limits (±7% for HOSE)
	ceiling := referencePrice * 1.07

	if limitPrice > ceiling {
		return &SimulatedFill{
			Success:         false,
			RejectionReason: fmt.Sprintf("limit price %.2f above ceiling %.2f", limitPrice, ceiling),
		}, nil
	}

	// If stock gapped to ceiling and stuck there, cannot buy
	if bar.High >= ceiling-1 && bar.Low >= ceiling-1 {
		return &SimulatedFill{
			Success:         false,
			RejectionReason: fmt.Sprintf("stock stuck at ceiling %.2f, no liquidity", ceiling),
		}, nil
	}

	// Calculate fill price (assume fill at min of limit or open, plus slippage)
	fillPrice := limitPrice
	if bar.Open < fillPrice {
		fillPrice = bar.Open
	}
	// Add pessimistic slippage
	fillPrice = fillPrice * (1 + s.slippage)

	// Ensure fill price doesn't exceed limit
	if fillPrice > limitPrice {
		fillPrice = limitPrice
	}

	// Calculate costs
	subtotal := fillPrice * float64(shares)
	commissionAmount := subtotal * s.commission
	totalCost := subtotal + commissionAmount

	return &SimulatedFill{
		Success:      true,
		FilledPrice:  fillPrice,
		FilledShares: shares,
		Commission:   commissionAmount,
		Slippage:     fillPrice - (fillPrice / (1 + s.slippage)), // backwards calc
		TotalCost:    totalCost,
	}, nil
}

// SimulateSell simulates selling shares
// Returns SimulatedFill with execution details or rejection reason
func (s *TradeSimulator) SimulateSell(
	symbol string,
	shares int,
	limitPrice float64,
	bar data.OHLCV,
	referencePrice float64,
) (*SimulatedFill, error) {
	// Check if limit price is within daily range
	// Sells can only execute if limit <= bar.High
	if limitPrice > bar.High {
		return &SimulatedFill{
			Success:         false,
			RejectionReason: fmt.Sprintf("limit price %.2f above day's high %.2f", limitPrice, bar.High),
		}, nil
	}

	// Check Vietnam price limits
	floor := referencePrice * 0.93

	if limitPrice < floor {
		return &SimulatedFill{
			Success:         false,
			RejectionReason: fmt.Sprintf("limit price %.2f below floor %.2f", limitPrice, floor),
		}, nil
	}

	// If stock gapped to floor and stuck there, apply maximum slippage
	atFloor := false
	if bar.High <= floor+1 && bar.Low <= floor+1 {
		atFloor = true
	}

	// Calculate fill price (assume fill at max of limit or open, minus slippage)
	fillPrice := limitPrice
	if bar.Open > fillPrice {
		fillPrice = bar.Open
	}

	// Apply pessimistic slippage
	if atFloor {
		// Maximum slippage at floor
		fillPrice = fillPrice * (1 - s.slippage*2)
	} else {
		fillPrice = fillPrice * (1 - s.slippage)
	}

	// Calculate proceeds
	subtotal := fillPrice * float64(shares)
	commissionAmount := subtotal * s.commission
	netProceeds := subtotal - commissionAmount

	return &SimulatedFill{
		Success:      true,
		FilledPrice:  fillPrice,
		FilledShares: shares,
		Commission:   commissionAmount,
		Slippage:     (fillPrice / (1 - s.slippage)) - fillPrice, // backwards calc
		TotalCost:    subtotal,
		NetProceeds:  netProceeds,
	}, nil
}

// ValidateWithinLimits checks if a price is within Vietnam daily limits
func (s *TradeSimulator) ValidateWithinLimits(price, refPrice float64, exchange string) error {
	var limitPercent float64
	switch exchange {
	case "HOSE":
		limitPercent = 0.07 // ±7%
	case "HNX":
		limitPercent = 0.10 // ±10%
	default:
		limitPercent = 0.07 // default to HOSE
	}

	ceiling := refPrice * (1 + limitPercent)
	floor := refPrice * (1 - limitPercent)

	if price > ceiling {
		return fmt.Errorf("price %.2f exceeds ceiling %.2f (%.0f%%)", price, ceiling, limitPercent*100)
	}
	if price < floor {
		return fmt.Errorf("price %.2f below floor %.2f (%.0f%%)", price, floor, limitPercent*100)
	}

	return nil
}

// IsAtCeiling checks if bar is stuck at ceiling
func (s *TradeSimulator) IsAtCeiling(bar data.OHLCV, refPrice float64) bool {
	ceiling := refPrice * 1.07
	return bar.High >= ceiling-1 && bar.Low >= ceiling-1
}

// IsAtFloor checks if bar is stuck at floor
func (s *TradeSimulator) IsAtFloor(bar data.OHLCV, refPrice float64) bool {
	floor := refPrice * 0.93
	return bar.High <= floor+1 && bar.Low <= floor+1
}

// DetectGapScenario detects if a bar gapped to ceiling or floor
func (s *TradeSimulator) DetectGapScenario(bar data.OHLCV, prevClose float64) string {
	ceiling := prevClose * 1.07
	floor := prevClose * 0.93

	// Check if gapped to ceiling (open at or near ceiling)
	if bar.Open >= ceiling-1 {
		if s.IsAtCeiling(bar, prevClose) {
			return "gap_to_ceiling"
		}
		return "gap_up"
	}

	// Check if gapped to floor (open at or near floor)
	if bar.Open <= floor+1 {
		if s.IsAtFloor(bar, prevClose) {
			return "gap_to_floor"
		}
		return "gap_down"
	}

	return "normal"
}
