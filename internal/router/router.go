package router

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nonobeam/golang-stock-trading/internal/config"
	"github.com/nonobeam/golang-stock-trading/internal/handler"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/internal/middleware"
)

type HandlerDeps struct{
	MarketHandler         *handler.MarketHandler
	AccountHandler        *handler.AccountHandler
	PositionHandler       *handler.PositionHandler
	SignalHandler         *handler.SignalHandler
	WatchlistHandler      *handler.WatchlistHandler
	RecommendationHandler *handler.RecommendationHandler
	StockPrefHandler      *handler.StockPrefHandler
	OTPHandler            *handler.OTPHandler
	JWTHandler            *handler.JWTHandler
}

func NewRouter(deps HandlerDeps, cfg *config.Config) *mux.Router {
	r := mux.NewRouter()

	r.Use(middleware.LoggerMiddleware)
	r.Use(middleware.CORSMiddleware(cfg.CORSOrigins))

	api := r.PathPrefix("/api").Subrouter()

	api.HandleFunc("/health", HealthCheck).Methods("GET")

	api.HandleFunc("/market/indices", deps.MarketHandler.GetIndices).Methods("GET", "OPTIONS")
	api.HandleFunc("/market/indices/{indexKey}/history", deps.MarketHandler.GetIndexHistory).Methods("GET", "OPTIONS")
	api.HandleFunc("/market/regime", deps.MarketHandler.GetMarketRegime).Methods("GET", "OPTIONS")
	api.HandleFunc("/market/quote/{symbol}", deps.MarketHandler.GetQuote).Methods("GET", "OPTIONS")

	api.HandleFunc("/account/info", deps.AccountHandler.GetAccountInfo).Methods("GET", "OPTIONS")
	api.HandleFunc("/account/summary", deps.AccountHandler.GetAccountSummary).Methods("GET", "OPTIONS")

	api.HandleFunc("/positions/active", deps.PositionHandler.GetActivePositions).Methods("GET", "OPTIONS")
	api.HandleFunc("/positions/summary", deps.PositionHandler.GetSummary).Methods("GET", "OPTIONS")

	api.HandleFunc("/signals", deps.SignalHandler.GetSignals).Methods("GET", "OPTIONS")

	api.HandleFunc("/watchlist", deps.WatchlistHandler.GetWatchlist).Methods("GET", "OPTIONS")
	api.HandleFunc("/watchlist", deps.WatchlistHandler.AddToWatchlist).Methods("POST", "OPTIONS")
	api.HandleFunc("/watchlist/{symbol}", deps.WatchlistHandler.RemoveFromWatchlist).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/watchlist/{symbol}/favorite", deps.WatchlistHandler.ToggleFavorite).Methods("PATCH", "OPTIONS")

	api.HandleFunc("/recommendations", deps.RecommendationHandler.GetRecommendation).Methods("POST", "OPTIONS")

	// Stock signal preferences
	api.HandleFunc("/preferences/stocks", deps.StockPrefHandler.GetAllPreferences).Methods("GET", "OPTIONS")
	api.HandleFunc("/preferences/stocks/{symbol}", deps.StockPrefHandler.GetPreference).Methods("GET", "OPTIONS")
	api.HandleFunc("/preferences/stocks/{symbol}", deps.StockPrefHandler.SetPreference).Methods("PUT", "OPTIONS")
	api.HandleFunc("/preferences/stocks/{symbol}", deps.StockPrefHandler.DeletePreference).Methods("DELETE", "OPTIONS")

	// OTP management
	api.HandleFunc("/otp", deps.OTPHandler.GetOTP).Methods("GET", "OPTIONS")
	api.HandleFunc("/otp", deps.OTPHandler.SetOTP).Methods("POST", "OPTIONS")

	// JWT token
	api.HandleFunc("/jwt-token", deps.JWTHandler.GetJWTToken).Methods("GET", "OPTIONS")

	logger.Info().Msg("Router configured with all endpoints")

	return r
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","service":"dashboard-api"}`))
}
