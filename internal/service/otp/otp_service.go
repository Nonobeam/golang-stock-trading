package otp

import (
	"context"
	"fmt"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/internal/service/telegram"
)

// Service handles OTP operations
type Service struct {
	telegram *telegram.BotService
}

// NewService creates a new OTP service
func NewService(telegram *telegram.BotService) *Service {
	return &Service{
		telegram: telegram,
	}
}

// Request requests new OTP from Telegram
func (s *Service) Request(ctx context.Context, timeout time.Duration) (string, error) {
	// Request from Telegram
	logger.Info().Msg("Requesting OTP from Telegram")
	if s.telegram == nil {
		return "", fmt.Errorf("Telegram bot not configured")
	}

	newOTP, err := s.telegram.RequestSmartOTP(timeout)
	if err != nil {
		return "", fmt.Errorf("failed to get OTP from Telegram: %w", err)
	}

	return newOTP, nil
}

// RequestNew requests new one from Telegram (wrapper for Request for backward compat/clarity of intent)
func (s *Service) RequestNew(ctx context.Context, timeout time.Duration, reason string) (string, error) {
	// Notify Telegram
	if s.telegram != nil {
		msg := fmt.Sprintf("⚠️ OTP Request\n\nReason: %s\n\nPlease send new 6-digit OTP.", reason)
		s.telegram.Broadcast(msg)
	}

	// Request new OTP
	return s.Request(ctx, timeout)
}
