package scoring

import "fmt"

// Liquidity filter thresholds for VN30 stocks (Criteria 2.10-2.12)
const (
	// Criterion 2.10: Minimum median turnover
	VN30_MIN_TURNOVER_VND = 50_000_000_000 // 50 billion VND

	// Criterion 2.10: Minimum median volume
	VN30_MIN_VOLUME = 1_000_000 // 1 million shares

	// Criterion 2.11: Maximum bid-ask spread
	VN30_MAX_SPREAD_PERCENT = 0.5 // 0.5%

	// Criterion 2.12: Zero tolerance for zero-volume days
	VN30_MAX_ZERO_DAYS = 0 // No zero-volume days in last 30 days

	// Deprecated - old thresholds (kept for backward compatibility)
	MinDailyVolume   = 500_000       // Old: Minimum 500K shares
	MinDailyTurnover = 2_000_000_000 // Old: Minimum 2B VND
	MaxZeroVolDays   = 0             // No zero-volume days allowed
)

// CheckLiquidity validates if a stock meets minimum liquidity requirements.
// For VN30 stocks: median volume >= 1M, median turnover >= 50B VND, no zero-volume days.
// Uses strict VN30 thresholds per Criteria 2.10-2.12.
func CheckLiquidity(medianVolume, medianTurnover float64, zeroVolumeDays int) LiquidityResult {
	return CheckLiquidityStrict(medianVolume, medianTurnover, 0, zeroVolumeDays)
}

// CheckLiquidityStrict validates liquidity with spread validation (Criteria 2.10-2.12).
func CheckLiquidityStrict(medianVolume, medianTurnover, currentSpread float64, zeroVolumeDays int) LiquidityResult {
	result := LiquidityResult{
		Passes:  true,
		Issues:  []string{},
		Details: []string{},
	}

	// Check 1: Median volume (Criterion 2.10)
	if medianVolume >= VN30_MIN_VOLUME {
		result.Details = append(result.Details,
			formatCheck(true, "Median volume: %.0f shares (≥%d)", medianVolume, VN30_MIN_VOLUME))
	} else {
		result.Passes = false
		result.Issues = append(result.Issues,
			formatMsg("Insufficient volume: %.0f < %d shares (Criterion 2.10)", medianVolume, VN30_MIN_VOLUME))
		result.Details = append(result.Details,
			formatCheck(false, "Median volume: %.0f shares", medianVolume))
	}

	// Check 2: Median turnover (Criterion 2.10)
	if medianTurnover >= VN30_MIN_TURNOVER_VND {
		result.Details = append(result.Details,
			formatCheck(true, "Median turnover: %.0f VND (≥50B)", medianTurnover))
	} else {
		result.Passes = false
		result.Issues = append(result.Issues,
			formatMsg("Insufficient turnover: %.0f < 50B VND (Criterion 2.10)", medianTurnover))
		result.Details = append(result.Details,
			formatCheck(false, "Median turnover: %.0f VND", medianTurnover))
	}

	// Check 3: Zero-volume days (Criterion 2.12)
	if zeroVolumeDays <= VN30_MAX_ZERO_DAYS {
		result.Details = append(result.Details,
			formatCheck(true, "No zero-volume days in last 30 days"))
	} else {
		result.Passes = false
		result.Issues = append(result.Issues,
			formatMsg("%d zero-volume days (max %d) - Criterion 2.12", zeroVolumeDays, VN30_MAX_ZERO_DAYS))
		result.Details = append(result.Details,
			formatCheck(false, "%d zero-volume days", zeroVolumeDays))
	}

	// Check 4: Bid-ask spread (Criterion 2.11) - only if spread provided
	if currentSpread > 0 {
		if currentSpread <= VN30_MAX_SPREAD_PERCENT {
			result.Details = append(result.Details,
				formatCheck(true, "Spread: %.2f%% (≤%.1f%%)", currentSpread, VN30_MAX_SPREAD_PERCENT))
		} else {
			result.Passes = false
			result.Issues = append(result.Issues,
				formatMsg("Spread too wide: %.2f%% > %.1f%% (Criterion 2.11)", currentSpread, VN30_MAX_SPREAD_PERCENT))
			result.Details = append(result.Details,
				formatCheck(false, "Spread: %.2f%%", currentSpread))
		}
	}

	return result
}

// formatCheck returns a detail string with check/cross prefix
func formatCheck(pass bool, format string, args ...interface{}) string {
	prefix := "✓ "
	if !pass {
		prefix = "✗ "
	}
	return prefix + formatMsg(format, args...)
}

// formatMsg formats a message with arguments
func formatMsg(format string, args ...interface{}) string {
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}
