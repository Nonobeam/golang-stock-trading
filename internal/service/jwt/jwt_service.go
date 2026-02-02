package jwt

import (
	"context"
	"fmt"

	"github.com/nonobeam/golang-stock-trading/internal/api"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	redisclient "github.com/nonobeam/golang-stock-trading/internal/redis"
)

// Service handles JWT token operations with Redis caching
type Service struct {
	redis    *redisclient.Client
	dnseAuth *api.DNSEAuthService
}

// NewService creates a new JWT token service
func NewService(redis *redisclient.Client, dnseAuth *api.DNSEAuthService) *Service {
	return &Service{
		redis:    redis,
		dnseAuth: dnseAuth,
	}
}

// Get retrieves JWT token from Redis
func (s *Service) Get(ctx context.Context) (string, error) {
	return s.redis.GetJWTToken(ctx)
}

// Set stores JWT token in Redis (permanent - no expiration)
func (s *Service) Set(ctx context.Context, token string) error {
	logger.Info().Msg("Storing JWT token in Redis")
	return s.redis.SetJWTToken(ctx, token)
}

// GetOrFetch gets JWT token from Redis or fetches from DNSE auth
func (s *Service) GetOrFetch(ctx context.Context) (string, error) {
	// Try Redis first
	token, err := s.redis.GetJWTToken(ctx)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get JWT token from Redis, will fetch from DNSE")
	}

	if token != "" {
		logger.Info().Msg("Using cached JWT token from Redis")
		return token, nil
	}

	// Fetch from DNSE auth service
	logger.Info().Msg("No JWT token in Redis, fetching from DNSE auth")
	if s.dnseAuth == nil {
		return "", fmt.Errorf("DNSE auth service not configured")
	}

	newToken, err := s.dnseAuth.GetToken()
	if err != nil {
		return "", fmt.Errorf("failed to get JWT token from DNSE: %w", err)
	}

	// Store in Redis (permanent)
	if err := s.redis.SetJWTToken(ctx, newToken); err != nil {
		logger.Error().Err(err).Msg("Failed to store JWT token in Redis")
		// Continue anyway with the token we got
	}

	return newToken, nil
}

// Refresh forces a refresh of the JWT token from DNSE
func (s *Service) Refresh(ctx context.Context) (string, error) {
	logger.Info().Msg("Forcing JWT token refresh from DNSE")
	
	if s.dnseAuth == nil {
		return "", fmt.Errorf("DNSE auth service not configured")
	}

	// Force login to get new token
	if err := s.dnseAuth.Login(); err != nil {
		return "", fmt.Errorf("failed to refresh JWT token: %w", err)
	}

	newToken, err := s.dnseAuth.GetToken()
	if err != nil {
		return "", fmt.Errorf("failed to get refreshed JWT token: %w", err)
	}

	// Store in Redis (permanent)
	if err := s.redis.SetJWTToken(ctx, newToken); err != nil {
		logger.Error().Err(err).Msg("Failed to store refreshed JWT token in Redis")
	}

	return newToken, nil
}
