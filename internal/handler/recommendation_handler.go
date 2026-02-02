package handler

import (
	"encoding/json"
	"net/http"

	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/internal/service/recommendation"
)

type RecommendationHandler struct {
	svc *recommendation.Service
}

func NewRecommendationHandler(svc *recommendation.Service) *RecommendationHandler {
	return &RecommendationHandler{svc: svc}
}

func (h *RecommendationHandler) GetRecommendation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	userID := int64(1)

	var req recommendation.RecommendationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = recommendation.RecommendationRequest{
			IncludePortfolio:    true,
			IncludeMarketRegime: true,
			IncludeSignals:      true,
		}
	}

	rec, err := h.svc.GenerateRecommendation(ctx, userID, req)
	if err != nil {
		logger.Error().Err(err).Int64("userID", userID).Msg("Failed to generate recommendation")
		http.Error(w, `{"error":"Failed to generate recommendation"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rec)
}
