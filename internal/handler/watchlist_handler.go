package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/internal/service/watchlist"
)

type WatchlistHandler struct {
	svc *watchlist.Service
}

func NewWatchlistHandler(svc *watchlist.Service) *WatchlistHandler {
	return &WatchlistHandler{svc: svc}
}

func (h *WatchlistHandler) GetWatchlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	userID := int64(1)

	items, err := h.svc.GetWatchlist(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Int64("userID", userID).Msg("Failed to get watchlist")
		http.Error(w, `{"error":"Failed to retrieve watchlist"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"items": items,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *WatchlistHandler) AddToWatchlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	userID := int64(1)

	var req struct {
		Symbol string `json:"symbol"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Symbol == "" {
		http.Error(w, `{"error":"Symbol is required"}`, http.StatusBadRequest)
		return
	}

	err := h.svc.AddToWatchlist(ctx, userID, req.Symbol)
	if err != nil {
		logger.Error().Err(err).Int64("userID", userID).Str("symbol", req.Symbol).Msg("Failed to add to watchlist")
		http.Error(w, `{"error":"Failed to add to watchlist"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"symbol":     req.Symbol,
		"message":    "Added to watchlist successfully",
		"isFavorite": false,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *WatchlistHandler) RemoveFromWatchlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	symbol := vars["symbol"]
	
	userID := int64(1)

	err := h.svc.RemoveFromWatchlist(ctx, userID, symbol)
	if err != nil {
		logger.Error().Err(err).Int64("userID", userID).Str("symbol", symbol).Msg("Failed to remove from watchlist")
		http.Error(w, `{"error":"Failed to remove from watchlist"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"symbol":  symbol,
		"message": "Removed from watchlist successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *WatchlistHandler) ToggleFavorite(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	symbol := vars["symbol"]
	
	userID := int64(1)

	var req struct {
		IsFavorite bool `json:"isFavorite"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	err := h.svc.ToggleFavorite(ctx, userID, symbol, req.IsFavorite)
	if err != nil {
		logger.Error().Err(err).Int64("userID", userID).Str("symbol", symbol).Msg("Failed to toggle favorite")
		http.Error(w, `{"error":"Failed to update favorite status"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"symbol":     symbol,
		"isFavorite": req.IsFavorite,
		"message":    "Favorite status updated successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
