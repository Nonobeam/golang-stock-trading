package ftd

import (
	"context"
	"fmt"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/api"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

// IngestionService handles fetching and storing market regime data.
type IngestionService struct {
	client *api.DNSEClient
	repo   *Repository
}

// NewIngestionService creates a new ingestion service.
func NewIngestionService(client *api.DNSEClient, repo *Repository) *IngestionService {
	return &IngestionService{
		client: client,
		repo:   repo,
	}
}

// SyncDailyData fetches VN-Index data for the given date (or date range) and updates the database.
// If 'to' date is not provided, it defaults to 'from' date (single day sync).
func (s *IngestionService) SyncDailyData(ctx context.Context, from time.Time, to *time.Time) error {
	endDate := from
	if to != nil {
		endDate = *to
	}

	logger.Info().
		Time("from", from).
		Time("to", endDate).
		Msg("Syncing VN-Index daily data")

	// 1. Fetch VN-Index history
	bars, err := s.client.GetVNIndexDaily(from, endDate)
	if err != nil {
		return fmt.Errorf("failed to fetch VN-Index data: %w", err)
	}

	logger.Info().Int("count", len(bars)).Msg("Fetched VN-Index bars")

	// 2. Process each bar
	for _, bar := range bars {
		// Calculate derived metrics if needed (e.g., 20-day avg volume)
		// For now, we just insert the raw data. 
		// The 20-day avg volume calculation requires looking back 20 days.
		// We'll implement a helper to calculate it on the fly or in a separate pass.
		
		regime := &MarketRegime{
			Date:       bar.Date,
			IndexValue: bar.Value,
			Volume:     bar.Volume,
			// Note: Daily history API only returns close value for index in standard message
			// If we need true OHLC, we might need a different API or accumulated intraday data.
			// For now, we leave O/H/L as nil unless we can derive them.
			// Actually, let's set them all to Value as fallback if needed, or keep nil.
			// Keeping nil means logic must handle nil.
			// RallyAttempt logic needs LOW.
			// If missing, use IndexValue (Close) as proxy.
			// Better: Populate Low/High/Open with IndexValue if missing to avoid nil checks everywhere
			Open: &bar.Value,
			High: &bar.Value,
			Low:  &bar.Value,
			// Other fields will be populated by the analysis logic later
		}

		// Calculate Volume vs Avg 20d
		// This requires querying DB for previous 20 entries
		// For efficiency, we might want to do this in batches or skip if backfilling large range
		avgVol, err := s.calculateAvgVolume(ctx, bar.Date, 20)
		if err == nil && avgVol > 0 {
			pct := (float64(bar.Volume) / float64(avgVol))
			regime.VolumeVsAvg20d = &pct
		} else if err != nil {
			logger.Warn().Err(err).Time("date", bar.Date).Msg("Failed to calculate avg volume")
		}

		if err := s.repo.UpsertMarketRegime(ctx, regime); err != nil {
			logger.Error().Err(err).Time("date", bar.Date).Msg("Failed to upsert market regime")
			// Continue to next bar rather than failing entire batch? 
			// For now let's return error to ensure data integrity
			return fmt.Errorf("failed to upsert regime for %s: %w", bar.Date.Format("2006-01-02"), err)
		}
	}

	return nil
}

// calculateAvgVolume calculates the average volume for the N trading days PRIOR to the given date.
func (s *IngestionService) calculateAvgVolume(ctx context.Context, date time.Time, periods int) (int64, error) {
	// Look back 40 days to be safe (to account for weekends/holidays)
	from := date.AddDate(0, 0, -40)
	
	// Get previous regimes
	prevRegimes, err := s.repo.GetMarketRegimes(ctx, from, date.AddDate(0, 0, -1))
	if err != nil {
		return 0, err
	}

	count := 0
	var totalVol int64
	
	// Iterate backwards from end
	for i := len(prevRegimes) - 1; i >= 0 && count < periods; i-- {
		totalVol += prevRegimes[i].Volume
		count++
	}

	if count == 0 {
		return 0, nil
	}

	return totalVol / int64(count), nil
}
