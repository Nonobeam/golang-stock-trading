package repository

import (
	"context"
	"database/sql"
)

type StockUniverseRepository struct {
	db *sql.DB
}

func NewStockUniverseRepository(db *sql.DB) *StockUniverseRepository {
	return &StockUniverseRepository{db: db}
}

func (r *StockUniverseRepository) GetActiveSymbols(ctx context.Context) ([]string, error) {
	query := `
		SELECT ticker
		FROM "stock-trading".stock_universe
		WHERE is_active = TRUE
		ORDER BY ticker
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
