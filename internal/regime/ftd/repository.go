package ftd

import (
	"context"
	"database/sql"
	"time"
)

// Repository handles database operations for market regime tracking.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new FTD repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// UpsertMarketRegime inserts or updates a market regime record.
func (r *Repository) UpsertMarketRegime(ctx context.Context, m *MarketRegime) error {
	query := `
		INSERT INTO market_regime_tracking (
			date, index_value, open, high, low, volume, volume_vs_avg_20d, rally_attempt_day, rally_attempt_baseline,
			is_ftd, ftd_strength, ftd_score, breadth_ratio, leader_participation_score, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
		ON CONFLICT (date) DO UPDATE SET
			index_value = EXCLUDED.index_value,
			open = EXCLUDED.open,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			volume = EXCLUDED.volume,
			volume_vs_avg_20d = EXCLUDED.volume_vs_avg_20d,
			rally_attempt_day = EXCLUDED.rally_attempt_day,
			rally_attempt_baseline = EXCLUDED.rally_attempt_baseline,
			is_ftd = EXCLUDED.is_ftd,
			ftd_strength = EXCLUDED.ftd_strength,
			ftd_score = EXCLUDED.ftd_score,
			breadth_ratio = EXCLUDED.breadth_ratio,
			leader_participation_score = EXCLUDED.leader_participation_score,
			updated_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, query,
		m.Date, m.IndexValue, m.Open, m.High, m.Low, m.Volume, m.VolumeVsAvg20d, m.RallyAttemptDay, m.RallyAttemptBaseline,
		m.IsFTD, m.FTDStrength, m.FTDScore, m.BreadthRatio, m.LeaderParticipationScore,
	)
	return err
}

// GetMarketRegime returns the market regime record for a specific date.
func (r *Repository) GetMarketRegime(ctx context.Context, date time.Time) (*MarketRegime, error) {
	query := `
		SELECT id, date, index_value, open, high, low, volume, volume_vs_avg_20d, rally_attempt_day, rally_attempt_baseline,
		       is_ftd, ftd_strength, ftd_score, breadth_ratio, leader_participation_score, created_at, updated_at
		FROM market_regime_tracking
		WHERE date = $1
	`
	row := r.db.QueryRowContext(ctx, query, date)
	return scanMarketRegime(row)
}

// GetLatestMarketRegime returns the most recent market regime record.
func (r *Repository) GetLatestMarketRegime(ctx context.Context) (*MarketRegime, error) {
	query := `
		SELECT id, date, index_value, open, high, low, volume, volume_vs_avg_20d, rally_attempt_day, rally_attempt_baseline,
		       is_ftd, ftd_strength, ftd_score, breadth_ratio, leader_participation_score, created_at, updated_at
		FROM market_regime_tracking
		ORDER BY date DESC
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query)
	return scanMarketRegime(row)
}

// GetMarketRegimes returns market regime records for a date range.
func (r *Repository) GetMarketRegimes(ctx context.Context, from, to time.Time) ([]*MarketRegime, error) {
	query := `
		SELECT id, date, index_value, open, high, low, volume, volume_vs_avg_20d, rally_attempt_day, rally_attempt_baseline,
		       is_ftd, ftd_strength, ftd_score, breadth_ratio, leader_participation_score, created_at, updated_at
		FROM market_regime_tracking
		WHERE date >= $1 AND date <= $2
		ORDER BY date ASC
	`
	rows, err := r.db.QueryContext(ctx, query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regimes []*MarketRegime
	for rows.Next() {
		m, err := scanMarketRegime(rows)
		if err != nil {
			return nil, err
		}
		regimes = append(regimes, m)
	}
	return regimes, nil
}

// Helper to scan row into MarketRegime struct
func scanMarketRegime(scanner interface {
	Scan(dest ...interface{}) error
}) (*MarketRegime, error) {
	m := &MarketRegime{}
	var rallyDay sql.NullInt32
	var rallyBaseline sql.NullFloat64
	var ftdStrength sql.NullString
	var ftdScore sql.NullInt32
	var breadthRatio sql.NullFloat64
	var volVsAvg sql.NullFloat64
	
	err := scanner.Scan(
		&m.ID, &m.Date, &m.IndexValue, &m.Open, &m.High, &m.Low, &m.Volume, &volVsAvg, &rallyDay, &rallyBaseline,
		&m.IsFTD, &ftdStrength, &ftdScore, &breadthRatio, &m.LeaderParticipationScore,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if rallyDay.Valid {
		day := int(rallyDay.Int32)
		m.RallyAttemptDay = &day
	}
	if rallyBaseline.Valid {
		m.RallyAttemptBaseline = &rallyBaseline.Float64
	}
	if ftdStrength.Valid {
		m.FTDStrength = ftdStrength.String
	}
	if ftdScore.Valid {
		m.FTDScore = int(ftdScore.Int32)
	}
	if breadthRatio.Valid {
		m.BreadthRatio = &breadthRatio.Float64
	}
	if volVsAvg.Valid {
		m.VolumeVsAvg20d = &volVsAvg.Float64
	}
	
	return m, nil
}
