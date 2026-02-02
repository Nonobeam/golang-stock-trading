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

// GetOpenPositions returns all open positions for a user
func (r *PositionRepository) GetOpenPositions(ctx context.Context, userID int64) ([]*Position, error) {
	query := `
		SELECT id, user_id, symbol, entry_date, entry_price, quantity,
			stop_loss, target_1, target_2, target_3,
			signal_type, score, notes, is_closed,
			exit_date, exit_price, exit_reason,
			pnl, pnl_percent, r_multiple,
			created_at, updated_at
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
			created_at, updated_at
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
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return pos, nil
}

// Close marks a position as closed with exit details
func (r *PositionRepository) Close(ctx context.Context, id string, exitPrice float64, exitDate time.Time, reason string) error {
	query := `
		UPDATE positions
		SET is_closed = TRUE,
			exit_date = $2,
			exit_price = $3,
			exit_reason = $4,
			pnl = (exit_price - entry_price) * quantity,
			pnl_percent = ((exit_price - entry_price) / entry_price) * 100,
			r_multiple = (exit_price - entry_price) / (entry_price - stop_loss),
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
