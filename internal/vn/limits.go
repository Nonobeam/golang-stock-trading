// Package vn provides Vietnam market-specific trading rules.
package vn

import (
	"errors"
	"math"

	"github.com/nonobeam/golang-stock-trading/internal/config"
)

// ErrPriceAboveCeiling is returned when order price exceeds ceiling.
var ErrPriceAboveCeiling = errors.New("price exceeds ceiling limit")

// ErrPriceBelowFloor is returned when order price is below floor.
var ErrPriceBelowFloor = errors.New("price below floor limit")

// PriceLimits holds ceiling and floor prices.
type PriceLimits struct {
	Reference float64
	Ceiling   float64
	Floor     float64
}

// GetDailyLimitPercent returns the configured daily limit (default: 0.07).
func GetDailyLimitPercent() float64 {
	cfg := config.Get()
	if cfg.Trading.VNDailyLimitPercent > 0 {
		return cfg.Trading.VNDailyLimitPercent
	}
	return 0.07
}

// GetSettlementDays returns the configured settlement days (default: 2).
func GetSettlementDays() int {
	cfg := config.Get()
	if cfg.Trading.VNSettlementDays > 0 {
		return cfg.Trading.VNSettlementDays
	}
	return 2
}

// CalculateLimits calculates ceiling and floor from reference price.
// Reference is typically previous closing price.
func CalculateLimits(reference float64) PriceLimits {
	limitPercent := GetDailyLimitPercent()
	ceiling := reference * (1 + limitPercent)
	floor := reference * (1 - limitPercent)

	// Round to VN price tick (varies by price range, simplified here)
	ceiling = RoundToTick(ceiling)
	floor = RoundToTick(floor)

	return PriceLimits{
		Reference: reference,
		Ceiling:   ceiling,
		Floor:     floor,
	}
}

// RoundToTick rounds price to the nearest valid tick size.
// VN market tick sizes:
// - Under 10,000: 10 VND
// - 10,000 - 49,950: 50 VND
// - 50,000+: 100 VND
func RoundToTick(price float64) float64 {
	var tick float64
	switch {
	case price < 10000:
		tick = 10
	case price < 50000:
		tick = 50
	default:
		tick = 100
	}

	return math.Round(price/tick) * tick
}

// ValidateOrderPrice checks if order price is within daily limits.
func ValidateOrderPrice(price float64, limits PriceLimits) error {
	if price > limits.Ceiling {
		return ErrPriceAboveCeiling
	}
	if price < limits.Floor {
		return ErrPriceBelowFloor
	}
	return nil
}

// AdjustStopToFloor adjusts stop loss if it falls below floor.
// Returns adjusted stop and warning message.
func AdjustStopToFloor(stop float64, limit PriceLimits) (float64, string) {
	if stop >= limit.Floor {
		return stop, ""
	}
	return limit.Floor, "stop adjusted to floor price - limited effectiveness"
}

// AdjustTargetToCeiling adjusts target if it exceeds ceiling.
func AdjustTargetToCeiling(target float64, limits PriceLimits) (float64, string) {
	if target <= limits.Ceiling {
		return target, ""
	}
	return limits.Ceiling, "target adjusted to ceiling price"
}

// CanReachTarget checks if target is reachable within daily limits.
func CanReachTarget(entry, target float64, limits PriceLimits) bool {
	if target > entry {
		return target <= limits.Ceiling
	}
	return target >= limits.Floor
}

// MaxGainToday calculates maximum possible gain from entry to ceiling.
func MaxGainToday(entry float64, limits PriceLimits) float64 {
	if entry == 0 {
		return 0
	}
	return (limits.Ceiling - entry) / entry
}

// MaxLossToday calculates maximum possible loss from entry to floor.
func MaxLossToday(entry float64, limits PriceLimits) float64 {
	if entry == 0 {
		return 0
	}
	return (entry - limits.Floor) / entry
}
