package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/utils"
)

// DailyBarsRepository handles daily OHLCV bar data persistence
type DailyBarsRepository struct {
	db *sql.DB
}

// NewDailyBarsRepository creates a new daily bars repository
func NewDailyBarsRepository(db *sql.DB) *DailyBarsRepository {
	return &DailyBarsRepository{db: db}
}

// UpsertDailyBar inserts or updates a daily bar record for a given symbol and date.
// It automatically calculates turnover from close price and volume.
//
// If a record with the same symbol and date already exists, it will be updated.
// Otherwise, a new record will be inserted.
func (r *DailyBarsRepository) UpsertDailyBar(
	ctx context.Context,
	symbol string,
	date time.Time,
	open, high, low, close float64,
	volume int64,
) error {
	// Calculate turnover using utility function
	turnover := utils.CalculateTurnover(close, float64(volume))

	query := `
		INSERT INTO "stock-trading".daily_bars (symbol, date, open, high, low, close, volume, turnover, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		ON CONFLICT (symbol, date)
		DO UPDATE SET
			open = EXCLUDED.open,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			close = EXCLUDED.close,
			volume = EXCLUDED.volume,
			turnover = EXCLUDED.turnover,
			updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, query, symbol, date, open, high, low, close, volume, turnover)
	return err
}
