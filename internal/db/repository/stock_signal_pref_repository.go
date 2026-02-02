package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

// StockSignalPrefRepository handles stock signal preference CRUD operations
type StockSignalPrefRepository struct {
	db *sql.DB
}

// NewStockSignalPrefRepository creates a new stock signal preference repository
func NewStockSignalPrefRepository(db *sql.DB) *StockSignalPrefRepository {
	return &StockSignalPrefRepository{db: db}
}

// GetByUserAndSymbol gets preference for a specific stock for a user
func (r *StockSignalPrefRepository) GetByUserAndSymbol(ctx context.Context, userID int64, symbol string) (*StockSignalPreference, error) {
	query := `
		SELECT id, user_id, symbol, min_signal_score, notes, created_at, updated_at
		FROM stock_signal_preferences
		WHERE user_id = $1 AND symbol = $2
	`

	pref := &StockSignalPreference{}
	err := r.db.QueryRowContext(ctx, query, userID, symbol).Scan(
		&pref.ID, &pref.UserID, &pref.Symbol, &pref.MinSignalScore, &pref.Notes,
		&pref.CreatedAt, &pref.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No preference set for this stock
	}
	if err != nil {
		return nil, err
	}

	return pref, nil
}

// GetAllByUser gets all stock preferences for a user
func (r *StockSignalPrefRepository) GetAllByUser(ctx context.Context, userID int64) ([]*StockSignalPreference, error) {
	query := `
		SELECT id, user_id, symbol, min_signal_score, notes, created_at, updated_at
		FROM stock_signal_preferences
		WHERE user_id = $1
		ORDER BY symbol
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefs []*StockSignalPreference
	for rows.Next() {
		pref := &StockSignalPreference{}
		err := rows.Scan(
			&pref.ID, &pref.UserID, &pref.Symbol, &pref.MinSignalScore, &pref.Notes,
			&pref.CreatedAt, &pref.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		prefs = append(prefs, pref)
	}

	return prefs, rows.Err()
}

// Upsert creates or updates a stock signal preference
func (r *StockSignalPrefRepository) Upsert(ctx context.Context, pref *StockSignalPreference) error {
	query := `
		INSERT INTO stock_signal_preferences (user_id, symbol, min_signal_score, notes, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id, symbol) 
		DO UPDATE SET 
			min_signal_score = EXCLUDED.min_signal_score,
			notes = EXCLUDED.notes,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		pref.UserID, pref.Symbol, pref.MinSignalScore, pref.Notes,
	).Scan(&pref.ID, &pref.CreatedAt, &pref.UpdatedAt)

	return err
}

// Delete removes a stock signal preference (user falls back to default)
func (r *StockSignalPrefRepository) Delete(ctx context.Context, userID int64, symbol string) error {
	query := `
		DELETE FROM stock_signal_preferences
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
		return fmt.Errorf("preference not found for user %d, symbol %s", userID, symbol)
	}

	return nil
}

// BulkGet gets preferences for multiple stocks for a user
func (r *StockSignalPrefRepository) BulkGet(ctx context.Context, userID int64, symbols []string) (map[string]*StockSignalPreference, error) {
	if len(symbols) == 0 {
		return make(map[string]*StockSignalPreference), nil
	}

	query := `
		SELECT id, user_id, symbol, min_signal_score, notes, created_at, updated_at
		FROM stock_signal_preferences
		WHERE user_id = $1 AND symbol = ANY($2)
	`

	rows, err := r.db.QueryContext(ctx, query, userID, pq.Array(symbols))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prefs := make(map[string]*StockSignalPreference)
	for rows.Next() {
		pref := &StockSignalPreference{}
		err := rows.Scan(
			&pref.ID, &pref.UserID, &pref.Symbol, &pref.MinSignalScore, &pref.Notes,
			&pref.CreatedAt, &pref.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		prefs[pref.Symbol] = pref
	}

	return prefs, rows.Err()
}
