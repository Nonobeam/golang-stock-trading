package risk

import (
	"errors"
	"math"

	"github.com/nonobeam/golang-stock-trading/internal/vn"
)

// Error types for stop loss calculations.
var (
	ErrStopTooWide      = errors.New("stop loss exceeds max stop limit")
	ErrStopTooTight     = errors.New("stop loss below minimum distance")
	ErrNoStopMethod     = errors.New("no stop method available")
	ErrInvalidSwingData = errors.New("insufficient data for swing low calculation")
	ErrStopWrongSide    = errors.New("stop on wrong side of entry")
)

// StopResult holds comprehensive stop loss calculation results.
type StopResult struct {
	StopLossPrice       float64 // Final stop price (rounded to tick)
	StopDistance        float64 // Distance from entry in VND
	StopDistancePercent float64 // Distance as percentage
	Method              string  // Which method was used

	// Technical details
	ATR            float64 // ATR if used
	Multiplier     float64 // Multiplier if used
	SwingLow       float64 // Swing low if detected
	TechnicalLevel float64 // Support/MA level if used
	Buffer         float64 // Buffer applied

	// Vietnam floor awareness
	FloorPrice       float64 // Today's floor
	IntendedStop     float64 // Original intended stop
	ReachableToday   bool    // Can execute at intended stop
	Warning          string  // Floor warning if any
	WorstCaseStop    float64 // Multi-floor-day worst case
	WorstCasePercent float64 // Worst case loss %

	// Pre-emptive alerts
	Alert1Price  float64 // 50% to stop - exit half
	Alert1Action string  // Action for alert 1
	Alert2Price  float64 // 70% to stop - exit rest
	Alert2Action string  // Action for alert 2

	// Validation
	IsValid          bool     // Passed all validation
	ValidationIssues []string // Any issues found
}

// ATRStop calculates stop loss based on ATR.
func ATRStop(params StopParams) (float64, error) {
	if params.ATR <= 0 || params.Multiplier <= 0 {
		return 0, ErrInvalidInput
	}

	stopDistance := params.ATR * params.Multiplier

	var stop float64
	if params.IsLong {
		stop = params.EntryPrice - stopDistance
	} else {
		stop = params.EntryPrice + stopDistance
	}

	// Validate max stop
	if err := ValidateMaxStop(params.EntryPrice, stop); err != nil {
		return 0, err
	}

	return stop, nil
}

// ATRStopDetailed returns full StopResult for ATR-based stop.
func ATRStopDetailed(params StopParams) (*StopResult, error) {
	if params.ATR <= 0 || params.Multiplier <= 0 {
		return nil, ErrInvalidInput
	}

	stopDistance := params.ATR * params.Multiplier

	var stop float64
	if params.IsLong {
		stop = params.EntryPrice - stopDistance
	} else {
		stop = params.EntryPrice + stopDistance
	}

	result := &StopResult{
		StopLossPrice:       vn.RoundToTick(stop),
		StopDistance:        stopDistance,
		StopDistancePercent: (stopDistance / params.EntryPrice) * 100,
		Method:              "ATR-based",
		ATR:                 params.ATR,
		Multiplier:          params.Multiplier,
		ReachableToday:      true,
	}

	// Validate
	result.IsValid, result.ValidationIssues = validateStopInternal(params.EntryPrice, result.StopLossPrice, params.IsLong)

	return result, nil
}

// PercentageStop calculates stop loss based on percentage.
func PercentageStop(params StopParams) (float64, error) {
	if params.Percentage <= 0 || params.Percentage > 1 {
		return 0, ErrInvalidInput
	}

	var stop float64
	if params.IsLong {
		stop = params.EntryPrice * (1 - params.Percentage)
	} else {
		stop = params.EntryPrice * (1 + params.Percentage)
	}

	if err := ValidateMaxStop(params.EntryPrice, stop); err != nil {
		return 0, err
	}

	return stop, nil
}

// PercentageStopDetailed returns full StopResult for percentage-based stop.
func PercentageStopDetailed(params StopParams) (*StopResult, error) {
	if params.Percentage <= 0 || params.Percentage > 1 {
		return nil, ErrInvalidInput
	}

	stopDistance := params.EntryPrice * params.Percentage
	var stop float64
	if params.IsLong {
		stop = params.EntryPrice - stopDistance
	} else {
		stop = params.EntryPrice + stopDistance
	}

	result := &StopResult{
		StopLossPrice:       vn.RoundToTick(stop),
		StopDistance:        stopDistance,
		StopDistancePercent: params.Percentage * 100,
		Method:              "Percentage-based",
		ReachableToday:      true,
	}

	result.IsValid, result.ValidationIssues = validateStopInternal(params.EntryPrice, result.StopLossPrice, params.IsLong)

	return result, nil
}

// TechnicalStop uses a technical level with optional buffer.
func TechnicalStop(params StopParams) (float64, error) {
	if params.TechnicalStop <= 0 {
		return 0, ErrInvalidInput
	}

	stop := params.TechnicalStop

	// Apply buffer below technical level for longs
	if params.Buffer > 0 {
		if params.IsLong {
			stop = params.TechnicalStop * (1 - params.Buffer)
		} else {
			stop = params.TechnicalStop * (1 + params.Buffer)
		}
	}

	if err := ValidateMaxStop(params.EntryPrice, stop); err != nil {
		return 0, err
	}

	return stop, nil
}

// SupportStop places stop below a support level with buffer.
func SupportStop(supportLevel float64, bufferPercent float64, bufferATR float64, isLong bool) (*StopResult, error) {
	if supportLevel <= 0 {
		return nil, ErrInvalidInput
	}

	var buffer float64
	var bufferType string

	if bufferATR > 0 {
		buffer = bufferATR
		bufferType = "ATR"
	} else if bufferPercent > 0 {
		buffer = supportLevel * bufferPercent
		bufferType = "Percentage"
	} else {
		buffer = supportLevel * 0.005 // Default 0.5%
		bufferType = "Default"
	}

	var stop float64
	if isLong {
		stop = supportLevel - buffer
	} else {
		stop = supportLevel + buffer
	}

	return &StopResult{
		StopLossPrice:       vn.RoundToTick(stop),
		StopDistance:        buffer,
		StopDistancePercent: (buffer / supportLevel) * 100,
		Method:              "Support Level (" + bufferType + " buffer)",
		TechnicalLevel:      supportLevel,
		Buffer:              buffer,
		ReachableToday:      true,
		IsValid:             true,
	}, nil
}

// MovingAverageStop places stop below a moving average with buffer.
func MovingAverageStop(maValue float64, maPeriod int, bufferPercent float64, bufferATR float64, isLong bool) (*StopResult, error) {
	if maValue <= 0 {
		return nil, ErrInvalidInput
	}

	var buffer float64
	var bufferType string

	if bufferATR > 0 {
		buffer = bufferATR
		bufferType = "ATR"
	} else if bufferPercent > 0 {
		buffer = maValue * bufferPercent
		bufferType = "Percentage"
	} else {
		buffer = maValue * 0.02 // Default 2%
		bufferType = "Default"
	}

	var stop float64
	if isLong {
		stop = maValue - buffer
	} else {
		stop = maValue + buffer
	}

	method := "MA-based"
	if maPeriod > 0 {
		method = string(rune(maPeriod)) + "-period MA"
	}

	return &StopResult{
		StopLossPrice:       vn.RoundToTick(stop),
		StopDistance:        buffer,
		StopDistancePercent: (buffer / maValue) * 100,
		Method:              method + " (" + bufferType + " buffer)",
		TechnicalLevel:      maValue,
		Buffer:              buffer,
		ReachableToday:      true,
		IsValid:             true,
	}, nil
}

// SwingLowStop calculates stop below recent swing low.
func SwingLowStop(params StopParams) (*StopResult, error) {
	lookback := params.LookbackPeriods
	if lookback <= 0 {
		lookback = GetSwingLookback()
	}

	if len(params.Lows) < lookback {
		return nil, ErrInvalidSwingData
	}

	// Get recent lows
	recentLows := params.Lows[len(params.Lows)-lookback:]

	// Find minimum (swing low)
	swingLow := recentLows[0]
	swingLowIndex := 0
	for i, low := range recentLows {
		if low < swingLow {
			swingLow = low
			swingLowIndex = i
		}
	}

	// Apply buffer (default 0.5% if not specified)
	bufferPercent := params.Buffer
	if bufferPercent <= 0 {
		bufferPercent = 0.005
	}

	buffer := swingLow * bufferPercent
	var stop float64
	if params.IsLong {
		stop = swingLow - buffer
	} else {
		stop = swingLow + buffer
	}

	stopDistance := math.Abs(params.EntryPrice - stop)

	result := &StopResult{
		StopLossPrice:       vn.RoundToTick(stop),
		StopDistance:        stopDistance,
		StopDistancePercent: (stopDistance / params.EntryPrice) * 100,
		Method:              "Swing Low",
		SwingLow:            swingLow,
		Buffer:              buffer,
		ReachableToday:      true,
	}

	// Add index info (relative to lookback)
	_ = swingLowIndex // Available for logging/debugging

	result.IsValid, result.ValidationIssues = validateStopInternal(params.EntryPrice, result.StopLossPrice, params.IsLong)

	return result, nil
}

// GapAdjustedStop widens stop for overnight positions in VN market.
func GapAdjustedStop(baseStopDistance float64) float64 {
	return baseStopDistance * GetGapRiskFactor()
}

// FloorAwareStop adjusts stop considering Vietnam daily floor limits.
func FloorAwareStop(entryPrice, intendedStop, referencePrice, dailyLimitPercent float64) (*StopResult, error) {
	if referencePrice <= 0 {
		return nil, ErrInvalidInput
	}

	if dailyLimitPercent <= 0 {
		dailyLimitPercent = vn.GetDailyLimitPercent()
	}

	// Calculate floor
	limits := vn.CalculateLimits(referencePrice)
	floorPrice := limits.Floor

	result := &StopResult{
		IntendedStop:   intendedStop,
		FloorPrice:     floorPrice,
		ReachableToday: intendedStop >= floorPrice,
	}

	if intendedStop < floorPrice {
		result.StopLossPrice = floorPrice
		result.Warning = "Intended stop below floor - adjusted to floor price"
		result.ReachableToday = false
	} else {
		result.StopLossPrice = vn.RoundToTick(intendedStop)
	}

	result.StopDistance = math.Abs(entryPrice - result.StopLossPrice)
	result.StopDistancePercent = (result.StopDistance / entryPrice) * 100
	result.Method = "Floor-aware"

	// Calculate worst case (3 consecutive floor days)
	worstCase := referencePrice * math.Pow(1-dailyLimitPercent, 3)
	result.WorstCaseStop = vn.RoundToTick(worstCase)
	result.WorstCasePercent = ((entryPrice - worstCase) / entryPrice) * 100

	result.IsValid = true
	return result, nil
}

// PreemptiveAlerts calculates early exit levels to avoid floor locks.
func PreemptiveAlerts(entryPrice, intendedStop float64) (*StopResult, error) {
	if entryPrice <= 0 || intendedStop <= 0 {
		return nil, ErrInvalidInput
	}

	stopDistance := math.Abs(entryPrice - intendedStop)

	// Alert 1: 50% of way to stop
	alert1Distance := stopDistance * 0.5
	alert1Price := entryPrice - alert1Distance

	// Alert 2: 70% of way to stop
	alert2Distance := stopDistance * 0.7
	alert2Price := entryPrice - alert2Distance

	return &StopResult{
		StopLossPrice:       intendedStop,
		StopDistance:        stopDistance,
		StopDistancePercent: (stopDistance / entryPrice) * 100,
		Method:              "Pre-emptive",
		Alert1Price:         vn.RoundToTick(alert1Price),
		Alert1Action:        "Exit 50% of position",
		Alert2Price:         vn.RoundToTick(alert2Price),
		Alert2Action:        "Exit remaining 50%",
		IsValid:             true,
	}, nil
}

// ValidateMaxStop checks if stop distance exceeds configured max stop percent.
func ValidateMaxStop(entry, stop float64) error {
	if entry == 0 {
		return ErrInvalidInput
	}

	stopPercent := math.Abs(entry-stop) / entry
	if stopPercent > GetMaxStopPercent() {
		return ErrStopTooWide
	}

	return nil
}

// ValidateStopFull performs comprehensive stop validation.
func ValidateStopFull(entryPrice, stopPrice float64, isLong bool) (*StopResult, error) {
	result := &StopResult{
		StopLossPrice: stopPrice,
	}

	result.IsValid, result.ValidationIssues = validateStopInternal(entryPrice, stopPrice, isLong)

	result.StopDistance = math.Abs(entryPrice - stopPrice)
	result.StopDistancePercent = (result.StopDistance / entryPrice) * 100

	if !result.IsValid {
		return result, errors.New(result.ValidationIssues[0])
	}

	return result, nil
}

// validateStopInternal performs internal validation checks.
func validateStopInternal(entryPrice, stopPrice float64, isLong bool) (bool, []string) {
	var issues []string

	// Check 1: Stop on correct side
	if isLong && stopPrice >= entryPrice {
		issues = append(issues, "stop on wrong side: must be below entry for long")
	}
	if !isLong && stopPrice <= entryPrice {
		issues = append(issues, "stop on wrong side: must be above entry for short")
	}

	// Check 2: Stop price is positive
	if stopPrice <= 0 {
		issues = append(issues, "stop price must be positive")
	}

	// Check 3: Stop not too wide
	if entryPrice > 0 {
		stopPercent := math.Abs(entryPrice-stopPrice) / entryPrice
		if stopPercent > GetMaxStopPercent() {
			issues = append(issues, "stop too wide: exceeds max stop percent")
		}

		// Check 4: Stop not too tight
		stopDistance := math.Abs(entryPrice - stopPrice)
		if stopPercent < GetMinStopPercent() {
			issues = append(issues, "stop too tight: below min stop percent")
		}
		if stopDistance < GetMinStopDistance() {
			issues = append(issues, "stop too tight: below min stop distance")
		}
	}

	return len(issues) == 0, issues
}

// StopDistance returns the absolute distance between entry and stop.
func StopDistance(entry, stop float64) float64 {
	return math.Abs(entry - stop)
}

// StopPercent returns stop distance as a percentage.
func StopPercent(entry, stop float64) float64 {
	if entry == 0 {
		return 0
	}
	return math.Abs(entry-stop) / entry
}

// CalculateStop selects the best stop method based on available params.
func CalculateStop(params StopParams) (float64, error) {
	// Priority: Swing Low > Technical > ATR > Percentage
	if len(params.Lows) >= GetSwingLookback() {
		result, err := SwingLowStop(params)
		if err == nil && result.IsValid {
			return result.StopLossPrice, nil
		}
	}
	if params.TechnicalStop > 0 {
		return TechnicalStop(params)
	}
	if params.ATR > 0 && params.Multiplier > 0 {
		return ATRStop(params)
	}
	if params.Percentage > 0 {
		return PercentageStop(params)
	}

	return 0, ErrNoStopMethod
}

// CalculateStopDetailed returns comprehensive stop result with all calculations.
func CalculateStopDetailed(params StopParams) (*StopResult, error) {
	var result *StopResult
	var err error

	// Determine effective lookback
	lookback := params.LookbackPeriods
	if lookback <= 0 {
		lookback = GetSwingLookback()
	}

	// Priority: Swing Low > Technical > ATR > Percentage
	if params.LookbackPeriods > 0 && len(params.Lows) >= lookback {
		result, err = SwingLowStop(params)
		if err == nil && result.IsValid {
			goto applyFloorAware
		}
	}

	if params.TechnicalStop > 0 {
		stop, stopErr := TechnicalStop(params)
		if stopErr == nil {
			result = &StopResult{
				StopLossPrice:       vn.RoundToTick(stop),
				StopDistance:        math.Abs(params.EntryPrice - stop),
				StopDistancePercent: (math.Abs(params.EntryPrice-stop) / params.EntryPrice) * 100,
				Method:              "Technical",
				TechnicalLevel:      params.TechnicalStop,
				Buffer:              params.Buffer,
				IsValid:             true,
			}
			goto applyFloorAware
		}
	}

	if params.ATR > 0 && params.Multiplier > 0 {
		result, err = ATRStopDetailed(params)
		if err == nil {
			goto applyFloorAware
		}
	}

	if params.Percentage > 0 {
		result, err = PercentageStopDetailed(params)
		if err == nil {
			goto applyFloorAware
		}
	}

	return nil, ErrNoStopMethod

applyFloorAware:
	// Apply floor awareness if reference price provided
	if params.ReferencePrice > 0 {
		floorResult, floorErr := FloorAwareStop(params.EntryPrice, result.StopLossPrice, params.ReferencePrice, params.DailyLimitPercent)
		if floorErr == nil {
			result.FloorPrice = floorResult.FloorPrice
			result.IntendedStop = result.StopLossPrice
			result.ReachableToday = floorResult.ReachableToday
			result.Warning = floorResult.Warning
			result.WorstCaseStop = floorResult.WorstCaseStop
			result.WorstCasePercent = floorResult.WorstCasePercent
			if !result.ReachableToday {
				result.StopLossPrice = floorResult.StopLossPrice
			}
		}
	}

	// Add pre-emptive alerts if enabled
	if params.EnablePreemptive || IsPreemptiveEnabled() {
		preemptive, _ := PreemptiveAlerts(params.EntryPrice, result.StopLossPrice)
		if preemptive != nil {
			result.Alert1Price = preemptive.Alert1Price
			result.Alert1Action = preemptive.Alert1Action
			result.Alert2Price = preemptive.Alert2Price
			result.Alert2Action = preemptive.Alert2Action
		}
	}

	return result, nil
}
