package position

import (
	"github.com/nonobeam/golang-stock-trading/internal/regime"
	"github.com/nonobeam/golang-stock-trading/internal/statistics"
)

// ToTrade converts a closed Position to a statistics.Trade for performance analysis.
// This should only be called on fully closed positions (SharesRemaining == 0).
func (p *Position) ToTrade() statistics.Trade {
	// Calculate total P&L
	var totalPnL float64
	for _, exit := range p.Exits {
		if p.PositionType == "long" {
			totalPnL += (exit.Price - p.EntryPrice) * float64(exit.Shares)
		} else {
			totalPnL += (p.EntryPrice - exit.Price) * float64(exit.Shares)
		}
	}

	// Calculate average exit price
	var avgExitPrice float64
	if len(p.Exits) > 0 {
		var totalSharesExited int
		var weightedPrice float64
		for _, exit := range p.Exits {
			weightedPrice += exit.Price * float64(exit.Shares)
			totalSharesExited += exit.Shares
		}
		if totalSharesExited > 0 {
			avgExitPrice = weightedPrice / float64(totalSharesExited)
		}
	}

	// Calculate P&L percent
	var pnlPercent float64
	if p.EntryPrice > 0 {
		pnlPercent = ((avgExitPrice - p.EntryPrice) / p.EntryPrice) * 100
		if p.PositionType == "short" {
			pnlPercent = ((p.EntryPrice - avgExitPrice) / p.EntryPrice) * 100
		}
	}

	// Get exit time from last exit
	var exitTime = p.LastUpdated
	if len(p.Exits) > 0 {
		exitTime = p.Exits[len(p.Exits)-1].Date
	}

	// Parse signal type to match statistics expectations
	signalType := p.SetupType
	if signalType == "" {
		signalType = "Unknown"
	}

	// Parse regime (default to empty/unknown)
	regimeType := regime.RegimeType("")

	return statistics.Trade{
		Symbol:      p.Ticker,
		EntryTime:   p.EntryDate,
		ExitTime:    exitTime,
		EntryPrice:  p.EntryPrice,
		ExitPrice:   avgExitPrice,
		Quantity:    p.Shares,
		PnL:         totalPnL,
		PnLPercent:  pnlPercent,
		InitialRisk: p.RiskPerShare,
		SignalType:  signalType,
		Regime:      regimeType,
		Score:       p.TradeScore,
	}
}

// ToTradeWithRegime converts a closed Position to a statistics.Trade with a specified regime.
func (p *Position) ToTradeWithRegime(regimeType regime.RegimeType) statistics.Trade {
	trade := p.ToTrade()
	trade.Regime = regimeType
	return trade
}
