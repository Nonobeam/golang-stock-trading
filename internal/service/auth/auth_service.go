package auth

import (
	"encoding/json"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/config"
	"github.com/nonobeam/golang-stock-trading/internal/errors"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/pkg/httpclient"
)

type AuthService struct {
	client       *httpclient.Client
	accessToken  string
	tradingToken string
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken string   `json:"token"`
	Roles       []string `json:"roles"`
	IsNeedReset bool     `json:"isNeedReset"`
}

type TradingTokenResponse struct {
	TradingToken string `json:"tradingToken"`
	ExpiresIn    int    `json:"expiresIn"`
}

func NewAuthService(cfg *config.Config) *AuthService {
	client := httpclient.New(cfg.DnseApiBaseUrl, 30*time.Second)
	return &AuthService{
		client: client,
	}
}

func (s *AuthService) Login(username, password string) (*LoginResponse, error) {
	logger.Info().Str("username", username).Msg("Attempting login")

	reqBody := LoginRequest{
		Username: username,
		Password: password,
	}

	respBody, err := s.client.Post("/auth-service/login", reqBody, nil)
	if err != nil {
		logger.Error().Err(err).Msg("Login failed")
		return nil, errors.Wrap(err, errors.ErrInvalidCredentials)
	}

	var resp LoginResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, errors.Wrap(err, errors.ErrInvalidResponse)
	}

	s.accessToken = resp.AccessToken
	s.client.SetToken(s.accessToken)

	logger.Info().Msg("Login successful")
	return &resp, nil
}

func (s *AuthService) GetTradingToken(otp string) (*TradingTokenResponse, error) {
	logger.Info().Msg("Exchanging Smart OTP for trading token")

	headers := map[string]string{
		"smart-otp": otp,
	}

	respBody, err := s.client.Post("/order-service/trading-token", nil, headers)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get trading token from API")
		return nil, errors.Wrap(err, errors.ErrInvalidTradingToken)
	}

	var resp TradingTokenResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, errors.Wrap(err, errors.ErrInvalidResponse)
	}

	s.tradingToken = resp.TradingToken
	logger.Info().Int("expiresIn", resp.ExpiresIn).Msg("Trading token received successfully")
	return &resp, nil
}

func (s *AuthService) SetTradingToken(token string) {
	s.tradingToken = token
	logger.Info().Msg("Trading token set successfully")
}

func (s *AuthService) GetAccessToken() string {
	return s.accessToken
}

func (s *AuthService) GetTradingTokenValue() string {
	return s.tradingToken
}

func (s *AuthService) GetClient() *httpclient.Client {
	return s.client
}
