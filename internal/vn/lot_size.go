package vn

import (
	"errors"
	"fmt"
)

// ErrInvalidLotSize is returned when quantity is not a valid lot size
var ErrInvalidLotSize = errors.New("quantity must be multiple of 100 shares")

// ErrBelowMinimum is returned when quantity is below minimum lot
var ErrBelowMinimum = errors.New("quantity below minimum 100 shares (1 lot)")

// DefaultLotSize is the standard lot size for Vietnam market (100 shares)
const DefaultLotSize = 100

// RoundToLotSize rounds down the number of shares to the nearest valid lot size.
// Returns the rounded quantity and the adjustment factor for risk recalculation.
//
// Example:
//
//	rounded, factor := RoundToLotSize(537, 100)  // returns (500, 0.931)
func RoundToLotSize(shares int, lotSize int) (int, float64) {
	if lotSize <= 0 {
		lotSize = DefaultLotSize
	}

	lots := shares / lotSize
	roundedShares := lots * lotSize

	// Calculate adjustment factor for risk recalculation
	var adjustmentFactor float64
	if shares > 0 {
		adjustmentFactor = float64(roundedShares) / float64(shares)
	}

	return roundedShares, adjustmentFactor
}

// ValidateLotSize validates that the quantity is a valid lot size for Vietnam market.
// Returns an error if the quantity is not a multiple of 100 or is below the minimum.
func ValidateLotSize(quantity int) error {
	if quantity < DefaultLotSize {
		return ErrBelowMinimum
	}

	if quantity%DefaultLotSize != 0 {
		return fmt.Errorf("%w: got %d shares, nearest valid: %d",
			ErrInvalidLotSize, quantity, (quantity/DefaultLotSize)*DefaultLotSize)
	}

	return nil
}

// GetLotSize returns the default lot size for Vietnam market
func GetLotSize() int {
	return DefaultLotSize
}
