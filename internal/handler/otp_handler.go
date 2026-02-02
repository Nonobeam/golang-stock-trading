package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/internal/service/otp"
)

type OTPHandler struct {
	otpService *otp.Service
}

func NewOTPHandler(otpService *otp.Service) *OTPHandler {
	return &OTPHandler{otpService: otpService}
}

// GetOTP handles GET /api/otp
func (h *OTPHandler) GetOTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	otpValue, err := h.otpService.Get(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get OTP")
		http.Error(w, `{"error":"Failed to get OTP"}`, http.StatusInternalServerError)
		return
	}

	exists := otpValue != ""
	var ttl time.Duration
	var expiresAt *time.Time

	if exists {
		ttl, err = h.otpService.GetTTL(ctx)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get OTP TTL")
		} else if ttl > 0 {
			exp := time.Now().Add(ttl)
			expiresAt = &exp
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"exists":      exists,
		"otp":         otpValue,
		"ttl_seconds": int(ttl.Seconds()),
		"expires_at":  expiresAt,
	})
}

// SetOTP handles POST /api/otp
func (h *OTPHandler) SetOTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		OTP string `json:"otp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate OTP format (6 digits)
	if len(req.OTP) != 6 {
		http.Error(w, `{"error":"OTP must be exactly 6 digits"}`, http.StatusBadRequest)
		return
	}

	// Validate all digits
	for _, c := range req.OTP {
		if c < '0' || c > '9' {
			http.Error(w, `{"error":"OTP must contain only digits"}`, http.StatusBadRequest)
			return
		}
	}

	// Set OTP (atomic replace)
	if err := h.otpService.Set(ctx, req.OTP); err != nil {
		logger.Error().Err(err).Msg("Failed to set OTP")
		http.Error(w, `{"error":"Failed to set OTP"}`, http.StatusInternalServerError)
		return
	}

	// Get TTL
	ttl, _ := h.otpService.GetTTL(ctx)
	var expiresAt *time.Time
	if ttl > 0 {
		exp := time.Now().Add(ttl)
		expiresAt = &exp
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "OTP set successfully",
		"otp":         req.OTP,
		"ttl_seconds": int(ttl.Seconds()),
		"expires_at":  expiresAt,
	})
}
