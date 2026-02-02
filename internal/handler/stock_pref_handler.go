package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

type StockPrefHandler struct {
	repo *repository.StockSignalPrefRepository
}

func NewStockPrefHandler(repo *repository.StockSignalPrefRepository) *StockPrefHandler {
	return &StockPrefHandler{repo: repo}
}

// GetAllPreferences handles GET /api/preferences/stocks
func (h *StockPrefHandler) GetAllPreferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := int64(1) // Default for development

	prefs, err := h.repo.GetAllByUser(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Int64("userID", userID).Msg("Failed to get stock preferences")
		http.Error(w, `{"error":"Failed to get preferences"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"preferences": prefs,
		"count":       len(prefs),
	})
}

// GetPreference handles GET /api/preferences/stocks/:symbol
func (h *StockPrefHandler) GetPreference(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	symbol := vars["symbol"]
	userID := int64(1) // Default for development

	pref, err := h.repo.GetByUserAndSymbol(ctx, userID, symbol)
	if err != nil {
		logger.Error().Err(err).Int64("userID", userID).Str("symbol", symbol).Msg("Failed to get preference")
		http.Error(w, `{"error":"Failed to get preference"}`, http.StatusInternalServerError)
		return
	}

	if pref == nil {
		http.Error(w, `{"error":"Preference not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pref)
}

// SetPreference handles PUT /api/preferences/stocks/:symbol
func (h *StockPrefHandler) SetPreference(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	symbol := vars["symbol"]
	userID := int64(1) // Default for development

	var req struct {
		MinSignalScore int     `json:"min_signal_score"`
		Notes          *string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate score range
	if req.MinSignalScore < 0 || req.MinSignalScore > 13 {
		http.Error(w, `{"error":"min_signal_score must be between 0 and 13"}`, http.StatusBadRequest)
		return
	}

	pref := &repository.StockSignalPreference{
		UserID:         userID,
		Symbol:         symbol,
		MinSignalScore: req.MinSignalScore,
		Notes:          req.Notes,
	}

	if err := h.repo.Upsert(ctx, pref); err != nil {
		logger.Error().Err(err).Int64("userID", userID).Str("symbol", symbol).Msg("Failed to upsert preference")
		http.Error(w, `{"error":"Failed to save preference"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(pref)
}

// DeletePreference handles DELETE /api/preferences/stocks/:symbol
func (h *StockPrefHandler) DeletePreference(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	symbol := vars["symbol"]
	userID := int64(1) // Default for development

	err := h.repo.Delete(ctx, userID, symbol)
	if err != nil {
		logger.Error().Err(err).Int64("userID", userID).Str("symbol", symbol).Msg("Failed to delete preference")
		http.Error(w, `{"error":"Failed to delete preference"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Preference deleted successfully",
		"symbol":  symbol,
	})
}
