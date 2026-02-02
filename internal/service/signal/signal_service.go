package signal

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
)

// SignalResponse represents a signal for Dashboard API
type SignalResponse struct {
	ID           string   `json:"id"`
	Symbol       string   `json:"symbol"`
	Name         string   `json:"name"`
	Exchange     string   `json:"exchange"`
	CurrentPrice float64  `json:"currentPrice"`
	SignalType   string   `json:"signalType"` // buy, sell, watch
	Strength     string   `json:"strength"` // weak, moderate, strong
	Score        int      `json:"score"`
	Indicators   []string `json:"indicators"`
	GeneratedAt  string   `json:"generatedAt"`
	ExpiresAt    string   `json:"expiresAt"`
	Reason       string   `json:"reason"`
}

// QueryParams for filtering signals
type QueryParams struct {
	Limit    int
	Sort     string // score, generatedAt
	Type     string // buy, sell, watch
	Strength string // weak, moderate, strong
}

// Service handles signal-related business logic
type Service struct {
	repo *repository.SignalHistoryRepository
}

// NewService creates a new signal service
func NewService(repo *repository.SignalHistoryRepository) *Service {
	return &Service{repo: repo}
}

// GetSignals retrieves signals with filtering and sorting
func (s *Service) GetSignals(ctx context.Context, userID int64, params QueryParams) ([]SignalResponse, int, error) {
	// Get signals from repository (last 7 days, score >= 6)
	minScore := 6
	signals, err := s.repo.GetRecent(ctx, &userID, 7, minScore)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get signals: %w", err)
	}

	// Transform to API responses
	responses := make([]SignalResponse, 0, len(signals))
	for _, sig := range signals {
		response := s.transformSignal(sig)
		
		// Apply type filter
		if params.Type != "" && response.SignalType != params.Type {
			continue
		}
		
		// Apply strength filter
		if params.Strength != "" && response.Strength != params.Strength {
			continue
		}
		
		responses = append(responses, *response)
	}

	// Sort
	s.sortSignals(responses, params.Sort)

	// Apply limit
	if params.Limit > 0 && params.Limit < len(responses) {
		responses = responses[:params.Limit]
	}

	return responses, len(responses), nil
}

// transformSignal converts DB signal to API response
func (s *Service) transformSignal(sig *repository.SignalHistory) *SignalResponse {
	// Determine signal type from signal_type field
	signalType := "buy"
	if sig.SignalType != "" {
		signalType = strings.ToLower(sig.SignalType)
		if strings.Contains(signalType, "buy") || strings.Contains(signalType, "pullback") || strings.Contains(signalType, "breakout") {
			signalType = "buy"
		} else if strings.Contains(signalType, "sell") {
			signalType = "sell"
		} else {
			signalType = "watch"
		}
	}

	// Determine strength from score
	strength := "moderate"
	if sig.Score >= 8 {
		strength = "strong"
	} else if sig.Score < 7 {
		strength = "weak"
	}

	// Generate indicators list
	indicators := []string{}
	if sig.Score >= 8 {
		indicators = append(indicators, "Strong Score")
	}
	if sig.Regime != nil && *sig.Regime != "" {
		indicators = append(indicators, "Regime: "+*sig.Regime)
	}
	indicators = append(indicators, "Entry: "+fmt.Sprintf("%.0f", sig.EntryPrice))

	// Generate reason/rationale
	reason := fmt.Sprintf("%s signal with score %d", signalType, sig.Score)
	if sig.Regime != nil {
		reason += fmt.Sprintf(" in %s regime", *sig.Regime)
	}

	// Calculate expiry (24 hours from generated)
	expiresAt := sig.DetectedAt.Add(24 * 60 * 60 * 1000000000) // 24 hours in nanoseconds

	return &SignalResponse{
		ID:           sig.ID,
		Symbol:       sig.Symbol,
		Name:         s.getStockName(sig.Symbol),
		Exchange:     "HOSE",
		CurrentPrice: sig.EntryPrice,
		SignalType:   signalType,
		Strength:     strength,
		Score:        sig.Score,
		Indicators:   indicators,
		GeneratedAt:  sig.DetectedAt.Format("2006-01-02T15:04:05.000Z"),
		ExpiresAt:    expiresAt.Format("2006-01-02T15:04:05.000Z"),
		Reason:       reason,
	}
}

// sortSignals sorts by specified field
func (s *Service) sortSignals(signals []SignalResponse, sortBy string) {
	switch sortBy {
	case "generatedAt":
		sort.Slice(signals, func(i, j int) bool {
			return signals[i].GeneratedAt > signals[j].GeneratedAt
		})
	case "score":
		fallthrough
	default:
		sort.Slice(signals, func(i, j int) bool {
			return signals[i].Score > signals[j].Score
		})
	}
}

// getStockName returns full company name for symbol
func (s *Service) getStockName(symbol string) string {
	names := map[string]string{
		"VCB": "Vietcombank",
		"VNM": "Vinamilk",
		"FPT": "FPT Corporation",
		"VIC": "Vingroup",
		"HPG": "Hoa Phat Group",
	}
	
	if name, ok := names[symbol]; ok {
		return name
	}
	return symbol
}
