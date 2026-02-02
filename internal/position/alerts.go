package position

import (
	"fmt"
)

// Alert types
const (
	AlertTypeStopHit     = "STOP_HIT"
	AlertTypeStopClose   = "STOP_CLOSE"
	AlertTypeTargetHit   = "TARGET_HIT"
	AlertTypeTargetClose = "TARGET_CLOSE"
	AlertTypeTimeLong    = "TIME_LONG"
	AlertTypeLargeProfit = "LARGE_PROFIT"
)

// Alert severities
const (
	SeverityHigh   = "HIGH"
	SeverityMedium = "MEDIUM"
	SeverityLow    = "LOW"
)

// CheckAlerts checks for various alert conditions on a position.
func CheckAlerts(position *Position, metrics *PositionMetrics) []Alert {
	var alerts []Alert

	// Stop alerts
	alerts = append(alerts, checkStopAlerts(position, metrics)...)

	// Target alerts
	alerts = append(alerts, checkTargetAlerts(position, metrics)...)

	// Time alerts
	alerts = append(alerts, checkTimeAlerts(position, metrics)...)

	// Profit alerts
	alerts = append(alerts, checkProfitAlerts(position, metrics)...)

	return alerts
}

// checkStopAlerts checks for stop loss related alerts.
func checkStopAlerts(position *Position, metrics *PositionMetrics) []Alert {
	var alerts []Alert

	// Stop hit
	if metrics.StopHit {
		alerts = append(alerts, Alert{
			Type:     AlertTypeStopHit,
			Severity: SeverityHigh,
			Message:  fmt.Sprintf("%s stop loss hit at %.0f", position.Ticker, position.CurrentPrice),
			Action:   "EXIT IMMEDIATELY",
		})
		return alerts // Don't add STOP_CLOSE if already hit
	}

	// Stop approaching (within 2%)
	if metrics.StopDistancePercent < 2.0 {
		alerts = append(alerts, Alert{
			Type:     AlertTypeStopClose,
			Severity: SeverityMedium,
			Message:  fmt.Sprintf("%s within %.1f%% of stop", position.Ticker, metrics.StopDistancePercent),
			Action:   "Monitor closely",
		})
	}

	return alerts
}

// checkTargetAlerts checks for target related alerts.
func checkTargetAlerts(position *Position, metrics *PositionMetrics) []Alert {
	var alerts []Alert

	for _, targetProgress := range metrics.TargetProgress {
		if targetProgress.TargetHit {
			alerts = append(alerts, Alert{
				Type:     AlertTypeTargetHit,
				Severity: SeverityLow,
				Message:  fmt.Sprintf("%s Target %d hit at %.0f", position.Ticker, targetProgress.TargetNumber, targetProgress.TargetPrice),
				Action:   fmt.Sprintf("Consider selling %d%% of position", targetProgress.PercentToSell),
			})
		} else if targetProgress.PercentComplete >= 90 {
			alerts = append(alerts, Alert{
				Type:     AlertTypeTargetClose,
				Severity: SeverityLow,
				Message:  fmt.Sprintf("%s Target %d %.0f%% complete", position.Ticker, targetProgress.TargetNumber, targetProgress.PercentComplete),
				Action:   "Prepare limit order",
			})
		}
	}

	return alerts
}

// checkTimeAlerts checks for time-based alerts.
func checkTimeAlerts(position *Position, metrics *PositionMetrics) []Alert {
	var alerts []Alert

	if metrics.DaysInTrade >= 30 {
		alerts = append(alerts, Alert{
			Type:     AlertTypeTimeLong,
			Severity: SeverityLow,
			Message:  fmt.Sprintf("%s held for %d days", position.Ticker, metrics.DaysInTrade),
			Action:   "Review position - consider time stop",
		})
	}

	return alerts
}

// checkProfitAlerts checks for large profit alerts.
func checkProfitAlerts(position *Position, metrics *PositionMetrics) []Alert {
	var alerts []Alert

	if metrics.RMultiple >= 4.0 {
		alerts = append(alerts, Alert{
			Type:     AlertTypeLargeProfit,
			Severity: SeverityLow,
			Message:  fmt.Sprintf("%s at +%.1fR", position.Ticker, metrics.RMultiple),
			Action:   "Consider taking profits or tightening stop",
		})
	}

	return alerts
}
