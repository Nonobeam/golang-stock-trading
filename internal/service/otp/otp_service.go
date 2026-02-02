package otp

import (
	"context"
	"fmt"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/logger"
	redisclient "github.com/nonobeam/golang-stock-trading/internal/redis"
	"github.com/nonobeam/golang-stock-trading/internal/service/telegram"
)

// Service handles OTP operations with Redis caching
type Service struct {
	redis    *redisclient.Client
	telegram *telegram.BotService
}

// NewService creates a new OTP service
func NewService(redis *redisclient.Client, telegram *telegram.BotService) *Service {
	return &Service{
		redis:    redis,
		telegram: telegram,
	}
}

// Get retrieves OTP from Redis
func (s *Service) Get(ctx context.Context) (string, error) {
	return s.redis.GetOTP(ctx)
}

// Set stores OTP in Redis (replaces existing)
func (s *Service) Set(ctx context.Context, otp string) error {
	logger.Info().Str("otp", otp).Msg("Storing OTP in Redis")
	return s.redis.SetOTP(ctx, otp)
}

// Delete removes OTP from Redis
func (s *Service) Delete(ctx context.Context) error {
	logger.Info().Msg("Deleting OTP from Redis")
	return s.redis.DeleteOTP(ctx)
}

// GetTTL gets remaining TTL for OTP
func (s *Service) GetTTL(ctx context.Context) (time.Duration, error) {
	return s.redis.GetOTPTTL(ctx)
}

// GetOrRequest gets OTP from Redis or requests new one from Telegram
func (s *Service) GetOrRequest(ctx context.Context, timeout time.Duration) (string, error) {
	// Try Redis first
	otp, err := s.redis.GetOTP(ctx)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get OTP from Redis, will request from Telegram")
	}
	
	if otp != "" {
		logger.Info().Msg("Using cached OTP from Redis")
		return otp, nil
	}

	// Request from Telegram
	logger.Info().Msg("No OTP in Redis, requesting from Telegram")
	if s.telegram == nil {
		return "", fmt.Errorf("Telegram bot not configured")
	}

	newOTP, err := s.telegram.RequestSmartOTP(timeout)
	if err != nil {
		return "", fmt.Errorf("failed to get OTP from Telegram: %w", err)
	}

	// Store in Redis
	if err := s.redis.SetOTP(ctx, newOTP); err != nil {
		logger.Error().Err(err).Msg("Failed to store OTP in Redis")
		// Continue anyway with the OTP we got
	}

	return newOTP, nil
}

// RequestNew deletes old OTP and requests new one from Telegram
func (s *Service) RequestNew(ctx context.Context, timeout time.Duration, reason string) (string, error) {
	// Delete old OTP
	if err := s.redis.DeleteOTP(ctx); err != nil {
		logger.Warn().Err(err).Msg("Failed to delete old OTP")
	}

	// Notify Telegram
	if s.telegram != nil {
		msg := fmt.Sprintf("⚠️ OTP Request\n\nReason: %s\n\nPlease send new 6-digit OTP.", reason)
		s.telegram.SendMessage(msg)
	}

	// Request new OTP
	return s.GetOrRequest(ctx, timeout)
}
