package recommendation

import (
	"context"
	"fmt"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	mlconfig "github.com/nonobeam/golang-stock-trading/internal/ml/config"
	"github.com/nonobeam/golang-stock-trading/internal/service/market"
	"github.com/nonobeam/golang-stock-trading/proto/ml"
)

// RecommendationRequest contains context for generating recommendation
type RecommendationRequest struct {
	IncludePortfolio     bool `json:"portfolio"`
	IncludeMarketRegime  bool `json:"marketRegime"`
	IncludeSignals       bool `json:"signals"`
}

// RecommendationResponse contains AI-generated trading recommendation
type RecommendationResponse struct {
	Symbol       string  `json:"symbol"`
	Action       string  `json:"action"` // buy, sell, hold
	Confidence   int     `json:"confidence"` // 0-100
	Rationale    string  `json:"rationale"`
	TargetPrice  float64 `json:"targetPrice"`
	StopLoss     float64 `json:"stopLoss"`
	Timeframe    string  `json:"timeframe"` // short-term, medium-term, long-term
	GeneratedAt  string  `json:"generatedAt"`
}

// Service handles AI recommendation generation
type Service struct {
	signalRepo     *repository.SignalHistoryRepository
	positionRepo   *repository.PositionRepository
	marketService  *market.Service
	mlClient       ml.MLPredictionServiceClient // ML service for predictions
}

// NewService creates a new recommendation service
func NewService(
	signalRepo *repository.SignalHistoryRepository,
	positionRepo *repository.PositionRepository,
	marketService *market.Service,
	mlClient ml.MLPredictionServiceClient,
) *Service {
	return &Service{
		signalRepo:     signalRepo,
		positionRepo:   positionRepo,
		marketService:  marketService,
		mlClient:       mlClient,
	}
}

// GenerateRecommendation creates an AI-powered trading recommendation
func (s *Service) GenerateRecommendation(ctx context.Context, userID int64, req RecommendationRequest) (*RecommendationResponse, error) {
	logger.Info().Int64("userID", userID).Msg("Generating recommendation")

	// Get portfolio context if requested
	var positionCount int
	var portfolioRisk string
	if req.IncludePortfolio {
		positions, err := s.positionRepo.GetOpenPositions(ctx, userID)
		if err == nil {
			positionCount = len(positions)
			if positionCount >= 5 {
				portfolioRisk = "high concentration"
			} else {
				portfolioRisk = "moderate exposure"
			}
		}
	}

	// Get market regime if requested  
	var regimeName string
	if req.IncludeMarketRegime && s.marketService != nil {
		regime, err := s.marketService.GetMarketRegime(ctx)
		if err == nil {
			regimeName = regime.Regime
		} else {
			regimeName = "choppy"
		}
	} else {
		regimeName = "trending-up"
	}

	// Get top signals if requested
	var topSignal *repository.SignalHistory
	if req.IncludeSignals {
		signals, err := s.signalRepo.GetRecent(ctx, &userID, 7, 7)
		if err == nil && len(signals) > 0 {
			topSignal = signals[0]
		}
	}

	// Generate recommendation based on context
	var symbol string
	var action string
	var confidence int
	var rationale string
	var targetPrice float64
	var stopLoss float64
	var timeframe string

	// If we have a top signal, use it
	if topSignal != nil {
		symbol = topSignal.Symbol
		
		// Default values from signal
		action = "buy"
		confidence = calculateConfidence(topSignal.Score, regimeName, positionCount)
		rationale = fmt.Sprintf("Strong %s signal (score %d) detected. ", topSignal.SignalType, topSignal.Score)
		targetPrice = topSignal.EntryPrice * 1.08 // Default 8% target
		stopLoss = topSignal.StopLoss
		timeframe = "short-term"

		// Use ML prediction if available
		if s.mlClient != nil {
			today := time.Now().Format("2006-01-02")
			// Call ML service with proper timeout
			mlCtx, cancel := context.WithTimeout(ctx, mlconfig.RecommendationMLTimeout)
			defer cancel()

			pred, err := s.mlClient.Predict(mlCtx, &ml.PredictRequest{
				Ticker: symbol,
				Date:   today,
			})
			
			if err == nil && pred.Success {
				// 1. Determine action based on p50 (expected return)
				if pred.P50 > mlconfig.BuyThreshold {
					action = "buy"
					rationale += fmt.Sprintf("ML model confirms positive outlook (+%.1f%% return). ", pred.P50*100)
				} else if pred.P50 > 0 {
					action = "hold"
					rationale += fmt.Sprintf("ML model suggests caution (only +%.1f%% return). ", pred.P50*100)
				} else {
					action = "hold" // Don't recommend buy if ML is negative
					rationale += fmt.Sprintf("ML model predicts negative movement (%.1f%%). ", pred.P50*100)
				}

				// 2. Adjust target price based on ML (with validation)
				newTarget := topSignal.EntryPrice * (1 + pred.P90)
				if newTarget > mlconfig.MinPrice {
					targetPrice = newTarget
				} else {
					rationale += "ML target invalid. Keeping technical target. "
				}

				// 3. Adjust confidence based on uncertainty (p90 - p10)
				uncertainty := pred.P90 - pred.P10
				if uncertainty > mlconfig.HighUncertainty {
					confidence -= 15 // High uncertainty penalty
					rationale += "High ML model uncertainty. "
				} else if uncertainty < 0.05 {
					confidence += 10 // Low uncertainty bonus
				}

				// Bounds Check
				if confidence > mlconfig.MaxConfidence {
					confidence = mlconfig.MaxConfidence
				}
				if confidence < mlconfig.MinConfidence {
					confidence = mlconfig.MinConfidence
				}
				
				rationale += fmt.Sprintf("ML confidence: %.0f%%. ", (1.0-(uncertainty/2.0))*100)
			} else {
				rationale += "ML prediction unavailable. "
				if err != nil {
					logger.Error().Err(err).Str("symbol", symbol).Msg("ML prediction failed during recommendation")
				}
			}
		}

		if regimeName != "" {
			rationale += fmt.Sprintf("Market regime is %s. ", regimeName)
		}
		if positionCount > 0 {
			rationale += fmt.Sprintf("Portfolio risk: %s. ", portfolioRisk)
		}
		
	} else {
		// Default recommendation when no signals
		symbol = "VNM"
		action = "hold"
		confidence = 55
		rationale = "No strong signals detected currently. "
		if regimeName == "choppy" || regimeName == "volatile" {
			rationale += "Market is in " + regimeName + " regime, suggesting caution. "
		}
		rationale += "Recommend holding current positions and waiting for clearer setups."
		targetPrice = 92500
		stopLoss = 86000
		timeframe = "medium-term"
	}

	return &RecommendationResponse{
		Symbol:      symbol,
		Action:      action,
		Confidence:  confidence,
		Rationale:   rationale,
		TargetPrice: targetPrice,
		StopLoss:    stopLoss,
		Timeframe:   timeframe,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}, nil
}

// calculateConfidence determines confidence score based on multiple factors
func calculateConfidence(signalScore int, regime string, positions int) int {
	confidence := signalScore * 10 // Base from signal score (70-100)

	// Adjust for regime
	if regime == "trending-up" {
		confidence += 5
	} else if regime == "choppy" || regime == "volatile" {
		confidence -= 10
	}

	// Adjust for portfolio concentration
	if positions >= 5 {
		confidence -= 5 // Too many positions
	}

	// Cap at 0-100
	if confidence > 100 {
		confidence = 100
	}
	if confidence < 0 {
		confidence = 0
	}

	return confidence
}
