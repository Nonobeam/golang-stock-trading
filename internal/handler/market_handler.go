package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/internal/service/market"
)

// MarketHandler handles market data endpoints
type MarketHandler struct {
	svc           *market.Service
	universeRepo  *repository.StockUniverseRepository
}

// NewMarketHandler creates a new market handler
func NewMarketHandler(svc *market.Service, db *sql.DB) *MarketHandler {
	return &MarketHandler{
		svc:          svc,
		universeRepo: repository.NewStockUniverseRepository(db),
	}
}

// GetIndices handles GET /api/market/indices
func (h *MarketHandler) GetIndices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	indices, err := h.svc.GetAllIndices(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get market indices")
		http.Error(w, `{"error":"Failed to retrieve market indices"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(indices)
}

// GetIndexHistory handles GET /api/market/indices/{indexKey}/history
func (h *MarketHandler) GetIndexHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	indexKey := vars["indexKey"]

	// Parse query parameters
	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "1m"
	}

	limit := 50 // default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		var l int
		if _, err := fmt.Sscanf(limitStr, "%d", &l); err == nil && l > 0 {
			limit = l
		}
	}

	history, err := h.svc.GetIndexHistory(ctx, indexKey, interval, limit)
	if err != nil {
		logger.Error().Err(err).Str("indexKey", indexKey).Msg("Failed to get index history")
		http.Error(w, `{"error":"Failed to retrieve index history"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// GetMarketRegime handles GET /api/market/regime
func (h *MarketHandler) GetMarketRegime(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	regime, err := h.svc.GetMarketRegime(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get market regime")
		http.Error(w, `{"error":"Failed to retrieve market regime"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(regime)
}

// GetQuote handles GET /api/market/quote/{symbol}
func (h *MarketHandler) GetQuote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	symbol := vars["symbol"]

	// TODO: Fetch real quote from DNSE API
	// For now, return demo data
	quote := map[string]interface{}{
		"symbol":        symbol,
		"price":         87500.0,
		"open":          86000.0,
		"high":          88200.0,
		"low":           85800.0,
		"volume":        1250000,
		"change":        1500.0,
		"changePercent": 1.74,
		"ceiling":       92880.0,
		"floor":         79920.0,
		"lastUpdate":    "2026-01-14T15:30:00Z",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quote)
}

// GetUniverse handles GET /api/market/universe — returns active stock symbols from stock_universe table
func (h *MarketHandler) GetUniverse(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	symbols, err := h.universeRepo.GetActiveSymbols(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get stock universe")
		http.Error(w, `{"error":"Failed to retrieve stock universe"}`, http.StatusInternalServerError)
		return
	}

	if symbols == nil {
		symbols = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(symbols)
}
