package handler

import (
	"encoding/json"
	"net/http"

	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/internal/service/position"
)

// PositionHandler handles position endpoints
type PositionHandler struct {
	svc *position.Service
}

// NewPositionHandler creates a new position handler
func NewPositionHandler(svc *position.Service) *PositionHandler {
	return &PositionHandler{svc: svc}
}

// GetActivePositions handles GET /api/positions/active
func (h *PositionHandler) GetActivePositions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Use default userID for development (no auth)
	userID := int64(1)

	positions, err := h.svc.GetActivePositions(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Int64("userID", userID).Msg("Failed to get positions")
		http.Error(w, `{"error":"Failed to retrieve positions"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"positions": positions,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetSummary handles GET /api/positions/summary
func (h *PositionHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Use default userID for development (no auth)
	userID := int64(1)

	summary, err := h.svc.GetPortfolioSummary(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Int64("userID", userID).Msg("Failed to get portfolio summary")
		http.Error(w, `{"error":"Failed to retrieve portfolio summary"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}
