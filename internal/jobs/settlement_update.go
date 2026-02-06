// Package jobs contains scheduled job implementations
package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/internal/vn"
	"github.com/rs/zerolog"
)

// SettlementUpdater handles daily settlement status updates
type SettlementUpdater struct {
	db     *sql.DB
	logger *zerolog.Logger
}

// NewSettlementUpdater creates a new settlement updater
func NewSettlementUpdater(db *sql.DB) *SettlementUpdater {
	return &SettlementUpdater{
		db:     db,
		logger: logger.Get(),
	}
}

// SettlementUpdateResult contains the result of a daily settlement update
type SettlementUpdateResult struct {
	TotalPositions       int
	UpdatedPositions     int
	TransitionedToLiquid int
	StillLocked          int
	Errors               []string
	UpdatedAt            time.Time
}

// RunDailySettlementUpdate updates settlement status for all active positions
func (u *SettlementUpdater) RunDailySettlementUpdate(ctx context.Context) (*SettlementUpdateResult, error) {
	u.logger.Info().Msg("Starting daily settlement status update")

	result := &SettlementUpdateResult{
		UpdatedAt: time.Now(),
		Errors:    make([]string, 0),
	}

	posRepo := repository.NewPositionRepository(u.db)

	// Get all open positions (across all users)
	// Note: We need to query without userID filter
	rows, err := u.db.QueryContext(ctx, `
		SELECT id, user_id, symbol, entry_date, entry_price, quantity,
			stop_loss, target_1, target_2, target_3,
			signal_type, score, notes, is_closed,
			exit_date, exit_price, exit_reason,
			pnl, pnl_percent, r_multiple,
			created_at, updated_at,
			total_entries, total_fees_paid, first_entry_date, last_entry_date,
			settlement_status, purchase_date, settlement_date, can_sell_date,
			locked_capital, liquid_capital, exchange
		FROM positions
		WHERE is_closed = FALSE
			AND settlement_status IS NOT NULL
		ORDER BY purchase_date ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query open positions: %w", err)
	}
	defer rows.Close()

	positions := make([]*repository.Position, 0)
	for rows.Next() {
		pos := &repository.Position{}
		err := rows.Scan(
			&pos.ID, &pos.UserID, &pos.Symbol, &pos.EntryDate, &pos.EntryPrice, &pos.Quantity,
			&pos.StopLoss, &pos.Target1, &pos.Target2, &pos.Target3,
			&pos.SignalType, &pos.Score, &pos.Notes, &pos.IsClosed,
			&pos.ExitDate, &pos.ExitPrice, &pos.ExitReason,
			&pos.PnL, &pos.PnLPercent, &pos.RMultiple,
			&pos.CreatedAt, &pos.UpdatedAt,
			&pos.TotalEntries, &pos.TotalFeesPaid, &pos.FirstEntryDate, &pos.LastEntryDate,
			&pos.SettlementStatus, &pos.PurchaseDate, &pos.SettlementDate, &pos.CanSellDate,
			&pos.LockedCapital, &pos.LiquidCapital, &pos.Exchange,
		)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("scan error: %v", err))
			continue
		}
		positions = append(positions, pos)
	}

	result.TotalPositions = len(positions)
	currentDate := time.Now()

	u.logger.Info().Int("total_positions", result.TotalPositions).Msg("Processing positions")

	for _, pos := range positions {
		// Skip positions without purchase date
		if pos.PurchaseDate == nil {
			u.logger.Warn().Str("position_id", pos.ID).Msg("Position missing purchase_date, skipping")
			continue
		}

		// Calculate new settlement status
		newStatus := vn.CalculateSettlementStatusFromDates(*pos.PurchaseDate, currentDate)
		oldStatus := ""
		if pos.SettlementStatus != nil {
			oldStatus = *pos.SettlementStatus
		}

		// Only update if status changed
		if string(newStatus) != oldStatus {
			// Calculate locked vs liquid capital
			var lockedCapital, liquidCapital float64
			positionValue := float64(pos.Quantity) * pos.EntryPrice

			if newStatus.IsLocked() {
				lockedCapital = positionValue
				liquidCapital = 0
			} else {
				lockedCapital = 0
				liquidCapital = positionValue
			}

			// Update position settlement status
			err = posRepo.UpdateSettlementStatus(ctx, pos.ID, string(newStatus), lockedCapital, liquidCapital)
			if err != nil {
				errMsg := fmt.Sprintf("failed to update position %s: %v", pos.ID, err)
				result.Errors = append(result.Errors, errMsg)
				u.logger.Error().Err(err).Str("position_id", pos.ID).Msg("Update failed")
				continue
			}

			result.UpdatedPositions++

			// Track transitions to LIQUID
			if newStatus == vn.Liquid && oldStatus != string(vn.Liquid) {
				result.TransitionedToLiquid++
				u.logger.Info().
					Str("position_id", pos.ID).
					Str("symbol", pos.Symbol).
					Str("old_status", oldStatus).
					Str("new_status", string(newStatus)).
					Msg("Position transitioned to LIQUID")
			}

			// Record daily settlement tracking snapshot
			exchange := ""
			if pos.Exchange != nil {
				exchange = *pos.Exchange
			} else {
				exchange = vn.GetExchangeFromTicker(pos.Symbol)
			}

			daysUntilLiquid := 0
			if newStatus.IsLocked() {
				daysUntilLiquid = vn.GetDaysUntilLiquid(*pos.PurchaseDate, currentDate)
			}

			lockedRisk := 0.0
			if newStatus.IsLocked() {
				multiplier := vn.GetLockedRiskMultiplierForExchange(exchange)
				lockedRisk = lockedCapital * multiplier
			}

			// Determine risk classification
			riskClass := u.classifyRisk(newStatus, daysUntilLiquid)

			tracking := &repository.PositionSettlementTracking{
				PositionID:         pos.ID,
				CheckDate:          currentDate,
				SettlementStatus:   string(newStatus),
				DaysUntilLiquid:    daysUntilLiquid,
				LockedValue:        lockedCapital,
				LockedRisk:         lockedRisk,
				RiskClassification: riskClass,
			}

			err = posRepo.RecordSettlementTracking(ctx, tracking)
			if err != nil {
				u.logger.Warn().Err(err).Str("position_id", pos.ID).Msg("Failed to record tracking snapshot")
			}
		}

		if newStatus.IsLocked() {
			result.StillLocked++
		}
	}

	u.logger.Info().
		Int("total", result.TotalPositions).
		Int("updated", result.UpdatedPositions).
		Int("transitioned_to_liquid", result.TransitionedToLiquid).
		Int("still_locked", result.StillLocked).
		Int("errors", len(result.Errors)).
		Msg("Settlement update completed")

	return result, nil
}

// classifyRisk determines risk classification based on settlement status
func (u *SettlementUpdater) classifyRisk(status vn.SettlementStatus, daysUntilLiquid int) string {
	switch status {
	case vn.LockedT0, vn.LockedT1:
		return "HIGH_RISK_LOCKED"
	case vn.LockedT2:
		return "MODERATE_RISK_NEAR_LIQUID"
	case vn.Liquid:
		return "LOW_RISK_LIQUID"
	default:
		return "LOW_RISK_LIQUID"
	}
}

// GetLastUpdateTime retrieves the most recent settlement update timestamp
func (u *SettlementUpdater) GetLastUpdateTime(ctx context.Context) (*time.Time, error) {
	var lastUpdate time.Time
	err := u.db.QueryRowContext(ctx, `
		SELECT MAX(created_at)
		FROM position_settlement_tracking
	`).Scan(&lastUpdate)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &lastUpdate, nil
}

// ShouldRunUpdate checks if update should run (not run in last 24 hours)
func (u *SettlementUpdater) ShouldRunUpdate(ctx context.Context) (bool, error) {
	lastUpdate, err := u.GetLastUpdateTime(ctx)
	if err != nil {
		return false, err
	}

	if lastUpdate == nil {
		return true, nil // Never run before
	}

	// Run if last update was more than 23 hours ago
	timeSinceUpdate := time.Since(*lastUpdate)
	return timeSinceUpdate > 23*time.Hour, nil
}
