package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/internal/service/signal"
)

type SignalHandler struct {
	svc *signal.Service
}

func NewSignalHandler(svc *signal.Service) *SignalHandler {
	return &SignalHandler{svc: svc}
}

func (h *SignalHandler) GetSignals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	userID := int64(1)

	params := signal.QueryParams{
		Limit:    10,
		Sort:     "score",
		Type:     r.URL.Query().Get("type"),
		Strength: r.URL.Query().Get("strength"),
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			params.Limit = limit
		}
	}

	if sortBy := r.URL.Query().Get("sort"); sortBy != "" {
		params.Sort = sortBy
	}

	signals, count, err := h.svc.GetSignals(ctx, userID, params)
	if err != nil {
		logger.Error().Err(err).Int64("userID", userID).Msg("Failed to get signals")
		http.Error(w, `{"error":"Failed to retrieve signals"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"signals": signals,
		"count":   count,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
