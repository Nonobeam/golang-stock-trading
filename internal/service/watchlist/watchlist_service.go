package watchlist

import (
	"context"
	"fmt"

	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

// WatchlistItemResponse represents a watchlist item with live quote data
type WatchlistItemResponse struct {
	Symbol         string    `json:"symbol"`
	AddedAt        int64     `json:"addedAt"`
	IsFavorite     bool      `json:"isFavorite"`
	Price          float64   `json:"price"`
	Change         float64   `json:"change"`
	ChangePercent  float64   `json:"changePercent"`
	SparklineData  []float64 `json:"sparklineData"`
}

// Service handles watchlist business logic
type Service struct {
	repo *repository.WatchlistRepository
	// TODO: Add quote service for real-time prices
}

// NewService creates a new watchlist service
func NewService(repo *repository.WatchlistRepository) *Service {
	return &Service{repo: repo}
}

// GetWatchlist retrieves user's watchlist with quote enrichment
func (s *Service) GetWatchlist(ctx context.Context, userID int64) ([]WatchlistItemResponse, error) {
	items, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get watchlist: %w", err)
	}

	responses := make([]WatchlistItemResponse, 0, len(items))
	for _, item := range items {
		// Enrich with quote data
		response := s.enrichWithQuote(item)
		responses = append(responses, *response)
	}

	return responses, nil
}

// AddToWatchlist adds a symbol to the user's watchlist
func (s *Service) AddToWatchlist(ctx context.Context, userID int64, symbol string) error {
	err := s.repo.Create(ctx, userID, symbol)
	if err != nil {
		return fmt.Errorf("failed to add to watchlist: %w", err)
	}

	logger.Info().Int64("userID", userID).Str("symbol", symbol).Msg("Added to watchlist")
	return nil
}

// RemoveFromWatchlist removes a symbol from the user's watchlist
func (s *Service) RemoveFromWatchlist(ctx context.Context, userID int64, symbol string) error {
	err := s.repo.Delete(ctx, userID, symbol)
	if err != nil {
		return fmt.Errorf("failed to remove from watchlist: %w", err)
	}

	logger.Info().Int64("userID", userID).Str("symbol", symbol).Msg("Removed from watchlist")
	return nil
}

// ToggleFavorite toggles the favorite status of a watchlist item
func (s *Service) ToggleFavorite(ctx context.Context, userID int64, symbol string, isFavorite bool) error {
	err := s.repo.UpdateFavorite(ctx, userID, symbol, isFavorite)
	if err != nil {
		return fmt.Errorf("failed to update favorite: %w", err)
	}

	logger.Info().Int64("userID", userID).Str("symbol", symbol).Bool("isFavorite", isFavorite).Msg("Updated favorite status")
	return nil
}

// enrichWithQuote fetches live quote data and sparkline
func (s *Service) enrichWithQuote(item *repository.WatchlistItem) *WatchlistItemResponse {
	// TODO: Fetch real quote from DNSE API or market service
	// For now, use demo data
	
	// Demo price data
	basePrice := 62800.0
	price := basePrice * (1.0 + (float64(len(item.Symbol)) * 0.01)) // Vary by symbol length
	change := price * 0.0125
	changePercent := 1.25

	// Generate sparkline (last 5-10 data points for mini chart)
	sparkline := s.generateSparkline(price, 7)

	return &WatchlistItemResponse{
		Symbol:        item.Symbol,
		AddedAt:       item.CreatedAt.UnixMilli(),
		IsFavorite:    item.IsActive, // Using is_active as proxy for is_favorite
		Price:         price,
		Change:        change,
		ChangePercent: changePercent,
		SparklineData: sparkline,
	}
}

// generateSparkline creates demo sparkline data
func (s *Service) generateSparkline(currentPrice float64, points int) []float64 {
	data := make([]float64, points)
	
	// Generate upward trend ending at current price
	for i := 0; i < points; i++ {
		// Start 3% below current, trend up
		progress := float64(i) / float64(points-1)
		data[i] = currentPrice * (0.97 + (0.03 * progress))
	}
	
	return data
}
