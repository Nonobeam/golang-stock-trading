package telegram

import (
	"fmt"
	"strings"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
	"github.com/nonobeam/golang-stock-trading/internal/vn"
)

// FormatSettlementStatusMessage formats settlement status for display
func FormatSettlementStatusMessage(pos *repository.Position, currentDate time.Time) string {
	var sb strings.Builder

	// Settlement status
	status := "LIQUID"
	if pos.SettlementStatus != nil {
		status = *pos.SettlementStatus
	}

	sb.WriteString(fmt.Sprintf("📊 Settlement Status: *%s*\n", status))

	// Days until liquid
	if pos.PurchaseDate != nil && status != "LIQUID" {
		daysUntilLiquid := vn.GetDaysUntilLiquid(*pos.PurchaseDate, currentDate)
		sb.WriteString(fmt.Sprintf("⏳ Days until liquid: *%d*\n", daysUntilLiquid))

		if pos.CanSellDate != nil {
			sb.WriteString(fmt.Sprintf("📅 Can sell from: *%s*\n", pos.CanSellDate.Format("2006-01-02")))
		}
	}

	// Capital breakdown
	if pos.LockedCapital != nil || pos.LiquidCapital != nil {
		sb.WriteString("\n💰 Capital Breakdown:\n")
		if pos.LockedCapital != nil && *pos.LockedCapital > 0 {
			sb.WriteString(fmt.Sprintf("  🔒 Locked: %s VND\n", formatVND(*pos.LockedCapital)))
		}
		if pos.LiquidCapital != nil && *pos.LiquidCapital > 0 {
			sb.WriteString(fmt.Sprintf("  Liquid: %s VND\n", formatVND(*pos.LiquidCapital)))
		}
	}

	// Exchange and risk
	if pos.Exchange != nil {
		sb.WriteString(fmt.Sprintf("\n🏢 Exchange: *%s*\n", *pos.Exchange))

		if pos.IsLocked() {
			lockedRisk := pos.GetLockedRisk()
			sb.WriteString(fmt.Sprintf("⚠️  Locked Risk: %s VND\n", formatVND(lockedRisk)))
			sb.WriteString("   (worst-case floor-hit scenario)\n")
		}
	}

	// Warning if stop loss cannot be executed
	if pos.IsLocked() {
		sb.WriteString("\n⚠️  *WARNING*: Stop loss cannot be executed until shares settle\n")
	}

	return sb.String()
}

// FormatLockedRiskReport formats locked risk summary for user's portfolio
func FormatLockedRiskReport(
	totalLockedRisk float64,
	maxAllowed float64,
	available float64,
	usedPercent float64,
	threshold float64,
	accountValue float64,
) string {
	var sb strings.Builder

	sb.WriteString("📊 *Locked Risk Budget Report*\n\n")

	// Usage bar
	bars := int(usedPercent / 10)
	if bars > 10 {
		bars = 10
	}
	progressBar := strings.Repeat("█", bars) + strings.Repeat("░", 10-bars)
	sb.WriteString(fmt.Sprintf("%s %.1f%%\n\n", progressBar, usedPercent))

	// Numbers
	sb.WriteString(fmt.Sprintf("Account Value: %s VND\n", formatVND(accountValue)))
	sb.WriteString(fmt.Sprintf("Total Locked Risk: %s VND\n", formatVND(totalLockedRisk)))
	sb.WriteString(fmt.Sprintf("Max Allowed (%.0f%%): %s VND\n", threshold*100, formatVND(maxAllowed)))
	sb.WriteString(fmt.Sprintf("Available: %s VND\n\n", formatVND(available)))

	// Warning if close to threshold
	if usedPercent > 80 {
		sb.WriteString("⚠️  *WARNING*: Approaching locked risk threshold!\n")
		sb.WriteString("New purchases may be restricted.\n\n")
	}

	return sb.String()
}

// FormatPositionLiquidNotification formats notification when position becomes liquid
func FormatPositionLiquidNotification(pos *repository.Position) string {
	var sb strings.Builder

	sb.WriteString("*Position Now Liquid*\n\n")
	sb.WriteString(fmt.Sprintf("Symbol: *%s*\n", pos.Symbol))
	sb.WriteString(fmt.Sprintf("Quantity: *%d* shares\n", pos.Quantity))

	totalValue := float64(pos.Quantity) * pos.EntryPrice
	if pos.LiquidCapital != nil {
		totalValue = *pos.LiquidCapital
	}
	sb.WriteString(fmt.Sprintf("Value: *%s VND*\n\n", formatVND(totalValue)))

	sb.WriteString("Your shares are now sellable.\n")
	sb.WriteString("Stop loss protection is active.\n")

	return sb.String()
}
