package ftd

import (
	"context"
	"fmt"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

// DowntrendDetector detects if the market is in a downtrend.
// Prerequisites for FTD: Market must be in a downtrend or correction.
type DowntrendDetector struct {
	repo *Repository
}

// NewDowntrendDetector creates a new downtrend detector.
func NewDowntrendDetector(repo *Repository) *DowntrendDetector {
	return &DowntrendDetector{repo: repo}
}

// DetectDowntrend analyzes market data to determine if a downtrend is active.
// It checks for:
// 1. RSI < 30 (Oversold)
// 2. Price breaking below 50-day SMA with volume
// 3. New 20-day lows
// 4. Support level tests
func (d *DowntrendDetector) DetectResult(ctx context.Context, date time.Time) (bool, []string, error) {
	// Need historical data for analysis (60 days)
	from := date.AddDate(0, 0, -60)
	regimes, err := d.repo.GetMarketRegimes(ctx, from, date)
	if err != nil {
		return false, nil, fmt.Errorf("failed to fetch history: %w", err)
	}

	if len(regimes) < 20 {
		return false, []string{"Insufficient data (<20 days)"}, nil
	}

	// Extract prices
	prices := make([]float64, len(regimes))
	for i, r := range regimes {
		prices[i] = r.IndexValue
	}

	reasons := []string{}
	isDowntrend := false

	// Check 1: RSI < 30
	rsi := CalculateRSI(prices, 14) // 14-day RSI
	if rsi < 30 {
		isDowntrend = true
		reasons = append(reasons, fmt.Sprintf("RSI Oversold (%.1f < 30)", rsi))
	}

	// Check 2: Price below 50-day SMA
	sma50 := CalculateSMA(prices, 50)
	currentPrice := prices[len(prices)-1]
	if sma50 > 0 && currentPrice < sma50 {
		// Strictly being below SMA50 isn't enough, but it's a bearish sign.
		// We usually look for the *break* or sustained trading below.
		// For FTD context, we just need to confirm we are NOT in a strong uptrend yet.
		reasons = append(reasons, fmt.Sprintf("Price below 50-day SMA (%.1f < %.1f)", currentPrice, sma50))
		// If we are significantly below, it supports downtrend
		if currentPrice < sma50*0.97 { // 3% below
			isDowntrend = true
		}
	}

	// Check 3: New 20-day low
	if IsNewLow(prices, 20) {
		isDowntrend = true
		reasons = append(reasons, "New 20-day low made")
	}

	// Check 4: Support Test
	atSupport, supportLevel := IsAtSupport(prices, 60, 2.0)
	if atSupport {
		// Testing support is often part of bottoming process (downtrend maturing)
		reasons = append(reasons, fmt.Sprintf("Testing support level at %.1f", supportLevel))
		isDowntrend = true
	}

	logger.Info().
		Time("date", date).
		Bool("is_downtrend", isDowntrend).
		Strs("reasons", reasons).
		Msg("Downtrend detection analysis")

	return isDowntrend, reasons, nil
}
