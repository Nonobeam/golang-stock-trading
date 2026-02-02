package handler

import (
	"encoding/json"
	"net/http"

	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/internal/service/jwt"
)

type JWTHandler struct {
	jwtService *jwt.Service
}

func NewJWTHandler(jwtService *jwt.Service) *JWTHandler {
	return &JWTHandler{jwtService: jwtService}
}

// GetJWTToken handles GET /api/jwt-token
func (h *JWTHandler) GetJWTToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tokenValue, err := h.jwtService.GetOrFetch(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get JWT token")
		http.Error(w, `{"error":"Failed to get JWT token"}`, http.StatusInternalServerError)
		return
	}

	exists := tokenValue != ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"exists": exists,
		"token":  tokenValue,
	})
}
