package main

import (
	"context"

	"github.com/nonobeam/golang-stock-trading/internal/service/account"
	"github.com/nonobeam/golang-stock-trading/internal/service/position"
	"github.com/nonobeam/golang-stock-trading/internal/service/telegram"
)

// BotAdapters bridges account and position services to telegram bot interfaces
type BotAdapters struct {
	accountService  *account.Service
	positionService *position.Service
	userID          int64 // Simplified: default user ID (1)
}

// NewBotAdapters creates a new adapter instance
func NewBotAdapters(accountService *account.Service, positionService *position.Service) *BotAdapters {
	return &BotAdapters{
		accountService:  accountService,
		positionService: positionService,
		userID:          1, // Default user
	}
}

// === RiskManager Implementation ===

func (a *BotAdapters) GetPortfolioRisk() float64 {
	summary, err := a.accountService.GetAccountSummary(context.Background(), a.userID)
	if err != nil {
		return 0
	}
	return summary.RiskPercent / 100 // Convert percentage to decimal
}

func (a *BotAdapters) GetDailyLoss() float64 {
	summary, err := a.accountService.GetAccountSummary(context.Background(), a.userID)
	if err != nil {
		return 0
	}
	// Simplified: treat negative DayPnL as loss
	if summary.DayPnL < 0 {
		info, err := a.accountService.GetAccountInfo(context.Background(), a.userID)
		if err != nil || info.Capital == 0 {
			return 0
		}
		return (summary.DayPnL * -1) / info.Capital
	}
	return 0
}

func (a *BotAdapters) GetCapitalUtilization() float64 {
	info, err := a.accountService.GetAccountInfo(context.Background(), a.userID)
	if err != nil || info.Capital == 0 {
		return 0
	}
	return info.PositionsValue / info.Capital
}

func (a *BotAdapters) GetMaxPortfolioRisk() float64 {
	return 0.06 // 6% max portfolio risk
}

func (a *BotAdapters) GetDailyLossLimit() float64 {
	return 0.02 // 2% daily loss limit
}

// === PositionTracker Implementation ===

func (a *BotAdapters) GetActivePositions() []telegram.Position {
	positions, err := a.positionService.GetActivePositions(context.Background(), a.userID)
	if err != nil {
		return []telegram.Position{}
	}

	result := make([]telegram.Position, len(positions))
	for i, pos := range positions {
		targets := []float64{}
		if pos.TargetPrice > 0 {
			targets = append(targets, pos.TargetPrice)
		}

		// Calculate progress
		progress := 0.0
		if pos.TargetPrice > 0 {
			totalMove := pos.TargetPrice - pos.EntryPrice
			currentMove := pos.CurrentPrice - pos.EntryPrice
			if totalMove != 0 {
				progress = (currentMove / totalMove) * 100
			}
		}

		// Calculate stop distance
		stopDistance := 0.0
		if pos.StopPrice > 0 {
			stopDistance = ((pos.CurrentPrice - pos.StopPrice) / pos.CurrentPrice) * 100
		}

		result[i] = telegram.Position{
			Symbol:         pos.Symbol,
			EntryPrice:     pos.EntryPrice,
			CurrentPrice:   pos.CurrentPrice,
			StopLoss:       pos.StopPrice,
			Targets:        targets,
			PositionSize:   pos.Shares,
			RMultiple:      pos.RMultiple,
			TargetProgress: progress,
			StopDistance:   stopDistance,
		}
	}

	return result
}

// === RestartHandler Implementation ===

// RestartHandlerFunc is a function type that implements telegram.RestartHandler
type RestartHandlerFunc func(ctx context.Context, otp string) error

// OnRestart implements telegram.RestartHandler interface
func (f RestartHandlerFunc) OnRestart(ctx context.Context, otp string) error {
	return f(ctx, otp)
}

