package risk

// VietnamAggressiveLimits returns aggressive risk limits for FTD/Bull market.
// Allows for larger positions and more total risk.
func VietnamAggressiveLimits() *RiskLimits {
	limits := DefaultRiskLimits()
	limits.MaxTotalRiskPercent = 8.0   // Increased from 6.0%
	limits.MaxPositions = 8            // Standard 8 positions
	limits.DailyLossLimitPercent = 2.5 // Looser stop on daily loss
	limits.MaxPositionSizePercent = 25.0 // Allow up to 25% for high conviction
	return limits
}

// UpdateLimits updates the risk limits based on market regime.
func (pm *PortfolioManager) UpdateLimits(regime string) {
	switch regime {
	case "FTD_CONFIRMED", "BULL":
		pm.Limits = VietnamAggressiveLimits()
	case "BEAR", "DOWNTREND":
		pm.Limits = VietnamConservativeLimits()
	default:
		// Default or Transition
		pm.Limits = DefaultRiskLimits()
	}
}
