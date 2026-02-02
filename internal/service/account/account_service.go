package account

import (
	"context"
	"fmt"

	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

type AccountInfoResponse struct {
	Capital          float64 `json:"capital"`
	Cash             float64 `json:"cash"`
	LockedCash       float64 `json:"lockedCash"`
	PositionsValue   float64 `json:"positionsValue"`
	BuyingPower      float64 `json:"buyingPower"`
	MarginUsed       float64 `json:"marginUsed"`
	MarginAvailable  float64 `json:"marginAvailable"`
}

type AccountSummaryResponse struct {
	TotalPnL       float64 `json:"totalPnL"`
	TotalPnLPercent float64 `json:"totalPnLPercent"`
	DayPnL         float64 `json:"dayPnL"`
	DayPnLPercent  float64 `json:"dayPnLPercent"`
	RiskExposure   float64 `json:"riskExposure"`
	RiskPercent    float64 `json:"riskPercent"`
}

type Service struct {
	positionRepo *repository.PositionRepository
	userConfigRepo *repository.UserConfigRepository
}

func NewService(positionRepo *repository.PositionRepository, userConfigRepo *repository.UserConfigRepository) *Service {
	return &Service{
		positionRepo: positionRepo,
		userConfigRepo: userConfigRepo,
	}
}

func (s *Service) GetAccountInfo(ctx context.Context, userID int64) (*AccountInfoResponse, error) {
	var config *repository.UserConfig
	userConfig, err := s.userConfigRepo.GetByUserID(ctx, userID)
	if err != nil || userConfig == nil {
		logger.Warn().Err(err).Int64("userID", userID).Msg("Failed to get user config, using defaults")
		config = &repository.UserConfig{
			InitialCapital: 100000000, // 100M VND default
		}
	} else {
		config = userConfig
	}

	positions, err := s.positionRepo.GetOpenPositions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	var positionsValue float64
	var lockedCash float64
	for _, pos := range positions {
		positionsValue += pos.EntryPrice * float64(pos.Quantity)
		lockedCash += pos.EntryPrice * float64(pos.Quantity)
	}

	// Calculate available cash (initial capital - locked in positions)
	cash := config.InitialCapital - positionsValue
	if cash < 0 {
		cash = 0
	}

	buyingPower := cash * 2.0

	marginUsed := positionsValue * 0.3 // 30% margin requirement
	marginAvailable := buyingPower - marginUsed

	capital := cash + positionsValue

	return &AccountInfoResponse{
		Capital:         capital,
		Cash:            cash,
		LockedCash:      lockedCash,
		PositionsValue:  positionsValue,
		BuyingPower:     buyingPower,
		MarginUsed:      marginUsed,
		MarginAvailable: marginAvailable,
	}, nil
}

func (s *Service) GetAccountSummary(ctx context.Context, userID int64) (*AccountSummaryResponse, error) {
	var config *repository.UserConfig
	userConfig, err := s.userConfigRepo.GetByUserID(ctx, userID)
	if err != nil || userConfig == nil {
		logger.Warn().Err(err).Int64("userID", userID).Msg("Failed to get user config")
		config = &repository.UserConfig{
			InitialCapital: 100000000, // 100M VND default  
		}
	} else {
		config = userConfig
	}

	positions, err := s.positionRepo.GetOpenPositions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	var totalPnL float64
	var totalRisk float64

	for _, pos := range positions {
		currentPrice := pos.EntryPrice * 1.025
		pnl := (currentPrice - pos.EntryPrice) * float64(pos.Quantity)
		totalPnL += pnl

		if pos.StopLoss > 0 {
			risk := (pos.EntryPrice - pos.StopLoss) * float64(pos.Quantity)
			totalRisk += risk
		}
	}

	// Use initial capital for P&L calculations
	totalPnLPercent := (totalPnL / config.InitialCapital) * 100
	riskPercent := (totalRisk / config.InitialCapital) * 100

	dayPnL := totalPnL * 0.1 // Assume 10% of total was made today
	dayPnLPercent := (dayPnL / config.InitialCapital) * 100

	return &AccountSummaryResponse{
		TotalPnL:        totalPnL,
		TotalPnLPercent: totalPnLPercent,
		DayPnL:          dayPnL,
		DayPnLPercent:   dayPnLPercent,
		RiskExposure:    totalRisk,
		RiskPercent:     riskPercent,
	}, nil
}
