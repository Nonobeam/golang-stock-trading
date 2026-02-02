package errors

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code       string
	Message    string
	HttpStatus int
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HttpStatus: httpStatus,
	}
}

func Wrap(err error, appErr *AppError) *AppError {
	return &AppError{
		Code:       appErr.Code,
		Message:    appErr.Message,
		HttpStatus: appErr.HttpStatus,
		Err:        err,
	}
}

var (
	ErrInvalidCredentials = New("AUTH001", "Invalid username or password", http.StatusUnauthorized)
	ErrUnauthorized       = New("AUTH002", "Unauthorized access", http.StatusUnauthorized)
	ErrTokenExpired       = New("AUTH003", "Token has expired", http.StatusUnauthorized)
	ErrOtpExpired         = New("AUTH004", "OTP has expired", http.StatusBadRequest)
	ErrOtpInvalid         = New("AUTH005", "Invalid OTP", http.StatusBadRequest)
)

var (
	ErrInternalServer = New("SYS001", "Internal server error", http.StatusInternalServerError)
	ErrBadRequest     = New("SYS002", "Bad request", http.StatusBadRequest)
	ErrTimeout        = New("SYS003", "Request timeout", http.StatusRequestTimeout)
)

// Notification errors
var (
	ErrEmailSendFailed       = New("NOTIF001", "Failed to send email", http.StatusInternalServerError)
	ErrTelegramConnection    = New("NOTIF002", "Failed to connect to Telegram", http.StatusInternalServerError)
	ErrTradingTokenTimeout   = New("NOTIF003", "Trading token request timed out", http.StatusRequestTimeout)
	ErrInvalidTradingToken   = New("NOTIF004", "Invalid trading token format", http.StatusBadRequest)
)

var (
	ErrApiCallFailed     = New("API001", "API call failed", http.StatusBadGateway)
	ErrInvalidResponse   = New("API002", "Invalid API response", http.StatusBadGateway)
	ErrTradingTokenError = New("API003", "Failed to get trading token", http.StatusBadGateway)
)

var (
	ErrWsConnectionFailed = New("WS001", "WebSocket connection failed", http.StatusBadGateway)
	ErrWsSubscribeFailed  = New("WS002", "Failed to subscribe to topic", http.StatusBadGateway)
)
