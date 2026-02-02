package position

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

// PositionResponse represents a position for Dashboard API
type PositionResponse struct {
	ID            string  `json:"id"`
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Exchange      string  `json:"exchange"`
	Shares        int     `json:"shares"`
	EntryPrice    float64 `json:"entryPrice"`
	CurrentPrice  float64 `json:"currentPrice"`
	EntryDate     string  `json:"entryDate"`
	StopPrice     float64 `json:"stopPrice"`
	TargetPrice   float64 `json:"targetPrice"`
	EntryValue    float64 `json:"entryValue"`
	CurrentValue  float64 `json:"currentValue"`
	GrossPnL      float64 `json:"grossPnL"`
	NetPnL        float64 `json:"netPnL"`
	NetPnLPercent float64 `json:"netPnLPercent"`
	RMultiple     float64 `json:"rMultiple"`
	Risk          float64 `json:"risk"`
	Status        string  `json:"status"` // green, yellow, red
	DaysHeld      int     `json:"daysHeld"`
}

// PortfolioSummaryResponse represents portfolio summary metrics
type PortfolioSummaryResponse struct {
	TotalPositions int     `json:"totalPositions"`
	TotalValue     float64 `json:"totalValue"`
	TotalPnL       float64 `json:"totalPnL"`
	TotalPnLPercent float64 `json:"totalPnLPercent"`
	AvgRMultiple   float64 `json:"avgRMultiple"`
	TotalRisk      float64 `json:"totalRisk"`
	RiskPercent    float64 `json:"riskPercent"`
}

// Service handles position-related business logic
type Service struct {
	repo *repository.PositionRepository
}

// NewService creates a new position service
func NewService(repo *repository.PositionRepository) *Service {
	return &Service{repo: repo}
}

// GetActivePositions retrieves all active positions for a user
func (s *Service) GetActivePositions(ctx context.Context, userID int64) ([]PositionResponse, error) {
	positions, err := s.repo.GetOpenPositions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	responses := make([]PositionResponse, 0, len(positions))
	for _, pos := range positions {
		response, err := s.transformPosition(pos)
		if err != nil {
			logger.Warn().Err(err).Str("positionID", pos.ID).Msg("Failed to transform position")
			continue
		}
		responses = append(responses, *response)
	}

	return responses, nil
}

// GetPortfolioSummary calculates aggregate portfolio metrics
func (s *Service) GetPortfolioSummary(ctx context.Context, userID int64) (*PortfolioSummaryResponse, error) {
	positions, err := s.repo.GetOpenPositions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	var totalValue float64
	var totalPnL float64
	var totalRisk float64
	var sumRMultiple float64
	var countWithR int

	for _, pos := range positions {
		// Calculate current value (using demo price)
		currentPrice := pos.EntryPrice * 1.025 // +2.5% demo
		currentValue := currentPrice * float64(pos.Quantity)
		entryValue := pos.EntryPrice * float64(pos.Quantity)
		
		totalValue += currentValue
		
		// P&L
		grossPnL := currentValue - entryValue
		netPnL := grossPnL - s.calculateFees(entryValue, currentValue)
		totalPnL += netPnL

		// Risk
		if pos.StopLoss > 0 {
			riskPerShare := math.Abs(pos.EntryPrice - pos.StopLoss)
			risk := riskPerShare * float64(pos.Quantity)
			totalRisk += risk

			// R-multiple
			if riskPerShare > 0 {
				rMultiple := netPnL / risk
				sumRMultiple += rMultiple
				countWithR++
			}
		}
	}

	avgRMultiple := 0.0
	if countWithR > 0 {
		avgRMultiple = sumRMultiple / float64(countWithR)
	}

	totalPnLPercent := 0.0
	if totalValue > 0 {
		totalPnLPercent = (totalPnL / (totalValue - totalPnL)) * 100
	}

	riskPercent := 0.0
	if totalValue > 0 {
		riskPercent = (totalRisk / totalValue) * 100
	}

	return &PortfolioSummaryResponse{
		TotalPositions:  len(positions),
		TotalValue:      totalValue,
		TotalPnL:        totalPnL,
		TotalPnLPercent: totalPnLPercent,
		AvgRMultiple:    avgRMultiple,
		TotalRisk:       totalRisk,
		RiskPercent:     riskPercent,
	}, nil
}

// transformPosition converts DB position to API response format
func (s *Service) transformPosition(pos *repository.Position) (*PositionResponse, error) {
	// Get current price (would call market service or DNSE API)
	// For demo, use +2.5% from entry
	currentPrice := pos.EntryPrice * 1.025

	entryValue := pos.EntryPrice * float64(pos.Quantity)
	currentValue := currentPrice * float64(pos.Quantity)
	grossPnL := currentValue - entryValue
	netPnL := grossPnL - s.calculateFees(entryValue, currentValue)
	netPnLPercent := (netPnL / entryValue) * 100

	// Calculate R-multiple
	riskPerShare := 0.0
	if pos.StopLoss > 0 {
		riskPerShare = math.Abs(pos.EntryPrice - pos.StopLoss)
	}
	
	risk := riskPerShare * float64(pos.Quantity)
	rMultiple := 0.0
	if risk > 0 {
		rMultiple = netPnL / risk
	}

	// Determine status
	status := "green"
	if currentPrice < pos.EntryPrice {
		status = "yellow"
	}
	if pos.StopLoss > 0 && currentPrice <= pos.StopLoss {
		status = "red"
	}

	// Days held
	daysHeld := int(time.Since(pos.EntryDate).Hours() / 24)

	// Get target price (use first target if available)
	targetPrice := 0.0
	if pos.Target1 != nil && *pos.Target1 > 0 {
		targetPrice = *pos.Target1
	}

	return &PositionResponse{
		ID:            pos.ID,
		Symbol:        pos.Symbol,
		Name:          s.getStockName(pos.Symbol),
		Exchange:      s.getExchange(pos.Symbol),
		Shares:        pos.Quantity,
		EntryPrice:    pos.EntryPrice,
		CurrentPrice:  currentPrice,
		EntryDate:     pos.EntryDate.Format(time.RFC3339),
		StopPrice:     pos.StopLoss,
		TargetPrice:   targetPrice,
		EntryValue:    entryValue,
		CurrentValue:  currentValue,
		GrossPnL:      grossPnL,
		NetPnL:        netPnL,
		NetPnLPercent: netPnLPercent,
		RMultiple:     rMultiple,
		Risk:          risk,
		Status:        status,
		DaysHeld:      daysHeld,
	}, nil
}

// calculateFees estimates trading fees (commission + tax)
func (s *Service) calculateFees(entryValue, exitValue float64) float64 {
	// Vietnam: 0.15% commission each way + 0.1% tax on sell
	commission := (entryValue * 0.0015) + (exitValue * 0.0015)
	tax := exitValue * 0.001
	return commission + tax
}

// getStockName returns full company name for symbol
func (s *Service) getStockName(symbol string) string {
	// TODO: Implement symbol lookup table
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

// getExchange determines exchange from symbol
func (s *Service) getExchange(symbol string) string {
	// TODO: Implement proper exchange detection
	// For now, assume all are HOSE (most VN30 stocks)
	return "HOSE"
}
