package position

// StopAdjustmentReason represents the reason for a stop adjustment.
type StopAdjustmentReason string

// Stop adjustment reasons
const (
	ReasonBreakeven          StopAdjustmentReason = "breakeven"
	ReasonTargetHit          StopAdjustmentReason = "target_hit"
	ReasonTrailingATR        StopAdjustmentReason = "trailing_atr"
	ReasonTrailingEMA        StopAdjustmentReason = "trailing_ema"
	ReasonTrailingPercentage StopAdjustmentReason = "trailing_percentage"
	ReasonTrailingSwing      StopAdjustmentReason = "trailing_swing"
	ReasonTimeStop           StopAdjustmentReason = "time_stop"
	ReasonVolatilityChange   StopAdjustmentReason = "volatility_change"
	ReasonManual             StopAdjustmentReason = "manual"
)

// TrailingMethod represents the trailing stop method.
type TrailingMethod string

// Trailing methods
const (
	TrailingMethodATR        TrailingMethod = "atr"
	TrailingMethodEMA        TrailingMethod = "ema"
	TrailingMethodPercentage TrailingMethod = "percentage"
	TrailingMethodSwing      TrailingMethod = "swing"
)

// StopAdjustmentRule contains configuration for stop adjustment rules.
type StopAdjustmentRule struct {
	// Breakeven rules
	MoveToBreakevenAtR float64 `json:"moveToBreakevenAtR"` // Move to BE at this R-multiple (default: 1.0)
	BreakevenBuffer    float64 `json:"breakevenBuffer"`    // Buffer above entry in percent (default: 0.5)

	// Target-based rules
	AdjustOnTargetHit    bool `json:"adjustOnTargetHit"`    // Adjust stop when target hit (default: true)
	MoveToPreviousTarget bool `json:"moveToPreviousTarget"` // Move to previous target level (default: true)

	// Trailing stop configuration
	TrailingMethod TrailingMethod `json:"trailingMethod"` // Default: "atr"

	// ATR trailing
	ATRMultiplier         float64 `json:"atrMultiplier"`         // Default: 1.5
	MinATRDistancePercent float64 `json:"minAtrDistancePercent"` // Don't trail tighter than this % (default: 2.0)

	// EMA trailing
	EMAPeriod        int     `json:"emaPeriod"`        // Default: 20
	EMABufferPercent float64 `json:"emaBufferPercent"` // Buffer below EMA in % (default: 1.0)

	// Percentage trailing
	TrailingPercentage float64 `json:"trailingPercentage"` // Trail this % below highest (default: 5.0)

	// Swing trailing
	SwingBufferPercent float64 `json:"swingBufferPercent"` // Buffer below swing low in % (default: 1.0)

	// Time stops
	EnableTimeStop bool    `json:"enableTimeStop"` // Enable time-based stop (default: true)
	TimeStopDays   int     `json:"timeStopDays"`   // Exit if no progress after N days (default: 30)
	TimeStopMinR   float64 `json:"timeStopMinR"`   // Minimum R to apply time stop (default: 0.5)

	// Volatility adjustments
	AdjustForVolatility     bool    `json:"adjustForVolatility"`     // Adapt ATR multiplier to volatility (default: true)
	VolatilityMultiplierMin float64 `json:"volatilityMultiplierMin"` // Min ATR multiplier (default: 1.0)
	VolatilityMultiplierMax float64 `json:"volatilityMultiplierMax"` // Max ATR multiplier (default: 2.5)

	// Safety rules
	NeverWidenStop      bool    `json:"neverWidenStop"`      // Never move stop down for longs (default: true)
	MinAdjustmentAmount float64 `json:"minAdjustmentAmount"` // Don't adjust for less than this VND (default: 500)
}

// DefaultStopAdjustmentRule returns rules with sensible defaults.
func DefaultStopAdjustmentRule() StopAdjustmentRule {
	return StopAdjustmentRule{
		// Breakeven
		MoveToBreakevenAtR: 1.0,
		BreakevenBuffer:    0.5,

		// Target-based
		AdjustOnTargetHit:    true,
		MoveToPreviousTarget: true,

		// Trailing
		TrailingMethod:        TrailingMethodATR,
		ATRMultiplier:         1.5,
		MinATRDistancePercent: 2.0,
		EMAPeriod:             20,
		EMABufferPercent:      1.0,
		TrailingPercentage:    5.0,
		SwingBufferPercent:    1.0,

		// Time stops
		EnableTimeStop: true,
		TimeStopDays:   30,
		TimeStopMinR:   0.5,

		// Volatility
		AdjustForVolatility:     true,
		VolatilityMultiplierMin: 1.0,
		VolatilityMultiplierMax: 2.5,

		// Safety
		NeverWidenStop:      true,
		MinAdjustmentAmount: 500,
	}
}

// Indicators contains technical indicators needed for stop management.
type Indicators struct {
	ATR      float64 `json:"atr"`
	EMA20    float64 `json:"ema20"`
	SwingLow float64 `json:"swingLow"`
}
