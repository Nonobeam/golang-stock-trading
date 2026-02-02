package jwt

import (
	"context"
	"fmt"

	"github.com/nonobeam/golang-stock-trading/internal/api"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

// Service handles JWT token operations
type Service struct {
	dnseAuth *api.DNSEAuthService
}

// NewService creates a new JWT token service
func NewService(dnseAuth *api.DNSEAuthService) *Service {
	return &Service{
		dnseAuth: dnseAuth,
	}
}

// GetOrFetch gets JWT token from DNSE auth
func (s *Service) GetOrFetch(ctx context.Context) (string, error) {
	// Fetch from DNSE auth service (it handles in-memory caching)
	if s.dnseAuth == nil {
		return "", fmt.Errorf("DNSE auth service not configured")
	}

	token, err := s.dnseAuth.GetToken()
	if err != nil {
		return "", fmt.Errorf("failed to get JWT token from DNSE: %w", err)
	}

	return token, nil
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

	token, err := s.dnseAuth.GetToken()
	if err != nil {
		return "", fmt.Errorf("failed to get refreshed JWT token: %w", err)
	}

	return token, nil
}
