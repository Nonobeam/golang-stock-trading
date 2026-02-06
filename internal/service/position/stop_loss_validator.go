package position

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
	"github.com/nonobeam/golang-stock-trading/internal/vn"
)

// StopLossValidator validates stop loss execution against settlement status
type StopLossValidator struct {
	db      *sql.DB
	posRepo *repository.PositionRepository
}

// NewStopLossValidator creates a new stop loss validator
func NewStopLossValidator(db *sql.DB) *StopLossValidator {
	return &StopLossValidator{
		db:      db,
		posRepo: repository.NewPositionRepository(db),
	}
}

// StopLossExecutability represents whether a stop loss can be executed
type StopLossExecutability struct {
	CanExecute          bool
	SettlementStatus    string
	Reason              string
	DaysUntilExecutable int
	CanSellDate         *time.Time
}

// CanExecuteStopLoss checks if a stop loss can be executed for a position
func (v *StopLossValidator) CanExecuteStopLoss(ctx context.Context, position *repository.Position) (*StopLossExecutability, error) {
	result := &StopLossExecutability{
		CanExecute: true,
	}

	// If no settlement status, assume backward compatibility (liquid)
	if position.SettlementStatus == nil {
		return result, nil
	}

	status := *position.SettlementStatus
	result.SettlementStatus = status

	// Check if position is locked
	if position.IsLocked() {
		result.CanExecute = false

		// Calculate days until executable
		if position.PurchaseDate != nil {
			result.DaysUntilExecutable = vn.GetDaysUntilLiquid(*position.PurchaseDate, time.Now())
		}

		if position.CanSellDate != nil {
			result.CanSellDate = position.CanSellDate
			result.Reason = fmt.Sprintf(
				"Shares in settlement (%s), cannot execute stop loss until %s (%d trading days)",
				status,
				position.CanSellDate.Format("2006-01-02"),
				result.DaysUntilExecutable,
			)
		} else {
			result.Reason = fmt.Sprintf(
				"Shares in settlement (%s), cannot execute stop loss",
				status,
			)
		}

		return result, nil
	}

	// Position is liquid, can execute
	return result, nil
}

// RecordTheoreticalBreach records a stop loss breach that could not be executed
func (v *StopLossValidator) RecordTheoreticalBreach(
	ctx context.Context,
	positionID string,
	stopPrice float64,
	actualPrice float64,
	settlementStatus string,
	daysUntilExecutable int,
) error {

	breach := &repository.TheoreticalStopBreach{
		PositionID:          positionID,
		BreachDate:          time.Now(),
		StopPrice:           stopPrice,
		ActualPrice:         actualPrice,
		SettlementStatus:    settlementStatus,
		DaysUntilExecutable: daysUntilExecutable,
	}

	return v.posRepo.RecordTheoreticalStopBreach(ctx, breach)
}

// GetExecutableStops returns all positions where stop loss can be executed
func (v *StopLossValidator) GetExecutableStops(ctx context.Context, userID int64, currentPrices map[string]float64) ([]*repository.Position, error) {
	// Get all open positions
	positions, err := v.posRepo.GetOpenPositions(ctx, userID)
	if err != nil {
		return nil, err
	}

	executable := make([]*repository.Position, 0)

	for _, pos := range positions {
		// Check if current price is at or below stop loss
		currentPrice, exists := currentPrices[pos.Symbol]
		if !exists || currentPrice > pos.StopLoss {
			continue
		}

		// Check if position is liquid (can execute)
		execResult, err := v.CanExecuteStopLoss(ctx, pos)
		if err != nil {
			return nil, err
		}

		if execResult.CanExecute {
			executable = append(executable, pos)
		}
	}

	return executable, nil
}

// GetTheoreticalStops returns positions where stop was breached but not executable
func (v *StopLossValidator) GetTheoreticalStops(ctx context.Context, userID int64, currentPrices map[string]float64) ([]*repository.Position, error) {
	// Get all open positions
	positions, err := v.posRepo.GetOpenPositions(ctx, userID)
	if err != nil {
		return nil, err
	}

	theoretical := make([]*repository.Position, 0)

	for _, pos := range positions {
		// Check if current price is at or below stop loss
		currentPrice, exists := currentPrices[pos.Symbol]
		if !exists || currentPrice > pos.StopLoss {
			continue
		}

		// Check if position is locked (cannot execute)
		execResult, err := v.CanExecuteStopLoss(ctx, pos)
		if err != nil {
			return nil, err
		}

		if !execResult.CanExecute {
			theoretical = append(theoretical, pos)
		}
	}

	return theoretical, nil
}

// ValidateAndRecordBreach validates stop loss breach and records if not executable
func (v *StopLossValidator) ValidateAndRecordBreach(
	ctx context.Context,
	position *repository.Position,
	currentPrice float64,
) (*StopLossExecutability, error) {

	// Check executability
	result, err := v.CanExecuteStopLoss(ctx, position)
	if err != nil {
		return nil, err
	}

	// If not executable, record as theoretical breach
	if !result.CanExecute {
		err = v.RecordTheoreticalBreach(
			ctx,
			position.ID,
			position.StopLoss,
			currentPrice,
			result.SettlementStatus,
			result.DaysUntilExecutable,
		)
		if err != nil {
			return result, fmt.Errorf("failed to record theoretical breach: %w", err)
		}
	}

	return result, nil
}

// GetStopLossStatus returns comprehensive stop loss status for a position
func (v *StopLossValidator) GetStopLossStatus(
	ctx context.Context,
	position *repository.Position,
	currentPrice float64,
) (map[string]interface{}, error) {

	execResult, err := v.CanExecuteStopLoss(ctx, position)
	if err != nil {
		return nil, err
	}

	status := map[string]interface{}{
		"position_id":        position.ID,
		"symbol":             position.Symbol,
		"stop_loss_price":    position.StopLoss,
		"current_price":      currentPrice,
		"can_execute":        execResult.CanExecute,
		"settlement_status":  execResult.SettlementStatus,
		"stop_breached":      currentPrice <= position.StopLoss,
	}

	if !execResult.CanExecute {
		status["reason"] = execResult.Reason
		status["days_until_executable"] = execResult.DaysUntilExecutable
		if execResult.CanSellDate != nil {
			status["can_sell_date"] = execResult.CanSellDate.Format("2006-01-02")
		}
	}

	return status, nil
}
