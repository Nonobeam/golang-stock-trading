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

// PositionEntryResponse represents a single transaction
type PositionEntryResponse struct {
	EntryID         string  `json:"entryId"`
	Date            string  `json:"date"`
	Price           float64 `json:"price"`
	Shares          int     `json:"shares"`
	Fee             float64 `json:"fee"`
	TransactionType string  `json:"type"`
	Value           float64 `json:"value"`
}

// PositionDetailsResponse includes entries history
type PositionDetailsResponse struct {
	PositionResponse
	Entries []PositionEntryResponse `json:"entries"`
}

// AddEntryRequest represents a request to buy more shares
type AddEntryRequest struct {
	UserID     int64   `json:"userId"`
	Symbol     string  `json:"symbol"`
	Price      float64 `json:"price"`
	Shares     int     `json:"shares"`
	Type       string  `json:"type"` // BUY_NEW or BUY_MORE
	Date       time.Time `json:"date"`
}

// CreatePositionRequest represents a request to open a new position
type CreatePositionRequest struct {
	UserID    int64     `json:"userId"`
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Shares    int       `json:"shares"`
	Date      time.Time `json:"date"`
	StopLoss  float64   `json:"stopLoss"`
	Target1   *float64  `json:"target1"`
	Target2   *float64  `json:"target2"`
	Target3   *float64  `json:"target3"`
	SignalType *string  `json:"signalType"`
	Score      *int     `json:"score"`
	Notes      *string  `json:"notes"`
}

// Service handles position-related business logic
type Service struct {
	repo *repository.PositionRepository
}

// NewService creates a new position service
func NewService(repo *repository.PositionRepository) *Service {
	return &Service{repo: repo}
}

// CreatePosition opens a new position and records the first entry
func (s *Service) CreatePosition(ctx context.Context, req CreatePositionRequest) (*PositionResponse, error) {
	// Calculate initial fee
	entryValue := req.Price * float64(req.Shares)
	entryFee := s.calculateEntryFee(entryValue)

	// A. Create Position Record
	pos := &repository.Position{
		UserID:     req.UserID,
		Symbol:     req.Symbol,
		EntryDate:  req.Date,
		EntryPrice: req.Price,
		Quantity:   req.Shares,
		StopLoss:   req.StopLoss,
		Target1:    req.Target1,
		Target2:    req.Target2,
		Target3:    req.Target3,
		SignalType: req.SignalType,
		Score:      req.Score,
		Notes:      req.Notes,
	}

	if err := s.repo.Create(ctx, pos); err != nil {
		return nil, fmt.Errorf("failed to create position: %w", err)
	}

	// B. Create First Entry Record
	entry := &repository.PositionEntry{
		UserID:          req.UserID,
		Ticker:          req.Symbol,
		EntryDate:       req.Date,
		EntryPrice:      req.Price,
		SharesPurchased: req.Shares,
		EntryFeePaid:    entryFee,
		TransactionType: "BUY_NEW",
	}

	if err := s.repo.CreateEntry(ctx, entry); err != nil {
		// Note: Position was created but entry failed. Ideally partial failure should be handled.
		// Since positions table is primary, we might just log error or return it.
		// For now, return error.
		return nil, fmt.Errorf("failed to create initial entry: %w", err)
	}

	return s.transformPosition(pos)
}

// PartialExitRequest represents a request to sell some shares
type PartialExitRequest struct {
	UserID    int64     `json:"userId"`
	Symbol    string    `json:"symbol"`
	Shares    int       `json:"shares"`
	Price     float64   `json:"price"`
	Date      time.Time `json:"date"`
}

// PartialExitResponse contains the result of a partial exit
type PartialExitResponse struct {
	SharesSold     int     `json:"sharesSold"`
	Price          float64 `json:"price"`
	Proceeds       float64 `json:"proceeds"`
	CostBasis      float64 `json:"costBasis"`
	RealizedPnL    float64 `json:"realizedPnL"`
	RemainingShares int    `json:"remainingShares"`
	EntryFeePortion float64 `json:"entryFeePortion"`
	ExitFee        float64 `json:"exitFee"`
}


// AddEntry adds shares to an existing position
func (s *Service) AddEntry(ctx context.Context, req AddEntryRequest) error {
	entryValue := req.Price * float64(req.Shares)
	entryFee := s.calculateEntryFee(entryValue)

	entry := &repository.PositionEntry{
		UserID:          req.UserID,
		Ticker:          req.Symbol,
		EntryDate:       req.Date,
		EntryPrice:      req.Price,
		SharesPurchased: req.Shares,
		EntryFeePaid:    entryFee,
		TransactionType: req.Type, // usually BUY_MORE
	}

	if err := s.repo.CreateEntry(ctx, entry); err != nil {
		return fmt.Errorf("failed to add entry: %w", err)
	}

	return nil
}

// PartialExit processes a partial sale of a position
func (s *Service) PartialExit(ctx context.Context, req PartialExitRequest) (*PartialExitResponse, error) {
	// 1. Get current position
	pos, err := s.repo.GetBySymbol(ctx, req.UserID, req.Symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get position: %w", err)
	}

	if pos.Quantity < req.Shares {
		return nil, fmt.Errorf("insufficient shares: have %d, trying to sell %d", pos.Quantity, req.Shares)
	}

	// 2. Calculate proportional data
	// Proportional Entry Fee = (shares_sold / total_shares) * total_fees_paid
	ratio := float64(req.Shares) / float64(pos.Quantity)
	proportionalEntryFee := ratio * pos.TotalFeesPaid

	// Exit Transaction Cost (Commission + Tax) on the SOLD amount
	exitValue := req.Price * float64(req.Shares)
	exitCost := s.calculateExitTransactionCost(exitValue)

	// Cost Basis of sold shares (using Weighted Average Price)
	costBasis := pos.EntryPrice * float64(req.Shares)

	// Realized P&L
	// P&L = Proceeds - CostBasis - ProportionalEntryFee - ExitCost
	realizedPnL := exitValue - costBasis - proportionalEntryFee - exitCost

	// 3. Update Position State in DB
	newQuantity := pos.Quantity - req.Shares
	newTotalFees := pos.TotalFeesPaid - proportionalEntryFee

	if err := s.repo.PartialExit(ctx, pos.ID, newQuantity, newTotalFees); err != nil {
		return nil, fmt.Errorf("failed to update position: %w", err)
	}

	return &PartialExitResponse{
		SharesSold:      req.Shares,
		Price:           req.Price,
		Proceeds:        exitValue,
		CostBasis:       costBasis,
		RealizedPnL:     realizedPnL,
		RemainingShares: newQuantity,
		EntryFeePortion: proportionalEntryFee,
		ExitFee:         exitCost,
	}, nil
}

// GetPositionDetails retrieves full position details including entries
func (s *Service) GetPositionDetails(ctx context.Context, userID int64, symbol string) (*PositionDetailsResponse, error) {
	// 1. Get fundamental position data
	pos, err := s.repo.GetBySymbol(ctx, userID, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get position: %w", err)
	}

	baseResponse, err := s.transformPosition(pos)
	if err != nil {
		return nil, err
	}

	// 2. Get entries history
	entries, err := s.repo.GetEntries(ctx, userID, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get entries: %w", err)
	}

	entryResponses := make([]PositionEntryResponse, len(entries))
	for i, e := range entries {
		entryResponses[i] = PositionEntryResponse{
			EntryID:         e.EntryID,
			Date:            e.EntryDate.Format(time.RFC3339),
			Price:           e.EntryPrice,
			Shares:          e.SharesPurchased,
			Fee:             e.EntryFeePaid,
			TransactionType: e.TransactionType,
			Value:           e.EntryPrice * float64(e.SharesPurchased),
		}
	}

	return &PositionDetailsResponse{
		PositionResponse: *baseResponse,
		Entries:          entryResponses,
	}, nil
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

// calculateEntryFee estimates entry fee only (0.15%)
func (s *Service) calculateEntryFee(entryValue float64) float64 {
	return entryValue * 0.0015
}

// calculateExitTransactionCost estimates exit costs (0.15% comm + 0.1% tax)
func (s *Service) calculateExitTransactionCost(exitValue float64) float64 {
	return exitValue * 0.0025
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

