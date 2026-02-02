package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// SignalHistoryRepository handles signal history CRUD operations
type SignalHistoryRepository struct {
	db *sql.DB
}

// NewSignalHistoryRepository creates a new signal history repository
func NewSignalHistoryRepository(db *sql.DB) *SignalHistoryRepository {
	return &SignalHistoryRepository{db: db}
}

// Create inserts a new signal
func (r *SignalHistoryRepository) Create(ctx context.Context, signal *SignalHistory) error {
	targetsJSON, err := json.Marshal(signal.Targets)
	if err != nil {
		return fmt.Errorf("failed to marshal targets: %w", err)
	}

	query := `
		INSERT INTO signals_history (
			symbol, signal_type, score, entry_price, stop_loss,
			targets, position_size, risk_amount, detected_at, regime
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`

	return r.db.QueryRowContext(ctx, query,
		signal.Symbol, signal.SignalType, signal.Score, signal.EntryPrice, signal.StopLoss,
		targetsJSON, signal.PositionSize, signal.RiskAmount, signal.DetectedAt, signal.Regime,
	).Scan(&signal.ID, &signal.CreatedAt)
}

// GetRecent returns recent signals with optional filters
func (r *SignalHistoryRepository) GetRecent(ctx context.Context, userID *int64, days int, minScore int) ([]*SignalHistory, error) {
	query := `
		SELECT id, symbol, signal_type, score, entry_price, stop_loss,
			targets, position_size, risk_amount, detected_at, regime,
			sent_to_user, user_action, user_id, created_at
		FROM signals_history
		WHERE detected_at >= NOW() - INTERVAL '%d days'
			AND score >= $1
			AND ($2::BIGINT IS NULL OR user_id = $2)
		ORDER BY detected_at DESC
		LIMIT 50
	`

	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(query, days), minScore, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var signals []*SignalHistory
	for rows.Next() {
		signal := &SignalHistory{}
		var targetsJSON []byte

		err := rows.Scan(
			&signal.ID, &signal.Symbol, &signal.SignalType, &signal.Score,
			&signal.EntryPrice, &signal.StopLoss, &targetsJSON,
			&signal.PositionSize, &signal.RiskAmount, &signal.DetectedAt, &signal.Regime,
			&signal.SentToUser, &signal.UserAction, &signal.UserID, &signal.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal JSONB targets
		if len(targetsJSON) > 0 {
			if err := json.Unmarshal(targetsJSON, &signal.Targets); err != nil {
				return nil, fmt.Errorf("failed to unmarshal targets: %w", err)
			}
		}

		signals = append(signals, signal)
	}

	return signals, rows.Err()
}

// MarkAsSent marks a signal as sent to user
func (r *SignalHistoryRepository) MarkAsSent(ctx context.Context, id string, userID int64) error {
	query := `
		UPDATE signals_history
		SET sent_to_user = TRUE, user_id = $2
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("signal not found: %s", id)
	}

	return nil
}

// RecordUserAction records user's action on a signal
func (r *SignalHistoryRepository) RecordUserAction(ctx context.Context, id string, action string) error {
	query := `
		UPDATE signals_history
		SET user_action = $2
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, action)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("signal not found: %s", id)
	}

	return nil
}

// GetBySymbol returns recent signals for a specific symbol
func (r *SignalHistoryRepository) GetBySymbol(ctx context.Context, symbol string, limit int) ([]*SignalHistory, error) {
	query := `
		SELECT id, symbol, signal_type, score, entry_price, stop_loss,
			targets, position_size, risk_amount, detected_at, regime,
			sent_to_user, user_action, user_id, created_at
		FROM signals_history
		WHERE symbol = $1
		ORDER BY detected_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, symbol, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var signals []*SignalHistory
	for rows.Next() {
		signal := &SignalHistory{}
		var targetsJSON []byte

		err := rows.Scan(
			&signal.ID, &signal.Symbol, &signal.SignalType, &signal.Score,
			&signal.EntryPrice, &signal.StopLoss, &targetsJSON,
			&signal.PositionSize, &signal.RiskAmount, &signal.DetectedAt, &signal.Regime,
			&signal.SentToUser, &signal.UserAction, &signal.UserID, &signal.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal JSONB targets
		if len(targetsJSON) > 0 {
			if err := json.Unmarshal(targetsJSON, &signal.Targets); err != nil {
				return nil, fmt.Errorf("failed to unmarshal targets: %w", err)
			}
		}

		signals = append(signals, signal)
	}

	return signals, rows.Err()
}
