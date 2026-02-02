package position

import (
	"fmt"
	"strings"
)

// PositionDashboard generates dashboard views of positions.
type PositionDashboard struct {
	tracker *PositionTracker
}

// NewDashboard creates a new position dashboard.
func NewDashboard(tracker *PositionTracker) *PositionDashboard {
	return &PositionDashboard{tracker: tracker}
}

// GenerateSummaryTable generates a text-based summary table of all positions.
func (d *PositionDashboard) GenerateSummaryTable() string {
	summary := d.tracker.GetAllPositionsSummary()

	if summary.NumPositions == 0 {
		return "No open positions"
	}

	var output strings.Builder

	// Header
	output.WriteString(strings.Repeat("=", 120))
	output.WriteString("\n")
	output.WriteString("OPEN POSITIONS SUMMARY\n")
	output.WriteString(strings.Repeat("=", 120))
	output.WriteString("\n")
	output.WriteString(fmt.Sprintf("Total Positions: %d | Total Value: %.0f VND | Total P&L: %.0f VND\n",
		summary.NumPositions, summary.TotalValue, summary.TotalPL))
	output.WriteString(strings.Repeat("=", 120))
	output.WriteString("\n\n")

	// Table header
	output.WriteString(fmt.Sprintf("%-8s %-10s %-10s %-8s %-12s %-8s %-7s %-6s %-8s\n",
		"Ticker", "Entry", "Current", "Shares", "P&L", "%", "R", "Days", "Stop %"))
	output.WriteString(strings.Repeat("-", 120))
	output.WriteString("\n")

	// Each position
	for _, pos := range summary.Positions {
		output.WriteString(fmt.Sprintf("%-8s %10.0f %10.0f %8d %12.0f %7.1f%% %6.2fR %6d %7.1f%%\n",
			pos.Ticker,
			pos.EntryPrice,
			pos.CurrentPrice,
			pos.SharesRemaining,
			pos.UnrealizedPL,
			pos.UnrealizedPLPercent,
			pos.RMultiple,
			pos.DaysInTrade,
			pos.StopDistancePercent))
	}

	output.WriteString(strings.Repeat("=", 120))
	output.WriteString("\n")

	return output.String()
}

// GenerateDetailedReport generates a detailed report for a single position.
func (d *PositionDashboard) GenerateDetailedReport(positionID string) string {
	position, err := d.tracker.GetPosition(positionID)
	if err != nil {
		return fmt.Sprintf("Position %s not found", positionID)
	}

	metrics, _ := d.tracker.GetPositionMetrics(positionID)

	var output strings.Builder

	output.WriteString(strings.Repeat("=", 80))
	output.WriteString("\n")
	output.WriteString(fmt.Sprintf("POSITION REPORT: %s\n", position.Ticker))
	output.WriteString(strings.Repeat("=", 80))
	output.WriteString("\n\n")

	// Entry details
	output.WriteString("ENTRY DETAILS:\n")
	output.WriteString(fmt.Sprintf("  Date: %s\n", position.EntryDate.Format("2006-01-02 15:04")))
	output.WriteString(fmt.Sprintf("  Entry Price: %.0f VND\n", position.EntryPrice))
	output.WriteString(fmt.Sprintf("  Shares: %d (Remaining: %d)\n", position.Shares, position.SharesRemaining))
	output.WriteString(fmt.Sprintf("  Position Value: %.0f VND\n", position.PositionValue))
	if position.SetupType != "" {
		output.WriteString(fmt.Sprintf("  Setup Type: %s\n", position.SetupType))
	}
	if position.TradeScore > 0 {
		output.WriteString(fmt.Sprintf("  Trade Score: %d\n", position.TradeScore))
	}
	output.WriteString("\n")

	// Current status
	output.WriteString("CURRENT STATUS:\n")
	output.WriteString(fmt.Sprintf("  Current Price: %.0f VND\n", position.CurrentPrice))
	output.WriteString(fmt.Sprintf("  Days in Trade: %d\n", metrics.DaysInTrade))
	output.WriteString(fmt.Sprintf("  Unrealized P&L: %.0f VND (%.2f%%)\n", metrics.UnrealizedPL, metrics.UnrealizedPLPercent))
	output.WriteString(fmt.Sprintf("  R-Multiple: %.2fR\n", metrics.RMultiple))

	if metrics.RealizedPL != 0 {
		output.WriteString(fmt.Sprintf("  Realized P&L: %.0f VND\n", metrics.RealizedPL))
		output.WriteString(fmt.Sprintf("  Total P&L: %.0f VND\n", metrics.TotalPL))
	}
	output.WriteString("\n")

	// Extremes
	output.WriteString("EXTREMES:\n")
	output.WriteString(fmt.Sprintf("  Highest: %.0f VND (MFE: +%.2f%% / +%.2fR)\n",
		metrics.HighestPrice, metrics.MFEPercent, metrics.MFE_R))
	output.WriteString(fmt.Sprintf("  Lowest: %.0f VND (MAE: -%.2f%% / -%.2fR)\n",
		metrics.LowestPrice, metrics.MAEPercent, metrics.MAE_R))
	output.WriteString("\n")

	// Stop loss
	output.WriteString("STOP LOSS:\n")
	output.WriteString(fmt.Sprintf("  Stop Price: %.0f VND\n", position.StopLoss))
	output.WriteString(fmt.Sprintf("  Distance: %.0f VND (%.2f%%)\n", metrics.StopDistance, metrics.StopDistancePercent))
	if metrics.StopHit {
		output.WriteString("  ⚠️  STOP HIT - EXIT IMMEDIATELY\n")
	}
	output.WriteString("\n")

	// Targets
	if len(metrics.TargetProgress) > 0 {
		output.WriteString("TARGETS:\n")
		for _, target := range metrics.TargetProgress {
			status := fmt.Sprintf("%.0f%% complete", target.PercentComplete)
			if target.TargetHit {
				status = "✓ HIT"
			}
			output.WriteString(fmt.Sprintf("  T%d: %.0f VND (%.1fR) - %s\n",
				target.TargetNumber, target.TargetPrice, target.RMultiple, status))
		}
		output.WriteString("\n")
	}

	// Partial exits
	if len(position.Exits) > 0 {
		output.WriteString("PARTIAL EXITS:\n")
		for _, exit := range position.Exits {
			output.WriteString(fmt.Sprintf("  %s: %d shares @ %.0f VND (%s)\n",
				exit.Date.Format("2006-01-02 15:04"), exit.Shares, exit.Price, exit.Reason))
		}
		output.WriteString("\n")
	}

	// Risk metrics
	output.WriteString("RISK METRICS:\n")
	output.WriteString(fmt.Sprintf("  Risk Remaining: %.0f VND (%.2f%%)\n", metrics.RiskRemaining, metrics.RiskRemainingPercent))
	output.WriteString(fmt.Sprintf("  Current R:R: %.2f:1\n", metrics.RiskRewardCurrent))
	output.WriteString("\n")

	output.WriteString(strings.Repeat("=", 80))
	output.WriteString("\n")

	return output.String()
}
