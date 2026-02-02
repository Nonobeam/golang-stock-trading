package handler

import (
	"encoding/json"
	"net/http"

	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/internal/service/account"
)

type AccountHandler struct {
	svc *account.Service
}

func NewAccountHandler(svc *account.Service) *AccountHandler {
	return &AccountHandler{svc: svc}
}

func (h *AccountHandler) GetAccountInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	userID := int64(1)

	info, err := h.svc.GetAccountInfo(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Int64("userID", userID).Msg("Failed to get account info")
		http.Error(w, `{"error":"Failed to retrieve account info"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (h *AccountHandler) GetAccountSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	userID := int64(1)

	summary, err := h.svc.GetAccountSummary(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Int64("userID", userID).Msg("Failed to get account summary")
		http.Error(w, `{"error":"Failed to retrieve account summary"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}
