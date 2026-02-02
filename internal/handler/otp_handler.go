package handler

import (
	"encoding/json"
	"net/http"

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
	// OTP caching is removed, so we always return "not found" or dummy
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"exists":      false,
		"otp":         "",
		"ttl_seconds": 0,
		"expires_at":  nil,
	})
}

// SetOTP handles POST /api/otp
func (h *OTPHandler) SetOTP(w http.ResponseWriter, r *http.Request) {
	// We no longer support setting OTP via API for caching
	// Return Method Not Allowed or just dummy success to satisfy frontend?
	// Let's return 404/Not Supported to indicate API change
	http.Error(w, `{"error":"OTP management via API is disabled"}`, http.StatusNotImplemented)
}
