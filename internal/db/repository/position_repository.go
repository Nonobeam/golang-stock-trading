package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// PositionRepository handles position CRUD operations
type PositionRepository struct {
	db *sql.DB
}

// NewPositionRepository creates a new position repository
func NewPositionRepository(db *sql.DB) *PositionRepository {
	return &PositionRepository{db: db}
}

// Create inserts a new position
func (r *PositionRepository) Create(ctx context.Context, pos *Position) error {
	query := `
		INSERT INTO positions (
			user_id, symbol, entry_date, entry_price, quantity,
			stop_loss, target_1, target_2, target_3,
			signal_type, score, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRowContext(ctx, query,
		pos.UserID, pos.Symbol, pos.EntryDate, pos.EntryPrice, pos.Quantity,
		pos.StopLoss, pos.Target1, pos.Target2, pos.Target3,
		pos.SignalType, pos.Score, pos.Notes,
	).Scan(&pos.ID, &pos.CreatedAt, &pos.UpdatedAt)
}

// CreateEntry inserts a new position entry
func (r *PositionRepository) CreateEntry(ctx context.Context, entry *PositionEntry) error {
	query := `
		INSERT INTO position_entries (
			user_id, ticker, entry_date, entry_price, shares_purchased,
			entry_fee_paid, transaction_type
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING entry_id, created_at
	`

	return r.db.QueryRowContext(ctx, query,
		entry.UserID, entry.Ticker, entry.EntryDate, entry.EntryPrice,
		entry.SharesPurchased, entry.EntryFeePaid, entry.TransactionType,
	).Scan(&entry.EntryID, &entry.CreatedAt)
}

// GetEntries returns all position entries for a stock
func (r *PositionRepository) GetEntries(ctx context.Context, userID int64, ticker string) ([]*PositionEntry, error) {
	query := `
		SELECT entry_id, user_id, ticker, entry_date, entry_price,
			shares_purchased, entry_fee_paid, transaction_type, created_at
		FROM position_entries
		WHERE user_id = $1 AND ticker = $2
		ORDER BY entry_date DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID, ticker)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*PositionEntry
	for rows.Next() {
		entry := &PositionEntry{}
		err := rows.Scan(
			&entry.EntryID, &entry.UserID, &entry.Ticker, &entry.EntryDate,
			&entry.EntryPrice, &entry.SharesPurchased, &entry.EntryFeePaid,
			&entry.TransactionType, &entry.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// GetOpenPositions returns all open positions for a user
func (r *PositionRepository) GetOpenPositions(ctx context.Context, userID int64) ([]*Position, error) {
	query := `
		SELECT id, user_id, symbol, entry_date, entry_price, quantity,
			stop_loss, target_1, target_2, target_3,
			signal_type, score, notes, is_closed,
			exit_date, exit_price, exit_reason,
			pnl, pnl_percent, r_multiple,
			created_at, updated_at,
			total_entries, total_fees_paid, first_entry_date, last_entry_date
		FROM positions
		WHERE user_id = $1 AND is_closed = FALSE
		ORDER BY entry_date DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []*Position
	for rows.Next() {
		pos := &Position{}
		err := rows.Scan(
			&pos.ID, &pos.UserID, &pos.Symbol, &pos.EntryDate, &pos.EntryPrice, &pos.Quantity,
			&pos.StopLoss, &pos.Target1, &pos.Target2, &pos.Target3,
			&pos.SignalType, &pos.Score, &pos.Notes, &pos.IsClosed,
			&pos.ExitDate, &pos.ExitPrice, &pos.ExitReason,
			&pos.PnL, &pos.PnLPercent, &pos.RMultiple,
			&pos.CreatedAt, &pos.UpdatedAt,
			&pos.TotalEntries, &pos.TotalFeesPaid, &pos.FirstEntryDate, &pos.LastEntryDate,
		)
		if err != nil {
			return nil, err
		}
		positions = append(positions, pos)
	}

	return positions, rows.Err()
}

// GetBySymbol returns an open position for a symbol
func (r *PositionRepository) GetBySymbol(ctx context.Context, userID int64, symbol string) (*Position, error) {
	query := `
		SELECT id, user_id, symbol, entry_date, entry_price, quantity,
			stop_loss, target_1, target_2, target_3,
			signal_type, score, notes, is_closed,
			exit_date, exit_price, exit_reason,
			pnl, pnl_percent, r_multiple,
			created_at, updated_at,
			total_entries, total_fees_paid, first_entry_date, last_entry_date
		FROM positions
		WHERE user_id = $1 AND symbol = $2 AND is_closed = FALSE
		LIMIT 1
	`

	pos := &Position{}
	err := r.db.QueryRowContext(ctx, query, userID, symbol).Scan(
		&pos.ID, &pos.UserID, &pos.Symbol, &pos.EntryDate, &pos.EntryPrice, &pos.Quantity,
		&pos.StopLoss, &pos.Target1, &pos.Target2, &pos.Target3,
		&pos.SignalType, &pos.Score, &pos.Notes, &pos.IsClosed,
		&pos.ExitDate, &pos.ExitPrice, &pos.ExitReason,
		&pos.PnL, &pos.PnLPercent, &pos.RMultiple,
		&pos.CreatedAt, &pos.UpdatedAt,
		&pos.TotalEntries, &pos.TotalFeesPaid, &pos.FirstEntryDate, &pos.LastEntryDate,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return pos, nil
}

// GetAllOpenBySymbol returns all open positions for a symbol
func (r *PositionRepository) GetAllOpenBySymbol(ctx context.Context, userID int64, symbol string) ([]*Position, error) {
	query := `
		SELECT id, user_id, symbol, entry_date, entry_price, quantity,
			stop_loss, target_1, target_2, target_3,
			signal_type, score, notes, is_closed,
			exit_date, exit_price, exit_reason,
			pnl, pnl_percent, r_multiple,
			created_at, updated_at,
			total_entries, total_fees_paid, first_entry_date, last_entry_date
		FROM positions
		WHERE user_id = $1 AND symbol = $2 AND is_closed = FALSE
		ORDER BY entry_date DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID, symbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []*Position
	for rows.Next() {
		pos := &Position{}
		err := rows.Scan(
			&pos.ID, &pos.UserID, &pos.Symbol, &pos.EntryDate, &pos.EntryPrice, &pos.Quantity,
			&pos.StopLoss, &pos.Target1, &pos.Target2, &pos.Target3,
			&pos.SignalType, &pos.Score, &pos.Notes, &pos.IsClosed,
			&pos.ExitDate, &pos.ExitPrice, &pos.ExitReason,
			&pos.PnL, &pos.PnLPercent, &pos.RMultiple,
			&pos.CreatedAt, &pos.UpdatedAt,
			&pos.TotalEntries, &pos.TotalFeesPaid, &pos.FirstEntryDate, &pos.LastEntryDate,
		)
		if err != nil {
			return nil, err
		}
		positions = append(positions, pos)
	}

	return positions, rows.Err()
}

// PartialExit updates position quantity and fees after a partial sell
func (r *PositionRepository) PartialExit(ctx context.Context, id string, newQuantity int, newTotalFees float64) error {
	query := `
		UPDATE positions
		SET quantity = $2,
			total_fees_paid = $3,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, newQuantity, newTotalFees)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("position not found: %s", id)
	}

	return nil
}

// Close marks a position as closed with exit details
func (r *PositionRepository) Close(ctx context.Context, id string, exitPrice float64, exitDate time.Time, reason string) error {
	// Note: entry_price in DB is now weighted average, so PnL calculation is correct for remaining quantity
	query := `
		UPDATE positions
		SET is_closed = TRUE,
			exit_date = $2,
			exit_price = $3,
			exit_reason = $4,
			pnl = (exit_price - entry_price) * quantity,
			pnl_percent = ((exit_price - entry_price) / entry_price) * 100,
			r_multiple = CASE WHEN (entry_price - stop_loss) != 0 THEN (exit_price - entry_price) / (entry_price - stop_loss) ELSE 0 END,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, exitDate, exitPrice, reason)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("position not found: %s", id)
	}

	return nil
}

// Update updates a position
func (r *PositionRepository) Update(ctx context.Context, pos *Position) error {
	query := `
		UPDATE positions
		SET stop_loss = $2,
			target_1 = $3,
			target_2 = $4,
			target_3 = $5,
			notes = $6,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		pos.ID, pos.StopLoss, pos.Target1, pos.Target2, pos.Target3, pos.Notes,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("position not found: %s", pos.ID)
	}

	return nil
}

// GetClosedPositions returns closed positions for a user with optional limit
func (r *PositionRepository) GetClosedPositions(ctx context.Context, userID int64, limit int) ([]*Position, error) {
	query := `
		SELECT id, user_id, symbol, entry_date, entry_price, quantity,
			stop_loss, target_1, target_2, target_3,
			signal_type, score, notes, is_closed,
			exit_date, exit_price, exit_reason,
			pnl, pnl_percent, r_multiple,
			created_at, updated_at
		FROM positions
		WHERE user_id = $1 AND is_closed = TRUE
		ORDER BY exit_date DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []*Position
	for rows.Next() {
		pos := &Position{}
		err := rows.Scan(
			&pos.ID, &pos.UserID, &pos.Symbol, &pos.EntryDate, &pos.EntryPrice, &pos.Quantity,
			&pos.StopLoss, &pos.Target1, &pos.Target2, &pos.Target3,
			&pos.SignalType, &pos.Score, &pos.Notes, &pos.IsClosed,
			&pos.ExitDate, &pos.ExitPrice, &pos.ExitReason,
			&pos.PnL, &pos.PnLPercent, &pos.RMultiple,
			&pos.CreatedAt, &pos.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		positions = append(positions, pos)
	}

	return positions, rows.Err()
}

// CreatePositionWithSettlement inserts a new position with settlement tracking fields
func (r *PositionRepository) CreatePositionWithSettlement(ctx context.Context, pos *Position) error {
	query := `
		INSERT INTO positions (
			user_id, symbol, entry_date, entry_price, quantity,
			stop_loss, target_1, target_2, target_3,
			signal_type, score, notes,
			settlement_status, purchase_date, settlement_date, can_sell_date,
			locked_capital, liquid_capital, exchange
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRowContext(ctx, query,
		pos.UserID, pos.Symbol, pos.EntryDate, pos.EntryPrice, pos.Quantity,
		pos.StopLoss, pos.Target1, pos.Target2, pos.Target3,
		pos.SignalType, pos.Score, pos.Notes,
		pos.SettlementStatus, pos.PurchaseDate, pos.SettlementDate, pos.CanSellDate,
		pos.LockedCapital, pos.LiquidCapital, pos.Exchange,
	).Scan(&pos.ID, &pos.CreatedAt, &pos.UpdatedAt)
}

// UpdateSettlementStatus updates the settlement status and related fields for a position
func (r *PositionRepository) UpdateSettlementStatus(ctx context.Context, id string, status string, lockedCapital, liquidCapital float64) error {
	query := `
		UPDATE positions
		SET settlement_status = $2,
			locked_capital = $3,
			liquid_capital = $4,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, status, lockedCapital, liquidCapital)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("position not found: %s", id)
	}

	return nil
}

// GetPositionsBySettlementStatus returns positions with a specific settlement status
func (r *PositionRepository) GetPositionsBySettlementStatus(ctx context.Context, userID int64, status string) ([]*Position, error) {
	query := `
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
		WHERE user_id = $1 AND settlement_status = $2 AND is_closed = FALSE
		ORDER BY entry_date DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []*Position
	for rows.Next() {
		pos := &Position{}
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
			return nil, err
		}
		positions = append(positions, pos)
	}

	return positions, rows.Err()
}

// GetLockedPositions returns all positions with locked settlement status (T0, T1, T2)
func (r *PositionRepository) GetLockedPositions(ctx context.Context, userID int64) ([]*Position, error) {
	query := `
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
		WHERE user_id = $1
			AND settlement_status IN ('LOCKED_T0', 'LOCKED_T1', 'LOCKED_T2')
			AND is_closed = FALSE
		ORDER BY purchase_date DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []*Position
	for rows.Next() {
		pos := &Position{}
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
			return nil, err
		}
		positions = append(positions, pos)
	}

	return positions, rows.Err()
}

// GetTotalLockedRisk calculates the total locked risk for a user across all locked positions
func (r *PositionRepository) GetTotalLockedRisk(ctx context.Context, userID int64) (float64, error) {
	query := `
		SELECT COALESCE(
			SUM(
				CASE
					WHEN exchange = 'HOSE' THEN locked_capital * 0.20
					WHEN exchange = 'HNX' THEN locked_capital * 0.30
					WHEN exchange = 'UPCOM' THEN locked_capital * 0.40
					ELSE locked_capital * 0.20
				END
			),
			0
		)
		FROM positions
		WHERE user_id = $1
			AND settlement_status IN ('LOCKED_T0', 'LOCKED_T1', 'LOCKED_T2')
			AND is_closed = FALSE
	`

	var totalLockedRisk float64
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&totalLockedRisk)
	if err != nil {
		return 0, err
	}

	return totalLockedRisk, nil
}

// RecordSettlementTracking inserts a daily settlement status snapshot
func (r *PositionRepository) RecordSettlementTracking(ctx context.Context, tracking *PositionSettlementTracking) error {
	query := `
		INSERT INTO position_settlement_tracking (
			position_id, check_date, settlement_status, days_until_liquid,
			locked_value, locked_risk, risk_classification
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (position_id, check_date) DO UPDATE SET
			settlement_status = EXCLUDED.settlement_status,
			days_until_liquid = EXCLUDED.days_until_liquid,
			locked_value = EXCLUDED.locked_value,
			locked_risk = EXCLUDED.locked_risk,
			risk_classification = EXCLUDED.risk_classification
		RETURNING tracking_id, created_at
	`

	return r.db.QueryRowContext(ctx, query,
		tracking.PositionID, tracking.CheckDate, tracking.SettlementStatus,
		tracking.DaysUntilLiquid, tracking.LockedValue, tracking.LockedRisk,
		tracking.RiskClassification,
	).Scan(&tracking.TrackingID, &tracking.CreatedAt)
}

// RecordTheoreticalStopBreach records a stop loss breach that could not be executed
func (r *PositionRepository) RecordTheoreticalStopBreach(ctx context.Context, breach *TheoreticalStopBreach) error {
	query := `
		INSERT INTO theoretical_stop_breaches (
			position_id, breach_date, stop_price, actual_price,
			settlement_status, days_until_executable
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING breach_id, created_at
	`

	return r.db.QueryRowContext(ctx, query,
		breach.PositionID, breach.BreachDate, breach.StopPrice, breach.ActualPrice,
		breach.SettlementStatus, breach.DaysUntilExecutable,
	).Scan(&breach.BreachID, &breach.CreatedAt)
}
