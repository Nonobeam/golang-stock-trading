package risk

import (
	"fmt"
	"math"
)

// CalculateTrailingStop calculates trailing stop levels for remaining position.
//
// Supports 4 methods: ATR, EMA, percentage, and swing low trailing.
//
// Example:
//
//	params := TrailingStopParams{
//	    EntryPrice: 52000,
//	    CurrentPrice: 64000,
//	    HighestPriceReached: 66000,
//	    Method: "atr",
//	    ATR: 2500,
//	    ATRMultiplier: 1.5,
//	}
//	result := CalculateTrailingStop(params)
func CalculateTrailingStop(params TrailingStopParams) TrailingStopResult {
	var trailingStop float64
	var methodDescription string

	switch params.Method {
	case "atr":
		if params.ATR <= 0 {
			return TrailingStopResult{Method: "Trailing Stop (ATR)"}
		}
		multiplier := params.ATRMultiplier
		if multiplier == 0 {
			multiplier = 1.5
		}
		trailingStop = params.HighestPriceReached - (params.ATR * multiplier)
		methodDescription = fmt.Sprintf("%.1f× ATR trail", multiplier)

	case "ema":
		if params.EMAValue <= 0 {
			return TrailingStopResult{Method: "Trailing Stop (EMA)"}
		}
		trailingStop = params.EMAValue
		methodDescription = "EMA trail"

	case "percentage":
		percentage := params.Percentage
		if percentage == 0 {
			percentage = 5.0
		}
		trainingDistance := params.HighestPriceReached * (percentage / 100)
		trailingStop = params.HighestPriceReached - trainingDistance
		methodDescription = fmt.Sprintf("%.1f%% trail", percentage)

	case "swing_low":
		if params.RecentSwingLow <= 0 {
			return TrailingStopResult{Method: "Trailing Stop (Swing Low)"}
		}
		buffer := params.RecentSwingLow * 0.01
		trailingStop = params.RecentSwingLow - buffer
		methodDescription = "Swing low trail"

	default:
		return TrailingStopResult{Method: "Trailing Stop"}
	}

	if trailingStop >= params.CurrentPrice {
		trailingStop = params.CurrentPrice * 0.98
	}

	lockedInProfit := 0.0
	if trailingStop < params.EntryPrice {
		trailingStop = params.EntryPrice
	} else {
		lockedInProfit = trailingStop - params.EntryPrice
	}

	distanceFromCurrent := params.CurrentPrice - trailingStop
	distancePercent := (distanceFromCurrent / params.CurrentPrice) * 100

	profitIfStopped := trailingStop - params.EntryPrice
	profitPercent := (profitIfStopped / params.EntryPrice) * 100

	recommendation := "Hold - trailing"
	if params.CurrentPrice <= trailingStop {
		recommendation = "EXIT - stop hit"
	}

	return TrailingStopResult{
		Method:                 fmt.Sprintf("Trailing Stop (%s)", methodDescription),
		EntryPrice:             params.EntryPrice,
		CurrentPrice:           params.CurrentPrice,
		HighestPriceReached:    params.HighestPriceReached,
		TrailingStop:           roundToNearest100(trailingStop),
		DistanceFromCurrent:    math.Abs(distanceFromCurrent),
		DistancePercent:        math.Abs(distancePercent),
		LockedInProfit:         lockedInProfit,
		ProfitPercentIfStopped: profitPercent,
		AboveEntry:             trailingStop >= params.EntryPrice,
		Recommendation:         recommendation,
	}
}
