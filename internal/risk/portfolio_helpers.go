package risk

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
)

// ExportSnapshot exports the latest portfolio risk snapshot to a JSON file.
func (pm *PortfolioManager) ExportSnapshot(filepath string) error {
	if len(pm.RiskSnapshots) == 0 {
		return fmt.Errorf("no snapshots to export")
	}

	latestSnapshot := pm.RiskSnapshots[len(pm.RiskSnapshots)-1]

	data, err := json.MarshalIndent(latestSnapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	err = ioutil.WriteFile(filepath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ExportAllSnapshots exports the full snapshot history to a JSON file.
func (pm *PortfolioManager) ExportAllSnapshots(filepath string) error {
	data, err := json.MarshalIndent(pm.RiskSnapshots, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshots: %w", err)
	}

	err = ioutil.WriteFile(filepath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// formatVND formats a VND amount with comma separators.
func formatVND(amount float64) string {
	amountStr := fmt.Sprintf("%.0f", amount)

	negative := false
	if strings.HasPrefix(amountStr, "-") {
		negative = true
		amountStr = amountStr[1:]
	}

	var result strings.Builder
	for i, digit := range amountStr {
		if i > 0 && (len(amountStr)-i)%3 == 0 {
			result.WriteString(",")
		}
		result.WriteRune(digit)
	}

	if negative {
		return "-" + result.String()
	}
	return result.String()
}

// formatVNDWithSign formats a VND amount with +/- sign and comma separators.
func formatVNDWithSign(amount float64) string {
	if amount > 0 {
		return "+" + formatVND(amount)
	} else if amount < 0 {
		return formatVND(amount)
	}
	return formatVND(amount)
}

// generateProgressBar creates a visual progress bar.
func generateProgressBar(current, max, width float64) string {
	percent := (current / max) * 100
	if percent > 100 {
		percent = 100
	}

	filled := int((percent / 100) * width)
	empty := int(width) - filled

	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

// pluralize returns "s" if count != 1, empty string otherwise.
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// GetConsecutiveLosses returns the current consecutive loss count.
func (pm *PortfolioManager) GetConsecutiveLosses() int {
	return pm.ConsecutiveLosses
}

// GetConsecutiveWins returns the current consecutive win count.
func (pm *PortfolioManager) GetConsecutiveWins() int {
	return pm.ConsecutiveWins
}

// RecordTradeResultEnhanced records win/loss/breakeven for consecutive tracking with error handling.
// This is an enhanced version that supports "breakeven" and returns errors for invalid inputs.
// Use this instead of RecordTradeResult for better error handling.
func (pm *PortfolioManager) RecordTradeResultEnhanced(result string) error {
	if result == "win" {
		pm.ConsecutiveWins++
		pm.ConsecutiveLosses = 0
	} else if result == "loss" {
		pm.ConsecutiveLosses++
		pm.ConsecutiveWins = 0
	} else if result == "breakeven" {
		// Keep current streaks intact
	} else {
		return fmt.Errorf("invalid result: %s (must be 'win', 'loss', or 'breakeven')", result)
	}
	return nil
}
