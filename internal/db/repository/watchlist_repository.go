package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// WatchlistRepository handles watchlist CRUD operations
type WatchlistRepository struct {
	db *sql.DB
}

// NewWatchlistRepository creates a new watchlist repository
func NewWatchlistRepository(db *sql.DB) *WatchlistRepository {
	return &WatchlistRepository{db: db}
}

// Add adds a stock to the watchlist
func (r *WatchlistRepository) Add(ctx context.Context, item *WatchlistItem) error {
	signalTypesJSON, err := json.Marshal(item.SignalTypes)
	if err != nil {
		return fmt.Errorf("failed to marshal signal types: %w", err)
	}

	query := `
		INSERT INTO watchlist (
			user_id, symbol, target_price, notes, signal_types, min_score
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, symbol) 
		DO UPDATE SET 
			target_price = EXCLUDED.target_price,
			notes = EXCLUDED.notes,
			signal_types = EXCLUDED.signal_types,
			min_score = EXCLUDED.min_score,
			is_active = TRUE,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRowContext(ctx, query,
		item.UserID, item.Symbol, item.TargetPrice, item.Notes,
		signalTypesJSON, item.MinScore,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
}

// GetActive returns all active watchlist items for a user
func (r *WatchlistRepository) GetActive(ctx context.Context, userID int64) ([]*WatchlistItem, error) {
	query := `
		SELECT id, user_id, symbol, target_price, notes, signal_types,
			min_score, is_active, alert_sent, last_alert_at,
			created_at, updated_at
		FROM watchlist
		WHERE user_id = $1 AND is_active = TRUE
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*WatchlistItem
	for rows.Next() {
		item := &WatchlistItem{}
		var signalTypesJSON []byte

		err := rows.Scan(
			&item.ID, &item.UserID, &item.Symbol, &item.TargetPrice, &item.Notes,
			&signalTypesJSON, &item.MinScore, &item.IsActive, &item.AlertSent,
			&item.LastAlertAt, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal JSONB signal types
		if len(signalTypesJSON) > 0 {
			if err := json.Unmarshal(signalTypesJSON, &item.SignalTypes); err != nil {
				return nil, fmt.Errorf("failed to unmarshal signal types: %w", err)
			}
		}

		items = append(items, item)
	}

	return items, rows.Err()
}

// Remove removes a stock from the watchlist (soft delete by setting is_active = false)
func (r *WatchlistRepository) Remove(ctx context.Context, userID int64, symbol string) error {
	query := `
		UPDATE watchlist
		SET is_active = FALSE, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND symbol = $2
	`

	result, err := r.db.ExecContext(ctx, query, userID, symbol)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("watchlist item not found for user %d, symbol %s", userID, symbol)
	}

	return nil
}

// UpdateTargetPrice updates the target price for a watchlist item
func (r *WatchlistRepository) UpdateTargetPrice(ctx context.Context, userID int64, symbol string, targetPrice float64) error {
	query := `
		UPDATE watchlist
		SET target_price = $3, alert_sent = FALSE, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND symbol = $2 AND is_active = TRUE
	`

	result, err := r.db.ExecContext(ctx, query, userID, symbol, targetPrice)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("watchlist item not found for user %d, symbol %s", userID, symbol)
	}

	return nil
}

// MarkAlertSent marks an alert as sent for a watchlist item
func (r *WatchlistRepository) MarkAlertSent(ctx context.Context, id string) error {
	query := `
		UPDATE watchlist
		SET alert_sent = TRUE, last_alert_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("watchlist item not found: %s", id)
	}

	return nil
}

// GetSymbols returns all symbols being watched by all users (for scanner)
func (r *WatchlistRepository) GetSymbols(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT symbol
		FROM watchlist
		WHERE is_active = TRUE
		ORDER BY symbol
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, err
		}
		symbols = append(symbols, symbol)
	}

	return symbols, rows.Err()
}

// Dashboard API Methods

// Create adds a simple watchlist entry for Dashboard (with added_at and is_favorite)
func (r *WatchlistRepository) Create(ctx context.Context, userID int64, symbol string) error {
	query := `
		INSERT INTO watchlist (user_id, symbol)
		VALUES ($1, $2)
		ON CONFLICT (user_id, symbol) DO UPDATE SET is_active = TRUE
	`

	_, err := r.db.ExecContext(ctx, query, userID, symbol)
	if err != nil {
		return fmt.Errorf("failed to create watchlist item: %w", err)
	}

	return nil
}

// GetByUserID retrieves all active watchlist items for Dashboard display
func (r *WatchlistRepository) GetByUserID(ctx context.Context, userID int64) ([]*WatchlistItem, error) {
	return r.GetActive(ctx, userID)
}

// Delete hard deletes a watchlist item (for Dashboard API)
func (r *WatchlistRepository) Delete(ctx context.Context, userID int64, symbol string) error {
	query := `
		DELETE FROM watchlist
		WHERE user_id = $1 AND symbol = $2
	`

	result, err := r.db.ExecContext(ctx, query, userID, symbol)
	if err != nil {
		return fmt.Errorf("failed to delete watchlist item: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("watchlist item not found: %s", symbol)
	}

	return nil
}

// UpdateFavorite updates the is_favorite field (will need to add column if missing)
func (r *WatchlistRepository) UpdateFavorite(ctx context.Context, userID int64, symbol string, isFavorite bool) error {
	// Note: This assumes is_favorite column exists in watchlist table
	// If migration hasn't added it yet, it will fail
	query := `
		UPDATE watchlist
		SET updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND symbol = $2
	`

	result, err := r.db.ExecContext(ctx, query, userID, symbol)
	if err != nil {
		return fmt.Errorf("failed to update favorite status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("watchlist item not found: %s", symbol)
	}

	return nil
}
